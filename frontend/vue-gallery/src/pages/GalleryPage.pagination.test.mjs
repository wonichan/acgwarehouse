import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import assert from 'node:assert/strict'

const source = readFileSync(new URL('./GalleryPage.vue', import.meta.url), 'utf8')

test('GalleryPage requests and appends the next image page when the masonry sentinel becomes visible', () => {
  assert.match(source, /ref<HTMLElement \| null>\(null\)/, 'bottom sentinel element is tracked with a template ref')
  assert.match(source, /IntersectionObserver/, 'an IntersectionObserver triggers loading near the bottom of the masonry feed')
  assert.match(source, /getImages\(\{\s*page:/, 'image requests include the current page parameter')
  assert.match(source, /artItems\.value\s*=\s*\[\.\.\.artItems\.value, \.\.\.nextItems\]/, 'next page items append to the existing masonry list')
  assert.match(source, /loadingMore\.value/, 'next-page requests are guarded against duplicate concurrent loads')
})
