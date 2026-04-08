# Duplicate Detection SSE Incremental Design

## Summary

The duplicate-detection flow must stop behaving like a single blocking request that leaves Flutter with an indefinite spinner. Instead, Go becomes the owner of a long-running duplicate-detection task, exposes structured progress over SSE to Flutter, and uses persisted image-level hash metadata so that the first full run is expensive but later runs are incremental.

The approved direction is:

- change duplicate detection from synchronous request/response to Go-owned async task creation
- stream real-time duplicate-detection progress from Go to Flutter over SSE
- keep Flutter connected only to Go, not directly to Python task semantics
- persist reusable single-image hash values on `images`
- pass cached hash metadata from Go to Python when it is still valid
- keep pairwise duplicate metrics such as `distance` on `duplicate_relations`
- reduce large-run cost by paging and batching rather than submitting the entire library in one giant payload

## Problem Statement

The current duplicate-detection path is correct in function but poor in user feedback and weak at scale.

Codebase evidence:

- `flutter_app/lib/services/duplicate_service.dart` triggers duplicate detection through a blocking backend request.
- `flutter_app/lib/providers/duplicate_provider.dart` and `flutter_app/lib/screens/duplicate_screen.dart` only model a boolean loading state, so the UI can do little more than spin.
- `internal/handler/duplicate_handler.go` still handles `POST /api/v1/duplicates/detect` synchronously and only returns after the entire operation completes.
- `internal/service/duplicate_service.go` loads images with `FindAll(1000000, 0, "id", "asc")`, submits all inputs to the sidecar at once, polls Python progress internally, and only logs the intermediate state.
- `internal/sidecar/client.go` sends `DetectionImageInput` with only `id`, `path`, `width`, `height`, `file_size`, and `format`, so Python receives no reusable hash metadata.
- `services/python-sidecar/compute/hashing.py` computes `SHA256` and `pHash` for every image on every run, with worker count effectively capped by `max_workers=4`.
- `internal/domain/image.go` and `internal/repository/schema.go` show that `images` currently store `phash` and `phash_hex`, but not `sha256`.
- `internal/domain/duplicate_group.go` and `internal/repository/duplicate_repository.go` show that `duplicate_relations` store `file_hash` and `phash_distance`.
- `services/python-sidecar/models/duplicates.py` returns `sha256`, `phash`, and `distance` per group member, but the current persistence path only writes reusable `pHash` back to `images`.

This leaves two practical gaps:

1. users cannot tell what stage the job is in or whether it is progressing
2. every large run behaves too much like a cold start, especially with tens of thousands of images on local Windows SSD storage

## Goals

1. Replace the indefinite spinner with structured, real-time progress in Flutter.
2. Keep Go as the only frontend-facing orchestration layer.
3. Preserve the existing Python sidecar model rather than rewriting duplicate detection into a different stack.
4. Make the first full-library run expensive but make later runs incremental whenever cached hashes are still valid.
5. Keep duplicate-group semantics correct by storing reusable single-image metadata separately from pairwise relation metadata.
6. Reduce memory pressure and long serialization stalls for `50k`-image runs.

## Non-Goals

1. No rewrite that removes the Python sidecar entirely.
2. No direct Flutter subscription to Python endpoints.
3. No attempt to persist `distance` as if it were a stable image-level attribute.
4. No speculative redesign of duplicate scoring or recommendation logic beyond what is required for incremental execution.
5. No requirement in this phase that long-running duplicate-detection tasks survive a full backend restart.

## Approved Direction

The design is approved with the following constraints:

- Go owns the lifecycle of the duplicate-detection task presented to Flutter.
- Python can still use its own internal task model, but it is hidden behind Go.
- Flutter receives task progress through SSE and may also query a status endpoint as a fallback.
- `SHA256` and `pHash` are image-level cache values and belong on `images`.
- `distance`, `is_recommended`, `recommendation_score`, and recommendation rationale remain relation-level values and stay on `duplicate_relations`.
- Python worker count must be configurable; `CPU core count × 2` may be used as the default starting point, but not as a hard-coded invariant.
- Go must stop sending the whole library as one giant sidecar request for large runs.

## Architecture Overview

### 1. Ownership Model

The new task boundary is:

`Flutter -> Go duplicate task -> Python sidecar work batches -> Go persistence -> Flutter completion`

Go becomes the source of truth for:

- task creation
- task status
- user-visible progress stage names
- aggregation across paged or batched sidecar work
- final success or failure payload seen by Flutter

Python remains responsible for:

- computing missing hashes
- grouping duplicates by threshold
- calculating recommendation scores and rationale

Flutter remains responsible for:

- creating the task
- subscribing to task events
- rendering phase, percentage, and counts
- handling completion and failure states

### 2. Task API Contract

The current `POST /api/v1/duplicates/detect` must change from a blocking endpoint into task creation.

Required backend contract:

- `POST /api/v1/duplicates/detect`
  - input: threshold and any future detection options
  - behavior: create a Go-owned duplicate task and return immediately
  - output: `task_id`, initial status, and initial progress payload
- `GET /api/v1/duplicates/tasks/:task_id`
  - behavior: return current status for polling fallback and reconnect recovery
- `GET /api/v1/duplicates/tasks/:task_id/events`
  - behavior: stream structured SSE events for real-time updates

This keeps Flutter independent from Python task IDs and allows Go to evolve the internal execution model without breaking the frontend.

### 3. SSE Event Model

SSE must carry structured task events rather than raw log lines.

Each event should include at minimum:

- `task_id`
- `type`
- `phase`
- `progress`
- `message`
- `processed`
- `total`
- `timestamp`

Required event phases:

1. `queued`
2. `preparing`
3. `hashing`
4. `grouping`
5. `persisting`
6. `completed`
7. `failed`

Event rules:

- progress must be monotonic within a task
- `message` should describe real current work, not generic log text
- heartbeat comments or keepalive events must be sent often enough to avoid idle disconnects during long runs
- reconnecting clients must be able to rehydrate from `GET /status` even if they miss stream history

The implementation should reuse the repo’s existing event-streaming style from `MonitoringEventBus` and `LogStreamService`, but duplicate detection should have its own task-specific event channel rather than piggybacking on monitoring or log text.

## Data Model Design

### 1. Image-Level Cache Fields

Reusable single-image values belong on `images`.

Required image-level fields:

- existing `phash`
- existing `phash_hex`
- new `sha256`
- a freshness key sufficient to validate reuse before Go passes cached values back to Python

This spec does not force one exact freshness-key shape, but the implementation must use an explicit and deterministic validity rule. The safest first version is a file-identity/freshness check derived from data Go can reliably obtain before reuse.

Recommended first-version validation inputs:

- image path
- current file size
- current filesystem modified timestamp

If cached metadata cannot be proven fresh, Go must treat the image as stale and request recomputation.

### 2. Relation-Level Fields

Pairwise duplicate results remain on `duplicate_relations`.

These values stay relation-scoped:

- `phash_distance`
- `is_recommended`
- `recommendation_score`
- `recommendation_rationale`
- `file_hash` if retained for read-model compatibility

Important rule:

- `distance` must not be treated as an image-level reusable cache value

Reasoning:

- `distance` is defined relative to a specific recommended image or pair
- recommendation choice can change when the dataset changes
- the same image can have different distances in different duplicate groups or reruns

### 3. `file_hash` Compatibility Rule

The current schema stores `duplicate_relations.file_hash`. That field may remain for duplicate-detail read models, but it must be treated as a denormalized relation copy, not the authoritative reusable hash cache.

The canonical reusable `SHA256` value becomes `images.sha256`.

## Go-to-Python Request Contract

### Existing Gap

`internal/sidecar/client.go` currently sends only image geometry and path information, so Python cannot skip work even when Go already knows the hashes.

### Required Contract Change

`DetectionImageInput` and Python `ImageInput` must be expanded to optionally carry cached metadata.

Required added fields:

- `sha256`
- `phash` or `phash_hex`
- freshness/validation metadata if the Python side needs it for diagnostics or fallback reasoning

Behavior rules:

- if Go has valid cached `sha256` and `pHash`, it passes them to Python
- if one or both values are missing or stale, Go sends empty values and Python computes them
- Python must not blindly trust stale cache values; the Go side is responsible for only sending cache entries that passed freshness validation

## Incremental Execution Model

### 1. Cold Run

On the first full run, most or all images will have missing reusable hash fields.

Expected behavior:

- Go paginates through the image library
- Go validates which images already have reusable hash metadata
- Python computes missing values
- Go persists refreshed `sha256` and `pHash` back to `images`
- Go persists duplicate-group results to `duplicate_groups` and `duplicate_relations`

### 2. Warm Run

On later runs, many images will already have valid cache entries.

Expected behavior:

- unchanged images reuse cached `sha256` and `pHash`
- only new or changed images trigger expensive hashing work
- Python still performs grouping against the current dataset using the full effective hash set
- group membership, recommendation choice, and pairwise distance are recalculated for the current run

### 3. Cache Invalidation Rules

Go must invalidate reuse when any required proof is missing or inconsistent.

At minimum, treat cache as stale when:

- `sha256` is empty
- `phash_hex` is empty
- the current file no longer exists
- file size has changed
- modified timestamp has changed
- the image path resolves to a different file than before

The implementation may add stronger validation later, but it must not silently reuse unverifiable hashes.

## Large-Library Execution Strategy

### 1. Paged Reads in Go

Go must stop relying on a single `FindAll(1000000, ...)` call as the effective execution strategy.

Required behavior:

- read images in pages from the repository
- aggregate progress across pages under one logical duplicate-detection task
- avoid materializing and serializing the entire library into one giant request buffer when the library is large

### 2. Bounded Sidecar Batches

Go should submit work to Python in bounded internal batches while presenting one logical task to Flutter.

This design is required to reduce:

- large JSON payload overhead
- memory spikes in Go and Python
- long wait time before the first meaningful progress event
- the chance that one failure forces a full-library restart from scratch

The exact batch size is an implementation choice, but it must be configurable and tuned for local desktop workloads.

### 3. Python Worker Count

Python hashing concurrency must be configurable.

Approved direction:

- allow `CPU core count × 2` as the initial default target

Required safety rule:

- do not bury that value as an unchangeable constant in code

Reasoning:

- hashing here is not purely CPU bound because it includes file IO and image decode
- desktop Windows SSD workloads can slow down under oversubscription
- the first version must be measurable and adjustable without another schema or API redesign

## Python-Side Behavior Changes

### 1. Hashing Stage

`services/python-sidecar/compute/hashing.py` must support per-image cache reuse.

Required behavior:

- if an input already includes valid cached `sha256` and `phash`, return them without recomputation
- if only one cached value is available, compute the missing one
- if neither is available, compute both
- continue returning per-image errors for unreadable or invalid files

### 2. Progress Granularity

The current progress mapping of roughly `hashing -> 60`, `grouping -> 70`, `scoring -> 90`, `completed -> 100` is too coarse for the user experience target.

Required progress model:

- reflect real processed counts, not just phase jumps
- distinguish cache hit reuse from actual hash computation work where practical
- provide enough detail for Go to tell Flutter how much of the job is already done

### 3. Result Contract

Python should continue returning:

- `sha256`
- `phash`
- `distance`
- recommendation fields

Go persists the reusable values back to `images` and the relation values back to `duplicate_relations`.

## Flutter UX Model

### 1. Start Detection

When the user triggers duplicate detection:

- Flutter sends the create-task request
- receives `task_id`
- immediately subscribes to the SSE endpoint
- renders the task as an in-progress operation rather than a blocking modal spinner

### 2. In-Progress View

The duplicate screen should show at least:

- current phase label
- numeric progress percent
- processed / total counts where available
- current message such as cache validation, hashing, grouping, or persistence

### 3. Completion

On completion:

- stop the SSE subscription
- refresh duplicate groups
- surface total groups found and any skipped image count

### 4. Failure and Reconnect

If the SSE stream drops:

- Flutter should call the task status endpoint to recover current state
- if the task is still active, reconnect the SSE stream
- if the task is terminal, render the final state without forcing the user to restart the scan blindly

## Error Handling Model

### Task-Level Errors

- if task creation fails, Flutter must surface the backend error immediately
- if Python task submission or polling fails, Go must move the task into a terminal `failed` state and emit a final failure event
- if result persistence fails after computation succeeds, the user-visible task still fails because the system state is not complete

### Item-Level Errors

- individual unreadable or missing files should be recorded as skipped item failures where possible
- item-level failures must not silently disappear from the final task summary

### Mid-Run Mutation

If a file changes, disappears, or becomes inaccessible during detection:

- treat that image as failed or skipped for the current run
- do not reuse old cache values when the current file state is ambiguous

## Implementation Boundaries

### Backend Areas Affected

- `internal/handler/duplicate_handler.go`
- `internal/handler/routes.go`
- `internal/service/duplicate_service.go`
- `internal/sidecar/client.go`
- task/event-stream support code in `internal/service/...`
- `internal/domain/image.go`
- `internal/domain/duplicate_group.go`
- `internal/repository/schema.go`
- `internal/repository/image_repository.go`
- duplicate persistence logic in `internal/repository/duplicate_repository.go`

### Python Areas Affected

- `services/python-sidecar/models/duplicates.py`
- `services/python-sidecar/routers/duplicates.py`
- `services/python-sidecar/compute/hashing.py`
- any supporting task/progress helper used by duplicate detection

### Flutter Areas Affected

- duplicate API client/service layer
- duplicate provider state model
- duplicate screen progress UI
- SSE client integration and lifecycle handling

## Verification Requirements

The implementation plan must verify all of the following:

1. task creation returns immediately for a large duplicate-detection run
2. Flutter receives live SSE updates during detection
3. duplicate detection still produces correct duplicate groups after the async redesign
4. `images.sha256` and `images.phash_hex` are persisted after a cold run
5. a second run reuses cached hashes for unchanged images
6. changed files invalidate cache and trigger recomputation
7. relation-level `distance` values are still present in duplicate-group responses and are not moved to `images`
8. large-library runs no longer depend on a single full-library sidecar payload

## Open Decisions Deferred to the Implementation Plan

The implementation plan must still decide these explicit details:

- the exact Go task-state storage shape for duplicate detection
- whether the duplicate task event stream uses a dedicated bus type or a thin shared streaming abstraction
- the exact freshness-key representation and where it is persisted
- the initial page size and sidecar batch size
- the exact default worker-count configuration and override mechanism
- the specific Flutter SSE package or stream implementation to use on Windows desktop

## Recommendation

The implementation should proceed with a Go-owned async duplicate task, dedicated SSE progress stream, image-level `sha256` persistence, paged and batched sidecar orchestration, and strict separation between reusable single-image cache values and relation-only duplicate metrics.

That gives the user-visible progress they need, preserves the existing Go + Python stack, and creates a correct path from one expensive cold run to later incremental runs.
