import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import assert from 'node:assert/strict'

const source = readFileSync(new URL('./GalleryPage.vue', import.meta.url), 'utf8')

test('GalleryPage requests and appends the next image page when the masonry sentinel becomes visible', () => {
  assert.match(source, /ref<HTMLElement \| null>\(null\)/, 'bottom sentinel element is tracked with a template ref')
  assert.match(source, /IntersectionObserver/, 'an IntersectionObserver triggers loading near the bottom of the masonry feed')
  assert.match(source, /getImages\(galleryImageQuery\(nextPage\)\)/, 'next-page image requests reuse the active sort query')
  assert.match(source, /artItems\.value\s*=\s*appendArtItems\(artItems\.value,\s*nextItems\)/, 'next page items append to the existing masonry list')
  assert.match(source, /appendMasonryItems\(nextItems\)/, 'next page items append to stable masonry columns without rebuilding old cards')
  assert.match(source, /loadingMore\.value/, 'next-page requests are guarded against duplicate concurrent loads')
})

test('GalleryPage uses stable JS masonry columns instead of CSS multi-column reflow', () => {
  assert.match(source, /masonryColumns\s*=\s*ref<ArtItem\[\]\[\]>/, 'masonry columns are tracked as explicit item arrays')
  assert.match(source, /shortestColumnIndex/, 'new cards are assigned to the shortest current column')
  assert.match(source, /ResizeObserver/, 'the masonry container is remeasured on responsive width changes')
  assert.match(source, /class="masonry-column"/, 'the template renders explicit stable masonry columns')
})

test('GalleryPage maps visible filters to real backend image sort parameters', () => {
  assert.match(source, /高分参考:\s*\{\s*sort:\s*'avg_score',\s*order:\s*'desc'\s*\}/, 'score filter sorts by avg_score desc')
  assert.match(source, /收藏热度:\s*\{\s*sort:\s*'favorite_count',\s*order:\s*'desc'\s*\}/, 'favorite filter sorts by favorite_count desc')
  assert.match(source, /最新:\s*\{\s*sort:\s*'created_at',\s*order:\s*'desc'\s*\}/, 'latest filter sorts by created_at desc')
  assert.match(source, /推荐:\s*\{\s*sort:\s*'view_count',\s*order:\s*'desc'\s*\}/, 'recommended filter sorts by view_count desc')
  assert.match(source, /getImages\(galleryImageQuery\(1\)\)/, 'first page requests use the active sort query')
})
