# Go CLI manage tags Technical Design

## Architecture

Add a focused Go CLI entrypoint under `cmd/tagctl`. The CLI parses local flags, opens SQLite through existing infrastructure, constructs existing repositories, and invokes repository methods to mutate tags.

The CLI should not call `conf.Load()` because that path validates web-serving security settings such as `JWT_SECRET`. Instead, expose a database-only config loader in `internal/conf` that reuses the existing private `loadDatabaseConfig()` defaults and environment handling.

## Components and Boundaries

### `internal/conf`

Add an exported database-only config function, for example:

```go
func LoadDatabase() DatabaseConfig
```

Responsibilities:
- Return the same `DatabaseConfig` produced by the existing private `loadDatabaseConfig()`.
- Preserve `SQLITE_PATH`, `SQLITE_BUSY_TIMEOUT_MS`, and connection defaults.
- Avoid validating unrelated web/security config.

### `cmd/tagctl`

Create a CLI package with a small `main()` and a testable `run(ctx, args, stdout, stderr)` style function.

Responsibilities:
- Parse flags:
  - `-a string`: add tag by name.
  - `-d string`: delete tag by name.
  - `-u string`: update current tag name.
  - `-name string`: update target tag name.
- Validate exactly one operation among add/delete/update.
- Validate update has `-name`.
- Open SQLite via `db.NewSQLite(conf.LoadDatabase())`.
- Create `ImageRepository` and `TagRepository` using `sqliteDB.Read` and `sqliteDB.Write`.
- Execute operation:
  - Add: `TagRepository.Create(ctx, do.Tag{Name: addName})`.
  - Delete: `FindByName(ctx, deleteName)`, then `Delete(ctx, tag.ID)`.
  - Update: `FindByName(ctx, oldName)`, then `Update(ctx, do.Tag{ID: tag.ID, Name: newName})`.
- Print concise success messages that include tag names and IDs when available.

## Data Flow

1. User runs `go run ./cmd/tagctl ...`.
2. CLI parses and validates flags.
3. CLI loads database config only.
4. CLI opens the existing SQLite WAL-backed read/write pools.
5. CLI uses existing repositories for all tag mutations.
6. Repository methods enforce tag name normalization, uniqueness, `updated_at`, and delete transaction behavior.

## Delete Consistency

`TagRepository.Delete(ctx, id)` already performs:

1. `DELETE FROM image_tag WHERE tag_id = ?`
2. `DELETE FROM tag WHERE id = ?`

inside one `writeDB.Transaction`, so the CLI should use this method after resolving the tag name to ID. This satisfies the requirement that images no longer carry the deleted tag.

## Error Handling

- Invalid CLI usage returns a non-zero exit code and writes a human-readable message to stderr.
- Missing/nonexistent tag names should surface as operation failures.
- Duplicate target names during update should fail through the database unique index and return non-zero.
- Resource cleanup errors from SQLite close should be reported if they occur after a successful operation.

## Compatibility

- Existing HTTP tag API behavior remains unchanged.
- Existing `conf.Load()` behavior remains unchanged for web/sync paths.
- The new DB-only config loader is additive.

## Rollback

Rollback consists of removing the new `cmd/tagctl` package and the additive `LoadDatabase` function if the CLI is not wanted. No schema migration is needed.
