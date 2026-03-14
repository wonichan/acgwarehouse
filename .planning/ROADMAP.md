# Roadmap: ACGWarehouse（二次元图片库）

**Created:** 2026-03-14
**Granularity:** Fine (细粒度)
**Total Phases:** 6
**Total Requirements:** 47

---

## Progress Overview

| Phase | Status | Plans | Progress |
|-------|--------|-------|----------|
| 1 | ○ | 0/0 | 0% |
| 2 | ○ | 0/0 | 0% |
| 3 | ○ | 0/0 | 0% |
| 4 | ○ | 0/0 | 0% |
| 5 | ○ | 0/0 | 0% |
| 6 | ○ | 0/0 | 0% |

**Status:** ○ Pending | ◆ In Progress | ✓ Complete

---

## Phase 1: 基础架构与图片扫描

**Goal:** 建立项目骨架、数据库设计、图片扫描入库基础功能

**Duration Estimate:** 2-3 weeks

**Requirements:**
- CORE-01, CORE-02, CORE-03, CORE-04
- IMPT-01, IMPT-02, IMPT-03, IMPT-04

**Success Criteria:**
1. Go 后端项目结构初始化完成，可编译运行
2. SQLite 数据库 schema 创建完成（images, tags, collections 表）
3. RESTful API 框架搭建完成（健康检查端点可访问）
4. 图片扫描命令可扫描指定文件夹并导入图片元数据
5. 文件夹监控功能可检测新增图片并自动入库

**Deliverables:**
- Go 后端项目骨架
- 数据库 schema 和迁移脚本
- 图片扫描服务
- 文件夹监控服务
- API 基础端点

**Pitfalls Addressed:**
- Pitfall 1: 数据库存 BLOB — 数据库只存元数据，图片存文件系统

---

## Phase 2: 缩略图与图片浏览

**Goal:** 实现缩略图生成、Flutter 前端基础、图片浏览功能

**Duration Estimate:** 2-3 weeks

**Requirements:**
- IMPT-05, IMPT-06, IMPT-07
- GALR-01, GALR-02, GALR-03, GALR-04, GALR-05

**Success Criteria:**
1. 缩略图生成服务可批量处理导入图片
2. 感知哈希计算服务可计算图片相似度哈希
3. Flutter 前端可显示图片网格视图
4. Flutter 前端可显示瀑布流视图
5. 用户可查看图片详情和元数据
6. 用户可按时间/名称/大小排序浏览

**Deliverables:**
- 缩略图生成服务
- 感知哈希计算服务
- Flutter 图片网格组件
- Flutter 瀑布流组件
- 图片详情页面

**Pitfalls Addressed:**
- Pitfall 3: Flutter 内存爆炸 — 使用缩略图和正确的图片回收

---

## Phase 3: AI 角色识别与标签

**Goal:** 集成 DeepDanbooru 实现 AI 自动打标签，完成标签管理功能

**Duration Estimate:** 3-4 weeks

**Requirements:**
- AIRE-01, AIRE-02, AIRE-03, AIRE-04, AIRE-05, AIRE-06
- TAGS-01, TAGS-02, TAGS-03, TAGS-04, TAGS-05

**Success Criteria:**
1. DeepDanbooru 服务部署并可访问
2. AI 识别服务可异步处理图片并生成标签
3. 标签带有置信度分数
4. 用户可查看/确认/修改 AI 生成的标签
5. 用户可手动添加/修改/删除标签
6. 用户可按标签筛选图片

**Deliverables:**
- DeepDanbooru 微服务部署
- AI 识别集成服务
- 标签管理 API
- Flutter 标签筛选组件
- 标签确认界面

**Pitfalls Addressed:**
- Pitfall 4: AI API 速率限制 — 异步队列处理
- Pitfall 5: 角色识别误判 — 置信度阈值

---

## Phase 4: 重复检测与搜索

**Goal:** 实现相似图片检测、去重、以图搜图功能

**Duration Estimate:** 2-3 weeks

**Requirements:**
- DUPD-01, DUPD-02, DUPD-03, DUPD-04, DUPD-05
- SRCH-01, SRCH-02, SRCH-03, SRCH-04, SRCH-05

**Success Criteria:**
1. 系统可检测完全相同的图片（文件哈希）
2. 系统可检测相似图片（感知哈希）
3. 用户可设置相似度阈值
4. 用户可查看重复图片组并选择保留/删除
5. 用户可按文件名/标签搜索图片
6. 用户可上传图片进行以图搜图

**Deliverables:**
- 重复检测服务
- 搜索服务（全文 + 标签）
- 以图搜图服务
- Flutter 搜索界面
- 重复图片管理界面

**Pitfalls Addressed:**
- Pitfall 2: 感知哈希误判 — 多种哈希算法组合

---

## Phase 5: 收藏夹与批量操作

**Goal:** 实现收藏夹/相册管理、批量操作功能

**Duration Estimate:** 1-2 weeks

**Requirements:**
- COLL-01, COLL-02, COLL-03, COLL-04, COLL-05
- BTCH-01, BTCH-02, BTCH-03, BTCH-04

**Success Criteria:**
1. 用户可创建/重命名/删除收藏夹
2. 用户可添加/移除收藏夹中的图片
3. 用户可设置收藏夹封面
4. 用户可批量选择图片
5. 用户可批量添加/删除标签
6. 用户可批量移动到收藏夹或删除

**Deliverables:**
- 收藏夹管理 API
- 批量操作 API
- Flutter 收藏夹界面
- Flutter 批量选择组件

---

## Phase 6: 优化与部署

**Goal:** 性能优化、PostgreSQL 迁移支持、Docker 部署

**Duration Estimate:** 2 weeks

**Requirements:**
- (优化类需求，无新功能需求)

**Success Criteria:**
1. PostgreSQL 数据库迁移脚本完成
2. Docker Compose 配置文件完成
3. 基础 Web 管理后台可访问
4. 性能测试通过（支持 10k+ 图片）
5. 部署文档完成

**Deliverables:**
- PostgreSQL 迁移脚本
- Docker Compose 配置
- 基础 Web 管理后台
- 性能优化报告
- 部署文档

**Pitfalls Addressed:**
- Pitfall 6: 扩展性问题 — PostgreSQL 支持大型库

---

## Phase Dependencies

```
Phase 1 (基础架构)
    ↓
Phase 2 (缩略图/浏览) ← depends on Phase 1 数据模型
    ↓
Phase 3 (AI/标签) ← depends on Phase 1-2 图片已导入
    ↓
Phase 4 (去重/搜索) ← depends on Phase 2-3 哈希和标签
    ↓
Phase 5 (收藏夹/批量) ← depends on Phase 1-4 核心功能稳定
    ↓
Phase 6 (优化/部署) ← depends on Phase 1-5 功能完整
```

---

## Research Flags

Phases needing deeper research during planning:

| Phase | Research Needed |
|-------|-----------------|
| Phase 3 | DeepDanbooru 部署方式和 API 集成细节 |
| Phase 4 | 向量索引实现（如果图片量 > 100k） |

Phases with standard patterns (skip research-phase):

| Phase | Reason |
|-------|--------|
| Phase 1 | Go 项目结构和数据库 schema 有成熟模式 |
| Phase 2 | Flutter 图片网格有成熟组件 |
| Phase 5 | 收藏夹是标准 CRUD 功能 |

---

*Roadmap created: 2026-03-14*
*Last updated: 2026-03-14*