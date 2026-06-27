import { ref, onUnmounted } from 'vue'

export function useMotion(threshold = 0.14, rootMargin = '0px 0px -8% 0px') {
  const isVisible = ref(false)
  let observer: IntersectionObserver | null = null
  let element: Element | null = null

  const observe = (el: Element) => {
    element = el
    
    // Check if reduced motion is preferred
    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (prefersReducedMotion) {
      isVisible.value = true
      return
    }

    if (!('IntersectionObserver' in window)) {
      isVisible.value = true
      return
    }

    observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            isVisible.value = true
            observer?.unobserve(entry.target)
          }
        })
      },
      { threshold, rootMargin }
    )

    observer.observe(el)
  }

  const unobserve = () => {
    if (observer && element) {
      observer.unobserve(element)
    }
  }

  onUnmounted(() => {
    unobserve()
  })

  return {
    isVisible,
    observe,
    unobserve
  }
}

export function useMotionDelay(index: number, maxDelay = 180) {
  const delay = Math.min(index * 45, maxDelay)
  return `${delay}ms`
}