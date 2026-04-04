# Phase 18: Independent Viewer & Filmstrip - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-05
**Phase:** 18-independent-viewer-filmstrip
**Areas discussed:** 窗口策略, filmstrip 范围, 查看器布局, 元信息内容, 双击放大, 切图状态, 键盘导航, 窗口记忆, filmstrip 密度

---

## 窗口策略

| Option | Description | Selected |
|--------|-------------|----------|
| 可多窗口 | 每次双击都能打开新的非模态查看器，主图库继续可用，也最接近 Windows Photos 方向 | ✓ |
| 单窗口复用 | 始终复用同一个查看器窗口，只替换当前图片 | |
| 智能复用 | 默认复用已有查看器，但提供显式“新开窗口”能力 | |

**User's choice:** 可多窗口
**Notes:** 明确要求独立查看器保持非阻塞，支持同时打开多个窗口。

---

## Filmstrip 范围

| Option | Description | Selected |
|--------|-------------|----------|
| 同文件夹 | 只展示当前图片所在文件夹的图片，和 ROADMAP 成功标准一致 | |
| 当前结果集 | 展示当前图库/搜索结果集中的相邻图片，切换更连续 | ✓ |
| 混合策略 | 优先同文件夹；若无法确定文件夹上下文，再退回当前结果集 | |

**User's choice:** 当前结果集
**Notes:** 已提醒这与 ROADMAP 的“同一文件夹” wording 不完全一致；用户明确保持当前结果集策略。

---

## 查看器布局

| Option | Description | Selected |
|--------|-------------|----------|
| 右侧固定栏 | 右侧固定元信息栏默认展开，主区看图、底部 filmstrip | ✓ |
| 右侧可折叠 | 右侧栏默认收起，需要时展开 | |
| 覆盖层 | 元信息做成覆盖层/抽屉，按按钮弹出 | |

**User's choice:** 右侧固定栏
**Notes:** 倾向桌面工作区式布局，而不是轻量覆盖层。

---

## 元信息内容

| Option | Description | Selected |
|--------|-------------|----------|
| 扩展信息 | 在 requirement 最小集之外，增加格式、路径、导入时间 | ✓ |
| 最小必需集 | 只保留文件名、分辨率、大小、标签 | |
| 分层展示 | 默认最小集，用户展开后再看更多信息 | |

**User's choice:** 扩展信息
**Notes:** 明确接受复用现有详情页的更完整元信息结构。

---

## 双击放大

| Option | Description | Selected |
|--------|-------------|----------|
| 适应窗口 ↔ 2x | 双击在适应窗口与 2x 放大之间切换 | ✓ |
| 适应窗口 ↔ 100% | 双击在适应窗口与原始尺寸之间切换 | |
| 阶梯式放大 | 1x → 2x → 3x 阶梯放大后再回到适应窗口 | |

**User's choice:** 适应窗口 ↔ 2x
**Notes:** 倾向延续当前已有的查看器实现语义。

---

## 切图状态

| Option | Description | Selected |
|--------|-------------|----------|
| 切图即重置 | 切到新图后重置为适应窗口 | ✓ |
| 保留缩放位置 | 保留上一张图片的缩放比例和拖拽位置 | |
| 只保留缩放比例 | 保留缩放比例但重置拖拽位置 | |

**User's choice:** 切图即重置
**Notes:** 希望切图行为稳定，不把上一张图片的视图状态带到新图。

---

## 键盘导航

| Option | Description | Selected |
|--------|-------------|----------|
| 基础快捷键 | `←/→` 切图，`Esc` 关闭当前查看器 | ✓ |
| 扩展快捷键 | 在基础上增加 `Home/End`、`Ctrl+W` 等桌面快捷键 | |
| 极简快捷键 | 尽量少快捷键，只保留 `Esc` | |

**User's choice:** 基础快捷键
**Notes:** 第一版不追求完整桌面快捷键矩阵，先锁定最常用路径。

---

## 窗口记忆

| Option | Description | Selected |
|--------|-------------|----------|
| 不记忆 | 每次按统一默认尺寸/位置打开 | ✓ |
| 记忆最近状态 | 新开的查看器沿用最近一次大小与位置 | |
| 每窗独立记忆 | 每个查看器窗口都独立记忆自己的状态 | |

**User's choice:** 不记忆
**Notes:** 第一版优先稳定交付，不把窗口状态持久化拉入范围。

---

## Filmstrip 密度

| Option | Description | Selected |
|--------|-------------|----------|
| 中等密度 | 中等尺寸缩略图，当前项明显高亮，兼顾信息量与可点击性 | ✓ |
| 低密度大缩略图 | 缩略图更大，同时可见数量更少 | |
| 高密度小缩略图 | 缩略图更小，同时可见数量更多 | |

**User's choice:** 中等密度
**Notes:** 倾向平衡“快速扫图”与“保持可点击可辨识”。

---

## the agent's Discretion

- 多窗口查看器的内部实例命名与标题格式
- filmstrip 的精确缩略图尺寸、边距与高亮样式
- 元信息栏字段排版、分组和视觉细节
- 缩放灵敏度与滚轮/触控板细节

## Deferred Ideas

None
