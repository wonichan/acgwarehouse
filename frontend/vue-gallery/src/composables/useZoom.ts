import { ref } from 'vue'

export function useZoom(min = 0.75, max = 1.7, step = 0.15) {
  const zoom = ref(1)

  const zoomIn = () => {
    zoom.value = Math.min(max, zoom.value + step)
  }

  const zoomOut = () => {
    zoom.value = Math.max(min, zoom.value - step)
  }

  const reset = () => {
    zoom.value = 1
  }

  return {
    zoom,
    zoomIn,
    zoomOut,
    reset
  }
}