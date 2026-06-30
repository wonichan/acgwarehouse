# 每日随机推荐前端原型

## Design Read

这是 ACGWarehouse 首页里的新增发现区块，不是完整 redesign。视觉语言应延续现有 `DESIGN.md`：暖色社区档案、橙色交互强调、圆角 panel、图片优先、元数据弱化。原型目标是让“随机探索”和“社区精选/热榜算法”在同一首页中清晰并存，同时保持图片为主、必要文字辅助。

加载参考：`frontend` design router、`taste-skill`、`high-end-visual-design`、`imagegen-frontend-web`，并读取项目 `DESIGN.md`。本原型不写代码、不启动实现。

## Placement

放在现有社区精选轮播之后、图库筛选/瀑布流之前：

```text
Hero / 首页说明
↓
本周社区焦点（社区精选轮播，算法/热榜）
↓
每日随机推荐（新增，随机/公平轮转）
↓
图库筛选 tabs
↓
图库瀑布流
```

理由：用户先看到算法精选，再看到随机探索，两者语义自然分层；每日推荐不会打断首屏 hero，也不会混入普通图库列表。

## Desktop Prototype (1280px)

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ container max 1180                                                         │
│                                                                            │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ panel-raised · daily-random-panel                                     │  │
│  │                                                                      │  │
│  │  ┌─────────────── left intro rail ───────────────┐ ┌──────────────┐ │  │
│  │  │  pill: 北京时间今日更新                       │ │ mini status  │ │  │
│  │  │  H2: 每日随机推荐                             │ │  10 张       │ │  │
│  │  │  copy:                                        │ │  全站同款    │ │  │
│  │  │  随机抽取今日可展示作品，让冷门作品也有机会被看见。 │ │  公平轮转    │ │  │
│  │  └──────────────────────────────────────────────┘ └──────────────┘ │  │
│  │                                                                      │  │
│  │  ┌──────── large feature card ────────┐ ┌──── card ────┐ ┌──── card ─┐ │
│  │  │ image 2:1 / highlighted first item │ │ image square │ │ image     │ │
│  │  │ filename                           │ │ filename     │ │ filename  │ │
│  │  │ #category #score #favorites        │ │ meta         │ │ meta      │ │
│  │  └────────────────────────────────────┘ └──────────────┘ └───────────┘ │
│  │  ┌──── card ────┐ ┌──── card ────┐ ┌──── card ────┐ ┌──── card ────┐ │
│  │  │ image        │ │ image        │ │ image        │ │ image        │ │
│  │  │ meta         │ │ meta         │ │ meta         │ │ meta         │ │
│  │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘ │
│  │  ┌──── card ────┐ ┌──── card ────┐ ┌──── card ────┐                 │
│  │  │ image        │ │ image        │ │ image        │                 │
│  │  │ meta         │ │ meta         │ │ meta         │                 │
│  │  └──────────────┘ └──────────────┘ └──────────────┘                 │
│  └──────────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────────┘
```

### Composition

- Section shell: one `.panel.panel-raised` with slightly warmer inner wash using existing tokens only.
- Header: asymmetric, not another carousel header.
  - Left: pill + title + one short explanatory sentence.
  - Right: compact stats capsule showing `10 张 / 全站同款 / 公平轮转`.
  - Do not add an inline note under the header; this project is image-first, so explanatory text must stay minimal.
- Grid: “feature + small cards” bento rhythm.
  - First item gets a wide feature card to create a daily anchor.
  - Remaining 9 items use compact cards in a 3-column / 4-column responsive grid depending available width.
- Cards should reuse `ArtCard` visual language, but for this section use `selectable=false` to prevent recommendation browsing from toggling batch selection.

## Visual Direction

### Token usage

- Background: inherit page `--bg`.
- Section surface: `--surface` with subtle `color-mix()` from `--surface-warm` and `--surface`.
- Accent: `--accent` only for pill, focus ring, and small random/fairness marker.
- Text: `--fg`, `--fg-2`, `--muted`.
- Borders/shadows: `--border`, `--border-soft`, `--elev-raised`.

### Distinction from Community Picks

Community picks currently say:

- “本周社区焦点”
- “社区精选轮播”
- heat score / rank / favorites

Daily random should say only the minimum needed:

- “北京时间今日更新”
- “每日随机推荐”
- one short sentence explaining random/fair rotation

Avoid words like “精选”、“热榜”、“热门” in the daily block title/copy, except when contrasting. Avoid operational notes like timezone strings or refresh mechanics in the visible section unless needed for an error/debug state.

## Suggested Copy

```text
Eyebrow: 北京时间今日更新
Title: 每日随机推荐
Body: 随机抽取今日可展示作品，让冷门作品也有机会被看见。
Stats capsule:
  10 张
  全站同款
  公平轮转
Empty title: 今日还没有可展示作品
Empty body: 当图库里有 active 图片后，这里会展示每日随机推荐。
Error title: 每日推荐暂时不可用
Error body: 图库仍可继续浏览，稍后可重试。
```

## States

### Loading

- Header skeleton remains visible so layout does not jump.
- Render 1 wide skeleton + 9 compact skeletons matching final card shape.
- No spinner.

```text
[北京时间今日更新] 每日随机推荐
████████████████████████  copy skeleton

[wide skeleton] [small skeleton] [small skeleton]
[small skeleton] ...
```

### Empty

- Use `.panel` inside the daily shell, not plain text.
- Keep the explanatory copy; replace grid with empty-state panel.

### Error

- Error isolated to daily block; do not set GalleryPage full-page `error` if images/rankings loaded.
- Show small secondary “重试” button if implementation adds isolated retry.

### Partial invalid images

- Apply displayable-image filter before rendering.
- If fewer than 10 displayable images remain, show only the available cards. Do not add explanatory/debug text in normal product UI.

## Responsive Behavior

### Tablet (~768px)

```text
┌ daily panel ─────────────────────────┐
│ header stacked but stats capsule row │
│ [wide feature spans full width]      │
│ [card] [card]                        │
│ [card] [card]                        │
└──────────────────────────────────────┘
```

- Header becomes vertical stack.
- Feature card spans full width.
- Compact cards use 2 columns.

### Mobile (~375px)

```text
┌ daily panel ───────────────┐
│ pill                       │
│ 每日随机推荐               │
│ copy                       │
│ stats chips wrap           │
│ [feature card]             │
│ [card]                     │
│ [card]                     │
│ ...                        │
└────────────────────────────┘
```

- Single column.
- No horizontal overflow.
- The first feature card is still visually larger through image aspect ratio, not grid span.
- Minimum tap target remains the existing ArtCard link area.

## Image Generation Reference Prompt

If we want an actual visual mock image later, generate exactly one horizontal section image:

```text
Section 1 of 1: Daily Random Recommendations for ACGWarehouse homepage.
Horizontal 16:9 website section mockup, warm community archive design system, soft yellow page background, white raised rounded panel, orange accent chips, image-first anime gallery cards. Place after an existing community carousel, but show only this one section. Asymmetric compact header: left pill "北京时间今日更新", title "每日随机推荐", one short Chinese sentence; right compact stats capsule "10 张 / 全站同款 / 公平轮转". No inline note, no operational refresh text. Below, a bento image grid dominates the section: one wide featured artwork card plus nine compact artwork cards, rounded panels, warm borders, soft shadows, metadata secondary. Distinct from heat ranking: no hot/rank labels, no carousel controls. Premium but codeable Vue web UI, responsive grid implied, generous spacing, no purple AI gradients, no emojis, no fake dashboard.
```

## Implementation Notes

- Prefer reusing `ArtCard` for cards, passed `selectable=false`.
- If the feature card needs a different aspect ratio than `ArtCard`, either add a local wrapper class or create a small daily-specific card component; avoid over-generalizing unless reused elsewhere.
- Do not alter `Carousel.vue` for this block.
- `GalleryPage.vue` should keep community `carouselSlides` and daily `dailyRecommendationItems` as separate state.
- Daily recommendation API failure should set `dailyRecommendationError`, not the existing page-level `error`.
