# 图片展示性能调研

## Sources

- web.dev: Optimize Cumulative Layout Shift, updated 2025-02-07.
  URL: https://web.dev/articles/optimize-cls
- web.dev: Browser-level image lazy loading for the web.
  URL: https://web.dev/articles/browser-level-image-lazy-loading
- MDN: Masonry layout, modified 2026-03-09.
  URL: https://developer.mozilla.org/en-US/docs/Web/CSS/Guides/Grid_layout/Masonry_layout
- Chrome for Developers: An alternative proposal for CSS masonry, updated 2026-02-13.
  URL: https://developer.chrome.com/blog/masonry
- Masonic: high-performance virtualized masonry layouts for React.
  URL: https://github.com/jaredLunde/masonic
- Product references inspected: Unsplash, Pexels, Pinterest Ideas, Flickr Explore.
  URLs: https://unsplash.com/ , https://www.pexels.com/ , https://www.pinterest.com/ideas/ , https://www.flickr.com/explore

## Findings

- 图片流首要问题是 CLS。web.dev 将无尺寸图片列为常见 CLS 根因，并建议给图片设置 `width` / `height` 或用 CSS `aspect-ratio` 预留空间。
- 懒加载图片更需要尺寸。web.dev 指出浏览器在图片加载前不知道尺寸；若未声明宽高，图片可能以 0x0 参与初始布局，既会增加布局偏移，也可能让图库误判所有图片都在首屏内。
- 原生 CSS masonry 尚不适合生产依赖。MDN 标记 masonry 为 limited availability / experimental，不是 Baseline；Chrome 文章也说明 CSS masonry 仍有规范形态与性能模型争议。
- 大规模优秀 masonry 实现会做位置缓存、容器测量、滚动窗口、overscan、resize observer 和 infinite loader。Masonic 的实现说明表明：高性能 masonry 的核心不是 CSS columns，而是 positioner/cache + virtualization；同时仍建议尽量预先测量图片。
- 图片产品形态可分两类：
  - Pinterest / Unsplash / Pexels 类：强调探索流，适合 masonry，但必须稳定占位，新增内容不应导致可见旧内容重排。
  - Flickr / Google Photos 类：更偏相册浏览，常见 justified row/grid，适合时间轴与批量浏览，但会牺牲部分瀑布流的视觉密度。

## Implications for This Project

- 当前 `columns` 瀑布流应废弃。它交给浏览器重排列，追加分页时无法保证旧卡片位置稳定。
- 不应等 CSS 原生 masonry。兼容性和规范状态不足，且当前项目无需冒险。
- 不必立即引入虚拟化库。当前分页为 20 张/页，先用稳定 JS 分列和真实宽高占位即可；若未来单页驻留图片达到数千级，再引入虚拟化/positioner。
- 修复目标应是：
  - 数据层携带 `width` / `height`。
  - 卡片层用 `aspect-ratio` 或图片属性稳定占位。
  - 布局层用确定性分列，追加时只把新卡片放入最短列，不重排旧卡片。
  - 验证层增加静态断言和构建检查；若能跑浏览器，再记录滚动加载时 layout shift。
