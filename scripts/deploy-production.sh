#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="/opt/acgwarehouse"
FRONTEND_DIR="$ROOT_DIR/frontend/vue-gallery"
BACKEND_SERVICE="acgwarehouse.service"
FRONTEND_SERVICE="acgwarehouse-frontend.service"
BACKEND_BIN="$ROOT_DIR/bin/web"
FRONTEND_BIN="$ROOT_DIR/bin/frontend-server"
ENV_FILE="$ROOT_DIR/.env"
FRONTEND_PORT="2017"
KILL_STALE=0
DRY_RUN=0
TMP_BACKEND=""

log() {
  printf '[deploy] %s\n' "$*"
}

warn() {
  printf '[deploy][warn] %s\n' "$*" >&2
}

die() {
  printf '[deploy][error] %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'USAGE'
Usage: scripts/deploy-production.sh [--kill-stale] [--dry-run] [--help]

Pull, build, package, deploy, and verify ACGWarehouse production services.
The frontend service serves the built Vue SPA and proxies /api/* to the backend.

Options:
  --kill-stale   Terminate stale non-systemd backend processes that hold the
                 production SQLite DB or Bleve index. Default is report-only.
  --dry-run      Run checks and print planned commands without changing git,
                 rebuilding binaries, or restarting services.
  --help         Show this help.
USAGE
}

cleanup() {
  if [[ -n "$TMP_BACKEND" && -e "$TMP_BACKEND" ]]; then
    rm -f "$TMP_BACKEND"
  fi
}
trap cleanup EXIT

run() {
  if (( DRY_RUN )); then
    printf '[deploy][dry-run]'
    printf ' %q' "$@"
    printf '\n'
    return 0
  fi
  "$@"
}

run_git() {
  if (( DRY_RUN )); then
    printf '[deploy][dry-run] GIT_MASTER=1 git'
    local arg
    for arg in "$@"; do
      printf ' %q' "$arg"
    done
    printf '\n'
    return 0
  fi
  GIT_MASTER=1 git "$@"
}

parse_args() {
  while (($#)); do
    case "$1" in
      --kill-stale)
        KILL_STALE=1
        ;;
      --dry-run)
        DRY_RUN=1
        ;;
      --help|-h)
        usage
        exit 0
        ;;
      *)
        die "unknown option: $1"
        ;;
    esac
    shift
  done
}

require_command() {
  local name="$1"
  command -v "$name" >/dev/null 2>&1 || die "required command not found: $name"
}

require_commands() {
  local commands=(git go npm upx systemctl curl ss pgrep ps readlink install mktemp)
  local command_name
  for command_name in "${commands[@]}"; do
    require_command "$command_name"
  done
}

load_env_value() {
  local key="$1"
  local fallback="${2:-}"
  local value=""
  if [[ -f "$ENV_FILE" ]]; then
    value=$(awk -F= -v key="$key" '
      $0 !~ /^[[:space:]]*#/ && $1 == key {
        sub(/^[^=]*=/, "")
        gsub(/^[[:space:]]+|[[:space:]]+$/, "")
        gsub(/^"|"$/, "")
        gsub(/^'\''|'\''$/, "")
        print
        exit
      }
    ' "$ENV_FILE")
  fi
  if [[ -z "$value" ]]; then
    value="$fallback"
  fi
  printf '%s' "$value"
}

absolute_path() {
  local path="$1"
  if [[ "$path" = /* ]]; then
    printf '%s' "$path"
  else
    printf '%s/%s' "$ROOT_DIR" "$path"
  fi
}

service_main_pid() {
  local service="$1"
  systemctl show "$service" -p MainPID --value 2>/dev/null || printf '0'
}

pid_in_service_cgroup() {
  local pid="$1"
  local service="$2"
  local cgroup
  [[ -r "/proc/$pid/cgroup" ]] || return 1
  cgroup=$(cat "/proc/$pid/cgroup" 2>/dev/null || true)
  [[ "$cgroup" == *"/${service}"* ]]
}

is_allowed_port_owner() {
  local pid="$1"
  local service="$2"
  local main_pid
  main_pid=$(service_main_pid "$service")
  [[ "$main_pid" != "0" && "$pid" == "$main_pid" ]] || pid_in_service_cgroup "$pid" "$service"
}

port_owner_pids() {
  local port="$1"
  ss -ltnp "sport = :$port" 2>/dev/null \
    | grep -oE 'pid=[0-9]+' \
    | cut -d= -f2 \
    | sort -u || true
}

process_command() {
  local pid="$1"
  if [[ -r "/proc/$pid/cmdline" ]]; then
    tr '\0' ' ' < "/proc/$pid/cmdline"
  else
    ps -p "$pid" -o command= 2>/dev/null || true
  fi
}

check_port() {
  local port="$1"
  local service="$2"
  local label="$3"
  local owners=()
  local pid
  mapfile -t owners < <(port_owner_pids "$port")
  if ((${#owners[@]} == 0)); then
    log "$label port $port is free"
    return 0
  fi
  for pid in "${owners[@]}"; do
    if is_allowed_port_owner "$pid" "$service"; then
      log "$label port $port is owned by expected service $service (pid $pid)"
    else
      die "$label port $port is occupied by unexpected pid $pid: $(process_command "$pid")"
    fi
  done
}

check_frontend_service_exec() {
  local unit
  unit=$(systemctl cat "$FRONTEND_SERVICE" 2>/dev/null || true)
  [[ "$unit" == *"$FRONTEND_BIN"* ]] || die "$FRONTEND_SERVICE must ExecStart $FRONTEND_BIN before deployment"
}

fd_points_to_resource() {
  local fd_target="$1"
  local resource="$2"
  [[ "$fd_target" == "$resource" || "$fd_target" == "$resource"/* || "$fd_target" == "$resource "* || "$fd_target" == "$resource"/*" "* ]]
}

resource_owner_pids() {
  local resource="$1"
  local proc_dir fd target pid
  for proc_dir in /proc/[0-9]*; do
    pid="${proc_dir##*/}"
    [[ -d "$proc_dir/fd" ]] || continue
    for fd in "$proc_dir"/fd/*; do
      [[ -e "$fd" ]] || continue
      target=$(readlink "$fd" 2>/dev/null || true)
      if fd_points_to_resource "$target" "$resource"; then
        printf '%s\n' "$pid"
        break
      fi
    done
  done | sort -u
}

is_expected_backend_pid() {
  local pid="$1"
  is_allowed_port_owner "$pid" "$BACKEND_SERVICE"
}

is_stale_backend_candidate() {
  local pid="$1"
  local cmd cwd
  cmd=$(process_command "$pid")
  cwd=$(readlink "/proc/$pid/cwd" 2>/dev/null || true)
  [[ "$cwd" == "$ROOT_DIR" && "$cmd" == *web* ]]
}

terminate_pid() {
  local pid="$1"
  log "terminating stale backend pid $pid: $(process_command "$pid")"
  if (( DRY_RUN )); then
    log "dry-run: would terminate pid $pid"
    return 0
  fi
  kill -TERM "$pid" 2>/dev/null || return 0
  for _ in {1..20}; do
    [[ -d "/proc/$pid" ]] || return 0
    sleep 0.5
  done
  warn "pid $pid did not exit after SIGTERM; sending SIGKILL"
  kill -KILL "$pid" 2>/dev/null || true
}

check_resource_owners() {
  local label="$1"
  local resource="$2"
  local owners=()
  local pid unexpected=0
  [[ -n "$resource" ]] || return 0
  if [[ ! -e "$resource" ]]; then
    log "$label resource does not exist yet: $resource"
    return 0
  fi
  mapfile -t owners < <(resource_owner_pids "$resource")
  if ((${#owners[@]} == 0)); then
    log "$label resource has no open fd owners: $resource"
    return 0
  fi
  for pid in "${owners[@]}"; do
    if is_expected_backend_pid "$pid"; then
      log "$label resource is owned by expected backend service pid $pid"
      continue
    fi
    if is_stale_backend_candidate "$pid"; then
      if (( KILL_STALE )); then
        terminate_pid "$pid"
      else
        warn "$label resource is held by stale backend candidate pid $pid: $(process_command "$pid")"
        unexpected=1
      fi
    else
      warn "$label resource is held by unexpected pid $pid: $(process_command "$pid")"
      unexpected=1
    fi
  done
  if (( unexpected )); then
    die "$label resource is occupied. Re-run with --kill-stale only if the listed stale backend processes are safe to terminate."
  fi
}

check_git_clean_for_pull() {
  local dirty_tracked
  dirty_tracked=$(GIT_MASTER=1 git status --porcelain --untracked-files=no)
  [[ -z "$dirty_tracked" ]] || die "tracked files are dirty; commit/stash/revert before deployment:\n$dirty_tracked"
}

pull_code() {
  log "syncing code from origin/main"
  check_git_clean_for_pull
  local previous_sha new_sha
  previous_sha=$(GIT_MASTER=1 git rev-parse HEAD)
  run_git pull --ff-only origin main
  new_sha=$(GIT_MASTER=1 git rev-parse HEAD)
  log "git revision: $previous_sha -> $new_sha"
}

build_backend() {
  log "building backend binary"
  run mkdir -p "$ROOT_DIR/bin"
  TMP_BACKEND=$(mktemp /tmp/acgwarehouse-web.XXXXXX)
  run env CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o "$TMP_BACKEND" ./cmd/web
  run upx --best --lzma "$TMP_BACKEND"
  if (( ! DRY_RUN )); then
    [[ -s "$TMP_BACKEND" ]] || die "backend build produced an empty binary"
    chmod 0755 "$TMP_BACKEND"
  fi
  run install -o www -g www -m 0755 "$TMP_BACKEND" "$BACKEND_BIN"
}

build_frontend_server() {
  log "building frontend server binary"
  local tmp_frontend
  tmp_frontend=$(mktemp /tmp/acgwarehouse-frontend.XXXXXX)
  run env CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o "$tmp_frontend" ./cmd/frontend
  run upx --best --lzma "$tmp_frontend"
  if (( ! DRY_RUN )); then
    [[ -s "$tmp_frontend" ]] || die "frontend server build produced an empty binary"
    chmod 0755 "$tmp_frontend"
  fi
  run install -o www -g www -m 0755 "$tmp_frontend" "$FRONTEND_BIN"
  rm -f "$tmp_frontend"
}

build_frontend() {
  log "building frontend"
  cd "$FRONTEND_DIR"
  run npm install
  run npm run build
  if (( ! DRY_RUN )); then
    [[ -f "$FRONTEND_DIR/dist/index.html" ]] || die "frontend build did not produce dist/index.html"
  fi
  cd "$ROOT_DIR"
}

restart_services() {
  log "restarting systemd services"
  run systemctl restart "$BACKEND_SERVICE"
  run systemctl restart "$FRONTEND_SERVICE"
}

wait_for_http() {
  local url="$1"
  local label="$2"
  local attempt
  for attempt in {1..30}; do
    if curl -fsS "$url" >/dev/null; then
      log "$label is healthy: $url"
      return 0
    fi
    sleep 1
  done
  die "$label did not become healthy: $url"
}

verify_services() {
  local backend_port="$1"
  log "verifying systemd services"
  systemctl is-active --quiet "$BACKEND_SERVICE" || die "$BACKEND_SERVICE is not active"
  systemctl is-active --quiet "$FRONTEND_SERVICE" || die "$FRONTEND_SERVICE is not active"
  check_port "$backend_port" "$BACKEND_SERVICE" "backend"
  check_port "$FRONTEND_PORT" "$FRONTEND_SERVICE" "frontend"
  wait_for_http "http://127.0.0.1:${backend_port}/api/v1/ping" "backend"
  wait_for_http "http://127.0.0.1:${FRONTEND_PORT}/" "frontend"
  wait_for_http "http://127.0.0.1:${FRONTEND_PORT}/api/v1/ping" "frontend api proxy"
  log "recent service logs"
  journalctl -u "$BACKEND_SERVICE" -u "$FRONTEND_SERVICE" --since '2 minutes ago' --no-pager || true
}

preflight_paths() {
  local sqlite_path="$1"
  local bleve_path="$2"
  [[ -d "$ROOT_DIR" ]] || die "repository directory not found: $ROOT_DIR"
  [[ -f "$ENV_FILE" ]] || die "environment file not found: $ENV_FILE"
  [[ -d "$FRONTEND_DIR" ]] || die "frontend directory not found: $FRONTEND_DIR"
  [[ -e "$sqlite_path" || -w "$(dirname "$sqlite_path")" ]] || die "sqlite path is not writable: $sqlite_path"
  [[ -e "$bleve_path" || -w "$(dirname "$bleve_path")" ]] || die "bleve path parent is not writable: $bleve_path"
}

preflight_limits() {
  local nofile
  nofile=$(ulimit -n)
  if [[ "$nofile" =~ ^[0-9]+$ && "$nofile" -lt 1024 ]]; then
    warn "open file limit is low: $nofile"
  fi
}

main() {
  parse_args "$@"
  require_commands
  cd "$ROOT_DIR"

  local backend_port sqlite_path bleve_path
  backend_port=$(load_env_value PORT 2018)
  FRONTEND_PORT=$(load_env_value FRONTEND_PORT 2017)
  sqlite_path=$(absolute_path "$(load_env_value SQLITE_PATH data/acgwarehouse.db)")
  bleve_path=$(absolute_path "$(load_env_value BLEVE_PATH data/bleve)")

  log "backend port: $backend_port"
  log "frontend port: $FRONTEND_PORT"
  log "sqlite path: $sqlite_path"
  log "bleve path: $bleve_path"
  (( KILL_STALE )) && warn "--kill-stale enabled: stale backend candidates may be terminated"
  (( DRY_RUN )) && warn "--dry-run enabled: no changes will be made"

  preflight_paths "$sqlite_path" "$bleve_path"
  preflight_limits
  check_frontend_service_exec
  check_port "$backend_port" "$BACKEND_SERVICE" "backend"
  check_port "$FRONTEND_PORT" "$FRONTEND_SERVICE" "frontend"
  check_resource_owners "sqlite" "$sqlite_path"
  check_resource_owners "bleve" "$bleve_path"

  pull_code
  build_backend
  build_frontend
  build_frontend_server
  restart_services
  verify_services "$backend_port"
  log "deployment completed successfully"
}

main "$@"
