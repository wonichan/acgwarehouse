# Repository Guidelines

## Project Structure & Module Organization

ACGWarehouse is a Go backend with a Flutter client. Backend entry points live in `cmd/server`, `cmd/scan`, and `cmd/migrate-thumbnails`. Core backend code is under `internal/`: `domain` for models, `repository` for SQLite access, `service` for business logic, `handler` for Gin HTTP/WebSocket routes, `worker` for background jobs, and `ai` for provider integrations. SQL migrations are in `migrations/`. The Flutter app is in `flutter_app/lib`, with tests in `flutter_app/test`. Static admin assets are in `web/admin`; Go e2e and performance tests are under `test/`. Deployment files are in `deploy/`, `Dockerfile`, and `docker-compose.yml`.

## Build, Test, and Development Commands

- `make run` or `go run ./cmd/server`: run the backend locally.
- `make build`: build the backend binary to `bin/server`.
- `make test` or `go test ./...`: run all Go tests.
- `go test ./test/perf/... -run ^$ -bench . -benchmem -count=1`: run performance benchmarks.
- `cd flutter_app && flutter pub get`: install Flutter dependencies.
- `cd flutter_app && flutter test`: run Flutter tests.
- `cd flutter_app && flutter analyze`: run Dart analysis with `flutter_lints`.
- `docker compose up -d`: start the packaged stack; `docker compose logs -f` follows logs.

## Coding Style & Naming Conventions

Format Go code with `gofmt`; keep packages aligned with the existing `internal` layers. Use lower-case Go package names, exported identifiers only for cross-package APIs, and `_test.go` for tests. Dart follows `flutter_lints`; use two-space indentation, `snake_case.dart` filenames, `PascalCase` classes/widgets, and `camelCase` members. Avoid editing generated Flutter platform files unless platform behavior is the target change.

## Testing Guidelines

Place Go unit tests beside implementation as `*_test.go`; use `test/e2e` for end-to-end flows and `test/perf` for benchmarks. Prefer focused repository/service/handler tests over broad setup-heavy tests. Flutter tests belong in `flutter_app/test`, mirroring `lib` paths where practical. Run `go test ./...` and relevant `flutter test` suites before submitting backend/client changes.

## Commit & Pull Request Guidelines

History uses Conventional Commit-style subjects such as `feat: ...`, `fix: ...`, and `feat(tag): ...`. Keep subjects imperative and specific. Pull requests should describe the behavior change, list test commands run, link issues or design notes from `docs/superpowers`, and include screenshots or recordings for visible Flutter/admin UI changes.

## Security & Configuration Tips

Do not commit real API keys, storage credentials, or local library paths. Use `deploy/config/config.example.yaml` as the template and keep local overrides in ignored config files. Treat files under `data/` as runtime state, not source fixtures, unless a test explicitly requires them.
