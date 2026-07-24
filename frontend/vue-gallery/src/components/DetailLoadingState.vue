<template>
  <div class="container detail-stage detail-loading" role="status" aria-label="图片加载中" aria-live="polite">
    <article class="viewer cinema-viewer detail-loading-viewer" aria-hidden="true">
      <div class="viewer-art detail-loading-art" data-viewer-art>
        <span class="detail-loading-sheen" aria-hidden="true"></span>
      </div>
    </article>

    <aside class="stack detail-loading-sidebar" aria-hidden="true">
      <div class="panel panel-raised detail-loading-panel">
        <div class="detail-loading-line detail-loading-line--eyebrow"></div>
        <div class="detail-loading-line detail-loading-line--title"></div>
        <div class="detail-loading-line detail-loading-line--body"></div>
        <div class="detail-loading-chips">
          <span></span>
          <span></span>
          <span></span>
        </div>
        <div class="divider"></div>
        <div class="grid-2">
          <div class="detail-loading-button"></div>
          <div class="detail-loading-button"></div>
        </div>
      </div>

      <div class="panel detail-loading-panel detail-loading-panel--compact">
        <div class="detail-loading-line detail-loading-line--eyebrow"></div>
        <div class="detail-loading-line detail-loading-line--subtitle"></div>
        <div class="detail-loading-control"></div>
        <div class="detail-loading-button detail-loading-button--wide"></div>
      </div>
    </aside>
  </div>
</template>

<style scoped>
.detail-loading {
  pointer-events: none;
}

.detail-loading-viewer {
  isolation: isolate;
  min-height: min(72vh, 760px);
  overflow: hidden;
  display: grid;
  place-items: center;
  position: relative;
  border-radius: var(--radius-lg);
  border: 1px solid color-mix(in oklab, var(--fg), transparent 78%);
  background:
    radial-gradient(circle at 50% 18%, color-mix(in oklab, var(--accent), transparent 88%), transparent 42%),
    linear-gradient(
      160deg,
      color-mix(in oklab, var(--fg), transparent 10%),
      color-mix(in oklab, var(--fg), transparent 4%)
    );
  box-shadow: var(--elev-raised);
}

.detail-loading-art {
  width: min(92%, 720px);
  max-height: min(68vh, 700px);
  min-height: 280px;
  border-radius: var(--radius-md);
  background: color-mix(in oklab, var(--fg), transparent 18%);
  position: relative;
  overflow: hidden;
  box-shadow: inset 0 0 0 1px color-mix(in oklab, var(--surface), transparent 72%);
}

.detail-loading-art::before,
.detail-loading-art::after {
  content: none;
}

.detail-loading-sheen {
  position: absolute;
  inset: -18%;
  z-index: 2;
  background: linear-gradient(
    110deg,
    transparent 28%,
    color-mix(in oklab, var(--surface), transparent 56%) 46%,
    transparent 64%
  );
  opacity: 0.72;
  transform: translateX(-42%);
  animation: detail-loading-sheen 1.45s var(--ease-standard) infinite;
}

.detail-loading-panel {
  display: grid;
  gap: var(--space-4);
}

.detail-loading-panel--compact {
  gap: var(--space-3);
}

.detail-loading-line,
.detail-loading-button,
.detail-loading-control,
.detail-loading-chips span {
  position: relative;
  overflow: hidden;
  border-radius: var(--radius-pill);
  background: color-mix(in oklab, var(--surface-warm), var(--surface) 58%);
}

.detail-loading-line::after,
.detail-loading-button::after,
.detail-loading-control::after,
.detail-loading-chips span::after {
  content: "";
  position: absolute;
  inset: 0;
  background: linear-gradient(
    100deg,
    transparent 20%,
    color-mix(in oklab, var(--surface), transparent 34%) 50%,
    transparent 80%
  );
  transform: translateX(-100%);
  animation: detail-loading-sheen 1.6s var(--ease-standard) infinite;
}

.detail-loading-line--eyebrow {
  width: 96px;
  height: 12px;
}

.detail-loading-line--title {
  width: min(88%, 280px);
  height: 42px;
  border-radius: var(--radius-md);
}

.detail-loading-line--subtitle {
  width: min(72%, 220px);
  height: 24px;
  border-radius: var(--radius-md);
}

.detail-loading-line--body {
  width: 100%;
  height: 18px;
  border-radius: var(--radius-md);
}

.detail-loading-chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin-top: var(--space-2);
}

.detail-loading-chips span {
  width: 82px;
  height: 34px;
}

.detail-loading-button,
.detail-loading-control {
  min-height: 44px;
  border-radius: var(--radius-md);
}

.detail-loading-button--wide {
  width: min(100%, 180px);
}

@keyframes detail-loading-sheen {
  to {
    transform: translateX(100%);
  }
}

@media (max-width: 744px) {
  .detail-loading-viewer {
    min-height: 420px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .detail-loading-sheen,
  .detail-loading-line::after,
  .detail-loading-button::after,
  .detail-loading-control::after,
  .detail-loading-chips span::after {
    animation: none;
    opacity: 0;
  }
}
</style>
