# Implement: Similar Image Recommendations

## 执行清单

### 后端

- [ ] **B1. 新增仓储接口方法**
  - 文件：`internal/ports/repositories.go`、`internal/service/image.go`
  - 在两个 `ImageRepository` 接口中添加 `FindSimilarByTagIDs` 和 `FindSimilarByCategory` 方法签名（保持同步）。
  - 验证：`go build ./...` 会因 repository 未实现而失败（预期，B2 修复）。

- [ ] **B2. 实现仓储方法**
  - 文件：`internal/repository/image.go`
  - 实现 `FindSimilarByTagIDs`：JOIN image_tag + GROUP BY image.id + ORDER BY COUNT DESC, view_count DESC + LIMIT。
  - 实现 `FindSimilarByCategory`：WHERE category + NOT IN excludeIDs + ORDER BY view_count DESC + LIMIT。
  - 边界：空入参返回空切片、`activeImages` 限定、`limit <= 0` 返回空。
  - 验证：`go build ./...` 通过。

- [ ] **B3. 重构 newDetailResponse 编排**
  - 文件：`internal/service/image.go`
  - 新增 `const similarImageLimit = 6`。
  - 新增 `findSimilarImages(ctx, image, tagIDs, limit)` 编排方法。
  - 重构 `newDetailResponse`：复用 `tags.ListByImageID` 单次查询派生 tagNames + tagIDs；调用 `findSimilarImages` 填充 `SimilarImages`。
  - `imageTagNames` 改为 `imageTags`（返回 `[]do.Tag`），或保留 `imageTagNames` 并新增 `imageTagIDs`。
  - 新增 `toImageResponseList([]do.Image) []dto.ImageResponse` 辅助方法（或复用 `newListResult` 内联逻辑）。
  - 验证：`go build ./...` 通过。

- [ ] **B4. 后端单测**
  - 文件：`internal/repository/image_test.go`（已存在，追加）
  - `TestFindSimilarByTagIDs`：多图多标签场景，验证按重叠数排序、排除自身、软删除不出现、空 tagIDs 返回空。
  - `TestFindSimilarByCategory`：同 category 排序、excludeIDs 去重、空 category 返回空。
  - 文件：`internal/service/image_test.go`（若不存在则新建）
  - Mock `ImageRepository`（实现新方法）+ Mock `ImageTagReader`。
  - 测试 `findSimilarImages`：标签重叠足够（不触发回退）、不足（触发 category 回退 + 去重）、无标签无 category（返回空）。
  - 验证：`go test ./internal/repository/... ./internal/service/... -run Similar -v` 通过。

- [ ] **B5. 全量后端测试**
  - 命令：`go test ./...`
  - 期望：全部通过（含已有测试）。

### 前端

- [ ] **F1. SimilarImagesPanel 清理 + moreLinkTag**
  - 文件：`frontend/vue-gallery/src/components/SimilarImagesPanel.vue`
  - 空状态文案：`"暂无相似作品"` / `"还没有相关的作品推荐"`（去除 `"后端返回 similar_images 为空时显示此状态"`）。
  - 新增 prop `moreLinkTag?: string`。
  - 「更多」链接：`moreLinkTag` 非空时 `to="/?tag=<encoded>"`，否则 `to="/search"`。用 `computed` 计算。
  - 验证：`npm run build` 类型通过。

- [ ] **F2. DetailPage 传递 moreLinkTag**
  - 文件：`frontend/vue-gallery/src/pages/DetailPage.vue:274`
  - 改为 `<SimilarImagesPanel :images="detail.similar_images" :more-link-tag="detail.tags[0]" />`。
  - 验证：`npm run build` 类型通过。

- [ ] **F3. GalleryPage tag 深链支持**
  - 文件：`frontend/vue-gallery/src/pages/GalleryPage.vue`
  - 新增 `import { useRoute } from 'vue-router'` + `const route = useRoute()`。
  - 读取 `route.query.tag`（string），存 `activeTag` ref。
  - `galleryImageQuery` 增加 `tag` 字段：`activeTag.value` 非空时传入。
  - `onMounted`/`onActivated` 触发加载时带上 tag。
  - 监听 `route.query.tag` 变化重新加载（watch）。
  - 验证：`npm run build` 类型通过。

- [ ] **F4. 全量前端构建**
  - 命令：`cd frontend/vue-gallery && npm run build`
  - 期望：构建通过，无类型错误。

## 验证命令汇总

```bash
# 后端
go build ./...
go test ./...

# 前端
cd frontend/vue-gallery && npm run build
```

## 风险文件 / 回滚点

- `internal/service/image.go`：`newDetailResponse` 是详情接口核心路径，改动需保证原有 tags/avg_score 等字段不受影响。回滚点：保留原 `imageTagNames` 逻辑可快速恢复。
- `internal/ports/repositories.go` + `internal/service/image.go` 接口：新增方法后所有 mock 实现须同步，否则编译失败。
- `frontend/vue-gallery/src/pages/GalleryPage.vue`：加 `useRoute` + watch 需注意 keep-alive `onActivated` 场景，避免重复加载。

## Review Gates

- B2 完成后：`go build ./...` 通过（接口实现完整）。
- B3 完成后：`go build ./...` 通过（service 编译通过）。
- B5 完成后：`go test ./...` 通过（后端全绿）。
- F4 完成后：`npm run build` 通过（前端全绿）。
- 全部完成后：`trellis-check` 子代理做 spec 合规 + 跨层数据流检查。
