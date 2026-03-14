# Project Research Summary

**Project:** ACGWarehouse（二次元图片库）
**Domain:** Anime/Manga Image Library Management System
**Researched:** 2026-03-14
**Confidence:** HIGH

## Executive Summary

ACGWarehouse 是一个面向二次元爱好者的图片库管理系统，采用 Go 后端 + Flutter 前端架构。研究分析了 6 个竞品（Hydrus、ImoutoRebirth、FEMBOY 等），确定了行业标准栈和技术模式。

核心发现：
1. **技术栈成熟**：Go + Flutter 组合适合图片处理密集型应用，govips/libvips 是业界标准的图片处理方案
2. **AI 集成是关键差异化**：DeepDanbooru 是动漫角色识别的行业标准，可作为独立微服务运行
3. **架构模式清晰**：分层客户端-服务器架构，后台工作线程处理扫描/缩略图/AI 等异步任务
4. **主要陷阱已识别**：图片存储、内存管理、AI 集成等方面有明确的预防策略

## Key Findings

### Recommended Stack

**后端核心 (Go):**
- **govips v2.16.0** — 图片处理，封装 libvips，比纯 Go 方案快 10 倍
- **Gin v1.10.0+** — Web 框架，最成熟的 Go Web 生态
- **ncruces/go-sqlite3 v0.20.0+** — 纯 Go SQLite 驱动，便于跨平台编译
- **pgx v5.7.0+** — PostgreSQL 驱动，支持连接池
- **vitali-fedulov/imagehash2 v1.0.3+** — 感知哈希，用于相似图片检测

**前端核心 (Flutter):**
- **Flutter 3.27.x+** — 跨平台 UI 框架
- **waterfall_flow** — 瀑布流布局，适合动漫图片的多样长宽比
- **flutter_riverpod v2.6.0+** — 状态管理，官方推荐替代 Provider
- **cached_network_image v3.4.0+** — 图片加载缓存

**AI 服务:**
- **DeepDanbooru** — 动漫角色识别行业标准，作为 Python 微服务运行

### Expected Features

**Must have (table stakes):**
- 图片导入/扫描 — 用户期望导入现有文件夹
- 基础图库浏览 — 网格视图、缩略图
- 文本搜索 — 按文件名/基本元数据查找
- 收藏夹/相册管理 — 分组图片
- 重复检测 — 大型收藏集必需

**Should have (competitive):**
- AI 角色识别 — 核心差异化功能
- 自动打标签 — 显著减少手动工作
- 以图搜图 — 大型收藏集非常有用
- 跨平台客户端 — Flutter 天然支持

**Defer (v2+):**
- 社交功能/分享 — 违反隐私优先定位
- 云同步 — 成本和复杂性高
- 视频管理 — 范围蔓延风险

### Architecture Approach

采用分层客户端-服务器架构：

**Major components:**
1. **Ingestion Layer** — 文件扫描和导入
2. **Processing Layer** — 缩略图、AI 分析、去重
3. **Storage Layer** — 文件系统存图片，数据库存元数据
4. **API Layer** — RESTful 端点供客户端访问
5. **Client Layer** — Flutter 前端，支持离线优先

**关键原则：**
- 图片存储在文件系统，数据库只存元数据
- AI 调用永远异步，通过任务队列处理
- Flutter 本地缓存实现离线优先体验

### Critical Pitfalls

1. **数据库存 BLOB** — 导致性能灾难，改用文件系统存储图片
2. **感知哈希误判** — 动漫图片构图相似易碰撞，使用多种哈希算法组合
3. **Flutter 内存爆炸** — 1000+ 图片 OOM，必须使用缩略图和正确的图片回收
4. **AI API 速率限制** — 扫描大型库时 429 错误，需要队列和重试机制
5. **角色识别误判** — 相似角色脸容易混淆，设置置信度阈值

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: 基础架构与图片扫描
**Rationale:** 核心数据模型和存储是所有后续功能的基础
**Delivers:** 项目骨架、数据库 schema、图片扫描入库
**Addresses:** 存储架构、数据库设计
**Avoids:** BLOB 存储陷阱

### Phase 2: 缩略图与图片浏览
**Rationale:** 用户需要先能看到图片才能进行其他操作
**Delivers:** 缩略图生成、基础图库视图（网格/瀑布流）
**Uses:** govips、waterfall_flow
**Implements:** 处理层、客户端 UI 层

### Phase 3: AI 角色识别与标签
**Rationale:** 核心差异化功能，依赖图片导入完成
**Delivers:** DeepDanbooru 集成、自动打标签、角色数据库
**Addresses:** AI 集成陷阱
**Uses:** Python 微服务架构

### Phase 4: 重复检测与以图搜图
**Rationale:** 需要感知哈希和 AI 嵌入已建立
**Delivers:** 相似图片检测、去重、以图搜图
**Addresses:** 哈希误判陷阱
**Uses:** imagehash2、向量索引

### Phase 5: 收藏夹与高级功能
**Rationale:** 基础功能完善后的增强
**Delivers:** 收藏夹/相册管理、智能收藏、标签浏览
**Uses:** Riverpod 状态管理

### Phase 6: 优化与部署
**Rationale:** 功能完善后的性能优化和生产部署
**Delivers:** PostgreSQL 迁移、Docker 部署、Web 管理后台
**Addresses:** 扩展性陷阱

### Phase Ordering Rationale

- 先建立数据模型和存储（Phase 1），再构建 UI（Phase 2）
- AI 功能依赖图片已导入（Phase 3 在 Phase 1 后）
- 搜索功能依赖标签和哈希（Phase 4 在 Phase 2-3 后）
- 增强功能在核心功能稳定后（Phase 5）
- 部署优化在最后（Phase 6）

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3:** AI 集成复杂，需要研究 DeepDanbooru API 和部署方式
- **Phase 4:** 向量索引和相似度搜索需要研究具体实现

Phases with standard patterns (skip research-phase):
- **Phase 1:** Go 项目结构和数据库 schema 有成熟模式
- **Phase 2:** Flutter 图片网格有成熟组件

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | 所有版本已通过官方文档验证 |
| Features | HIGH | 6 个竞品分析，功能分类清晰 |
| Architecture | HIGH | 参考 Immich/PhotoPrism 生产架构 |
| Pitfalls | HIGH | 基于真实问题和解决方案 |

**Overall confidence:** HIGH

### Gaps to Address

- **AI 服务部署**: DeepDanbooru 具体部署配置需要在 Phase 3 规划时研究
- **向量索引**: 大规模相似度搜索的具体实现需要进一步调研（如果图片量 > 100k）

## Sources

### Primary (HIGH confidence)
- Context7 — Go (govips, Gin, pgx), Flutter (Riverpod, waterfall_flow)
- GitHub — 竞品代码分析 (Hydrus, ImoutoRebirth, FEMBOY, Mangatsu)
- 官方文档 — libvips, DeepDanbooru

### Secondary (MEDIUM confidence)
- 行业文章 — 图片库架构最佳实践
- 社区讨论 — 动漫图片管理常见问题

---
*Research completed: 2026-03-14*
*Ready for roadmap: yes*