# Go CLI manage tags Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI that adds, deletes, and updates tags in SQLite by tag name while preserving existing repository behavior.

**Architecture:** Add an exported DB-only config seam, then add a `cmd/tagctl` package with testable argument parsing and operation execution. Reuse `db.NewSQLite`, `ImageRepository`, and `TagRepository`; delete resolves name to ID and then calls the existing transactional repository delete.

**Tech Stack:** Go 1.26.4, standard `flag` package, GORM-backed repository layer, SQLite via existing `internal/infra/db`.

---

## Files

- Modify: `internal/conf/conf.go`
  - Add exported `LoadDatabase() DatabaseConfig` wrapper around existing `loadDatabaseConfig()`.
- Create: `cmd/tagctl/main.go`
  - CLI entrypoint, flag parsing, SQLite/repository wiring, operations.
- Create: `cmd/tagctl/main_test.go`
  - Unit/integration tests for CLI behavior against temporary SQLite database.
- Modify: `internal/repository/tag_test.go`
  - Add repository regression coverage that delete removes `image_tag` associations if not already covered by CLI tests.

## Task 1: Add DB-only config loader

- [ ] Add failing test if an appropriate config test exists; otherwise inspect `internal/conf/conf_test.go` and add one there.
- [ ] Implement `func LoadDatabase() DatabaseConfig` in `internal/conf/conf.go` by returning `loadDatabaseConfig()`.
- [ ] Verify it honors `SQLITE_PATH` without requiring `JWT_SECRET`.
- [ ] Run: `go test ./internal/conf`
- [ ] Expected: PASS.

## Task 2: Add tagctl CLI core

- [ ] Create `cmd/tagctl/main.go`.
- [ ] Define operation parsing with exactly one of `-a`, `-d`, `-u`.
- [ ] Require `-name` only when `-u` is set.
- [ ] Keep `run(ctx, args, stdout, stderr) error` testable; `main()` should call it and `os.Exit(1)` on error.
- [ ] Implement DB/repository wiring:
  - `cfg := conf.LoadDatabase()`
  - `sqliteDB, err := db.NewSQLite(cfg)`
  - `imageRepo := repository.NewImageRepository(sqliteDB.Read, sqliteDB.Write)`
  - `tagRepo := repository.NewTagRepository(sqliteDB.Read, sqliteDB.Write, imageRepo)`
- [ ] Implement add via `tagRepo.Create(ctx, do.Tag{Name: addName})`.
- [ ] Implement delete via `tagRepo.FindByName(ctx, deleteName)` then `tagRepo.Delete(ctx, tag.ID)`.
- [ ] Implement update via `tagRepo.FindByName(ctx, oldName)` then `tagRepo.Update(ctx, do.Tag{ID: tag.ID, Name: newName})`.
- [ ] Print concise success messages.

## Task 3: Test CLI behavior

- [ ] In `cmd/tagctl/main_test.go`, use `t.TempDir()` and `t.Setenv("SQLITE_PATH", tempPath)` so tests use isolated SQLite files.
- [ ] Test add creates a tag and returns success.
- [ ] Test add with existing name succeeds consistently with repository idempotent create behavior.
- [ ] Test update by old name renames the tag.
- [ ] Test delete by name removes the tag.
- [ ] Test invalid usage: no operation, multiple operations, update without `-name`, blank add name, nonexistent delete/update target.
- [ ] Run: `go test ./cmd/tagctl`
- [ ] Expected: PASS.

## Task 4: Test delete association cleanup

- [ ] Add or confirm a test that creates an image, creates a tag, assigns the tag to the image, deletes the tag, and then verifies `ListByImageID(image.ID)` returns no tags.
- [ ] Put this in `internal/repository/tag_test.go` unless CLI tests already directly verify the association behavior.
- [ ] Run: `go test ./internal/repository -run TagRepository`
- [ ] Expected: PASS.

## Task 5: Full verification

- [ ] Run Go formatting: `gofmt -w internal/conf/conf.go cmd/tagctl/main.go cmd/tagctl/main_test.go internal/repository/tag_test.go`.
- [ ] Run package tests: `go test ./internal/conf ./internal/repository ./cmd/tagctl`.
- [ ] Run full backend tests if package tests pass: `go test ./...`.
- [ ] Manually smoke test with temporary DB path:
  - `SQLITE_PATH=/tmp/tagctl-smoke.db go run ./cmd/tagctl -a "smoke-tag"`
  - `SQLITE_PATH=/tmp/tagctl-smoke.db go run ./cmd/tagctl -u "smoke-tag" -name "smoke-tag-new"`
  - `SQLITE_PATH=/tmp/tagctl-smoke.db go run ./cmd/tagctl -d "smoke-tag-new"`
- [ ] Expected: each smoke command exits 0 and prints a success message.

## Risk Notes

- Do not use `conf.Load()` in `tagctl`; it requires unrelated web security config and would make local DB maintenance harder.
- Do not hand-write delete SQL in `cmd/tagctl`; use repository delete to preserve transaction semantics.
- Avoid adding interactive prompts because the approved requirement is direct delete execution.
- Avoid committing unless explicitly asked.
