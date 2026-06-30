# 每日推荐执行计划

## Preconditions

- 当前 task 保持 `planning`，本文件通过用户 review 后才能 `task.py start`。
- 实现前运行 `trellis-before-dev`，读取后端和前端相关 spec。
- 先写/调整测试，再实现最小代码路径。

## Implementation Checklist

### 1. 后端领域与持久化模型

- [ ] 新增 `internal/model/do/daily_recommendation.go`，定义今日推荐领域对象和列表结果。
- [ ] 新增 `internal/model/po/daily_recommendation.go`，定义 `daily_recommendation`、`daily_recommendation_pool`，必要时定义 `daily_recommendation_state`。
- [ ] 新增 `internal/model/dto/daily_recommendation.go`，定义 HTTP 响应 DTO。
- [ ] 更新 `internal/infra/db/sqlite.go` AutoMigrate，包含新增 PO。

### 2. Repository 公平随机逻辑（先测试）

- [ ] 新增 repository 测试：首次请求生成今日 10 张，结果按 position 稳定。
- [ ] 新增 repository 测试：同一北京时间日期多次请求返回同一组结果。
- [ ] 新增 repository 测试：图片池大于 10 时，连续多日按洗牌周期推进，不重复抽同一小批。
- [ ] 新增 repository 测试：active 总数小于 10 时返回全部且不重复。
- [ ] 新增 repository 测试：软删除图片从新结果和补齐结果中过滤。
- [ ] 新增 repository 测试：新增/恢复 active 图片加入当前周期剩余池。
- [ ] 实现 `internal/repository/daily_recommendation.go`，使用事务维护今日结果和随机池。
- [ ] 为随机洗牌提供测试 seam：确定性 shuffler 或可注入随机源。

### 3. Service 层

- [ ] 新增 service 测试：北京时间日期边界，UTC 时间跨日但北京时间未/已跨日的场景。
- [ ] 新增 service 测试：DTO 图片 URL 拼接与 response shape。
- [ ] 实现 `internal/service/daily_recommendation.go`：默认 limit=10、timezone=Asia/Shanghai、repository 调用、DTO 转换。
- [ ] 确认 service 不依赖 GORM/PO，依赖接口。

### 4. Handler / Router / Wiring

- [ ] 新增 `internal/handler/daily_recommendation.go`，公开 `ListToday` 或 `Today` handler。
- [ ] 更新 `internal/handler/router/router.go`：`Services` 增加 DailyRecommendation service，注册 `GET /daily-recommendations`。
- [ ] 更新 `cmd/web/main.go`：创建 daily recommendation repository/service 并注入 router。
- [ ] 新增 route smoke 测试：`GET /api/v1/daily-recommendations` 返回 200、envelope、date/timezone/list shape。
- [ ] 新增 route smoke 测试：repository/service 错误时返回 500。

### 5. 前端 API Client

- [ ] 更新 `frontend/vue-gallery/src/api/types.ts`：新增 DailyRecommendation response 类型。
- [ ] 更新 `frontend/vue-gallery/src/api/client.ts`：新增 `getDailyRecommendations()`，调用 `/daily-recommendations`，导出到 `api`。
- [ ] 保持所有 type import 使用 `import type`。

### 6. 首页展示

- [ ] 更新 `frontend/vue-gallery/src/pages/GalleryPage.vue`：并行加载 images、rankings、daily recommendations。
- [ ] 将 displayable image 过滤抽成可复用 `hasDisplayableImageItem(image: ImageItem)`。
- [ ] 新增 `dailyRecommendationItems`、`dailyRecommendationError` 等状态，避免 daily 失败导致图库整页失败。
- [ ] 在社区精选附近新增“每日随机推荐”区块，展示 10 张 `ArtCard` 小网格，推荐 `selectable=false`。
- [ ] 增加 loading/empty/error 降级文案：不渲染 broken image。
- [ ] 保持移动端布局可用，不破坏现有 masonry 和 infinite scroll。

### 7. Documentation / Contracts

- [ ] 如实现中发现新的可复用约定，使用 `trellis-update-spec` 更新 `.trellis/spec/`。
- [ ] PRD convergence pass：实现前最终整理 `prd.md`，移除已解决 open questions。

## Validation Commands

后端：

```bash
go test ./...
```

前端：

```bash
cd frontend/vue-gallery && npm run build
```

可选 API smoke（服务启动后）：

```bash
curl -s http://localhost:8080/api/v1/daily-recommendations
curl -s http://localhost:8080/api/v1/rankings?period=week&size=3
```

## Review Gates

- Gate 1：repository/service 测试先通过，再接 handler/wiring。
- Gate 2：后端 `go test ./...` 通过后再做前端集成。
- Gate 3：前端 `npm run build` 通过后，运行 Trellis check / 代码审查。
- Gate 4：确认现有社区精选仍调用 `/rankings`，每日推荐调用新接口，两者文案和展示分离。

## Rollback Plan

- 后端新增内容集中在 daily recommendation 新文件、router/cmd wiring 和 AutoMigrate 新表；回滚时可移除路由和 wiring，保留未使用表不影响现有功能。
- 前端回滚可移除 `getDailyRecommendations()` 调用和首页区块，现有图库/社区精选继续工作。
- 不修改现有 ranking 表或 ranking job，避免热榜回滚风险。

## Files to Watch

- `internal/repository/image.go`：active 图片过滤规则需复用语义，不要复制出不一致条件。
- `internal/handler/router/router.go`：新增 service 字段时测试 harness 可能需要补零值处理。
- `cmd/web/main.go`：wiring 顺序保持清晰，不要启动新的后台 job；本期采用 lazy generation。
- `frontend/vue-gallery/src/pages/GalleryPage.vue`：避免 `Promise.all` 中 daily 失败导致整页失败。
- `frontend/vue-gallery/src/components/ArtCard.vue`：daily 区块使用 `selectable=false`，避免推荐区点击卡片外部触发批量选择。
