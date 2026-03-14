# Requirements: ACGWarehouse（二次元图片库）

**Defined:** 2026-03-14
**Core Value:** 让用户能够高效地管理和检索二次元图片库，通过 AI 自动化减少手动整理的工作量，实现"存入即整理"的体验。

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### 核心架构 (Core Infrastructure)

- [ ] **CORE-01**: 系统支持 Go 后端项目结构初始化
- [ ] **CORE-02**: 系统支持 SQLite 数据库（开发/单用户）和 PostgreSQL（生产/多用户）双模式
- [ ] **CORE-03**: 系统支持 RESTful API 基础框架（Gin）
- [ ] **CORE-04**: 系统支持配置文件管理（数据库连接、存储路径、AI 服务配置）

### 图片扫描入库 (Image Import)

- [ ] **IMPT-01**: 用户可以扫描指定文件夹并导入图片
- [ ] **IMPT-02**: 用户可以监控指定文件夹，自动导入新增图片
- [ ] **IMPT-03**: 系统支持常见图片格式（JPG、PNG、WebP、GIF）
- [ ] **IMPT-04**: 系统提取图片元数据（尺寸、格式、创建时间、EXIF）
- [ ] **IMPT-05**: 系统生成并存储缩略图（多种尺寸）
- [ ] **IMPT-06**: 系统计算并存储图片感知哈希（用于相似检测）
- [ ] **IMPT-07**: 用户可以查看导入进度和状态

### 图片浏览 (Gallery)

- [ ] **GALR-01**: 用户可以以网格视图浏览图片
- [ ] **GALR-02**: 用户可以以瀑布流视图浏览图片
- [ ] **GALR-03**: 用户可以查看图片详情（大图、元数据）
- [ ] **GALR-04**: 用户可以按时间/名称/大小排序
- [ ] **GALR-05**: 用户可以分页浏览大型图片库

### AI 角色识别 (AI Recognition)

- [ ] **AIRE-01**: 系统自动识别动漫角色并生成角色标签
- [ ] **AIRE-02**: 系统自动识别画师并生成画师标签
- [ ] **AIRE-03**: 系统自动识别原作/系列并生成原作标签
- [ ] **AIRE-04**: 系统为每个标签提供置信度分数
- [ ] **AIRE-05**: 用户可以查看 AI 识别结果并确认/修改
- [ ] **AIRE-06**: 系统异步处理 AI 识别任务，不阻塞用户操作

### 标签管理 (Tags)

- [ ] **TAGS-01**: 用户可以手动添加/修改/删除标签
- [ ] **TAGS-02**: 系统支持标签分类（角色、画师、原作、通用、元数据）
- [ ] **TAGS-03**: 用户可以按标签筛选图片
- [ ] **TAGS-04**: 系统支持标签搜索（模糊匹配）
- [ ] **TAGS-05**: 用户可以查看标签统计（使用次数）

### 重复检测 (Duplicate Detection)

- [ ] **DUPD-01**: 系统检测完全相同的图片（文件哈希）
- [ ] **DUPD-02**: 系统检测相似图片（感知哈希）
- [ ] **DUPD-03**: 用户可以设置相似度阈值
- [ ] **DUPD-04**: 用户可以查看重复图片组
- [ ] **DUPD-05**: 用户可以手动选择保留/删除重复图片

### 搜索功能 (Search)

- [ ] **SRCH-01**: 用户可以按文件名搜索图片
- [ ] **SRCH-02**: 用户可以按标签搜索图片
- [ ] **SRCH-03**: 用户可以组合多个标签搜索（AND/OR）
- [ ] **SRCH-04**: 用户可以上传图片进行以图搜图
- [ ] **SRCH-05**: 搜索结果支持排序和分页

### 收藏夹/相册 (Collections)

- [ ] **COLL-01**: 用户可以创建收藏夹/相册
- [ ] **COLL-02**: 用户可以添加/移除收藏夹中的图片
- [ ] **COLL-03**: 用户可以重命名/删除收藏夹
- [ ] **COLL-04**: 用户可以设置收藏夹封面
- [ ] **COLL-05**: 系统显示收藏夹统计（图片数量）

### 批量操作 (Batch Operations)

- [ ] **BTCH-01**: 用户可以批量选择图片
- [ ] **BTCH-02**: 用户可以批量添加/删除标签
- [ ] **BTCH-03**: 用户可以批量移动到收藏夹
- [ ] **BTCH-04**: 用户可以批量删除图片

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### 高级搜索 (Advanced Search)

- **ASRC-01**: 用户可以保存搜索条件为智能收藏夹
- **ASRC-02**: 用户可以按图片尺寸范围搜索
- **ASRC-03**: 用户可以按创建时间范围搜索

### Booru 集成 (Booru Integration)

- **BOOR-01**: 系统可以从 Danbooru/Gelbooru 同步标签
- **BOOR-02**: 用户可以手动触发标签同步

### 高级管理 (Advanced Management)

- **ADVM-01**: 系统支持图片评分功能
- **ADVM-02**: 系统支持图片备注功能
- **ADVM-03**: 系统支持 NSFW/SFW 内容过滤

### 部署 (Deployment)

- **DEPL-01**: 系统支持 Docker 部署
- **DEPL-02**: 系统提供 Web 管理后台

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| 社交功能（分享/关注） | 违反隐私优先的个人图库定位；复杂性爆炸 |
| 云同步/云存储 | 成本高、复杂度高、隐私顾虑 |
| 视频管理 | 范围蔓延风险；元数据需求完全不同 |
| 图片编辑 | 不是核心价值；系统工具已存在 |
| 内置浏览器/下载器 | 法律灰色地带；维护负担重 |
| 插件系统（v1） | 过早抽象；API 不稳定 |
| 多用户账户 | 增加认证复杂性；不是主要用例 |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CORE-01 | Phase 1 | Pending |
| CORE-02 | Phase 1 | Pending |
| CORE-03 | Phase 1 | Pending |
| CORE-04 | Phase 1 | Pending |
| IMPT-01 | Phase 1 | Pending |
| IMPT-02 | Phase 1 | Pending |
| IMPT-03 | Phase 1 | Pending |
| IMPT-04 | Phase 1 | Pending |
| IMPT-05 | Phase 2 | Pending |
| IMPT-06 | Phase 2 | Pending |
| IMPT-07 | Phase 2 | Pending |
| GALR-01 | Phase 2 | Pending |
| GALR-02 | Phase 2 | Pending |
| GALR-03 | Phase 2 | Pending |
| GALR-04 | Phase 2 | Pending |
| GALR-05 | Phase 2 | Pending |
| AIRE-01 | Phase 3 | Pending |
| AIRE-02 | Phase 3 | Pending |
| AIRE-03 | Phase 3 | Pending |
| AIRE-04 | Phase 3 | Pending |
| AIRE-05 | Phase 3 | Pending |
| AIRE-06 | Phase 3 | Pending |
| TAGS-01 | Phase 3 | Pending |
| TAGS-02 | Phase 3 | Pending |
| TAGS-03 | Phase 3 | Pending |
| TAGS-04 | Phase 3 | Pending |
| TAGS-05 | Phase 3 | Pending |
| DUPD-01 | Phase 4 | Pending |
| DUPD-02 | Phase 4 | Pending |
| DUPD-03 | Phase 4 | Pending |
| DUPD-04 | Phase 4 | Pending |
| DUPD-05 | Phase 4 | Pending |
| SRCH-01 | Phase 4 | Pending |
| SRCH-02 | Phase 4 | Pending |
| SRCH-03 | Phase 4 | Pending |
| SRCH-04 | Phase 4 | Pending |
| SRCH-05 | Phase 4 | Pending |
| COLL-01 | Phase 5 | Pending |
| COLL-02 | Phase 5 | Pending |
| COLL-03 | Phase 5 | Pending |
| COLL-04 | Phase 5 | Pending |
| COLL-05 | Phase 5 | Pending |
| BTCH-01 | Phase 5 | Pending |
| BTCH-02 | Phase 5 | Pending |
| BTCH-03 | Phase 5 | Pending |
| BTCH-04 | Phase 5 | Pending |

**Coverage:**
- v1 requirements: 47 total
- Mapped to phases: 47
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-14*
*Last updated: 2026-03-14 after initial definition*