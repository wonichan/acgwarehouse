# 快速任务 40：Flutter 桌面端补齐“收藏”入口与查看页

**日期：** 2026-04-08  
**描述：** 后端与右键“收藏到合集”流程已存在，但 Windows/Fluent 桌面左侧导航没有“收藏”入口，用户无法查看已收藏内容。

## 现状证据

- `flutter_app/lib/app/fluent_app_shell.dart` 使用 `NavigationPane` 渲染桌面左侧菜单，当前没有“收藏”项。
- `flutter_app/lib/providers/navigation_provider.dart` 集中维护导航索引与标题。
- `flutter_app/lib/widgets/fluent_gallery_content.dart` 的右键菜单包含 `收藏`，会打开 `ImageCollectionPickerDialog`。
- `flutter_app/lib/widgets/image_collection_picker_dialog.dart` 允许选择已有合集或新建合集后收藏。
- `flutter_app/lib/services/collection_service.dart` 已支持：
  - `fetchCollections()`
  - `fetchCollectionImages()`
  - `createCollection()`
  - `addImageToCollection()`
- 仓库中未发现现成的合集/收藏浏览页。

## 关键判断

- 当前产品语义是“收藏到合集”，不是单独的 favorites 存储模型。
- 因此最小正确修复不是新增后端 favorites 概念，而是把现有合集能力作为“收藏”入口暴露出来。
- 为避免打断现有 `NavigationProvider` 常量语义，新增导航项优先采用追加方式。

## 目标

1. 在 Flutter Windows/Fluent 左侧导航中新增“收藏”入口。
2. 新增一个可查看合集及合集内图片的桌面页面。
3. 保持右键收藏流程与现有 collection API 不变。
4. 在可能的低成本范围内保持 Material 侧导航/索引一致性，避免共享导航状态语义漂移。

## 非目标

- 不新增后端 favorites 表或专属 favorites API。
- 不重构现有收藏到合集的对话框流程。
- 不扩展到复杂的合集管理（重命名、删除、拖拽排序等）。
