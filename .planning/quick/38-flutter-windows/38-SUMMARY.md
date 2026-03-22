## Summary

修复了 Flutter Windows 桌面端标签筛选功能显示占位符的问题。

### Changes Made

1. **fluent_screens.dart**
   - 添加导入：`TagProvider`, `Tag` 模型
   - 重写 `_showTagFilterDialog()` 方法
   - 添加 `_TagFilterDialogContent` StatefulWidget 实现完整的标签筛选对话框

### Features Implemented

- 标签搜索/过滤
- 标签多选（带复选框）
- 显示已选择标签数量
- 未打标签图片筛选开关
- 清空选择功能
- 应用筛选功能

### Technical Details

- 使用 Fluent UI 组件：ContentDialog, TextBox, Checkbox, ToggleSwitch, ProgressRing
- 集成 TagProvider 管理标签数据和选择状态
- 集成 ImageListProvider 应用筛选条件到图片列表
- 本地状态管理确保用户在点击"应用"前可以预览选择
