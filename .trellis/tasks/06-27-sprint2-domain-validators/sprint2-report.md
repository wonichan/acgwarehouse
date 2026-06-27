# Sprint 2 完成报告 - 领域模型验证逻辑提取

## 完成时间
2026-06-27

## 修改摘要

### 1. 分析 DO 对象行为方法

**识别的行为方法：**
- `do.Image.NormalizeForCreate()` - 设置 Status、CreatedAt、LastModified 默认值
- `do.Collection.NormalizeForCreate()` - 设置 Visibility、CreatedAt 默认值
- `do.Image.IsActive()` - 纯状态判断（保留）
- `do.CollectionVisibility.IsValid()` - 纯枚举验证（保留）
- `do.RankingPeriod.IsValid()` / `Window()` - 纯计算（保留）
- `do.UserRole.IsValid()` / `User.Public()` - 纯操作（保留）

**提取决策：**
- `NormalizeForCreate` 方法包含业务规则（默认值设置），应提取到独立的 Validator 包
- 其他方法为纯值计算或状态判断，保留在 DO 中是合理的

### 2. 创建 validators 包

**新增文件：**
- `internal/validators/normalize.go`

**实现的函数：**
```go
func NormalizeImageForCreate(image do.Image, now time.Time) do.Image
func NormalizeCollectionForCreate(collection do.Collection, now time.Time) do.Collection
```

### 3. 更新 Repository 层

**修改文件：**
- `internal/repository/image.go`
  - 导入 `internal/validators`
  - `UpsertByCOSKey` 使用 `validators.NormalizeImageForCreate()`
  
- `internal/repository/collection.go`
  - 导入 `internal/validators`
  - `Create` 使用 `validators.NormalizeCollectionForCreate()`

### 4. 移除 DO 中的 NormalizeForCreate 方法

**修改文件：**
- `internal/model/do/image.go` - 移除 `NormalizeForCreate` 方法
- `internal/model/do/collection.go` - 移除 `NormalizeForCreate` 方法

## 验证结果

### 编译验证
```bash
go build ./internal/...
```
✅ 编译通过

### 测试验证
```bash
go test ./... -short
```
✅ 所有测试通过

### 架构验证

**修复前：**
```
do.Image.NormalizeForCreate() 包含业务规则  ❌ 行为混入值对象
do.Collection.NormalizeForCreate() 包含业务规则  ❌ 行为混入值对象
```

**修复后：**
```
validators.NormalizeImageForCreate()  ✅ 业务规则独立
validators.NormalizeCollectionForCreate()  ✅ 业务规则独立
do.Image / do.Collection 为纯值类型  ✅ 无行为方法
```

## 保留的方法（合理）

以下方法保留在 DO 中，因为它们是纯值计算或状态判断：

- `Image.IsActive()` - 纯状态判断，无副作用
- `CollectionVisibility.IsValid()` - 纯枚举验证
- `RankingPeriod.IsValid()` / `Window()` - 纯计算
- `UserRole.IsValid()` - 纯枚举验证
- `User.Public()` - 返回新对象，类似值对象复制

这些方法不涉及业务规则或默认值设置，保留在 DO 中符合值对象设计原则。

## 影响

- ✅ DO 对象更纯粹：仅包含值定义和纯计算方法
- ✅ 业务规则集中：默认值设置逻辑集中在 validators 包
- ✅ 可测试性提升：验证逻辑可独立测试
- ✅ 可复用性提升：validators 函数可在多处复用

## 下一步（可选）

Sprint 3 任务（优先级 4-6）：
- 添加测试性 Seam，支持 Repository mock
- Service 测试无需数据库 fixture
