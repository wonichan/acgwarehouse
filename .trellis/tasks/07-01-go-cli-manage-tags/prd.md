# Go CLI manage tags

## Goal

Create a Go command-line tool for local administrators to add, delete, and update global tags in the backend SQLite database without using the HTTP API. Delete operations must remove the tag and its image associations so images no longer contain that tag.

## Background

- The backend stores SQLite data at `data/acgwarehouse.db` by default and supports `SQLITE_PATH` override (`README.md`, `internal/conf/conf.go:168`).
- The tag table is named `tag` and stores `id`, `name`, `usage_count`, `created_at`, and `updated_at` (`internal/model/po/tag.go:5`).
- Tag names are unique, non-empty, and limited to 64 characters by existing repository validation (`internal/repository/tag.go:229`).
- Existing repository behavior supports create-or-return-existing (`internal/repository/tag.go:40`), update by ID (`internal/repository/tag.go:89`), find by name (`internal/repository/tag.go:76`), and delete tag plus `image_tag` rows transactionally (`internal/repository/tag.go:108`).
- Existing `conf.Load()` validates web security config, including `JWT_SECRET` (`internal/conf/conf.go:125`, `internal/conf/conf.go:201`), so the CLI needs a database-only config path instead of requiring web credentials.
- Existing Go entrypoints live under `cmd/*`, including `cmd/sync` and `cmd/web` (`README.md`).

## Requirements

- Provide a Go CLI entrypoint for tag management.
- Add mode must be `-a "标签名"` and create or reuse a tag by name.
- Delete mode must be `-d "标签名"` and locate the tag by unique name.
- Update mode must be `-u "旧标签名" -name "新标签名"` and locate the tag by current unique name.
- Delete must execute directly by default, without interactive confirmation.
- Delete must remove related `image_tag` rows and the `tag` row in one transactional operation.
- The CLI must use the same SQLite default path and environment override as the backend: default `data/acgwarehouse.db`, override via `SQLITE_PATH`.
- The CLI must not require unrelated web/server credentials such as `JWT_SECRET`.
- The CLI must reject ambiguous operation combinations, e.g. more than one of `-a`, `-d`, and `-u` in the same invocation.
- The CLI must print useful success/error messages and return non-zero on invalid input or failed database operations.
- The implementation should reuse existing repository/config/database code where practical instead of duplicating SQL rules.

## Acceptance Criteria

- [ ] `go run ./cmd/tagctl -a "新标签"` creates the tag, or returns the existing tag consistently with current repository behavior.
- [ ] `go run ./cmd/tagctl -d "旧标签"` locates by name, deletes the tag, and deletes all matching `image_tag` rows transactionally.
- [ ] `go run ./cmd/tagctl -u "旧标签" -name "新标签"` locates by old name, renames the tag, and updates `updated_at`.
- [ ] Invocations with zero operations, multiple operations, missing `-name` for update, blank names, over-length names, or nonexistent names fail with non-zero exit codes and useful messages.
- [ ] The CLI uses `SQLITE_PATH` when set and otherwise uses `data/acgwarehouse.db`.
- [ ] The CLI can run without configuring `JWT_SECRET`.
- [ ] Relevant Go tests pass, including repository tests for delete association cleanup and CLI/config tests for argument behavior.

## Out of Scope

- No frontend changes.
- No HTTP API behavior changes unless required for shared code reuse.
- No interactive delete confirmation.
- No automatic preset tag list is included unless requested separately.
