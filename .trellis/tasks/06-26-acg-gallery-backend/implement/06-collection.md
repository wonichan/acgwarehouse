## 阶段 6 · 收藏

- [ ] po/do/dto: collection、collection_item；repository。
- [ ] service/collection：收藏夹 CRUD（owner 校验）、可见性、加入/移除图片（同图唯一）、favorite_count 去重更新 + favorite 事件、浏览公开收藏夹。
- [ ] handler/collection.go + 路由。
- 验证：AC-R5（私有/公开可见性、非 owner 403、去重收藏数）。

- [ ] **codegraph sync**：本阶段完成后同步索引（`codegraph sync`）。
