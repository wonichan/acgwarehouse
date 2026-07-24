# 执行计划：前端视觉升级（切片 C）

## 前置

- [ ] 用户评审通过 `prd.md` + `design.md` + 本文件
- [ ] `implement.jsonl` / `check.jsonl` 已填真实 spec 条目（非 seed）
- [ ] `python3 ./.trellis/scripts/task.py start`（进入 in_progress 后再写代码）
- [ ] 实现前加载 `trellis-before-dev`；UI 工作加载 design taste / redesign skills（见 frontend component-guidelines）

## 有序清单

### Phase A — 基础层

1. **依赖**  
   - 在 `frontend/vue-gallery` 安装 `lucide-vue-next`  
   - 确认 `package.json` / lock 更新  

2. **图标约定**  
   - 新增 `AppIcon.vue`（可选但推荐）或文档化 import 约定  
   - 替换 `ArtCard` 选择勾 `✓` → Lucide Check  

3. **AppHeader**  
   - 导航当前态强化  
   - 搜索/账户旁图标  
   - 小屏折叠菜单（aria-expanded、Esc 关闭）  
   - 三端目视  

4. **ArtCard polish**  
   - hover/focus 层次；不改 selection 语义  
   - 确认 Collections 等复用卡片处无破版  

5. **清理**  
   - 删除无引用 `HelloWorld.vue`（若仍无引用）  

**A 门禁**：Header + ArtCard 可演示；`npm run build` 通过。

### Phase B — Detail 旗舰

6. **Detail 布局**  
   - 主图主导；真实比例 / contain；zoom 控件 Lucide 化  
7. **默认影院衬**  
   - viewer 局部暗衬 token `color-mix`  
   - 侧栏保持可读 surface  
8. **DetailLoadingState**  
   - 与影院衬协调，防闪白  
9. **回归**  
   - 未登录收藏/标签 AuthRequired  
   - 评分保存、similar 面板  

**B 门禁**：详情默认影院衬 + 核心操作可达；build 通过。

### Phase C — Gallery 策展

10. **Hero**  
    - 层级/间距/CTA；carousel 空错态保持  
11. **Feed 节奏**  
    - section 标题/toolbar 视觉；不改 filter 与 masonry 逻辑  
12. **DailyRecommendations**  
    - 仅区块节奏，必要时轻量标题  

**C 门禁**：首页策展感可辨；无限滚动与 tag 深链仍正确；build 通过。

### Phase D — 总验

13. 三端走查：Header、Gallery、Detail  
14. reduced-motion 抽查  
15. `npm run build`  
16. `trellis-check` / 任务验收对照 PRD AC1–AC7  

## 验证命令

```bash
cd frontend/vue-gallery
npm install
npm run build
```

可选：`npm run dev` 手测路由 `/`、`/detail?id=<valid>`、`/search`、`/account`（Header 回归）。

## 风险文件

| 文件 | 风险 |
|------|------|
| `src/assets/app.css` | 误改全局导致全站回归 — 尽量 scoped |
| `src/pages/GalleryPage.vue` | 误触 masonry/observer — 只改 template/样式层次 |
| `src/pages/DetailPage.vue` | 影院衬对比度 / picker 挂载 |
| `src/components/AppHeader.vue` | 小屏菜单可访问性 |
| `package.json` | 依赖锁定 |

## 回滚点

- A 完成后可单独回滚图标+Header  
- B 完成后可单独回滚 Detail  
- C 完成后可单独回滚 Gallery 模板样式  

## 明确不做（实现时勿蔓延）

- 工具页 redesign  
- 全站 dark / 观影切换  
- webfont  
- 后端或新 API  
