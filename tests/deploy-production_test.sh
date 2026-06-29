#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="$ROOT_DIR/scripts/deploy-production.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[[ -f "$SCRIPT" ]] || fail "deploy script is missing"
[[ -x "$SCRIPT" ]] || fail "deploy script is not executable"

bash -n "$SCRIPT"

help_output="$($SCRIPT --help)"
[[ "$help_output" == *"--kill-stale"* ]] || fail "help output must mention --kill-stale"
[[ "$help_output" == *"--dry-run"* ]] || fail "help output must mention --dry-run"

if grep -q 'serve -s dist' "$SCRIPT"; then
  fail "production frontend must not use static serve fallback because /api/* must reach backend"
fi

[[ "$help_output" == *"serves the built Vue SPA and proxies /api/*"* ]] || fail "help output must describe API proxy behavior"
grep -q 'systemctl cat "\$FRONTEND_SERVICE"' "$SCRIPT" || fail "deploy script must verify frontend service ExecStart"
grep -q 'frontend api proxy' "$SCRIPT" || fail "deploy script must verify API through frontend proxy"
grep -q 'FRONTEND_PORT=$(load_env_value FRONTEND_PORT 2017)' "$SCRIPT" || fail "deploy script must read FRONTEND_PORT from environment file"

printf 'PASS: deploy-production.sh static checks\n'
