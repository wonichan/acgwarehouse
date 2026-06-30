# 每日推荐技术设计

## 范围

本设计实现一个公开的“今日每日推荐”读取能力：全站统一、北京时间自然日稳定、每天最多 10 张、来源为 active 图片池的公平随机选择。它不替换现有热榜/社区精选，也不暴露历史每日推荐查询。

## 架构边界

- HTTP 层：新增公开只读路由，负责 query 解析、调用 service、返回统一 `Response{code,data,msg}`。
- Service 层：实现今日日期边界、默认数量、DTO 转换、错误语义；依赖 repository 接口，不依赖 GORM 细节。
- Repository 层：实现公平随机池、今日结果持久化、active 图片过滤与并发安全事务。
- Model 层：新增 DO/PO/DTO 类型，避免 `po` 穿透到 service/handler 响应。
- Frontend API 层：新增 `getDailyRecommendations()`，走 `/api/v1` 相对路径和现有 envelope/list 解析。
- Frontend 页面层：`GalleryPage.vue` 首页新增独立区块，与社区精选轮播并存。

## API Contract

### Endpoint

`GET /api/v1/daily-recommendations`

公开无 Auth。本期不支持按日期查询；如果传入未来扩展参数，当前实现不需要消费。

### Response

外层沿用：

```json
{
  "code": 0,
  "data": {
    "date": "2026-06-30",
    "timezone": "Asia/Shanghai",
    "total": 10,
    "list": [
      {
        "id": 1,
        "cos_key": "thumbnails/example.png",
        "filename": "example.png",
        "url": "https://cdn.example.com/thumbnails/example.png",
        "size": 123,
        "last_modified": "2026-06-29T00:00:00Z",
        "width": 800,
        "height": 1200,
        "category": "...",
        "avg_score": 80,
        "rating_count": 1,
        "favorite_count": 2,
        "view_count": 3,
        "created_at": "2026-06-29T00:00:00Z"
      }
    ]
  },
  "msg": ""
}
```

`total` 表示本次返回 `list` 长度，不表示全站图片总数。

## Persistence Model

新增两类持久化对象：

1. `daily_recommendation`
   - `date` string，主键组成部分，格式 `YYYY-MM-DD`，以 Asia/Shanghai 计算。
   - `image_id` int64，主键组成部分。
   - `position` int，今日展示顺序，1-based。
   - `cycle` int64，生成结果时所属随机周期。
   - `created_at` time.Time。
   - 外键/关联到 `image`，查询时 join active 图片。

2. `daily_recommendation_pool`
   - `image_id` int64，主键。
   - `cycle` int64，所属随机周期。
   - `position` int，当前周期内洗牌后的顺序。
   - `created_at` time.Time。

可选新增 `daily_recommendation_state` 保存当前 `cycle`。若不建 state 表，可通过 `max(cycle)` 从 pool/result 推导，但 state 表更清晰。若实现 state 表：

- `key` string primary key，例如 `global`。
- `cycle` int64。
- `updated_at` time.Time。

所有新增 PO 需要加入 `internal/infra/db/sqlite.go` 的 `AutoMigrate`。

## Fair Random Algorithm

### Date boundary

- Service 使用 `time.LoadLocation("Asia/Shanghai")`。
- 今日 key 使用 `now.In(location).Format("2006-01-02")`。
- 数据库时间戳仍用 UTC 存储。

### Read path

`DailyRecommendationService.Today(ctx, now)`：

1. 计算北京时间 date key。
2. 调用 repository `GetOrCreateToday(ctx, date, limit=10, nowUTC)`。
3. Repository 如果已有该 date 的 active 今日结果，按 `position` 返回。
4. 如果没有结果或结果因软删除过滤后不足 10，则在写事务内生成/补齐今日结果。
5. Service 将 DO 转 DTO，拼接 COS URL。

### Pool generation

Repository 在事务内维护一个“洗牌袋”：

1. 查询全部 active image IDs，稳定排序作为输入集合。
2. 查询当前 cycle 的剩余 pool rows，并删除不再 active 的 pool rows。
3. 将新增/恢复的 active image IDs 加入当前 cycle 剩余 pool：
   - 不重置已有 pool；
   - 新增 ID 使用随机 position 插入或附加后再对剩余池洗牌；
   - 必须保证新增 ID 有机会在当前周期被抽中。
4. 从剩余 pool 随机/洗牌顺序取最多 `needed` 张，写入 `daily_recommendation`，并从 pool 删除这些 image IDs。
5. 如果剩余 pool 不足 `needed`，开启 `cycle + 1`：
   - 用当前 active image IDs 重新洗牌生成新 pool；
   - 继续取剩余 needed 张；
   - 新周期不排除上一周期末尾未抽完以外的 active 图，因为上一周期已经耗尽或不足以满足当日数量。
6. 如果 active 图片总数小于 10，则返回全部 active 图片，不无限循环补重复。

随机性要求：使用 Go 标准库随机洗牌即可，但实现必须支持测试中注入确定性随机源或 shuffler，避免测试依赖概率。

### Soft delete / restore behavior

- 今日结果读取时 join/filter active 图片。若今日结果中的图片已软删除，返回时不展示该图片。
- 如果过滤后当日结果不足 10，允许同一 date 补齐新的 active 图片。
- Restore 后图片重新进入 active 集合，并在下一次生成/补齐时加入当前周期剩余池。

## Backend Files Likely Touched

- `internal/model/do/daily_recommendation.go`
- `internal/model/po/daily_recommendation.go`
- `internal/model/dto/daily_recommendation.go`
- `internal/ports/repositories.go` or service-local interface, consistent with current patterns.
- `internal/repository/daily_recommendation.go`
- `internal/service/daily_recommendation.go`
- `internal/handler/daily_recommendation.go`
- `internal/handler/router/router.go`
- `internal/infra/db/sqlite.go`
- `cmd/web/main.go`
- tests under `internal/repository`, `internal/service`, `internal/handler/router`

## Frontend Design

Detailed planning prototype: `frontend-prototype.md` in this task directory. That file is the source of truth for section placement, desktop/tablet/mobile wireframes, visual copy, loading/empty/error states, and the optional image-generation prompt.

### API types/client

- Add `DailyRecommendationResponse` or `DailyRecommendationListResponse` in `frontend/vue-gallery/src/api/types.ts`:
  - `date: string`
  - `timezone: 'Asia/Shanghai' | string`
  - `total: number`
  - `list: readonly ImageItem[]`
- Add `getDailyRecommendations(): Promise<DailyRecommendationListResponse>` in `client.ts`.
- Export method via `api` object.

### GalleryPage

- Load daily recommendations in parallel with existing `getImages` and `getRankings` during `loadGallery()`.
- Keep failures scoped: if daily recommendations fail but images/rankings succeed, the page should not necessarily become full-page error. Recommended behavior:
  - Existing gallery failure remains full-page error.
  - Daily block can show an inline empty/error state.
- Map `ImageItem` to existing `ArtItem` via `imageToArtItem`.
- Filter displayable images with the same image display predicate used for rankings, generalized to `ImageItem`.
- Render a new section near the community carousel:
  - eyebrow: `北京时间今日更新`
  - title: `每日随机推荐`
  - copy: explains random exploration/fair rotation, not heat ranking.
  - layout: responsive grid or horizontal card list showing up to 10 `ArtCard` items with `selectable=false` unless product wants batch-selection there.

### Styling

- Reuse existing panel/card/masonry primitives where possible.
- Do not modify `Carousel.vue` unless needed; daily recommendations are a separate card grid, not a second carousel.
- Maintain desktop/tablet/mobile responsive layout.

## Compatibility / Migration

- Existing `/api/v1/rankings` behavior remains unchanged.
- AutoMigrate creates new tables on startup; no manual migration script required in current project pattern.
- If table is empty, first request to `/daily-recommendations` lazily creates today's result.
- No config/env variable required for MVP; count and timezone are constants in service design unless later made configurable.

## Risks and Mitigations

- **Concurrent first requests for the same day**: use a DB transaction and primary key on `(date,image_id)` plus position uniqueness or idempotent re-read after conflict.
- **Random fairness tests flake**: inject deterministic shuffler/random source in service/repository tests.
- **Small image pools**: cap at active count and never duplicate image IDs in one day's result.
- **Deleted images in cached today result**: always join/filter active images on read and allow same-day补齐.
- **Frontend load coupling**: daily recommendation error should be isolated so core gallery remains usable.
