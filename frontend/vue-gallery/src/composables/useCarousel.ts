import { ref, computed, onMounted, onBeforeUnmount, onActivated, onDeactivated } from 'vue'
import type { CarouselSlide } from '@/types'

const DEFAULT_INTERVAL_MS = 3000

export interface UseCarouselOptions {
  readonly interval?: number
  readonly autoplay?: boolean
}

export function useCarousel(slides: CarouselSlide[], options: UseCarouselOptions = {}) {
  const interval = options.interval ?? DEFAULT_INTERVAL_MS
  const autoplayRequested = options.autoplay ?? true

  const currentIndex = ref(0)
  let timerId: ReturnType<typeof setTimeout> | null = null
  let hovering = false
  let focused = false
  let mounted = false
  let reducedMotion = false
  let reducedMotionQuery: MediaQueryList | null = null

  function isMultiSlide(): boolean {
    return slides.length > 1
  }

  function detectReducedMotion(): boolean {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return false
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches
  }

  function canAutoplay(): boolean {
    if (!autoplayRequested) return false
    if (!mounted) return false
    if (!isMultiSlide()) return false
    if (reducedMotion) return false
    if (hovering || focused) return false
    if (typeof document !== 'undefined' && document.visibilityState === 'hidden') return false
    return true
  }

  function clearTimer(): void {
    if (timerId !== null) {
      clearTimeout(timerId)
      timerId = null
    }
  }

  function scheduleNext(): void {
    clearTimer()
    if (!canAutoplay()) return
    timerId = setTimeout(() => {
      timerId = null
      if (!canAutoplay()) return
      currentIndex.value = (currentIndex.value + 1) % slides.length
      scheduleNext()
    }, interval)
  }

  const next = (): void => {
    if (!isMultiSlide()) return
    currentIndex.value = (currentIndex.value + 1) % slides.length
    scheduleNext()
  }

  const prev = (): void => {
    if (!isMultiSlide()) return
    currentIndex.value = (currentIndex.value - 1 + slides.length) % slides.length
    scheduleNext()
  }

  const goto = (index: number): void => {
    if (index < 0 || index >= slides.length) return
    currentIndex.value = index
    scheduleNext()
  }

  const pause = (): void => {
    hovering = true
    clearTimer()
  }

  const resume = (): void => {
    hovering = false
    scheduleNext()
  }

  const pauseByFocus = (): void => {
    focused = true
    clearTimer()
  }

  const resumeByFocus = (): void => {
    focused = false
    scheduleNext()
  }

  function handleVisibilityChange(): void {
    if (document.visibilityState === 'hidden') {
      clearTimer()
    } else {
      scheduleNext()
    }
  }

  function handleReducedMotionChange(event: MediaQueryListEvent): void {
    reducedMotion = event.matches
    if (reducedMotion) {
      clearTimer()
    } else {
      scheduleNext()
    }
  }

  function attachEnvironmentListeners(): void {
    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', handleVisibilityChange)
    }
    if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
      reducedMotionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
      reducedMotion = reducedMotionQuery.matches
      if (typeof reducedMotionQuery.addEventListener === 'function') {
        reducedMotionQuery.addEventListener('change', handleReducedMotionChange)
      } else {
        reducedMotionQuery.addListener(handleReducedMotionChange)
      }
    }
  }

  function detachEnvironmentListeners(): void {
    if (typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
    if (reducedMotionQuery !== null) {
      if (typeof reducedMotionQuery.removeEventListener === 'function') {
        reducedMotionQuery.removeEventListener('change', handleReducedMotionChange)
      } else {
        reducedMotionQuery.removeListener(handleReducedMotionChange)
      }
      reducedMotionQuery = null
    }
  }

  onMounted(() => {
    mounted = true
    reducedMotion = detectReducedMotion()
    attachEnvironmentListeners()
    scheduleNext()
  })

  onActivated(() => {
    mounted = true
    reducedMotion = detectReducedMotion()
    attachEnvironmentListeners()
    scheduleNext()
  })

  onDeactivated(() => {
    mounted = false
    clearTimer()
    detachEnvironmentListeners()
  })

  onBeforeUnmount(() => {
    mounted = false
    clearTimer()
    detachEnvironmentListeners()
  })

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
    goto,
    pause,
    resume,
    pauseByFocus,
    resumeByFocus,
  }
}