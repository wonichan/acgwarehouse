# 计划：修复Flutter缩略图不显示问题

## 问题根因分析
根据代码分析，可能原因：
1. **空字符串问题**：后端使用 `COALESCE(thumbnail_small_url, '')` 返回空字符串，Flutter只检查 `null || isEmpty`
2. **URL协议头缺失**：bucketURL可能缺少 `https://` 前缀
3. **缩略图URL未保存**：数据库中字段可能为空

## 任务列表

### 任务1：修复后端返回空字符串问题
- **文件**: `internal/repository/image_repository.go`
- **问题**: `COALESCE(thumbnail_small_url, '')` 返回空字符串而非NULL
- **修复**: 将 `COALESCE(thumbnail_small_url, '')` 改为直接返回字段值（允许NULL）
- **验证**: 重新执行缩略图生成任务后，检查数据库字段是否为有效URL

### 任务2：修复URL协议头问题
- **文件**: `internal/service/cos_service.go`
- **问题**: bucketURL可能缺少 `https://` 前缀
- **修复**: 确保Upload返回的完整URL包含协议头
- **验证**: 检查COS上缩略图的完整访问URL

### 任务3：验证Flutter端显示逻辑
- **文件**: `flutter_app/lib/widgets/image_grid.dart`
- **确认**: CachedNetworkImage 的 placeholder 和 errorWidget 配置正确
- **验证**: 检查实际网络请求是否正确发出

## 执行顺序
1. 先修复任务1（后端空字符串问题）
2. 重新运行缩略图生成任务
3. 检查数据库字段是否有正确的URL
4. 如果URL正确但仍不显示，检查任务2和3