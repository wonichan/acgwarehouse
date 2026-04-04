# Phase 19: Tag Management - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-05
**Phase:** 19-tag-management
**Areas discussed:** 合并与主工作流、删除确认规则、标签模型边界、批量操作深度、跨页面联动

---

## 合并与主工作流

| Option | Description | Selected |
|--------|-------------|----------|
| 列表+合并面板 | 列表为主视图，合并时打开目标标签选择/确认面板 | ✓ |
| 双栏治理台 | 左列表右详情区，治理能力集中在右栏 | |
| 独立合并向导 | 把重复标签合并做成独立工具/向导 | |

**User's choice:** 列表 + 合并面板
**Notes:** 用户希望沿用当前列表型页面骨架，不把 Phase 19 直接升级成重型治理后台；合并要成为明确工作流。

---

## 删除确认规则

| Option | Description | Selected |
|--------|-------------|----------|
| 仅删未使用标签 | 仅允许删除 `usageCount = 0` 的标签，确认中明确展示影响图片数 | ✓ |
| 允许强制删除 | 任意标签都能删，并连带移除关联 | |
| 删除前必须迁移 | 有引用的标签必须先迁移到替代标签 | |

**User's choice:** 仅删未使用标签
**Notes:** 用户要的是安全治理，不是危险删除；成功标准中的“显示受影响图片数”需要显式落到确认文案里。

---

## 标签模型边界

| Option | Description | Selected |
|--------|-------------|----------|
| 只做核心治理 | 只覆盖统计、改名、合并、删除 | |
| 加分类和别名 | 在核心治理上纳入 `primaryCategory` 与 alias | ✓ |
| 扩展到完整体系 | 进一步纳入 taxonomy / 颜色语义 / 完整分类体系 | |

**User's choice:** 加分类和别名
**Notes:** 用户希望本阶段把 `primaryCategory` 和 alias 一起正式治理，但不把完整 taxonomy 体系锁进当前阶段。

---

## 批量操作深度

| Option | Description | Selected |
|--------|-------------|----------|
| 单条为主+批量清理 | 只做高价值轻量批量能力 | |
| 完整批量治理 | 纳入批量清理、批量分类/别名整理、批量合并候选处理 | ✓ |
| 只做单条操作 | 完全不做批量 | |

**User's choice:** 完整批量治理
**Notes:** 已明确向用户提示这会扩大到“治理效率层”，但用户仍要求把完整批量能力锁进 Phase 19。

---

## 跨页面联动

| Option | Description | Selected |
|--------|-------------|----------|
| 跳转到受影响图片 | 从治理页跳到带预设筛选的图库/搜索结果 | ✓ |
| 页内预览联动 | 在治理页内嵌预览受影响图片 | |
| 仅显示统计 | 不做跨页联动 | |

**User's choice:** 跳转到受影响图片
**Notes:** 用户希望治理动作和实际图片影响之间有可验证闭环，但不希望为此把治理页做成重型图片工作区。

---

## the agent's Discretion

- 合并面板的具体表现形态（侧面板/对话框/嵌入详情区）
- 批量工具条的具体视觉布局
- 列表中统计字段、排序与搜索控件的具体信息密度

## Deferred Ideas

- 完整 taxonomy / 层级分类体系治理
- 标签颜色语义与分组视觉系统
- 页内重型图片预览工作区
- 强制删除任意标签并同时删除全部关联
