# Phase 4: 重复检测与搜索 - Research

**Gathered:** 2026-03-16
**Status:** Research complete

---

## 标准技术栈

### 后端 (Go)

| 领域 | 推荐技术 | 原因 |
|------|----------|------|
| 感知哈希计算 | `github.com/corona10/goimagehash` | 业界标准 pHash/aHash/dHash 实现，支持汉明距离计算 |
| 文件哈希 | `crypto/sha256` (标准库) | SHA-256 检测完全相同文件，标准库无需额外依赖 |
| 全文搜索 | SQLite FTS5 | 内置全文索引，支持中文分词，无需额外服务 |
| 中文分词 | `github.com/yanyiwu/gojieba` | 结巴分词 Go 版本，FTS5 可集成 |

### 前端 (Flutter)

| 领域 | 推荐技术 | 原因 |
|------|----------|------|
| 搜索界面 | 现有 `gallery_screen.dart` 模式 | 复用网格/瀑布流视图 |
| 搜索框 | AppBar 搜索组件 + `SearchDelegate` | Material Design 标准模式 |
| 文件上传 | `file_picker` + `dio` | 支持文件选择、拖拽、进度显示 |
| 图片拖拽 | `desktop_drop` | 支持桌面端拖拽上传 |

---

## 架构模式

### 1. 重复检测服务

**检测流程：**
```
触发检测 → 遍历所有图片 → 计算文件哈希(SHA256) → 计算感知哈希(pHash)
  → 比较汉明距离 → 分组(传递性) → 存储检测结果 → 返回重复组
```

**关键算法：**

1. **完全相同检测**：SHA256 文件哈希完全匹配
2. **相似图片检测**：pHash 汉明距离 ≤ 阈值（默认 10）
3. **传递性分组**：A~B 且 B~C → {A, B, C} 同组

**数据库设计：**

```sql
-- 重复组表
CREATE TABLE duplicate_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    recommended_image_id INTEGER NOT NULL,  -- 推荐保留的图片（分辨率最高）
    similarity_threshold INTEGER DEFAULT 10, -- 使用的阈值
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (recommended_image_id) REFERENCES images(id)
);

-- 重复关系表
CREATE TABLE duplicate_relations (
    group_id INTEGER NOT NULL,
    image_id INTEGER NOT NULL,
    is_recommended INTEGER DEFAULT 0,  -- 是否为推荐保留
    file_hash TEXT,                     -- SHA256
    phash_distance INTEGER,             -- 与推荐图的汉明距离
    PRIMARY KEY (group_id, image_id),
    FOREIGN KEY (group_id) REFERENCES duplicate_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE
);

CREATE INDEX idx_duplicate_relations_image ON duplicate_relations(image_id);
CREATE INDEX idx_duplicate_relations_file_hash ON duplicate_relations(file_hash);
```

### 2. 全文搜索服务

**FTS5 虚拟表设计：**

```sql
-- 全文索引虚拟表
CREATE VIRTUAL TABLE IF NOT EXISTS images_fts USING fts5(
    image_id UNINDEXED,  -- 关联图片 ID
    filename,            -- 文件名
    tags,                -- 标签文本（包括别名）
    content=''           -- 空内容表，使用触发器同步
);

-- 同步触发器（images 表变更时更新索引）
CREATE TRIGGER IF NOT EXISTS images_ai AFTER INSERT ON images BEGIN
    INSERT INTO images_fts(image_id, filename, tags)
    VALUES (new.id, new.filename, '');
END;

CREATE TRIGGER IF NOT EXISTS images_ad AFTER DELETE ON images BEGIN
    DELETE FROM images_fts WHERE image_id = old.id;
END;

CREATE TRIGGER IF NOT EXISTS images_au AFTER UPDATE ON images BEGIN
    UPDATE images_fts SET filename = new.filename WHERE image_id = old.id;
END;
```

**中文分词集成：**

```go
// 使用 gojieba 分词
import "github.com/yanyiwu/gojieba"

func tokenize(text string) string {
    jieba := gojieba.NewJieba()
    defer jieba.Free()
    words := jieba.Cut(text, true)  // 精确模式
    return strings.Join(words, " ")
}
```

### 3. 以图搜图服务

**流程：**
```
上传图片 → 计算 pHash → 查询数据库 → 按汉明距离排序 → 返回相似图片
```

**API 设计：**

```
POST /api/v1/search/by-image
Content-Type: multipart/form-data

参数：
- file: 图片文件（可选，与 url 二选一）
- url: 图片 URL（可选，与 file 二选一）
- limit: 返回数量（默认 20）

响应：
{
    "results": [
        {
            "image": { ... },
            "similarity": 0.95  // 相似度百分比
        }
    ],
    "query_thumbnail": "data:image/..."  // 查询图缩略图
}
```

---

## 不要重复造轮子

| 功能 | 使用现有代码 | 位置 |
|------|--------------|------|
| 图片网格视图 | `ImageGrid` widget | `flutter_app/lib/widgets/image_grid.dart` |
| 瀑布流视图 | `ImageMasonry` widget | `flutter_app/lib/widgets/image_masonry.dart` |
| Gallery 页面模式 | `GalleryScreen` | `flutter_app/lib/screens/gallery_screen.dart` |
| API 服务模式 | `ApiService` | `flutter_app/lib/services/api_service.dart` |
| Provider 状态管理 | `ImageListProvider` | `flutter_app/lib/providers/image_provider.dart` |
| 图片元数据提取 | `MetadataService` | `internal/service/metadata_service.go` |
| 图片仓储模式 | `ImageRepository` | `internal/repository/image_repository.go` |
| API 处理器模式 | `ImageHandler` | `internal/handler/image_handler.go` |

---

## 常见陷阱

### 1. 感知哈希误判

**问题**：单一哈希算法可能产生误判（如缩放、旋转图片被误判为相同）

**解决方案**：
- 组合多种哈希：pHash（主要）+ dHash（辅助验证）
- 设置合理阈值：汉明距离 ≤ 10 为相似（经验值）
- 用户可调整阈值以平衡精度/召回

### 2. FTS5 中文分词问题

**问题**：SQLite FTS5 默认分词器不支持中文

**解决方案**：
- 使用 `gojieba` 在应用层分词后存入 FTS5
- 或使用 FTS5 的 `unicode61` 分词器作为简单方案
- 本项目选择：先使用 `unicode61`（简单），后续可升级 gojieba

### 3. 大图库性能问题

**问题**：图片数量 > 10k 时，全量比较 O(n²) 性能下降

**解决方案**：
- 使用 pHash 索引：建立 B-tree 索引加速查询
- 增量检测：只检测新增图片与已有图片的相似关系
- 批量处理：分批次处理，避免内存溢出

### 4. 以图搜图上传临时文件管理

**问题**：上传的图片需要临时存储，可能占用大量空间

**解决方案**：
- 使用系统临时目录（`os.TempDir()`）
- 设置定时清理任务（每小时清理超过 1 小时的临时文件）
- 上传后立即计算哈希，无需持久化原图

---

## 验证架构

### Must-Haves（必须实现）

** truths（可观察行为）：**
1. 用户可以检测完全相同的图片（文件哈希匹配）
2. 用户可以检测相似图片（感知哈希匹配）
3. 用户可以设置相似度阈值
4. 用户可以查看重复图片组并选择保留/删除
5. 用户可以按文件名搜索图片
6. 用户可以按标签搜索图片
7. 用户可以组合搜索词和标签筛选
8. 用户可以上传图片进行以图搜图
9. 搜索结果支持排序和分页

**artifacts（必须存在）：**
- `internal/service/duplicate_service.go` — 重复检测服务
- `internal/service/search_service.go` — 全文搜索服务
- `internal/handler/duplicate_handler.go` — 重复检测 API
- `internal/handler/search_handler.go` — 搜索 API
- `internal/repository/duplicate_repository.go` — 重复组仓储
- `flutter_app/lib/screens/search_screen.dart` — 搜索界面
- `flutter_app/lib/screens/duplicate_screen.dart` — 重复管理界面

**key_links（关键连接）：**
- `duplicate_service.go` → `image_repository.go`（读取图片列表）
- `search_service.go` → `images_fts`（全文索引查询）
- `search_handler.go` → `search_service.go`（API 调用服务）
- `SearchScreen` → `ApiService.searchImages()`（前端调用后端）

### 验证策略

| 功能 | 自动化验证方式 |
|------|----------------|
| 文件哈希计算 | 单元测试：已知文件 → 预期哈希 |
| pHash 计算 | 单元测试：相同图片 → 相同哈希 |
| 汉明距离比较 | 单元测试：给定两个哈希 → 预期距离 |
| FTS5 搜索 | 集成测试：插入测试数据 → 搜索关键词 → 验证结果 |
| 以图搜图 | API 测试：上传图片 → 返回相似图片列表 |
| 重复检测 | 端到端测试：导入重复图片 → 检测 → 验证分组 |

---

## 实现优先级

### Wave 1: 基础设施（可并行）

1. **Plan 01: 重复检测后端层**
   - 文件哈希计算
   - pHash 计算
   - 汉明距离比较
   - 重复组数据库表
   - 重复检测 API

2. **Plan 02: 搜索后端层**
   - FTS5 虚拟表创建
   - 搜索 API
   - 标签搜索集成

### Wave 2: 前端界面（依赖 Wave 1）

3. **Plan 03: 重复检测前端**
   - 重复组列表界面
   - 批量选择删除
   - 进度显示

4. **Plan 04: 搜索前端层**
   - 搜索界面
   - 搜索结果展示
   - 以图搜图界面

### Wave 3: 集成验证

5. **Plan 05: 端到端集成**
   - 完整流程测试
   - 性能验证
   - 用户体验优化

---

## 外部依赖

| 依赖 | 用途 | 安装方式 |
|------|------|----------|
| `github.com/corona10/goimagehash` | 感知哈希计算 | `go get` |
| `github.com/yanyiwu/gojieba` | 中文分词（可选） | `go get` |
| `file_picker` | Flutter 文件选择 | `flutter pub add` |
| `desktop_drop` | Flutter 拖拽支持 | `flutter pub add` |
| `dio` | Flutter HTTP 上传 | `flutter pub add` |

---

## 用户配置需求

无需外部服务配置。所有功能使用本地计算和数据库，无需 API Key。

---

*研究完成时间：2026-03-16*