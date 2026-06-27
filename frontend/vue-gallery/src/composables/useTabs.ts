import { ref } from 'vue'

export function useTabs<T extends string>(tabs: T[], initialTab?: T) {
  const activeTab = ref<T>(initialTab ?? tabs[0])

  const isActive = (tab: T) => activeTab.value === tab

  const switchTab = (tab: T) => {
    activeTab.value = tab
  }

  const nextTab = () => {
    const currentIndex = tabs.indexOf(activeTab.value)
    const nextIndex = (currentIndex + 1) % tabs.length
    activeTab.value = tabs[nextIndex]
  }

  const prevTab = () => {
    const currentIndex = tabs.indexOf(activeTab.value)
    const prevIndex = (currentIndex - 1 + tabs.length) % tabs.length
    activeTab.value = tabs[prevIndex]
  }

  return {
    activeTab,
    isActive,
    switchTab,
    nextTab,
    prevTab
  }
}