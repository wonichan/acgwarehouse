<!-- TRELLIS:START -->
# Trellis Instructions

These instructions are for AI assistants working in this project.

This project is managed by Trellis. The working knowledge you need lives under `.trellis/`:

- `.trellis/workflow.md` — development phases, when to create tasks, skill routing
- `.trellis/spec/` — package- and layer-scoped coding guidelines (read before writing code in a given layer)
- `.trellis/workspace/` — per-developer journals and session traces
- `.trellis/tasks/` — active and archived tasks (PRDs, research, jsonl context)

If a Trellis command is available on your platform (e.g. `/trellis:finish-work`, `/trellis:continue`), prefer it over manual steps. Not every platform exposes every command.

If you're using Codex or another agent-capable tool, additional project-scoped helpers may live in:
- `.agents/skills/` — reusable Trellis skills
- `.codex/agents/` — optional custom subagents

Managed by Trellis. Edits outside this block are preserved; edits inside may be overwritten by a future `trellis update`.

<!-- TRELLIS:END -->

<!-- CODEGRAPH_START -->
## CodeGraph

In repositories indexed by CodeGraph (a `.codegraph/` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tool** (when available): `codegraph_explore` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them, including dynamic-dispatch hops grep can't follow. Name a file or symbol in the query to read its current line-numbered source. If it's listed but deferred, load it by name via tool search.
- **Shell** (always works): `codegraph explore "<symbol names or question>"` prints the same output.

If there is no `.codegraph/` directory, skip CodeGraph entirely — indexing is the user's decision.
<!-- CODEGRAPH_END -->

<!-- frontend_START -->
## Frontend Visual Rules

适用范围：`frontend/vue-gallery` 及任何用户可见 UI。与根目录 `DESIGN.md`、`.trellis/spec/frontend/` 一并遵守；冲突时 **token / 色板 / 间距以 DESIGN.md 为准**，本节约束品质与交互上限。

1. **图标**：统一使用 Lucide，界面全程禁止表情符号（emoji）。
2. **设计标准**：对标 Awwwards / FWA / CSS Design Awards 的完成度与细节品质；不是照搬装置艺术站，而是在图库产品可用的前提下做到同级 polish。
3. **身份与双轨**：
   - 全站主身份：`DESIGN.md` 的 warm community archive（暖色档案、橙强调、图优先、元数据次之）。
   - 旗舰页（首页 hero、详情观影等）允许更强的叙事、实验排版与沉浸布局；搜索 / 账户 / 表单等工具页保持清晰、可扫读，不牺牲效率。
4. **创意与画布**：浏览向页面可将浏览器当作交互画布——先锋版式、流畅物理感动效、冲击力文字层次均可；动效只用 `transform` / `opacity`，并尊重 `prefers-reduced-motion` 与现有 `--motion-*` token。
5. **沉浸式体验**：图、代码驱动的渲染与 loading / 转场应一体；主图不被次要控件抢戏；禁止为「炫技」把评分、标签、收藏等核心路径藏进难发现手势。
6. **实现约束**：优先现有 design-system class 与 CSS 变量；新样式写组件 `<style scoped>`，勿向臃肿全局 `app.css` 堆 feature 选择器；三端（desktop / tablet / mobile）布局均须可用。
<!-- frontend_END -->
