import { reactive, toRefs } from 'vue'

interface ToastState {
  message: string
  isOpen: boolean
}

const toastState = reactive<ToastState>({
  message: '',
  isOpen: false
})

let timer: number | null = null

export function useToast() {
  const show = (message: string, duration = 1800) => {
    toastState.message = message
    toastState.isOpen = true
    if (timer) clearTimeout(timer)
    timer = window.setTimeout(() => {
      toastState.isOpen = false
    }, duration)
  }

  const hide = () => {
    toastState.isOpen = false
    if (timer) clearTimeout(timer)
  }

  return {
    ...toRefs(toastState),
    show,
    hide
  }
}