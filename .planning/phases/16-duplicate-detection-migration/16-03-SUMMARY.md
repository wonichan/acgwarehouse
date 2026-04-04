---
phase: 16-duplicate-detection-migration
plan: 03
subsystem: api
tags: [go, python-sidecar, duplicate-detection, orchestration, handler]

requires:
  - phase: 16-01
    provides: Python 侧重复检测异步任务接口与计算结果结构
  - phase: 16-02
    provides: Go 侧 schema/domain/repository 对 phash_hex 与 recommendation 的承载能力
provides:
  - Go sidecar HTTP 客户端（submit/poll/fetch）
  - DuplicateService 编排链路（提交任务、轮询进度、拉取结果、落库）
  - Handler 侧车状态预检查与 503 可诊断返回
  - 旧 Go 计算实现删除与 goimagehash 依赖移除
affects: [phase-17, duplicate-handler, duplicate-service, sidecar-runtime]

tech-stack:
  added: []
  patterns: [go-orchestrates-python-compute, sync-api-async-internal, persist-phash-then-replace-groups]

key-files:
  created: []
  modified:
    - internal/sidecar/client.go
    - internal/sidecar/client_test.go
    - internal/service/duplicate_service.go
    - internal/service/duplicate_service_test.go
    - internal/handler/duplicate_handler.go
    - internal/handler/duplicate_handler_test.go
    - internal/handler/routes.go
    - internal/app/bootstrap.go
    - internal/app/app.go
    - test/e2e/duplicate_test.go
    - go.mod
    - go.sum
  deleted:
    - internal/service/hash_service.go
    - internal/service/hash_service_test.go

key-decisions:
  - "Go 对外保持同步 `/api/v1/duplicates/detect`，内部通过 submit→poll→fetch 编排 Python 异步任务。"
  - "仅在成功拉取 Python 结果并回写 phash_hex 后，才删除并重建 duplicate_groups，避免失败时清空历史结果。"
  - "Handler 在检测前强制检查 sidecar runtime 就绪态，未就绪直接 503 返回状态与诊断详情。"

patterns-established:
  - "侧车调用统一经 `internal/sidecar/client.go` 收敛，HTTP 错误统一带状态码诊断。"
  - "推荐依据以结构化 JSON 从 Python 透传到 API，不再输出转义字符串。"

requirements-completed: [COMP-03, COMP-04]

duration: 14 min
completed: 2026-04-04
---

# Phase 16 Plan 03: Go↔Python 重复检测编排接线总结

**Go 已完成重复检测链路迁移：由 Python 侧车负责计算，Go 负责任务编排与持久化，并在侧车不可用时提供可诊断 503。**

## Performance

- **Duration:** 14 min
- **Started:** 2026-04-04T14:22:12+08:00
- **Completed:** 2026-04-04T14:36:42+08:00
- **Tasks:** 3
- **Files modified:** 14

## Accomplishments
- 新增 `internal/sidecar/client.go`，实现 `SubmitDetection` / `PollProgress` / `FetchResults`，并补齐 409/4xx/5xx 诊断处理。
- 重构 `internal/service/duplicate_service.go`：改为 sidecar 编排模式，回写 `phash_hex`，持久化 recommendation 字段，并确保失败时不清空旧重复组。
- 删除旧 Go 计算路径：移除 `internal/service/hash_service.go` 与 `goimagehash` 依赖。
- 更新 handler 与路由接线：检测前检查 `sidecarRuntime.State()==ready`，未就绪返回 `503` 与 `state/details`。
- 保持对外 API 合约不变：`POST /api/v1/duplicates/detect` 仍返回同步结果（`message` + `groups_found`）。

## Task Commits

Each task was committed atomically:

1. **Task 1: 创建 sidecar HTTP 客户端** - `56cfa18` (feat)
2. **Task 2: 重构服务并删除旧计算代码** - `b345dcc` (feat)
3. **Task 3: handler 预检查与契约测试补齐** - `84e3b74` (test)

**Plan metadata:** pending final docs commit

## Files Created/Modified
- `internal/sidecar/client.go` - Go→Python 重复检测接口客户端。
- `internal/service/duplicate_service.go` - sidecar 编排、结果落库、phash_hex 回写。
- `internal/handler/duplicate_handler.go` - 侧车状态预检查、阈值规则更新（40/256）。
- `internal/app/bootstrap.go` - 侧车 baseURL/client 初始化与 runtime launcher/probe 接线。
- `internal/app/app.go` / `internal/handler/routes.go` - SidecarRuntime 依赖注入与 handler 构造更新。
- `test/e2e/duplicate_test.go` - e2e 场景改为 sidecar mock 驱动。
- `go.mod` / `go.sum` - 删除 `github.com/corona10/goimagehash`。

## Decisions Made
- 维持同步 API、异步内部实现的边界，避免对 Flutter/现有客户端引入额外协议变更。
- 将“删除旧重复组”后置到结果 fetch + phash 回写成功之后，避免失败路径下数据回退问题。

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- 并行 `git add` 触发过一次 `.git/index.lock` 竞争，已改为串行 staging，未影响代码与测试结果。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 16 的三份计划（16-01/16-02/16-03）已全部完成，可进入 Phase 17。
- 重复检测主链路已稳定切换到 Python 计算侧车，后续 UI Phase 可直接消费结构化 recommendation 数据。

## Self-Check: PASSED
- `go test ./internal/... -count=1` 通过。
- `go test ./test/e2e/... -run Duplicate -count=1 -v` 通过。
- `go mod tidy && go build ./...` 通过。
