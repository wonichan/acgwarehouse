# 技术设计 - 后端架构重构

## 边界

本次重构涉及以下模块边界：

- `internal/ports/` — 新增，定义 Repository 接口契约
- `internal/validators/` — 新增，提取验证逻辑
- `internal/service/` — 修改，移除对 repository 包的导入
- `internal/repository/` — 修改，修复内部实例化问题
- `internal/model/do/` — 修改，移除行为方法
- `cmd/` — 可能修改，wiring 层注入调整

## 契约

### Repository 接口契约

定义在 `internal/ports/repositories.go`：

```go
type ImageRepository interface {
    FindActiveByID(ctx context.Context, id int64) (do.Image, error)
    FindActiveByIDs(ctx context.Context, ids []int64) ([]do.Image, error)
    UpsertByCOSKey(ctx context.Context, image do.Image) (do.Image, error)
    SoftDelete(ctx context.Context, id int64, now time.Time) error
}

type TagRepository interface {
    Create(ctx context.Context, tag do.Tag) (do.Tag, error)
    FindByID(ctx context.Context, id int64) (do.Tag, error)
    ListByImageID(ctx context.Context, imageID int64) ([]do.Tag, error)
    AssignToImages(ctx context.Context, imageIDs, tagIDs []int64) ([]do.Image, error)
    UnassignFromImages(ctx context.Context, imageIDs, tagIDs []int64) ([]do.Image, error)
}
```

接口签名与现有实现保持一致，不改变行为。

### Validator 约

定义在 `internal/validators/`：

```go
type ImageValidator interface {
    ValidateForCreate(image do.Image) error
    IsActive(image do.Image) bool
}

type CollectionValidator interface {
    ValidateInput(collection do.Collection) error
    ValidateVisibility(visibility do.CollectionVisibility) error
}
```

Validator 返回 error 或 bool，不返回修改后的对象。

## 数据流

### 重构前

```
Handler → Service → Repository (imports GORM, po)
                ↓
            do.Image.IsActive() [行为在DO]
```

依赖问题：
- Service 直接导入 repository 包 → ORM 泄漏
- TagRepository 内部创建 ImageRepository → 违反 DI

### 重构后

```
Handler → Service → ports.ImageRepository (interface only)
    ↓           ↓
  wiring    implements
    ↓           ↓
   DI → Repository (GORM, po)
                ↓
         validators.ImageValidator.IsActive() [行为分离]
```

改进：
- Service 仅依赖 ports 包接口
- Repository 实现注入在 wiring 层
- 验证逻辑独立于 DO

## 权衡

### 选择：分离接口到 ports 包

**优点：**
- Service 屏蔽 ORM 细节
- 测试可 mock Repository
- 符合 Clean Architecture DIP

**代价：**
- 新增一个包，文件数量增加
- wiring 层需要显式注入
- 短期重构成本

**替代方案：**
- 在 service 包中定义接口（现状）
  - 问题：service 包仍导入 repository 包获取实现
- 使用依赖注入框架（wire, fx）
  - 过度：当前规模无需框架

**决定：** 采用 ports 包方案，手动 DI 在 main/wiring 层。

### 选择：Validator vs Builder 模式

**Validator 优点：**
- 纯函数，易测试
- 无状态，可复用
- 与 DO 解耦

**Builder 优点：**
- 流式 API，可读性好
- 可封装创建逻辑
- 支持默认值

**决定：** 优先使用 Validator，Builder 仅用于复杂创建场景（如 Image 创建）。

## 兼容性

### API 兼容性

HTTP Handler 不变，路由不变，DTO 不变。外部 API 100%兼容。

### 内部兼容性

Service 方法签名不变，但依赖来源改变（从 repository 包到 ports 包）。

Repository 实现不变，仅修改构造逻辑。

### 测试兼容性

现有测试必须继续通过：
- Repository 测试使用真实 SQLite（保持）
- Service 测试使用 memory mock（改为依赖 ports 接口）

## 推出计划

### Phase 1：依赖 Disorder 修复（无破坏）

1. 创建 `internal/ports/repositories.go`
2. Service 层改为导入 ports 包
3. Repository 实现接口（隐式实现，无需改动）
4. 修复 TagRepository 内部实例化 → 注入
5. 调整 wiring 层注入顺序
6. 运行测试验证

**回滚：** 移除 ports 包，恢复原导入。

### Phase 2：领域模型扭曲修复（无破坏）

1. 创建 `internal/validators/` 包
2. 从 DO 提取验证方法到 Validator
3. DO 移除行为方法
4. Service 调用 Validator 替代 DO 方法
5. 运行测试验证

**回滚：** 恢复 DO 方法，移除 Validator 包。

### Phase 3：测试性 Seam（可选）

1. 评估测试替身需求
2. 为需要 mock 的 Repository 创建 fake 实现
3. Service 测试使用 fake 替代 memory mock
4. 验证测试时间减少

**回滚：** 恢复原测试配置。