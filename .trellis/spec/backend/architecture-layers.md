# Backend Architecture Layers

> 架构分层与依赖方向规范，基于 Sprint 1-2 重构经验。

---

## 分层架构

```
cmd/                    # 应用入口，wiring 层
  └── main.go           # 依赖注入组装
handler/                # HTTP 处理层
  └── *.go              # 请求解析、响应序列化
service/                # 业务逻辑层
  └── *.go              # 用例编排，不依赖基础设施
ports/                  # 接口契约层（新增）
  └── repositories.go   # Repository 接口定义
repository/             # 基础设施层
  └── *.go              # GORM 实现，DB 访问
validators/             # 业务规则层（新增）
  └── normalize.go      # 创建时规范化逻辑
model/
  ├── do/               # 领域对象（纯值类型）
  ├── po/               # 持久化对象
  └── dto/              # 数据传输对象
```

---

## 依赖方向

### 正确的依赖方向

```
handler → service → ports (接口)
               ↑
          implements
               |
          repository (GORM)
               |
          validators
```

**原则**：
- Service 层依赖 `ports` 包的接口，不依赖 `repository` 实现
- Repository 层实现 `ports` 包的接口
- 领域层（service/ports/do）不依赖基础设施层（repository）

### 禁止的反向依赖

```go
// ❌ 错误：Service 导入 repository 包
import "github.com/yachiyo/acgwarehouse/internal/repository"

// ✅ 正确：Service 导入 ports 包接口
import "github.com/yachiyo/acgwarehouse/internal/ports"
```

---

## Ports 包模式

### 接口定义

`internal/ports/repositories.go`:

```go
package ports

// ImageRepository 定义图片持久化访问接口。
type ImageRepository interface {
    FindByID(id int64) (do.Image, error)
    UpsertByCOSKey(image do.Image) (do.Image, error)
    // ...
}

// 领域错误定义在 ports 包
var (
    ErrImageNotFound      = errors.New("image not found")
    ErrCollectionNotFound = errors.New("collection not found")
)
```

### Repository 实现

`internal/repository/image.go`:

```go
package repository

import "github.com/yachiyo/acgwarehouse/internal/ports"

type ImageRepository struct {
    db *gorm.DB
}

// 确保 ImageRepository 实现 ports.ImageRepository 接口
var _ ports.ImageRepository = (*ImageRepository)(nil)

func (r *ImageRepository) FindByID(id int64) (do.Image, error) {
    // GORM 实现...
}
```

### Service 使用

`internal/service/image.go`:

```go
package service

import "github.com/yachiyo/acgwarehouse/internal/ports"

type ImageService struct {
    repo ports.ImageRepository  // 依赖接口，不依赖实现
}
```

---

## 依赖注入

### 禁止内部实例化

```go
// ❌ 错误：Repository 内部创建另一个 Repository
type TagRepository struct {
    db *gorm.DB
}

func (r *TagRepository) FindActiveImagesWithTags() {
    imgRepo := repository.NewImageRepository(r.db)  // 禁止！
}

// ✅ 正确：通过构造函数注入
type TagRepository struct {
    db       *gorm.DB
    imgRepo  *ImageRepository  // 注入依赖
}

func NewTagRepository(db *gorm.DB, imgRepo *ImageRepository) *TagRepository {
    return &TagRepository{db: db, imgRepo: imgRepo}
}
```

### Wiring 层组装

`cmd/web/main.go`:

```go
func main() {
    db := setupDatabase()
    
    // 构造依赖链
    imageRepo := repository.NewImageRepository(db)
    tagRepo := repository.NewTagRepository(db, imageRepo)
    imageService := service.NewImageService(imageRepo)
    
    // 注入到 Handler
    handler := handler.NewImageHandler(imageService)
}
```

---

## 领域对象规范

### DO 对象为纯值类型

```go
// ✅ 正确：DO 仅包含值定义
type Image struct {
    ID       int64
    COSKey   string
    Status   ImageStatus
    // ...
}

// ✅ 正确：纯状态判断可保留
func (i Image) IsActive() bool {
    return i.Status == "" || i.Status == ImageStatusActive
}
```

### 业务规则提取到 Validators

```go
// ❌ 错误：DO 包含业务规则
func (i *Image) NormalizeForCreate() {
    i.Status = ImageStatusActive  // 默认值逻辑
    i.CreatedAt = time.Now()
}

// ✅ 正确：业务规则在独立包
// internal/validators/normalize.go
func NormalizeImageForCreate(image do.Image, now time.Time) do.Image {
    if image.Status == "" {
        image.Status = do.ImageStatusActive
    }
    if image.CreatedAt.IsZero() {
        image.CreatedAt = now
    }
    return image
}

// Repository 使用
func (r *ImageRepository) UpsertByCOSKey(image do.Image) (do.Image, error) {
    image = validators.NormalizeImageForCreate(image, time.Now())
    // GORM 操作...
}
```

---

## 类型别名兼容

Service 层保留类型别名导入是可接受的：

```go
// service/image.go
import (
    "github.com/yachiyo/acgwarehouse/internal/ports"
    "github.com/yachiyo/acgwarehouse/internal/repository"  // 仅用于类型别名
)

// 类型别名（不引入运行时依赖）
type RepositoryImageQuery = repository.ImageListQuery
```

**原因**：避免大量类型定义重复，且仅用于类型声明，不引入运行时依赖。

---

## 测试性 Seam

### Service 测试无需数据库

```go
// service/image_test.go
func TestImageService(t *testing.T) {
    // 使用 memory mock 实现 ports.ImageRepository
    mockRepo := &mockImageRepository{...}
    svc := service.NewImageService(mockRepo)
    
    // 测试不依赖真实数据库
}
```

---

## 设计决策记录

### Sprint 1: 为什么创建 ports 包？

**问题**：Service 层直接导入 repository 包，ORM 泄漏到领域层。

**方案**：
1. 直接在 Service 中定义接口 ❌ 接口分散
2. 创建独立的 ports 包 ✅ 接口集中管理

**决策**：创建 `internal/ports/` 包集中定义所有 Repository 接口。

**好处**：
- 接口与实现分离
- Service 测试可使用 mock
- 领域层不依赖基础设施

### Sprint 2: 为什么提取 NormalizeForCreate？

**问题**：DO 对象包含默认值设置逻辑，违反值对象原则。

**方案**：
1. 保留在 DO 中 ❌ DO 变成有行为的对象
2. 移到 Service 层 ❌ Service 承担过多职责
3. 创建独立 validators 包 ✅ 业务规则集中

**决策**：创建 `internal/validators/` 包，提取创建时规范化逻辑。

---

## 验收标准

重构完成后应满足：

- [ ] Service 层不导入 `internal/repository` 包（类型别名除外）
- [ ] Repository 通过构造函数注入依赖
- [ ] DO 对象无业务规则方法
- [ ] 所有测试通过
- [ ] Health score >= 80

---

## 相关文档

- [Go Naming and Style](./go-naming-and-style.md)
- [Go Error Handling](./go-error-handling.md)
- [Go Best Practices](./go-best-practices.md)
