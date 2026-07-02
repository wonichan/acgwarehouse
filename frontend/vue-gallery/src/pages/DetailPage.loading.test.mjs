import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import assert from 'node:assert/strict'

const detailPageSource = readFileSync(new URL('./DetailPage.vue', import.meta.url), 'utf8')
const loadingStateSource = readFileSync(new URL('../components/DetailLoadingState.vue', import.meta.url), 'utf8')

test('DetailPage uses an animation-only loading component instead of technical API copy', () => {
  assert.match(detailPageSource, /<DetailLoadingState v-else-if="loading" \/>/, 'detail loading branch renders the dedicated loading component')
  assert.doesNotMatch(detailPageSource, /\/api\/v1\/images/, 'detail page must not expose the image API route in loading UI')
  assert.doesNotMatch(detailPageSource, /获取图片、标签、评分和相似推荐/, 'detail page must not describe backend fetch internals')
  assert.doesNotMatch(detailPageSource, /读取真实作品详情/, 'detail page must not show the old text-only loading heading')
})

test('DetailLoadingState communicates loading visually while preserving reduced-motion support', () => {
  assert.match(loadingStateSource, /class="viewer-art detail-loading-art"/, 'loader reserves the main image viewer area')
  assert.match(loadingStateSource, /detail-loading-sheen/, 'loader includes a visual shimmer layer')
  assert.match(loadingStateSource, /@keyframes detail-loading-sheen/, 'loader defines a shimmer animation')
  assert.match(loadingStateSource, /@media \(prefers-reduced-motion: reduce\)/, 'loader disables continuous motion for reduced-motion users')
  assert.doesNotMatch(loadingStateSource, /\/api\/v1\//, 'loader must not expose backend routes')
  assert.doesNotMatch(loadingStateSource, />\s*正在加载\s*</, 'loader must not render visible loading text')
})
