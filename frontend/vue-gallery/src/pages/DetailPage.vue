<script setup lang="ts">
import { ref } from 'vue'
import { useZoom } from '@/composables/useZoom'
import { useToast } from '@/composables/useToast'

const { zoom, zoomIn, zoomOut, reset } = useZoom()
const { show } = useToast()

const customTag = ref('雨夜参考')
const customScore = ref('5')

const handleSave = () => {
  show('标签与评分已保存')
}

const handleFavorite = () => {
  show('已加入收藏夹')
}

const handleDownload = () => {
  show('示意图下载已开始')
}
</script>

<template>
  <main>
    <section class="section" data-od-id="detail-viewer">
      <div class="container detail-stage">
        <!-- Viewer Panel -->
        <article class="panel viewer panel-raised" aria-label="图片放大查看器">
          <div class="viewer-art" :style="`--zoom: ${zoom}`" data-viewer-art></div>
          <div class="zoom-controls" aria-label="缩放控制">
            <button class="btn btn-secondary btn-small" @click="zoomOut">缩小</button>
            <button class="btn btn-primary btn-small" @click="zoomIn">放大</button>
            <button class="btn btn-secondary btn-small" @click="reset">复位</button>
          </div>
        </article>

        <!-- Side Panel -->
        <aside class="stack">
          <div class="panel panel-raised">
            <p class="eyebrow">作品详情</p>
            <h1 style="font-size: clamp(var(--text-2xl), 4.6vw, var(--text-3xl));">樱雨街角的夜间补光</h1>
            <p class="lead">社区用户上传的原创风格示意图，适合作为雨夜场景、角色站姿和暖色补光参考。</p>
            <div class="kicker-row">
              <span class="tag is-hot">4.8 分</span>
              <span class="tag">雨景</span>
              <span class="tag">制服</span>
              <span class="tag">夜色</span>
            </div>
            <div class="divider"></div>
            <div class="grid-2">
              <button class="btn btn-primary" @click="handleFavorite">收藏到相册</button>
              <button class="btn btn-secondary" @click="handleDownload">下载示意图</button>
            </div>
          </div>

          <div class="panel">
            <div class="panel-head">
              <div>
                <p class="eyebrow">自定义标签</p>
                <h3>为作品打标</h3>
              </div>
              <span class="meta">个人可见</span>
            </div>
            <div class="form-grid">
              <label class="field">
                新增标签
                <input class="input" v-model="customTag" />
              </label>
              <label class="field">
                个人评分
                <select class="select" v-model="customScore">
                  <option>5 分</option>
                  <option>4 分</option>
                  <option>3 分</option>
                </select>
              </label>
              <button class="btn btn-primary" @click="handleSave">保存标签与评分</button>
            </div>
          </div>

          <div class="panel">
            <div class="panel-head">
              <div>
                <p class="eyebrow">相似推荐</p>
                <h3>同类作品</h3>
              </div>
              <RouterLink class="btn btn-secondary btn-small" to="/search">更多</RouterLink>
            </div>
            <div class="grid-2">
              <div class="thumb"></div>
              <div class="thumb"></div>
            </div>
          </div>
        </aside>
      </div>
    </section>
  </main>
</template>