# Phase 6: Optimization and Deployment - Research

**Created:** 2026-03-18
**Status:** Ready for planning

---

## Executive Summary

Phase 6 should be planned as a **single-machine SQLite hardening + browse-path optimization + same-origin admin dashboard + Docker Compose packaging** phase, not as a platform expansion phase.

The clearest implementation path is:
- keep **Go + Gin + SQLite** as the only backend runtime path for this phase;
- package a **single app container** with host bind mounts for config, database, image library, and any generated thumbnail/cache data;
- add a **minimal single-page admin dashboard** served by the Go server itself;
- protect `/admin` and `/api/v1/admin/*` with **simple local/internal protection** (recommended: one shared Basic Auth credential in YAML);
- treat **gallery browse latency and thumbnail delivery** as the primary optimization target, with background jobs optimized only enough to remain stable and observable.

This phase must explicitly note that `.planning/ROADMAP.md` still contains legacy wording about broader deployment concerns, while Phase 6 is now **SQLite-only** and **single-machine self-host only**.

---

## 1. Scope Guardrails to Preserve

These constraints are already decided and should be copied into the eventual plan as non-negotiable boundaries:

- **SQLite only** for Phase 6. Do not plan PostgreSQL delivery, migration work, or dual-path verification here.
- **Single-machine Docker Compose** is the primary deployment path. Target is one `docker compose up -d` experience.
- **YAML-first config**. `.env` may remain optional override glue, but must not be the primary operator workflow.
- **Host-visible persistence** is required. Database and image-library paths must be obvious on the host for backup and debugging.
- **Admin dashboard is operational, not business-facing**. It is a single-page dashboard for status + a few safe actions.
- **Protection is local/internal only**. No multi-user auth, RBAC, OAuth, SSO, or internet-grade security system.
- **Do not turn the dashboard into a Flutter web client replacement**.

---

## 2. Repository Findings That Matter for Planning

### 2.1 Deployment and bootstrap status

Current code inspection shows:

- `cmd/server/main.go`
  - loads `config.yaml` from a fixed default path only;
  - opens SQLite directly with `sql.Open("sqlite3", cfg.Database.Path)`;
  - does **not** apply SQLite runtime pragmas;
  - does **not** configure an explicit SQLite connection-pool policy;
  - starts Gin with `r.Run(...)`, so there is **no explicit graceful shutdown / signal handling** path for Docker stop;
  - wires health, image, tag, search, duplicate, AI job routes, but no admin surface.
- `cmd/scan/main.go`
  - already supports `-config`, which is a useful pattern the server does not yet follow.
- `internal/handler/health_handler.go`
  - `/health` and `/ready` are present, but both are shallow and do not validate database/path/job-manager state.
- `internal/service/watcher_service.go`
  - file watching exists, but is **not wired into the server bootstrap** today.
- `internal/worker/job_manager.go`
  - supports `Start`, `Stop`, and `AddJob`, but has no pause/retry/requeue controls yet.
- There is currently **no Dockerfile**, **no compose file**, and **no `.dockerignore`**.

### 2.2 Browse-path performance status

The current browse path has several planning-relevant gaps:

- `internal/handler/image_handler.go`
  - supports `limit`, `offset`, and `tag_ids` only;
  - ignores `sort_by`, `sort_dir`, and `cursor` style requests.
- `internal/repository/image_repository.go`
  - `FindAll` is still `ORDER BY id LIMIT ? OFFSET ?`;
  - there are no browse-oriented sort indexes for `created_at`, `filename`, or `file_size`.
- `internal/repository/schema.go`
  - has basic indexes, FTS5, and tag/search tables;
  - does **not** currently show a local thumbnail-path model or browse-specific composite indexes for the gallery list path.
- `flutter_app/lib/services/api_service.dart`
  - already sends `cursor`, `sort_by`, `sort_dir`;
  - expects response field `items`, while `image_handler.go` returns `images`.
- `flutter_app/lib/models/image.dart`
  - expects `thumbnail_small_url` and `thumbnail_large_url`.
- `flutter_app/lib/widgets/image_grid.dart` and `flutter_app/lib/widgets/image_masonry.dart`
  - already assume thumbnail URLs are available for efficient rendering.

This means Phase 6 planning should treat **API contract alignment + thumbnail delivery path validation** as a prerequisite for meaningful performance work. Otherwise the team risks optimizing around placeholder or mismatched behavior.

### 2.3 Background-task operability status

- The admin dashboard requirements want safe actions such as scan / retry failed jobs / pause background tasks.
- Current code already provides useful building blocks:
  - `cmd/scan/main.go`
  - `internal/service/scanner_service.go`
  - `internal/repository/job_repository.go`
  - `internal/handler/ai_tag_handler.go`
  - `internal/worker/job_manager.go`
- But there is still no unified admin-oriented API for:
  - queue summary;
  - recent failures;
  - pause/resume state;
  - retrying failed jobs;
  - launching scans as observable background work.

### 2.4 Deployment documentation / config hygiene status

- `.gitignore` already excludes local DB files and local config variants.
- The repo does **not** currently show a `config.example.yaml` / deployment example config.
- The checked-in `config.yaml` contains a concrete `ai.api_key`-like value and should **not** be used as a published deployment template without redaction.

---

## 3. Recommended Standard Pattern for Phase 6

### 3.1 Primary architecture recommendation

**Recommended path:** one Go server process, one container, one SQLite file, same-origin admin page.

Why this is the best fit for the phase:

- It directly matches the decided constraints: SQLite-only, single-machine, Compose-first, simple protection, single-page dashboard.
- It reuses the current backend structure instead of introducing a second frontend stack or a reverse-proxy-heavy deployment.
- It keeps operations simple: one service to build, one service to run, one place to expose health/admin/API endpoints.
- It avoids accidentally turning the dashboard into a second product surface.

### 3.2 Service layout

Recommended Compose shape:

- **`app` service only** as the primary path.
- No separate database service.
- No Kubernetes assumptions.
- No required reverse proxy in the baseline path.

Optional helper commands can still exist for maintenance, but the base deployment should remain one service.

### 3.3 Admin UI delivery pattern

**Recommended:** serve admin assets from the Go server itself.

Recommended implementation styles, in order:

1. **Best fit:** a small static single-page dashboard under a path like `/admin`, backed by JSON APIs under `/api/v1/admin/*`.
2. **Asset packaging:** either embed assets into the Go binary with `go:embed`, or copy them into the image at build time.

**Do not use Flutter Web for the Phase 6 dashboard as the primary recommendation.** It is heavier, raises build/deploy complexity, and increases the risk of scope drift into a full browser client.

### 3.4 Protection pattern

**Recommended:** simple HTTP Basic Auth for admin routes, configured in YAML.

Why Basic Auth is the best fit here:

- browser-native, so no custom login flow is needed;
- enough for localhost/LAN-only admin access;
- minimal code and minimal operational burden;
- keeps the dashboard single-page and operational.

Recommended scope of protection:

- `/admin`
- `/api/v1/admin/*`
- optionally scan / retry / pause action endpoints if exposed outside the admin namespace.

### 3.5 SQLite runtime pattern

**Recommended baseline SQLite settings for this phase:**

- `PRAGMA journal_mode = WAL`
- `PRAGMA foreign_keys = ON`
- `PRAGMA busy_timeout = 5000` (or similar)
- `PRAGMA synchronous = NORMAL`
- explicit small connection policy on the Go side instead of leaving `database/sql` defaults implicit

This is the right baseline because Phase 6 is 
the best practical improvement without making the architecture more complex.

### 3.6 Persistence layout pattern

Recommended host-visible directory layout:

```text
./deploy/
  docker-compose.yml
  config.example.yaml

./data/
  db/acgwarehouse.db
  thumbnails/
  reports/

./library/
  ... scanned image folders ...
```

Notes:

- `config.yaml` should be bind-mounted read-only.
- SQLite DB file must live in a bind-mounted host path.
- Image scan roots should be mounted from host-visible paths.
- If local thumbnails are part of the deployed browse path, they should also be host-visible.

---

## 4. Implementation Options and Recommendation

### 4.1 Admin dashboard implementation options

| Option | Pros | Cons | Recommendation |
|------|------|------|----------------|
| Go-served static page + JSON admin API | Simplest deploy, same-origin, low scope, fits operational UI | Requires small amount of hand-written web UI | **Recommended** |
| Flutter Web admin page | Reuses Flutter knowledge | Heavier build/deploy, larger blast radius, easy scope drift | Not recommended for this phase |
| Separate JS SPA + separate build pipeline | Flexible UI | Extra pipeline/service complexity | Not recommended |

### 4.2 Manual scan action options

| Option | Pros | Cons | Recommendation |
|------|------|------|----------------|
| Trigger in-process async scan job from server | Observable, testable, same logs/metrics, same auth path | Requires server wiring for scanner service | **Recommended** |
| Shell out to `cmd/scan` inside container | Reuses CLI immediately | Harder to observe/control, brittle in containers | Avoid as primary design |
| Dedicated always-on scan sidecar | Separation of concerns | Extra service/operator complexity | Not needed now |

### 4.3 Dashboard protection options

| Option | Pros | Cons | Recommendation |
|------|------|------|----------------|
| Single shared Basic Auth credential | Simple, browser-native, fits LAN use | Not suitable for public internet | **Recommended** |
| Shared bearer token in header | Simple backend | Awkward operator UX in browser | Secondary fallback only |
| Full user auth system | Powerful | Out of scope, high complexity | Explicitly reject |

### 4.4 Docker packaging options

| Option | Pros | Cons | Recommendation |
|------|------|------|----------------|
| Single multi-stage Dockerfile for Go app + admin assets | Simple release artifact, aligns with Compose-first | Slightly larger build setup | **Recommended** |
| Multi-container app + proxy split | More flexible | Unnecessary for this phase | Not recommended |

---

## 5. Concrete Planning Guidance for Performance Work

### 5.1 Optimize the correct path first

Phase 6 should optimize in this order:

1. **Gallery list API** (query shape, pagination, sort semantics, indexes)
2. **Thumbnail delivery path** (URL contract, local/static serving, cache headers, file layout)
3. **Flutter gallery loading behavior** (cursor flow, incremental loading, refresh correctness, scroll smoothness)
4. **Background work isolation** (AI/scan jobs must not visibly degrade browsing)
5. **Search / duplicate / other secondary paths only if measurement shows they impact day-to-day browsing**

Do **not** spend the first optimization slice on duplicate detection internals or generalized micro-optimizations. The acceptance target is "10k+ images with smooth daily browsing", not "every code path benchmarked equally."

### 5.2 API contract work that should be planned early

Before performance tuning is considered complete, the following contract issues should be resolved:

- `cursor` vs `offset` semantics must be unified.
- `sort_by` and `sort_dir` must be truly implemented on `/api/v1/images`.
- Response shape for gallery list must be unified (`images` vs `items`).
- Thumbnail URL fields expected by Flutter must have a real backend/source-of-truth path.

This is both a correctness task and a performance task.

### 5.3 Pagination recommendation

**Recommended:** use cursor/keyset pagination for the gallery list path once sort semantics are locked.

Why:

- current offset pagination is simple but degrades at larger offsets;
- the frontend already thinks in cursor terms;
- cursor pagination is a better match for endless-scroll style browsing;
- it reduces the chance of inconsistent page boundaries when new images arrive.

Recommended stable sort tuples:

- default browse: `(created_at DESC, id DESC)`
- filename sort: `(filename ASC|DESC, id ASC|DESC)`
- file size sort: `(file_size ASC|DESC, id ASC|DESC)`

### 5.4 Indexing guidance

Phase 6 should plan browse-focused index review, not just "add indexes everywhere."

Likely useful additions:

- `images(created_at, id)` for newest-first browse
- `images(filename, id)` for filename sort
- `images(file_size, id)` for size sort
- `image_tags(tag_id, image_id)` for tag-filter browse queries
- any thumbnail lookup path/index if thumbnails become first-class local storage metadata

Indexes should be chosen only after locking the final query shape.

### 5.5 Background work isolation guidance

For this phase, background work only needs to be "stable and observable enough":

- scans and AI jobs should run asynchronously;
- admin actions should enqueue work instead of holding the HTTP request open;
- job state should be visible in the dashboard;
- a paused background state should stop new work consumption without breaking browse requests.

A minimal pause model is enough: a single process-wide pause flag in the job manager is sufficient for this phase.

---

## 6. Concrete Planning Guidance for Deployment Work

### 6.1 Docker Compose baseline

The baseline deployment should include:

- a multi-stage `Dockerfile`
- a `docker-compose.yml`
- a `config.example.yaml`
- a short quickstart document
- healthcheck wired to `/ready`
- bind mounts for config, DB, library, and any local thumbnails/reports
- `restart: unless-stopped`

### 6.2 Server bootstrap improvements worth planning

Deployment planning should include small but important runtime-hardening work in `cmd/server/main.go`:

- support `-config` path like `cmd/scan/main.go`
- build a real `http.Server` instead of only `r.Run(...)`
- graceful shutdown on SIGTERM/SIGINT
- SQLite init helper that applies pragmas and DB pool settings
- richer readiness checks
- optional startup wiring for watcher/scanner behavior if deployment mode needs it

### 6.3 Recommended config additions

A minimal YAML-first deployment config likely needs additions similar to:

```yaml
server:
  host: 0.0.0.0
  port: 8080
  env: production

database:
  type: sqlite
  path: /data/db/acgwarehouse.db
  busy_timeout_ms: 5000
  journal_mode: wal

storage:
  scan_roots:
    - /library
  thumbnails_path: /data/thumbnails

admin:
  enabled: true
  base_path: /admin
  username: admin
  password: change-me
```

The exact field names can change, but the plan should reserve room for:

- SQLite deployment tuning
- thumbnail/local cache path
- admin protection config
- predictable container paths

### 6.4 Recommended admin data model

The dashboard home page should prioritize exactly the scope defined in context:

- service health / readiness status
- queue/job status summary
- image-library size summary
- storage/path configuration summary
- recent errors
- safe actions

Recommended admin summary blocks:

- **System:** version, uptime, env, DB path, WAL enabled, process status
- **Library:** image count, scan roots, last scan time/result
- **Jobs:** ready/running/failed/finished counts, paused/not paused
- **Storage:** DB file size, thumbnail directory size, free disk estimate if practical
- **Errors:** recent failed jobs / recent scan failures

### 6.5 Safe action set for this phase

Recommended actions to plan explicitly:

- trigger manual scan
- retry failed jobs
- pause background jobs
- resume background jobs
- refresh status

Do not expand the action set into gallery/tag/content management from the dashboard.

---

## 7. Common Pitfalls for This Phase

| Pitfall | Why it matters here | Planning guidance |
|------|------|----------------|
| Optimizing before API contract alignment | Current gallery API/frontend expectations already diverge | Lock list response, thumbnail fields, and pagination semantics first |
| Leaving SQLite at default runtime settings | Can produce avoidable `database is locked` issues and weaker browse concurrency | Plan a single SQLite init layer with WAL + busy timeout + explicit pool config |
| Using deep offset pagination for endless scroll | Becomes slower as library size grows | Prefer cursor/keyset pagination for the gallery path |
| Building the admin dashboard as a second client app | Scope will drift into full web product work | Keep it server-served, single-page, operational only |
| Making admin actions synchronous | Scan/retry/pause endpoints will feel unreliable and block UI | Queue work and return observable job state |
| Exposing admin with wildcard CORS + no protection | Fine for local dev, poor default for LAN use | Serve admin same-origin and protect admin routes with simple auth |
| Hiding persistence inside container-only paths or named volumes | Conflicts with host-visible persistence requirement | Use explicit bind mounts and document paths clearly |
| Forgetting restart semantics for in-memory queue | Jobs in `ready/running` state can become ambiguous after container restart | Either requeue on startup or document the exact restart behavior and surface it in admin UI |
| Publishing real secrets in YAML examples | Current repo already shows a key-like value in `config.yaml` | Ship redacted examples only |

---

## 8. Likely File / Module Touch Points

These are the most likely areas Phase 6 planning should assume will change.

| Area | Likely files |
|------|--------------|
| Server bootstrap / deployment hardening | `cmd/server/main.go`, possible new bootstrap helper in `internal/app/` or `internal/bootstrap/` |
| Config extensions | `internal/config/config.go`, `config.yaml`, new `config.example.yaml` |
| SQLite init / readiness | new helper near DB bootstrap, `internal/handler/health_handler.go` |
| Gallery browse optimization | `internal/handler/image_handler.go`, `internal/repository/image_repository.go`, `internal/repository/schema.go` |
| Admin API | new `internal/service/admin_service.go`, new `internal/handler/admin_handler.go`, `internal/handler/routes.go` |
| Job controls | `internal/worker/job_manager.go`, `internal/repository/job_repository.go`, possibly scanner/job wiring |
| Scan trigger wiring | `internal/service/scanner_service.go`, `cmd/server/main.go`, placeholder scan endpoint in `internal/handler/routes.go` |
| Admin UI assets | new `web/admin/` or similar asset directory |
| Deployment assets | new `Dockerfile`, new `.dockerignore`, new `docker-compose.yml`, docs under `docs/` or `.planning/phases/06-optimization-deployment/` |
| Flutter browse fixes if needed | `flutter_app/lib/services/api_service.dart`, `flutter_app/lib/providers/image_provider.dart`, gallery widgets/screens |

---

## 9. Recommended Plan Slices

A good Phase 6 plan will probably split cleanly into these slices:

### Slice A - Runtime and SQLite hardening

Goals:
- add config-path support to server bootstrap;
- add graceful shutdown;
- add SQLite init pragmas + explicit pool config;
- improve `/ready` to check actual runtime dependencies.

Why first:
- Docker deployment and reliable admin reporting both depend on this.

### Slice B - Browse-path contract and query optimization

Goals:
- unify list API contract;
- implement real sort support;
- switch or prepare for cursor-based pagination;
- add only the indexes needed for final list queries;
- ensure thumbnail URLs/local delivery path are real and measurable.

Why second:
- this is the user-facing performance acceptance path for 10k+ images.

### Slice C - Admin API and background controls

Goals:
- add admin summary endpoints;
- add recent-error / queue-summary endpoints;
- implement pause/resume/retry/scan trigger flows.

Why third:
- it delivers DEPL-02 in the intended narrow operational scope.

### Slice D - Single-page admin UI

Goals:
- build one status dashboard page;
- wire the small safe-action set;
- keep same-origin and protected.

Why fourth:
- UI should land after API semantics are stable.

### Slice E - Containerization, docs, and performance report

Goals:
- add Dockerfile + Compose;
- ship example config and host path conventions;
- add deployment guide and operator quickstart;
- produce reproducible performance report + benchmark method.

Why last:
- packaging and docs should describe the actual final runtime shape, not a moving target.

---
## 10. Validation Architecture (Nyquist-oriented)

This section is derivable and should be usable as the starting point for a later `06-VALIDATION.md`.

### 10.1 Validation objectives

Phase 6 validation should prove four things:

1. **Deployment correctness** - the app starts reliably via Compose with host-visible persistence.
2. **Browse-path performance** - 10k+ image browsing remains smooth enough for daily use.
3. **Operational correctness** - admin summary and safe actions reflect real runtime state.
4. **Restart resilience** - restart/redeploy does not lose DB/library state and leaves job state understandable.

### 10.2 Suggested test layers

| Layer | What to validate | Tooling |
|------|------------------|---------|
| Config/bootstrap | config path, SQLite pragmas, readiness, graceful shutdown | `go test` |
| Repository/service | browse query semantics, sort, cursor, queue summary, retry/pause logic | `go test` |
| Handler/API | admin endpoints, scan trigger, auth protection, readiness payloads | `httptest` + `go test` |
| Flutter/client contract | gallery list contract, cursor flow, refresh behavior if Flutter is touched | `flutter test` |
| Compose smoke | container start, healthcheck green, bind mounts persist data | shell script / documented smoke steps |
| Performance verification | 10k dataset load, list latency, dashboard responsiveness, manual scroll check | reproducible script + manual smoke |

### 10.3 Suggested automated validation commands

Backend quick run after each backend slice:

```bash
go test ./cmd/server ./internal/config ./internal/handler ./internal/repository ./internal/service ./internal/worker
```

Full backend suite before phase sign-off:

```bash
go test ./...
```

Flutter verification only if Phase 6 touches the existing gallery client:

```bash
cd flutter_app && flutter test
```

### 10.4 Recommended Wave 0 validation assets

The eventual plan should likely create these missing verification assets early:

- a **10k dataset seeding method** (script or command sequence)
- a **compose smoke-check script** or documented command set
- targeted tests for:
  - SQLite init behavior
  - gallery cursor/sort contract
  - admin auth middleware
  - admin summary endpoint
  - pause/resume/retry semantics
  - scan trigger endpoint

### 10.5 Manual acceptance checks

Minimum manual checks for phase sign-off:

1. `docker compose up -d` starts successfully on a clean machine with edited YAML only.
2. `/health` and `/ready` both behave correctly under normal startup and DB-path failure cases.
3. Admin dashboard loads under simple protection and shows real queue/library/path status.
4. Manual scan action returns immediately, creates observable background work, and does not freeze browse endpoints.
5. With a 10k+ dataset, gallery browsing feels smooth in normal desktop use.
6. Container restart preserves DB, library mounts, and leaves job state understandable.

### 10.6 Suggested performance report contents

The performance report for this phase should be reproducible and small:

- dataset shape (image count, thumbnail strategy, host machine profile)
- exact config used
- endpoint timings for gallery list path
- scroll smoke notes
- known limits / not-yet-optimized paths
- before/after comparison if practical

This is enough for Phase 6. It does not need to become a long-term benchmarking platform.

---

## 11. Recommended Planning Assumptions to Lock In

If the planning phase needs defaults, these are the safest defaults to adopt:

- **Deployment shape:** one `app` container only
- **Admin UI:** Go-served single-page dashboard, not Flutter Web
- **Admin protection:** one shared Basic Auth credential in YAML
- **Gallery pagination target:** cursor/keyset on stable sort tuples
- **SQLite target mode:** WAL + busy timeout + explicit pool config
- **Manual scan implementation:** in-process async job, not shelling out to CLI
- **Persistence layout:** bind-mounted config, DB, library, and thumbnail/report paths

These defaults align with the user decisions in `06-CONTEXT.md` and keep the phase small enough to finish cleanly.

---

*Research completed: 2026-03-18*
*Status: Ready for planning*
