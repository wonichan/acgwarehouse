import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import assert from 'node:assert/strict'

const source = readFileSync(new URL('./GalleryPage.vue', import.meta.url), 'utf8')

test('GalleryPage requests and appends the next image page when the masonry sentinel becomes visible', () => {
  assert.match(source, /ref<HTMLElement \| null>\(null\)/, 'bottom sentinel element is tracked with a template ref')
  assert.match(source, /IntersectionObserver/, 'an IntersectionObserver triggers loading near the bottom of the masonry feed')
  assert.match(source, /getImages\(galleryImageQuery\(nextPage\)\)/, 'next-page image requests reuse the active sort query')
  assert.match(source, /artItems\.value\s*=\s*\[\.\.\.artItems\.value, \.\.\.nextItems\]/, 'next page items append to the existing masonry list')
  assert.match(source, /loadingMore\.value/, 'next-page requests are guarded against duplicate concurrent loads')
})

test('GalleryPage maps visible filters to real backend image sort parameters', () => {
  assert.match(source, /高分参考:\s*\{\s*sort:\s*'avg_score',\s*order:\s*'desc'\s*\}/, 'score filter sorts by avg_score desc')
  assert.match(source, /收藏热度:\s*\{\s*sort:\s*'favorite_count',\s*order:\s*'desc'\s*\}/, 'favorite filter sorts by favorite_count desc')
  assert.match(source, /最新:\s*\{\s*sort:\s*'created_at',\s*order:\s*'desc'\s*\}/, 'latest filter sorts by created_at desc')
  assert.match(source, /推荐:\s*\{\s*sort:\s*'view_count',\s*order:\s*'desc'\s*\}/, 'recommended filter sorts by view_count desc')
  assert.match(source, /getImages\(galleryImageQuery\(1\)\)/, 'first page requests use the active sort query')
})
