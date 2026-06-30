<script setup lang="ts">
import type { CarouselSlide } from '@/types'
import { useCarousel } from '@/composables/useCarousel'

const props = defineProps<{
  slides: CarouselSlide[]
}>()

const { currentIndex, offset, next, prev, goto } = useCarousel(props.slides)

function carouselStatus(index: number, total: number): string {
  return `第 ${index + 1} 张，共 ${total} 张社区焦点作品`
}

function formatMetric(value: number): string {
  if (!Number.isFinite(value)) return '0'
  if (Math.abs(value) >= 100) return value.toFixed(0)
  return value.toFixed(1)
}

function imageAspectRatio(slide: CarouselSlide): string {
  const width = slide.imageWidth
  const height = slide.imageHeight
  if (width === undefined || height === undefined || width <= 0 || height <= 0) {
    return '4 / 5'
  }
  return `${width} / ${height}`
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'ArrowLeft') {
    event.preventDefault()
    prev()
  }
  if (event.key === 'ArrowRight') {
    event.preventDefault()
    next()
  }
}
</script>

<template>
  <aside
    class="panel panel-raised community-carousel-panel"
    role="region"
    aria-labelledby="carousel-title"
    aria-roledescription="carousel"
    tabindex="0"
    @keydown="handleKeydown"
  >
    <div class="panel-head community-carousel-head">
      <div>
        <p class="eyebrow">本周社区焦点</p>
        <h3 id="carousel-title">社区精选轮播</h3>
      </div>
      <div class="carousel-controls" aria-label="切换本周社区焦点">
        <button class="btn btn-secondary btn-small carousel-action" type="button" @click="prev" aria-label="查看上一个社区焦点">上一张</button>
        <button class="btn btn-primary btn-small carousel-action" type="button" @click="next" aria-label="查看下一个社区焦点">下一张</button>
      </div>
    </div>

    <div class="carousel-viewport">
      <div class="carousel-track" :style="{ '--carousel-offset': offset }">
        <article
          v-for="(slide, index) in slides"
          :key="slide.id"
          class="carousel-slide"
          :class="{ 'is-active': index === currentIndex }"
          role="group"
          aria-roledescription="slide"
          :aria-label="`${index + 1} / ${slides.length}：本周社区焦点作品`"
          :aria-hidden="index !== currentIndex"
        >
          <div class="focus-card">
            <RouterLink
              class="focus-art-link"
              :to="{ name: 'detail', query: { id: slide.id } }"
              :aria-label="`查看本周第 ${index + 1} 位作品详情`"
            >
              <span class="focus-art" :class="`focus-art-${slide.artVariant}`">
                <img
                  v-if="slide.imageUrl"
                  :src="slide.imageUrl"
                  :alt="`本周社区焦点第 ${index + 1} 张作品`"
                  :style="{ aspectRatio: imageAspectRatio(slide) }"
                  loading="lazy"
                />
              </span>
              <span class="focus-art-hint">查看详情</span>
            </RouterLink>
            <div class="focus-copy">
              <div class="focus-label-row">
                <span class="tag" :class="{ 'is-hot': slide.tagType === 'hot' }">{{ slide.tag }}</span>
                <span class="carousel-position">{{ index + 1 }} / {{ slides.length }}</span>
              </div>
              <h4>本周第 {{ index + 1 }} 位社区焦点</h4>
              <p>{{ slide.description }}</p>
              <div class="focus-stats" :aria-label="`${slide.title}数据`">
                <span><strong>{{ formatMetric(slide.score) }}</strong> 热度</span>
                <span><strong>{{ slide.favorites }}</strong> 收藏</span>
              </div>
            </div>
          </div>
        </article>
      </div>
    </div>

    <div class="carousel-footer">
      <div class="carousel-progress" aria-hidden="true">
        <span
          class="carousel-progress-bar"
          :style="{ '--carousel-progress': `${((currentIndex + 1) / Math.max(slides.length, 1)) * 100}%` }"
        ></span>
      </div>
      <div class="carousel-rail" role="group" aria-label="本周社区焦点快速导航">
        <button
          v-for="(slide, index) in slides"
          :key="slide.id"
          class="carousel-rail-dot"
          type="button"
          :aria-label="`查看第 ${index + 1} 张社区焦点作品`"
          :aria-current="index === currentIndex"
          @click="goto(index)"
        >
          <span class="carousel-rail-num">{{ index + 1 }}</span>
        </button>
      </div>
      <p class="carousel-status" aria-live="polite">{{ carouselStatus(currentIndex, slides.length) }}</p>
    </div>
  </aside>
</template>

<style scoped>
.community-carousel-panel {
  position: relative;
  isolation: isolate;
  overflow: hidden;
  container-type: inline-size;
  padding: clamp(var(--space-5), 3vw, var(--space-8));
  border-color: color-mix(in oklab, var(--border), var(--surface) 12%);
  background:
    linear-gradient(145deg, color-mix(in oklab, var(--surface), var(--surface-warm) 10%), var(--surface) 64%),
    var(--surface);
  box-shadow:
    0 26px 64px color-mix(in oklab, var(--fg), transparent 88%),
    inset 0 1px 0 color-mix(in oklab, var(--surface), transparent 18%);
}

.community-carousel-panel::before {
  content: "";
  position: absolute;
  inset: var(--space-3);
  z-index: -1;
  border-radius: calc(var(--radius-lg) - 6px);
  background:
    radial-gradient(circle at 18% 8%, color-mix(in oklab, var(--surface-warm), transparent 18%), transparent 34%),
    radial-gradient(circle at 88% 16%, color-mix(in oklab, var(--accent), transparent 84%), transparent 28%);
  opacity: 0.82;
  pointer-events: none;
}

.community-carousel-head {
  align-items: center;
  gap: var(--space-5);
  margin-bottom: clamp(var(--space-5), 3vw, var(--space-8));
}

.community-carousel-head h3 {
  margin: 0;
  color: var(--fg);
  font-size: clamp(var(--text-xl), 3.6vw, var(--text-2xl));
  line-height: var(--leading-tight);
  text-wrap: balance;
}

.carousel-controls {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  flex: 0 0 auto;
}

.carousel-action {
  min-width: 76px;
  transition:
    transform 260ms cubic-bezier(0.16, 1, 0.3, 1),
    box-shadow 260ms cubic-bezier(0.16, 1, 0.3, 1),
    background 260ms cubic-bezier(0.16, 1, 0.3, 1);
}

.carousel-action:hover {
  transform: translateY(-1px);
}

.carousel-action:active {
  transform: translateY(1px) scale(0.98);
}

.carousel-viewport {
  overflow: hidden;
  border-radius: calc(var(--radius-lg) + 6px);
  border: 1px solid color-mix(in oklab, var(--border), var(--surface) 10%);
  background:
    linear-gradient(150deg, color-mix(in oklab, var(--surface-warm), var(--surface) 36%), var(--surface) 72%);
  box-shadow:
    inset 0 1px 0 color-mix(in oklab, var(--surface), transparent 8%),
    inset 0 -1px 0 color-mix(in oklab, var(--border), transparent 28%);
}

.carousel-track {
  display: flex;
  transform: translateX(var(--carousel-offset, 0%));
  transition: transform 440ms cubic-bezier(0.16, 1, 0.3, 1);
  will-change: transform;
}

.carousel-slide {
  flex: 0 0 100%;
  min-width: 0;
  padding: clamp(var(--space-4), 3vw, var(--space-6));
  opacity: 0.5;
  transition: opacity var(--motion-base) var(--ease-standard);
}

.carousel-slide.is-active {
  opacity: 1;
}

.carousel-slide[aria-hidden="true"] {
  pointer-events: none;
}

.focus-card {
  display: grid;
  grid-template-columns: minmax(220px, 0.92fr) minmax(0, 1.08fr);
  gap: clamp(var(--space-5), 4vw, var(--space-8));
  align-items: stretch;
  min-height: clamp(320px, 34vw, 430px);
  padding: clamp(var(--space-3), 2vw, var(--space-4));
  border-radius: calc(var(--radius-lg) + 2px);
  background: color-mix(in oklab, var(--surface), transparent 6%);
  box-shadow:
    0 18px 36px color-mix(in oklab, var(--fg), transparent 92%),
    inset 0 0 0 1px color-mix(in oklab, var(--surface), transparent 20%);
}

.focus-art-link {
  position: relative;
  display: grid;
  align-items: center;
  min-width: 0;
  min-height: 0;
  color: inherit;
  text-decoration: none;
  border-radius: var(--radius-lg);
  outline: none;
}

.focus-art-link:focus-visible {
  box-shadow: var(--focus-ring);
}

.focus-art {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  min-height: clamp(260px, 31vw, 390px);
  position: relative;
  overflow: hidden;
  border-radius: var(--radius-lg);
  background:
    linear-gradient(135deg, color-mix(in oklab, var(--accent), var(--surface) 82%), var(--surface));
  box-shadow:
    0 18px 34px color-mix(in oklab, var(--fg), transparent 88%),
    inset 0 0 0 1px color-mix(in oklab, var(--surface), transparent 34%);
}

.focus-art img {
  width: auto;
  height: auto;
  max-width: 100%;
  max-height: min(100%, 420px);
  object-fit: contain;
  position: relative;
  z-index: 1;
  transform: scale(1.01);
  transition:
    transform 520ms cubic-bezier(0.16, 1, 0.3, 1),
    filter 520ms cubic-bezier(0.16, 1, 0.3, 1);
}

.focus-art-link:hover .focus-art img {
  transform: scale(1.045);
  filter: saturate(1.04);
}

.focus-art::before,
.focus-art::after {
  content: "";
  position: absolute;
  border-radius: 999px;
}

.focus-art:has(img)::before,
.focus-art:has(img)::after {
  content: none;
}

.focus-art::before {
  width: 48%;
  aspect-ratio: 1;
  left: 14%;
  top: 18%;
  background: color-mix(in oklab, var(--surface), transparent 16%);
  box-shadow: 84px 48px 0 color-mix(in oklab, var(--fg), transparent 88%);
}

.focus-art::after {
  width: 72%;
  height: 28%;
  right: -18%;
  bottom: 12%;
  background: color-mix(in oklab, var(--accent), var(--fg) 14%);
  transform: rotate(-16deg);
  opacity: 0.72;
}

.focus-art-character {
  background: linear-gradient(135deg, color-mix(in oklab, var(--surface-warm), var(--accent) 18%), color-mix(in oklab, var(--surface), var(--fg) 8%));
}

.focus-art-album {
  background: linear-gradient(135deg, color-mix(in oklab, var(--surface), var(--success) 12%), color-mix(in oklab, var(--surface-warm), var(--accent) 18%));
}

.focus-art-hint {
  position: absolute;
  right: var(--space-4);
  bottom: var(--space-4);
  z-index: 2;
  display: inline-flex;
  align-items: center;
  min-height: 34px;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-pill);
  background: color-mix(in oklab, var(--fg), transparent 12%);
  color: var(--surface);
  font-size: var(--text-xs);
  font-weight: 900;
  box-shadow: 0 10px 24px color-mix(in oklab, var(--fg), transparent 78%);
  opacity: 0;
  transform: translateY(6px);
  transition:
    opacity 260ms cubic-bezier(0.16, 1, 0.3, 1),
    transform 260ms cubic-bezier(0.16, 1, 0.3, 1);
}

.focus-art-link:hover .focus-art-hint,
.focus-art-link:focus-visible .focus-art-hint {
  opacity: 1;
  transform: translateY(0);
}

.focus-copy {
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-width: 0;
  gap: var(--space-4);
  padding: clamp(var(--space-2), 2vw, var(--space-5)) var(--space-2);
}

.focus-label-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
}

.carousel-position {
  color: var(--muted);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  font-weight: 900;
  font-variant-numeric: tabular-nums;
}

.focus-copy h4 {
  margin: 0;
  color: var(--fg);
  font-size: clamp(var(--text-xl), 3.4vw, var(--text-2xl));
  line-height: var(--leading-tight);
  letter-spacing: 0;
  text-wrap: balance;
}

.focus-copy p {
  margin: 0;
  max-width: 38ch;
  color: var(--fg-2);
  font-size: clamp(var(--text-base), 1.8vw, var(--text-lg));
  line-height: 1.62;
  overflow-wrap: anywhere;
}

.focus-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, max-content));
  gap: var(--space-3);
  margin-top: var(--space-2);
}

.focus-stats span {
  display: inline-flex;
  align-items: baseline;
  gap: var(--space-2);
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid color-mix(in oklab, var(--border), var(--surface) 16%);
  border-radius: var(--radius-md);
  background: color-mix(in oklab, var(--surface), transparent 4%);
  color: var(--muted);
  font-size: var(--text-sm);
  font-weight: 800;
  box-shadow: inset 0 1px 0 color-mix(in oklab, var(--surface), transparent 10%);
}

.focus-stats strong {
  color: var(--fg);
  font-family: var(--font-mono);
  font-size: var(--text-base);
  font-variant-numeric: tabular-nums;
}

.carousel-footer {
  display: grid;
  grid-template-columns: minmax(120px, 1fr) auto auto;
  align-items: center;
  gap: var(--space-4);
  margin-top: var(--space-5);
}

.carousel-progress {
  height: 6px;
  overflow: hidden;
  border-radius: var(--radius-pill);
  background: color-mix(in oklab, var(--border-soft), var(--surface) 22%);
  box-shadow: inset 0 0 0 1px color-mix(in oklab, var(--border), transparent 40%);
}

.carousel-progress-bar {
  display: block;
  width: var(--carousel-progress, 0%);
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--fg), color-mix(in oklab, var(--accent), var(--fg) 14%));
  transition: width 440ms cubic-bezier(0.16, 1, 0.3, 1);
}

.carousel-rail {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-2);
  min-width: 0;
}

.carousel-rail-dot {
  display: inline-grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border: 1px solid color-mix(in oklab, var(--border), var(--surface) 6%);
  border-radius: 50%;
  background: color-mix(in oklab, var(--surface), transparent 4%);
  color: var(--fg-2);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  font-weight: 900;
  font-variant-numeric: tabular-nums;
  transition:
    transform 220ms cubic-bezier(0.16, 1, 0.3, 1),
    background 220ms cubic-bezier(0.16, 1, 0.3, 1),
    color 220ms cubic-bezier(0.16, 1, 0.3, 1),
    border-color 220ms cubic-bezier(0.16, 1, 0.3, 1);
}

.carousel-rail-dot:hover {
  transform: translateY(-1px);
  border-color: color-mix(in oklab, var(--accent), var(--border) 45%);
}

.carousel-rail-dot:active {
  transform: translateY(1px) scale(0.96);
}

.carousel-rail-dot[aria-current="true"] {
  background: var(--fg);
  color: var(--surface);
  border-color: var(--fg);
}

.carousel-rail-dot:focus-visible {
  box-shadow: var(--focus-ring);
}

.carousel-status {
  margin: 0;
  color: var(--muted);
  font-size: var(--text-xs);
  white-space: nowrap;
}

@media (max-width: 920px) {
  .focus-card {
    grid-template-columns: minmax(190px, 0.86fr) minmax(0, 1.14fr);
    gap: var(--space-5);
    min-height: 320px;
  }

  .focus-art {
    min-height: clamp(230px, 36vw, 340px);
  }

  .focus-copy p {
    max-width: 42ch;
    font-size: var(--text-base);
  }

  .carousel-footer {
    grid-template-columns: 1fr;
    align-items: stretch;
  }

  .carousel-rail {
    justify-content: flex-start;
    overflow-x: auto;
    padding-bottom: var(--space-1);
    scrollbar-width: thin;
  }

  .carousel-status {
    white-space: normal;
  }
}

@container (max-width: 620px) {
  .focus-card {
    grid-template-columns: 1fr;
    gap: var(--space-4);
    min-height: auto;
  }

  .focus-art-link {
    height: auto;
  }

  .focus-art {
    height: auto;
    min-height: clamp(240px, 68cqw, 380px);
  }

  .focus-art img {
    width: auto;
    max-height: min(72cqw, 380px);
  }

  .focus-copy {
    padding: var(--space-2);
  }

  .focus-copy p {
    max-width: 44ch;
  }

  .focus-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .focus-stats span {
    justify-content: center;
  }

  .carousel-footer {
    grid-template-columns: 1fr;
    align-items: stretch;
  }

  .carousel-rail {
    justify-content: flex-start;
    overflow-x: auto;
    padding-bottom: var(--space-1);
  }

  .carousel-status {
    white-space: normal;
  }
}

@media (max-width: 744px) {
  .community-carousel-head {
    align-items: stretch;
  }

  .carousel-controls {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    width: 100%;
  }

  .carousel-action {
    width: 100%;
    min-height: 42px;
  }

  .carousel-slide {
    padding: var(--space-3);
  }

  .focus-card {
    grid-template-columns: 1fr;
    gap: var(--space-4);
    min-height: auto;
    padding: var(--space-3);
  }

  .focus-art {
    min-height: clamp(230px, 68vw, 380px);
  }

  .focus-copy {
    padding: var(--space-2);
  }

  .focus-copy h4 {
    font-size: clamp(var(--text-lg), 7vw, var(--text-xl));
  }

  .focus-stats {
    grid-template-columns: 1fr 1fr;
  }

  .focus-stats span {
    justify-content: center;
  }

  .carousel-rail-dot {
    width: 40px;
    height: 40px;
    flex: 0 0 auto;
  }
}

@media (max-width: 420px) {
  .community-carousel-panel {
    padding: var(--space-4);
  }

  .carousel-viewport {
    border-radius: var(--radius-lg);
  }

  .focus-stats {
    grid-template-columns: 1fr;
  }

  .focus-art-hint {
    opacity: 1;
    transform: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .carousel-track,
  .carousel-action,
  .focus-art img,
  .focus-art-hint,
  .carousel-progress-bar,
  .carousel-rail-dot {
    transition-duration: 1ms;
  }
}
</style>
