import { ref, computed } from 'vue'
import type { CarouselSlide } from '@/types'

export function useCarousel(slides: CarouselSlide[]) {
  const currentIndex = ref(0)

  const next = () => {
    currentIndex.value = (currentIndex.value + 1) % slides.length
  }

  const prev = () => {
    currentIndex.value = (currentIndex.value - 1 + slides.length) % slides.length
  }

  const goto = (index: number) => {
    currentIndex.value = index
  }

  const currentSlide = computed(() => slides[currentIndex.value])
  
  const offset = computed(() => `${currentIndex.value * -100}%`)
  
  const statusText = computed(() => {
    const slide = slides[currentIndex.value]
    return `第 ${currentIndex.value + 1} 张，共 ${slides.length} 张：${slide.title}`
  })

  return {
    currentIndex,
    currentSlide,
    offset,
    statusText,
    next,
    prev,
    goto
  }
}