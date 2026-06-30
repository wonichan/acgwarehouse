import { onBeforeUnmount, ref } from 'vue'
import type { Ref } from 'vue'
import type { ArtItem } from '@/types'

const DEFAULT_MIN_COLUMN_WIDTH = 240
const DEFAULT_MAX_COLUMNS = 4
const DEFAULT_COLUMN_GAP = 16
const ART_CARD_BODY_ESTIMATED_HEIGHT = 104
const ART_CARD_FALLBACK_HEIGHTS: Record<ArtItem['previewVariant'], number> = {
  default: 210,
  tall: 320,
  wide: 180,
}

interface MasonryLayoutOptions {
  readonly minColumnWidth?: number
  readonly maxColumns?: number
  readonly columnGap?: number
}

export function useMasonryLayout(
  sourceItems: Ref<readonly ArtItem[]>,
  options: MasonryLayoutOptions = {},
) {
  const minColumnWidth = options.minColumnWidth ?? DEFAULT_MIN_COLUMN_WIDTH
  const maxColumns = options.maxColumns ?? DEFAULT_MAX_COLUMNS
  const columnGap = options.columnGap ?? DEFAULT_COLUMN_GAP

  const masonry = ref<HTMLElement | null>(null)
  const columnCount = ref(1)
  const columns = ref<ArtItem[][]>([[]])
  const columnHeights = ref<number[]>([0])
  let resizeObserver: ResizeObserver | null = null

  function emptyColumns(count: number): ArtItem[][] {
    return Array.from({ length: count }, () => [])
  }

  function columnWidthFor(count: number): number {
    const containerWidth = masonry.value?.clientWidth ?? minColumnWidth
    const totalGap = Math.max(0, count - 1) * columnGap
    return Math.max(minColumnWidth, (containerWidth - totalGap) / count)
  }

  function estimateItemHeight(item: ArtItem, columnWidth: number): number {
    if (item.imageWidth !== undefined && item.imageHeight !== undefined && item.imageWidth > 0 && item.imageHeight > 0) {
      return (columnWidth * item.imageHeight / item.imageWidth) + ART_CARD_BODY_ESTIMATED_HEIGHT
    }
    return ART_CARD_FALLBACK_HEIGHTS[item.previewVariant] + ART_CARD_BODY_ESTIMATED_HEIGHT
  }

  function shortestColumnIndex(heights: readonly number[]): number {
    return heights.reduce((shortest, height, index) => height < heights[shortest] ? index : shortest, 0)
  }

  function appendItems(nextItems: readonly ArtItem[]): void {
    const count = columnCount.value
    const nextColumns = columns.value.length === count
      ? columns.value.map(column => [...column])
      : emptyColumns(count)
    const nextHeights = columnHeights.value.length === count
      ? [...columnHeights.value]
      : Array.from({ length: count }, () => 0)
    const itemColumnWidth = columnWidthFor(count)

    for (const item of nextItems) {
      const targetColumn = shortestColumnIndex(nextHeights)
      nextColumns[targetColumn].push(item)
      nextHeights[targetColumn] += estimateItemHeight(item, itemColumnWidth) + columnGap
    }

    columns.value = nextColumns
    columnHeights.value = nextHeights
  }

  function rebuild(): void {
    const count = columnCount.value
    columns.value = emptyColumns(count)
    columnHeights.value = Array.from({ length: count }, () => 0)
    appendItems(sourceItems.value)
  }

  function calculateColumnCount(width: number): number {
    if (width <= 0) return columnCount.value
    const nextCount = Math.floor((width + columnGap) / (minColumnWidth + columnGap))
    return Math.max(1, Math.min(maxColumns, nextCount))
  }

  function updateColumnCount(width: number): void {
    const nextCount = calculateColumnCount(width)
    if (nextCount === columnCount.value) return
    columnCount.value = nextCount
    rebuild()
  }

  function observe(): void {
    if (!masonry.value || resizeObserver !== null) return

    updateColumnCount(masonry.value.clientWidth)
    if (typeof ResizeObserver === 'undefined') return

    resizeObserver = new ResizeObserver(entries => {
      const width = entries[0]?.contentRect.width ?? masonry.value?.clientWidth ?? 0
      updateColumnCount(width)
    })
    resizeObserver.observe(masonry.value)
  }

  onBeforeUnmount(() => {
    if (resizeObserver !== null) {
      resizeObserver.disconnect()
      resizeObserver = null
    }
  })

  return {
    masonry,
    columnCount,
    columns,
    rebuild,
    observe,
  }
}
