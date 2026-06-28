<script setup lang="ts">
import type { CarouselSlide } from '@/types'
import { useCarousel } from '@/composables/useCarousel'

const props = defineProps<{
  slides: CarouselSlide[]
}>()

const { currentIndex, offset, statusText, next, prev, goto } = useCarousel(props.slides)

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
        <button class="btn btn-secondary btn-small" type="button" @click="prev" aria-label="查看上一个社区焦点">上一张</button>
        <button class="btn btn-primary btn-small" type="button" @click="next" aria-label="查看下一个社区焦点">下一张</button>
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
          :aria-label="`${index + 1} / ${slides.length}：${slide.title}`"
          :aria-hidden="index !== currentIndex"
        >
          <div class="focus-card">
            <div class="focus-art" :class="`focus-art-${slide.artVariant}`">
              <img v-if="slide.imageUrl" :src="slide.imageUrl" :alt="slide.title" loading="lazy" />
            </div>
            <div class="focus-copy">
              <div class="focus-label-row">
                <span class="tag" :class="{ 'is-hot': slide.tagType === 'hot' }">{{ slide.tag }}</span>
                <span class="carousel-position">{{ index + 1 }} / {{ slides.length }}</span>
              </div>
              <h4>{{ slide.title }}</h4>
              <p>{{ slide.description }}</p>
              <div class="focus-stats" :aria-label="`${slide.title}数据`">
                <span><strong>{{ slide.score }}</strong> 热度</span>
                <span><strong>{{ slide.favorites }}</strong> 收藏</span>
              </div>
            </div>
          </div>
        </article>
      </div>
    </div>

    <div class="carousel-footer">
      <div class="carousel-rail" role="group" aria-label="本周社区焦点快速导航">
        <button
          v-for="(slide, index) in slides"
          :key="slide.id"
          class="carousel-rail-chip"
          type="button"
          :aria-label="`查看第 ${index + 1} 张：${slide.title}`"
          :aria-current="index === currentIndex"
          @click="goto(index)"
        >
          <span class="carousel-rail-num">{{ index + 1 }}</span>
          <span class="carousel-rail-title">{{ slide.title }}</span>
        </button>
      </div>
      <p class="carousel-status" aria-live="polite">{{ statusText }}</p>
    </div>
  </aside>
</template>
