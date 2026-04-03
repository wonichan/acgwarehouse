# ACG 图库技术实施方案 (Go + Python + Flutter)

> 版本: 1.0
> 日期: 2025年4月
> 架构: Go 主控 + Python 计算 + Flutter UI

---

## 1. 总体架构

### 1.1 职责划分

| 层级 | 技术 | 核心职责 |
|-----|------|---------|
| **UI 层** | Flutter Desktop | 界面渲染、状态管理、用户交互、缩略图展示 |
| **主控层** | Go Backend | 业务逻辑、数据库、文件导入、工作流编排、AI 标签调用、Python 进程管理 |
| **计算层** | Python Service | 图像指纹、重复检测、相似度计算、质量评估 |

### 1.2 通信方式

```
Flutter Desktop <──HTTP/WebSocket──> Go Backend <──HTTP (localhost)──> Python Service
```

- **Flutter <-> Go**: Local HTTP REST API + WebSocket (实时进度推送)
- **Go <-> Python**: Go 启动 Python HTTP 服务，通过本地 HTTP 调用

**为什么不用 gRPC/FFI?**
- gRPC: 增加不必要的 proto 代码生成负担
- FFI/CGO: 交叉编译痛苦，Python GIL 容易导致崩溃
- HTTP: 调试简单、跨语言通用、本地环回开销可忽略

---

## 2. 模块边界与数据流

### 2.1 Flutter (UI 层)

**负责:**
- 图库主页面 (瀑布流/网格视图)
- 图片详情页 (大图、元数据、标签编辑)
- 标签管理页 (树形层级、别名、颜色)
- 导入中心 (监控目录、导入规则、历史记录)
- 重复处理页 (重复组对比、批量保留)
- 桌面壁纸模式 (可选扩展)

**状态管理**: Riverpod + Freezed
**路由**: go_router
**图片加载**: cached_network_image + 虚拟滚动

**禁止:**
- ❌ 直接文件系统操作
- ❌ 直接数据库读写
- ❌ 直接调用 AI 模型
- ❌ 直接运行图像算法

### 2.2 Go (主控层)

**负责:**
- **数据库管理**: SQLite + FTS5 全文搜索
- **文件导入**: 目录扫描、监控、规则匹配、任务队列
- **缩略图服务**: 调度生成、缓存管理
- **标签系统**: CRUD、层级管理、别名、筛选表达式
- **AI 标签编排**: 调用云端/本地大模型、标签清洗、置信度处理
- **重复检测编排**: 批次管理、调用 Python、结果落库
- **Python 进程管理**: 启动、健康检查、优雅关闭

**核心表:**
- `images` - 图片主表
- `tags` - 标签表 (支持层级)
- `image_tags` - 图片标签关联
- `folders` - 监控目录
- `import_jobs` - 导入任务
- `duplicate_groups` - 重复组
- `duplicate_candidates` - 重复候选

### 2.3 Python (计算层)

**负责:**
- **精确重复检测**: MD5/SHA1 哈希
- **视觉相似检测**: pHash/dHash 感知哈希
- **重复组聚类**: 汉明距离阈值分组
- **保留建议**: 按分辨率/清晰度/完整度推荐

**禁止:**
- ❌ 业务状态持久化
- ❌ 直接操作数据库
- ❌ 标签管理逻辑
- ❌ 文件系统操作 (只读计算)

---

## 3. 核心数据流

### 3.1 图片导入流程

```
[Flutter] 发起导入请求
    ↓ HTTP POST /imports
[Go] 扫描文件系统
    ↓
[Go] 写入 SQLite (images表)
    ↓
[Go] 分发缩略图生成任务
    ↓
[Go] 调用 AI 标签服务 (可选)
    ↓
[Go] WebSocket 推送进度 → [Flutter] 更新UI
```

### 3.2 标签筛选流程

```
[Flutter] 选择筛选条件 (标签A AND 标签B NOT 标签C)
    ↓ HTTP POST /filters/search
[Go] 解析筛选表达式
    ↓
[Go] 查询 SQLite + FTS + image_tags
    ↓
[Go] 返回分页结果 + 聚合统计
    ↓ HTTP Response
[Flutter] 渲染筛选结果
```

### 3.3 重复检测流程

```
[Flutter] 点击"开始检测"
    ↓ HTTP POST /duplicates/scan
[Go] 获取待检测图片列表
    ↓ HTTP POST (localhost:random_port)
[Python] 计算文件哈希 + pHash
    ↓
[Python] 聚类相似图片组
    ↓
[Python] 返回重复组 + 相似度 + 建议
    ↓ HTTP Response
[Go] 写入 duplicate_groups / duplicate_candidates
    ↓ WebSocket 推送
[Flutter] 展示重复组确认界面
```

### 3.4 去重执行流程

```
[Flutter] 用户确认保留策略
    ↓ HTTP POST /duplicates/apply
[Go] 执行移动操作 (到回收站/归档目录)
    ↓
[Go] 更新 images 状态 (软删除)
    ↓
[Go] 清理缩略图缓存
    ↓ HTTP Response
[Flutter] 刷新图库视图
```

---

## 4. API 设计

### 4.1 Flutter <-> Go API

| 接口 | 方法 | 描述 |
|-----|------|------|
| `/images` | GET | 分页获取图片列表 |
| `/images/:id` | GET | 获取单张图片详情 |
| `/imports` | POST | 发起导入任务 |
| `/jobs/:id` | GET | 查询任务进度 |
| `/tags` | GET/POST | 标签列表/创建 |
| `/tags/:id` | PUT/DELETE | 标签更新/删除 |
| `/filters/search` | POST | 标签筛选搜索 |
| `/duplicates/scan` | POST | 启动重复检测 |
| `/duplicates/groups` | GET | 获取重复组列表 |
| `/duplicates/apply` | POST | 应用去重策略 |
| `/ws` | WebSocket | 实时进度推送 |

### 4.2 Go <-> Python API

Python 作为常驻 HTTP 服务，监听 `127.0.0.1:0` (随机端口)

| 接口 | 方法 | 描述 |
|-----|------|------|
| `/dedupe/hash` | POST | 计算文件哈希 |
| `/dedupe/phash` | POST | 计算感知哈希 |
| `/dedupe/find-groups` | POST | 查找重复组 |
| `/dedupe/recommend` | POST | 保留建议 |
| `/health` | GET | 健康检查 |

---

## 5. 数据库设计 (SQLite)

```sql
-- 图片主表
CREATE TABLE images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT UNIQUE NOT NULL,
    filename TEXT NOT NULL,
    file_hash TEXT,
    phash TEXT,
    width INTEGER,
    height INTEGER,
    file_size INTEGER,
    format TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified_at TIMESTAMP,
    imported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'active', -- active | archived | deleted
    thumbnail_path TEXT
);

-- 标签表 (支持层级)
CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    aliases TEXT, -- JSON: ["Rem", "レム"]
    color TEXT,
    parent_id INTEGER,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_id) REFERENCES tags(id)
);

-- 图片标签关联
CREATE TABLE image_tags (
    image_id INTEGER,
    tag_id INTEGER,
    confidence REAL DEFAULT 1.0,
    source TEXT DEFAULT 'manual', -- manual | ai | imported
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (image_id, tag_id),
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- 监控目录
CREATE TABLE folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT UNIQUE NOT NULL,
    auto_import BOOLEAN DEFAULT FALSE,
    auto_tags TEXT, -- JSON: [tag_id1, tag_id2]
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 导入任务
CREATE TABLE import_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    folder_id INTEGER,
    status TEXT DEFAULT 'pending', -- pending | processing | completed | failed
    total_count INTEGER DEFAULT 0,
    processed_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    FOREIGN KEY (folder_id) REFERENCES folders(id)
);

-- 重复组
CREATE TABLE duplicate_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    detection_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    similarity_threshold REAL DEFAULT 0.85,
    status TEXT DEFAULT 'pending', -- pending | resolved | ignored
    recommended_keep_id INTEGER,
    FOREIGN KEY (recommended_keep_id) REFERENCES images(id)
);

-- 重复候选
CREATE TABLE duplicate_candidates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER,
    image_id INTEGER,
    similarity_score REAL,
    file_hash_match BOOLEAN,
    phash_distance INTEGER,
    status TEXT DEFAULT 'pending', -- pending | keep | delete | ignore
    FOREIGN KEY (group_id) REFERENCES duplicate_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE
);

-- 全文搜索虚拟表
CREATE VIRTUAL TABLE images_fts USING fts5(
    filename,
    content='images',
    content_rowid='id'
);
```

---

## 6. AI 标签方案

### 6.1 架构决策

**由 Go 编排，不由 Python 直接处理**

原因：
- 标签生成是业务能力，不只是图像算法
- Go 更适合任务队列、限流、缓存、审计日志
- Python 保持无状态，只负责纯计算

### 6.2 流程

```
[Go] 导入图片完成
    ↓
[Go] 生成缩略图 (减少传输)
    ↓
[Go] 调用 AI 服务 (云端 API 或本地模型)
    ↓
[Go] 接收原始候选标签
    ↓
[Go] 标签清洗 (去重、映射、过滤)
    ↓
[Go] 写入 image_tags (source=ai, confidence=0.92)
    ↓
[Go] WebSocket 通知 UI 更新
```

### 6.3 标签类型体系

```
一级分类:
├── 角色相关 (character)
│   ├── 角色名
│   └── 属性 (发色、服饰)
├── 作品来源 (source)
│   ├── 作品名
│   └── 类型 (动画/游戏/漫画)
├── 场景氛围 (scene)
│   ├── 场景 (城市/自然/室内)
│   └── 氛围 (温馨/战斗/日常)
├── 图像类型 (type)
│   ├── 官方图
│   ├── 同人图
│   ├── 截图
│   └── Cosplay
└── 质量标签 (quality)
    ├── 高清
    ├── 4K
    └── 收藏级
```

---

## 7. Python 重复检测服务

### 7.1 服务启动

Go 启动 Python 进程:
```go
// Go 代码
pythonCmd := exec.Command("python", "dedupe_service.py")
pythonCmd.Stdout = os.Stdout
pythonCmd.Stderr = os.Stderr
pythonCmd.Start()

// Python 启动后打印监听端口到 stdout
// Go 读取端口，建立 HTTP 连接
```

### 7.2 算法流程

```python
# Python 伪代码
class DedupeService:
    def find_duplicates(self, image_paths):
        # 阶段1: 精确重复 (MD5)
        exact_duplicates = self.find_exact_duplicates(image_paths)
        
        # 阶段2: 视觉相似 (pHash)
        phashes = {path: compute_phash(path) for path in image_paths}
        visual_groups = self.group_by_hamming_distance(phashes, threshold=5)
        
        # 阶段3: 保留建议
        for group in visual_groups:
            group.recommended_keep = self.select_best_to_keep(group)
        
        return {
            exact: exact_duplicates,
            visual: visual_groups
        }
    
    def select_best_to_keep(self, group):
        # 优先级: 分辨率 > EXIF完整度 > 文件大小 > 清晰度
        return max(group, key=lambda img: (
            img.width * img.height,
            len(img.exif_data),
            img.file_size,
            -img.blur_score
        ))
```

### 7.3 依赖

```txt
# requirements.txt
Pillow==10.0.0
imagehash==4.3.1
fastapi==0.104.0
uvicorn==0.24.0
numpy==1.24.0
```

---

## 8. Flutter UI 结构

### 8.1 页面划分

```
lib/
├── screens/
│   ├── gallery_screen.dart      # 图库主页面
│   ├── detail_screen.dart       # 图片详情
│   ├── tags_screen.dart         # 标签管理
│   ├── import_screen.dart       # 导入中心
│   ├── duplicates_screen.dart   # 重复处理
│   └── settings_screen.dart     # 设置
├── state/
│   ├── gallery_provider.dart    # 图库状态
│   ├── tags_provider.dart       # 标签状态
│   └── import_provider.dart     # 导入状态
├── api/
│   ├── gallery_api.dart         # API 客户端
│   └── websocket_client.dart    # WebSocket 连接
├── widgets/
│   ├── image_grid.dart          # 图片网格
│   ├── image_thumbnail.dart     # 缩略图
│   ├── tag_tree.dart            # 标签树
│   └── duplicate_comparator.dart # 重复对比
└── models/
    ├── image_model.dart
    └── tag_model.dart
```

### 8.2 关键依赖

```yaml
dependencies:
  flutter:
    sdk: flutter
  
  # 状态管理
  flutter_riverpod: ^2.4.0
  freezed_annotation: ^2.4.0
  
  # 网络
  dio: ^5.3.0
  web_socket_channel: ^2.4.0
  
  # 图片
  cached_network_image: ^3.3.0
  photo_view: ^0.14.0
  
  # 桌面
  window_manager: ^0.5.1
  desktop_drop: ^0.4.0
  
  # 存储
  sqflite_common_ffi: ^2.3.0
  
  # 路由
  go_router: ^12.0.0

dev_dependencies:
  build_runner: ^2.4.0
  freezed: ^2.4.0
  json_serializable: ^6.7.0
```

---

## 9. Windows 打包方案

### 9.1 分发目标

- **单文件分发**: 用户无需安装 Python/Go
- **绿色软件**: 解压即用
- **自动更新**: 可选 (后续迭代)

### 9.2 打包流程

```
# 1. Python 打包 (PyInstaller)
cd services/python-dedupe
pyinstaller --onefile --name dedupe_service app.py
# 输出: dist/dedupe_service.exe

# 2. Go 打包
cd services/go-backend
go build -o acg_backend.exe .
# 输出: acg_backend.exe

# 3. Flutter 打包
cd apps/flutter_desktop
flutter build windows --release
# 输出: build/windows/x64/runner/Release/

# 4. 合并
cd build/windows/x64/runner/Release/
# 复制 acg_backend.exe 和 dedupe_service.exe 到同一目录
# 打包为 zip/installer
```

### 9.3 启动顺序

```
[Flutter App] 启动
    ↓
[Flutter] 拉起 Go: acg_backend.exe
    ↓
[Go] 拉起 Python: dedupe_service.exe (随机端口)
    ↓
[Go] 读取 Python 端口，建立 HTTP 连接
    ↓
[Go] 启动 HTTP 服务，监听 localhost:8080
    ↓
[Flutter] 连接 localhost:8080，应用就绪
```

### 9.4 优雅关闭

```
[Flutter] 关闭
    ↓
[Flutter] 发送 POST /shutdown 到 Go
    ↓
[Go] 关闭 Python HTTP 连接
    ↓
[Go] 发送 SIGTERM 到 Python 进程
    ↓
[Go] 关闭 SQLite 连接
    ↓
[Go] 退出
    ↓
[Flutter] 完全退出
```

---

## 10. Flutter 桌面贴图功能 (扩展)

### 10.1 可行性评估

**可以做，但有限制**:

| 功能 | 可行性 | 实现方式 |
|-----|--------|---------|
| 透明无边框窗口 | ✅ 高 | `window_manager` + `flutter_acrylic` |
| 窗口拖拽/缩放 | ✅ 高 | Flutter GestureDetector |
| 左右滑动切图 | ✅ 高 | Flutter PageView |
| 始终置底 (桌面层) | ⚠️ 中 | 需要 Win32 插件 |
| 点击穿透 | ⚠️ 中 | 需要 Win32 插件 |
| 真正"壁纸"层级 | ❌ 低 | 需注入 Explorer，不推荐 |

### 10.2 推荐实现

**方案: 浮动图片窗**
- 透明无边框窗口
- 可拖拽、可缩放
- 支持左右滑动切换图片
- 始终显示在最上层 (类似悬浮窗)
- 不追求"壁纸"层级 (避免系统不稳定)

**所需插件**:
```yaml
dependencies:
  window_manager: ^0.5.1
  flutter_acrylic: ^1.1.0
```

**关键代码**:
```dart
await windowManager.ensureInitialized();

WindowOptions windowOptions = WindowOptions(
  size: Size(400, 300),
  center: true,
  backgroundColor: Colors.transparent,
  skipTaskbar: true,
  titleBarStyle: TitleBarStyle.hidden,
  alwaysOnTop: true,  // 始终置顶
);

await windowManager.waitUntilReadyToShow(windowOptions, () async {
  await windowManager.show();
  await windowManager.setAsFrameless();
});
```

---

## 11. MVP 分期建议

### Phase 1 (2-3周)
- [x] Go 基础 API + SQLite 表结构
- [x] 文件导入 + 目录监控
- [x] Flutter 图库主页 + 详情页
- [x] 基础标签 CRUD
- [ ] Python 精确重复检测 (MD5)

### Phase 2 (2-3周)
- [x] Python 感知哈希重复检测
- [x] 重复组确认界面
- [x] 标签筛选搜索 (基础)
- [x] 缩略图生成与缓存
- [ ] AI 标签接入 (云端 API)

### Phase 3 (2-3周)
- [x] 标签层级管理
- [x] 高级筛选 (布尔组合)
- [x] 自动导入规则
- [x] Windows 打包发布
- [ ] 桌面贴图功能 (可选)

---

## 12. 避坑指南

### 12.1 进程管理

**僵尸进程风险**:
- Go 必须监听 SIGTERM，确保 Python 正常退出
- 实现心跳检测: Go 定期 ping Python，超时自动重启
- Flutter 关闭时主动通知 Go 清理

**端口冲突**:
- 绝不硬编码端口 (8080, 5000)
- Go 监听 `127.0.0.1:0` (随机端口)
- Python 同样使用随机端口
- 通过 stdout/stdin 传递实际端口号

### 12.2 打包问题

**杀毒软件误报**:
- PyInstaller 打包的 exe 易被误报
- 建议: 使用嵌入式 Python + 脚本启动
- 或考虑代码签名 (后续)

**依赖缺失**:
- Python 服务必须包含所有依赖
- 使用 `--hidden-import` 确保 PyInstaller 包含所有模块

### 12.3 性能优化

**大图处理**:
- 导入时生成多尺寸缩略图 (128x128, 256x256, 512x512)
- 大图查看时使用分块加载
- Python 处理图片时使用内存限制

**数据库性能**:
- 图片表添加索引 (path, file_hash, status)
- 使用事务批量写入
- 定期 VACUUM

---

## 13. 参考资源

### Flutter Desktop
- [window_manager](https://pub.dev/packages/window_manager) - 窗口控制
- [flutter_acrylic](https://pub.dev/packages/flutter_acrylic) - 透明效果
- [desktop_drop](https://pub.dev/packages/desktop_drop) - 拖拽导入

### Go Backend
- [Gin](https://gin-gonic.com/) - HTTP 框架
- [gorm](https://gorm.io/) - ORM
- [fsnotify](https://github.com/fsnotify/fsnotify) - 文件监控

### Python Computer Vision
- [imagehash](https://github.com/JohannesBuchner/imagehash) - 感知哈希
- [Pillow](https://pillow.readthedocs.io/) - 图像处理
- [FastAPI](https://fastapi.tiangolo.com/) - HTTP 服务

---

## 总结

本方案采用 **Flutter(前端) + Go(主控) + Python(计算)** 的分层架构：

1. **职责清晰**: Flutter 只管 UI，Go 管业务，Python 只管图像算法
2. **通信简单**: HTTP + WebSocket，调试容易，跨语言通用
3. **部署简单**: 打包成独立 exe，用户无需配置环境
4. **可扩展**: 后续可接入本地 AI 模型、云端服务、更多平台

**立即开始**: 建议从 Phase 1 开始，2-3 周内完成核心骨架，验证架构可行性。
