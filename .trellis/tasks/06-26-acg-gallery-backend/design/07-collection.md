# 设计 · 收藏

> 决策见 prd.md R5。表见地基 §3 (collection/collection_item)。

## 要点
- 多命名收藏夹，visibility=private|public。
- collection_item 同夹同图唯一。
- 收藏/取消：更新 image.favorite_count（**去重用户数**），发 favorite 事件。
- owner 校验：非 owner 管理返回 403；public 任何人可读，private 仅 owner。
