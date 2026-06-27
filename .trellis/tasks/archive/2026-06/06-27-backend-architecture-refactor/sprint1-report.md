# Sprint 1 完成报告 - 依赖 Disorder 修复

## 完成时间
2026-06-27

## 修改摘要

### 1. 创建 ports 包定义 Repository 接口契约

**新增文件：**
- `internal/ports/repositories.go` - 定义所有 Repository 接口和领域错误

**接口定义：**
- `ImageRepository` - 图片持久化访问
- `CollectionRepository` - 收藏夹持久化访问
- `TagRepository` - 标签持久化访问
- `RankingRepository` - 热榜缓存访问
- `RatingRepository` - 评分持久化访问
- `UserRepository` - 用户持久化访问
- `ImageEventRepository` - 图片事件持久化访问
- `ImageSearcher`, `ViewRecorder`, `ImageTagReader`, `ImageIndexer` - 辅助接口

**领域错误定义：**
- `ErrCollectionNotFound`, `ErrCollectionForbidden`
- `ErrImageNotFound`, `ErrTagNotFound`
- `ErrUserNotFound`, `ErrUsernameExists`

### 2. Service 层改为导入 ports 包

**修改文件：**
- `internal/service/errors.go` - 新增，集中定义 Service 层共享错误
- `internal/service/collection.go` - 移除 repository 导入，使用 ports
- `internal/service/tag.go` - 移除 repository 导入，使用 ports
- `internal/service/user.go` - 移除 repository 导入，使用 ports
- `internal/service/image.go` - 移除 ErrImageNotFound 定义，使用 ports
- `internal/service/rating.go` - 移除 ErrImageNotFound 定义

**保留的 repository 导入（仅用于类型别名）：**
- `internal/service/image.go` - `RepositoryImageQuery = repository.ImageListQuery`
- `internal/service/ranking.go` - `RepositoryRankingQuery = repository.RankingListQuery`

### 3. Repository 层错误定义改为引用 ports

**修改文件：**
- `internal/repository/collection.go` - ErrCollectionNotFound/Forbidden 引用 ports
- `internal/repository/image.go` - ErrImageNotFound 引用 ports
- `internal/repository/tag.go` - ErrTagNotFound 引用 ports
- `internal/repository/user.go` - ErrUserNotFound/UsernameExists 引用 ports

### 4. 修复 TagRepository 内部实例化问题

**修改文件：**
- `internal/repository/tag.go`
  - `TagRepository` 结构体新增 `imgRepo *ImageRepository` 字段
  - `NewTagRepository` 构造函数新增 `imgRepo` 参数
  - `FindActiveImagesWithTags` 方法使用注入的 `imgRepo` 而非内部创建

### 5. 调整 wiring 层注入

**修改文件：**
- `cmd/web/main.go` - `NewTagRepository` 调用传入 `imageRepo` 参数

### 6. 修复测试文件

**修改文件：**
- `internal/repository/tag_test.go` - 所有 `NewTagRepository` 调用添加 `imageRepo` 参数
- `internal/repository/image_test.go` - `NewTagRepository` 调用添加 `imageRepo` 参数

## 验证结果

### 编译验证
```bash
go build ./internal/...
go build ./cmd/...
```
✅ 编译通过

### 测试验证
```bash
go test ./... -short
```
✅ 所有测试通过

### 架构验证

**修复前：**
- Service 层导入 repository 包 → ORM 泄漏到领域层
- TagRepository 内部创建 ImageRepository → 违反依赖注入原则

**修复后：**
- Service 层仅导入 ports 包接口 → 领域层与基础设施解耦
- TagRepository 通过构造函数注入 ImageRepository → 符合依赖注入原则
- Repository 层错误定义引用 ports → 错误契约集中管理

## 未完成项

以下导入仍保留用于类型别名兼容性（合理）：
- `internal/service/image.go` 导入 repository 用于 `RepositoryImageQuery` 类型别名
- `internal/service/ranking.go` 导入 repository 用于 `RepositoryRankingQuery` 类型别名

这些类型别名可在后续 Sprint 中通过在 ports 包中定义查询类型来完全移除。

## 下一步

Sprint 2 任务（优先级 4-6）：
- 从 Domain Objects 中提取验证逻辑到 `internal/validators/`
- 保持 DO 对象为纯值类型
- 为创建时规范化引入 Builder 模式

## 影响

- ✅ 依赖方向正确：Service → ports ← Repository
- ✅ 可测试性提升：Service 层可使用 mock Repository
- ✅ 架构边界清晰：领域层不依赖基础设施
- ✅ 错误契约集中：所有领域错误定义在 ports 包
