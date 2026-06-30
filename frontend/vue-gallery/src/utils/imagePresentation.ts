import type { ArtItem } from '@/types'
import type { ImageItem } from '@/api/client'

const DISPLAYABLE_URL_SUFFIX = '/'

export function hasDisplayableImageItem(image: ImageItem): boolean {
  const imageUrl = image.url.trim()
  return image.size > 0 && image.width > 0 && image.height > 0 && imageUrl.length > 0 && !imageUrl.endsWith(DISPLAYABLE_URL_SUFFIX)
}

export function imageToArtItem(img: ImageItem): ArtItem {
  let previewVariant: 'default' | 'tall' | 'wide' = 'default'
  if (img.width > 0 && img.height > 0) {
    const ratio = img.width / img.height
    if (ratio < 0.8) previewVariant = 'tall'
    else if (ratio > 1.3) previewVariant = 'wide'
  }

  const score = Number.isFinite(img.avg_score) ? img.avg_score : 0
  const category = img.category || '未分类'

  return {
    id: String(img.id),
    title: img.filename,
    tags: [category, `${score.toFixed(1)}/100`],
    score,
    favorites: img.favorite_count,
    previewVariant,
    imageWidth: img.width > 0 ? img.width : undefined,
    imageHeight: img.height > 0 ? img.height : undefined,
    imageUrl: img.url,
  }
}
