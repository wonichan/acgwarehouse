import { reactive, computed } from 'vue'

interface SelectionState {
  selectedIds: Set<string>
}

const selectionState = reactive<SelectionState>({
  selectedIds: new Set<string>()
})

export function useSelection() {
  const selectedCount = computed(() => selectionState.selectedIds.size)

  const isSelected = (id: string) => selectionState.selectedIds.has(id)

  const toggle = (id: string) => {
    if (selectionState.selectedIds.has(id)) {
      selectionState.selectedIds.delete(id)
    } else {
      selectionState.selectedIds.add(id)
    }
  }

  const select = (id: string) => {
    selectionState.selectedIds.add(id)
  }

  const deselect = (id: string) => {
    selectionState.selectedIds.delete(id)
  }

  const clear = () => {
    selectionState.selectedIds.clear()
  }

  return {
    selectedIds: selectionState.selectedIds,
    selectedCount,
    isSelected,
    toggle,
    select,
    deselect,
    clear
  }
}