# Phase 2: 缩略图、基础浏览与 AI 复核界面底座 - Research

**研究时间：** 2026-03-14
**状态：** 完成

## 研究摘要

本研究覆盖 Phase 2 实现所需的核心技术栈：Go 缩略图生成与感知哈希、腾讯云 COS 存储集成、Flutter 前端框架搭建、图片网格/瀑布流组件、图片缓存与缩放交互。

---

## 标准技术栈

### 后端 Go 库

| 库 | 用途 | 版本建议 | 状态 |
|---|------|---------|------|
| `disintegration/imaging` | 图片缩放、格式转换、JPEG 质量控制 | latest | 推荐 ✓ |
| `corona10/goimagehash` | 感知哈希计算 (pHash) | latest | 推荐 ✓ |
| `tencentyun/cos-go-sdk-v5` | 腾讯云 COS 上传 | latest | 推荐 ✓ |

### Flutter 包

| 包 | 用途 | 版本建议 | 状态 |
|---|------|---------|------|
| `flutter_staggered_grid_view` | 瀑布流布局 (MasonryGridView) | ^0.7.0 | 推荐 ✓ |
| `cached_network_image` | 网络图片缓存 | ^3.4.0 | 推荐 ✓ |
| `extended_image` | 图片缩放/拖动交互 | ^10.0.0 | 推荐 ✓ |
| `http` | HTTP 请求 | ^1.2.0 | 推荐 ✓ |
| `provider` | 状态管理 | ^6.1.0 | 推荐 ✓ |

---

## 架构模式

### 1. 缩略图生成服务

**实现模式：** 按需生成 + COS 上传

```go
// 使用 disintegration/imaging 进行高质量缩放
import "github.com/disintegration/imaging"

// 缩略图生成流程
func GenerateThumbnail(src image.Image, targetWidth int, quality int) ([]byte, error) {
    // 使用 Lanczos 滤镜进行高质量缩放
    resized := imaging.Resize(src, targetWidth, 0, imaging.Lanczos)
    
    // 编码为 JPEG 并控制质量
    var buf bytes.Buffer
    err := imaging.Encode(&buf, resized, imaging.JPEG, imaging.JPEGQuality(quality))
    return buf.Bytes(), err
}
```

**关键决策：**
- 小缩略图 (~200px): JPEG 质量 85
- 大缩略图 (~600px): JPEG 质量 90
- 使用 Lanczos 滤镜确保二次元图片边缘清晰

### 2. 感知哈希计算服务

**实现模式：** 导入时同步计算，存入 images.phash 字段

```go
import "github.com/corona10/goimagehash"

func ComputePHash(img image.Image) (uint64, error) {
    // 计算 pHash (对压缩、缩放、颜色变化鲁棒性强)
    hash, err := goimagehash.PerceptionHash(img)
    if err != nil {
        return 0, err
    }
    return hash.GetHash(), nil
}

// 相似度比较
func CompareImages(hash1, hash2 uint64) int {
    h1 := goimagehash.NewImageHash(hash1, goimagehash.PHash)
    h2 := goimagehash.NewImageHash(hash2, goimagehash.PHash)
    distance, _ := h1.Distance(h2)
    return distance // 距离越小越相似，默认阈值 12
}
```

**关键决策：**
- 主算法：pHash (Perception Hash)
- 默认相似阈值：12 (用户可调整)
- 存储位置：`images.phash` (int64/uint64)

### 3. 腾讯云 COS 上传

**实现模式：** Go SDK 直接上传

```go
import "github.com/tencentyun/cos-go-sdk-v5"

// COS 客户端初始化
func NewCOSClient(bucketURL, secretID, secretKey string) *cos.Client {
    u, _ := url.Parse(bucketURL)
    b := &cos.BaseURL{BucketURL: u}
    return cos.NewClient(b, &http.Client{
        Transport: &cos.AuthorizationTransport{
            SecretID:  secretID,
            SecretKey: secretKey,
        },
    })
}

// 上传缩略图
func UploadThumbnail(client *cos.Client, key string, data []byte) error {
    ctx := context.Background()
    _, err := client.Object.Put(ctx, key, bytes.NewReader(data), &cos.ObjectPutOptions{
        ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
            ContentType: "image/jpeg",
        },
    })
    return err
}
```

**存储桶信息：**
- 存储桶：`acgwarehouse-1301393037`
- 域名：`literal:${COS_BUCKET_URL:-}`
- 对象 Key 格式：`thumbnails/{image_id}_{size}.jpg`

### 4. Flutter 前端架构

**项目结构：**
```
lib/
├── main.dart
├── app.dart
├── config/
│   └── api_config.dart
├── models/
│   ├── image.dart
│   └── pagination.dart
├── services/
│   ├── api_service.dart
│   └── image_service.dart
├── providers/
│   └── image_provider.dart
├── screens/
│   ├── gallery_screen.dart
│   └── image_detail_screen.dart
└── widgets/
    ├── image_grid.dart
    ├── image_masonry.dart
    └── image_viewer.dart
```

### 5. Flutter 网格视图

**实现模式：** GridView.builder + cached_network_image

```dart
import 'package:flutter/material.dart';
import 'package:cached_network_image/cached_network_image.dart';

class ImageGrid extends StatelessWidget {
  final List<ImageModel> images;
  
  @override
  Widget build(BuildContext context) {
    return GridView.builder(
      gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 3, // 根据屏幕宽度自适应
        mainAxisSpacing: 4,
        crossAxisSpacing: 4,
      ),
      itemCount: images.length,
      itemBuilder: (context, index) {
        return GestureDetector(
          onTap: () => _navigateToDetail(images[index]),
          child: CachedNetworkImage(
            imageUrl: images[index].thumbnailUrl,
            placeholder: (context, url) => CircularProgressIndicator(),
            errorWidget: (context, url, error) => Icon(Icons.error),
            fit: BoxFit.cover,
          ),
        );
      },
    );
  }
}
```

### 6. Flutter 瀑布流视图

**实现模式：** flutter_staggered_grid_view MasonryGridView

```dart
import 'package:flutter_staggered_grid_view/flutter_staggered_grid_view.dart';

class ImageMasonry extends StatelessWidget {
  final List<ImageModel> images;
  
  @override
  Widget build(BuildContext context) {
    return MasonryGridView.count(
      crossAxisCount: 2, // 根据屏幕宽度自适应
      mainAxisSpacing: 4,
      crossAxisSpacing: 4,
      itemCount: images.length,
      itemBuilder: (context, index) {
        return GestureDetector(
          onTap: () => _navigateToDetail(images[index]),
          child: CachedNetworkImage(
            imageUrl: images[index].thumbnailUrl,
            placeholder: (context, url) => Container(
              height: 150,
              color: Colors.grey[200],
            ),
            errorWidget: (context, url, error) => Container(
              height: 150,
              color: Colors.red[100],
            ),
            fit: BoxFit.cover,
          ),
        );
      },
    );
  }
}
```

### 7. Flutter 图片详情页缩放

**实现模式：** extended_image 支持双指缩放和拖动

```dart
import 'package:extended_image/extended_image.dart';

class ImageViewer extends StatelessWidget {
  final String imageUrl;
  
  @override
  Widget build(BuildContext context) {
    return ExtendedImage.network(
      imageUrl,
      fit: BoxFit.contain,
      mode: ExtendedImageMode.gesture,
      initGestureConfigHandler: (state) {
        return GestureConfig(
          minScale: 0.9,
          animationMinScale: 0.7,
          maxScale: 3.0,
          animationMaxScale: 3.5,
          speed: 1.0,
          inertialSpeed: 100.0,
          initialScale: 1.0,
          inPageView: false,
        );
      },
    );
  }
}
```

### 8. API 分页设计

**实现模式：** 游标分页 (Cursor-based Pagination)

```go
// 请求参数
type ListImagesRequest struct {
    Cursor   string `form:"cursor"`   // 上一页最后一条记录的游标
    Limit    int    `form:"limit"`    // 每页数量，默认 20
    SortBy   string `form:"sort_by"`  // created_at, filename, file_size
    SortDir  string `form:"sort_dir"` // asc, desc
}

// 响应结构
type ListImagesResponse struct {
    Images     []Image `json:"images"`
    NextCursor string  `json:"next_cursor"`
    HasMore    bool    `json:"has_more"`
}

// 游标编码 (base64 JSON)
type Cursor struct {
    LastID    int64     `json:"last_id"`
    LastValue any       `json:"last_value"` // 排序字段的值
}
```

---

## 不要重复造轮子

### 现有代码可复用

| 组件 | 文件 | 复用方式 |
|-----|------|---------|
| Image 模型 | `internal/domain/image.go` | 已有 PHash 字段，可直接使用 |
| 异步任务管理 | `internal/worker/job_manager.go` | 注册 `thumbnail_generate` 任务类型 |
| 元数据提取 | `internal/service/metadata_service.go` | 扩展支持缩略图生成 |
| 图片仓储 | `internal/repository/image_repository.go` | 添加分页查询和更新方法 |
| API 路由 | `internal/handler/routes.go` | 替换 placeholderHandler |
| 数据库 Schema | `internal/repository/schema.go` | 添加 thumbnail_url 字段 |

### 成熟开源方案

| 需求 | 方案 | 优先级 |
|-----|------|-------|
| 图片缩放 | `disintegration/imaging` | 必须 |
| 感知哈希 | `corona10/goimagehash` | 必须 |
| COS 上传 | `tencentyun/cos-go-sdk-v5` | 必须 |
| 网格布局 | Flutter `GridView.builder` | 必须 |
| 瀑布流 | `flutter_staggered_grid_view` | 必须 |
| 图片缓存 | `cached_network_image` | 必须 |
| 图片缩放 | `extended_image` | 必须 |
| 状态管理 | `provider` | 推荐 |

---

## 常见陷阱

### 1. Flutter 内存爆炸 (Pitfall 3)

**问题：** 大图直接加载到 GridView 导致内存溢出

**解决方案：**
- 使用缩略图 URL 而非原图
- 使用 `cached_network_image` 自动缓存管理
- 设置合理的缓存大小限制
- 使用 `ListView.builder` / `GridView.builder` 实现懒加载

```dart
// 正确做法：使用缩略图 + 缓存
CachedNetworkImage(
  imageUrl: image.thumbnailSmallUrl, // ~200px 缩略图
  memCacheWidth: 200, // 限制内存缓存尺寸
  fit: BoxFit.cover,
)
```

### 2. COS 上传失败无重试

**问题：** 网络不稳定时上传失败，无自动重试

**解决方案：**
- 实现指数退避重试机制
- 记录失败任务到 async_jobs 表
- 提供手动重试接口

```go
func UploadWithRetry(client *cos.Client, key string, data []byte, maxRetries int) error {
    var lastErr error
    for i := 0; i < maxRetries; i++ {
        err := UploadThumbnail(client, key, data)
        if err == nil {
            return nil
        }
        lastErr = err
        time.Sleep(time.Second * time.Duration(1<<i)) // 指数退避
    }
    return lastErr
}
```

### 3. 感知哈希误判

**问题：** 单一 pHash 算法在某些场景下误判

**解决方案：**
- Phase 4 可引入多种哈希 (aHash, dHash) 组合
- 当前阶段使用单一 pHash + 合理阈值 (默认 12)
- 用户可在设置中调整阈值

### 4. 缩略图生成阻塞导入

**问题：** 批量导入时同步生成缩略图阻塞流程

**解决方案：**
- 导入时不生成缩略图，仅计算 pHash
- 缩略图按需生成：首次访问时触发
- 生成后上传 COS 并缓存 URL

---

## 数据库变更

### images 表扩展

```sql
-- 添加缩略图 URL 字段
ALTER TABLE images ADD COLUMN thumbnail_small_url TEXT;
ALTER TABLE images ADD COLUMN thumbnail_large_url TEXT;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_images_created_at ON images(created_at);
CREATE INDEX IF NOT EXISTS idx_images_filename ON images(filename);
CREATE INDEX IF NOT EXISTS idx_images_file_size ON images(file_size);
```

---

## 外部服务配置

### 腾讯云 COS

需要在 `config.yaml` 中配置：

```yaml
cos:
  bucket_url: "literal:${COS_BUCKET_URL:-}"
  secret_id: "${COS_SECRET_ID}"     # 从环境变量读取
  secret_key: "${COS_SECRET_KEY}"   # 从环境变量读取
```

### 用户设置要求

用户需要：
1. 在腾讯云控制台获取 Secret ID 和 Secret Key
2. 设置环境变量 `COS_SECRET_ID` 和 `COS_SECRET_KEY`
3. 确认存储桶 `acgwarehouse-1301393037` 已创建

---

## 验证架构

### 自动化测试策略

| 组件 | 测试类型 | 验证点 |
|-----|---------|--------|
| 缩略图服务 | 单元测试 | 尺寸正确、质量参数生效 |
| pHash 计算 | 单元测试 | 相同图片相同哈希、相似图片距离小 |
| COS 上传 | 集成测试 | Mock COS API 或使用测试桶 |
| 分页 API | 单元测试 | 游标正确、边界情况 |
| Flutter 组件 | Widget 测试 | 渲染正确、交互响应 |

### 集成验证点

1. 缩略图生成并上传 COS → URL 可访问
2. pHash 计算 → 存入数据库 → 相似度查询可用
3. Flutter 网格/瀑布流 → 图片正确显示
4. 图片详情页 → 缩放交互正常
5. 分页滚动 → 无限加载正常

---

## 阶段依赖分析

### 依赖 Phase 1 的组件

| Phase 1 组件 | Phase 2 使用方式 |
|-------------|-----------------|
| `internal/domain/image.go` | 扩展添加缩略图 URL 字段 |
| `internal/repository/image_repository.go` | 添加分页查询方法 |
| `internal/service/metadata_service.go` | 扩展支持缩略图生成 |
| `internal/worker/job_manager.go` | 注册缩略图生成任务 |
| `internal/handler/routes.go` | 实现真实的 API 端点 |

### 为 Phase 3 提供的组件

| Phase 2 组件 | Phase 3 使用方式 |
|-------------|-----------------|
| 图片详情页 | AI 标签展示和确认入口 |
| 标签状态占位组件 | AI 标签填充和交互 |
| 缩略图 URL | 标签标注界面展示 |

---

## 研究结论

### 技术风险：低

所有核心技术均有成熟的开源实现，无需自研。

### 实现复杂度：中等

- Go 后端：扩展现有服务 + COS 集成
- Flutter 前端：全新项目，需初始化架构

### 关键成功因素

1. 正确配置 COS 凭证
2. Flutter 内存管理（使用缩略图而非原图）
3. 游标分页正确实现（避免数据重复/遗漏）
4. 异步任务正确处理（失败重试机制）

---

*研究完成时间：2026-03-14*