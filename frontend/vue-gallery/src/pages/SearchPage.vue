<script setup lang="ts">
import { ref } from 'vue'
import { useToast } from '@/composables/useToast'

const { show } = useToast()

const keyword = ref('雨夜 角色')
const tags = ref('雨景, 制服, 暖光')
const scoreRange = ref('4.5 分以上')

const searchSummary = ref('正在展示「雨夜 角色」相关结果，评分范围：4.5 分以上')

const handleSearch = () => {
  searchSummary.value = `正在展示「${keyword.value}」相关结果，评分范围：${scoreRange.value}`
  show('筛选条件已应用')
}
</script>

<template>
  <main>
    <!-- Hero Section -->
    <section class="section hero" data-od-id="search-hero">
      <div class="container hero-grid">
        <div>
          <p class="eyebrow">智能搜索 · 文件名 / 标签 / 评分</p>
          <h1>把模糊灵感变成可筛选结果。</h1>
          <p class="lead">输入文件名片段、标签组合或评分区间，快速定位适合收藏、打标和下载的作品。</p>
        </div>
        <div class="panel panel-raised form-grid">
          <label class="field">
            关键词
            <input class="input" v-model="keyword" />
          </label>
          <label class="field">
            标签
            <input class="input" v-model="tags" />
          </label>
          <label class="field">
            评分范围
            <select class="select" v-model="scoreRange">
              <option>4.5 分以上</option>
              <option>4.0 - 4.5 分</option>
              <option>不限评分</option>
            </select>
          </label>
          <button class="btn btn-primary" @click="handleSearch">应用筛选</button>
        </div>
      </div>
    </section>

    <!-- Results Section -->
    <section class="section" data-od-id="search-results">
      <div class="container grid-main">
        <div class="stack">
          <div class="panel">
            <div class="row-between">
              <div>
                <p class="eyebrow">筛选结果</p>
                <h2>相关作品</h2>
              </div>
              <span class="meta" data-search-summary>{{ searchSummary }}</span>
            </div>
          </div>
          <div class="results-list">
            <article class="result-row">
              <div class="thumb"></div>
              <div>
                <h3>樱雨街角的夜间补光</h3>
                <p class="meta">文件名：sakura-rain-street-042.png · 标签：雨景、制服、夜色</p>
                <div class="kicker-row">
                  <span class="tag is-hot">4.8 分</span>
                  <span class="tag">已收藏</span>
                </div>
              </div>
              <RouterLink class="btn btn-secondary btn-small" to="/detail">查看</RouterLink>
            </article>
            <article class="result-row">
              <div class="thumb"></div>
              <div>
                <h3>咖啡店窗边回头瞬间</h3>
                <p class="meta">文件名：window-cafe-turn-018.png · 标签：角色、暖光、室内</p>
                <div class="kicker-row">
                  <span class="tag">4.7 分</span>
                  <span class="tag">可加入相册</span>
                </div>
              </div>
              <RouterLink class="btn btn-secondary btn-small" to="/detail">查看</RouterLink>
            </article>
            <article class="result-row">
              <div class="thumb"></div>
              <div>
                <h3>蓝色车站候车室</h3>
                <p class="meta">文件名：blue-station-platform-009.png · 标签：场景、夜色、构图参考</p>
                <div class="kicker-row">
                  <span class="tag">4.6 分</span>
                  <span class="tag">未打标</span>
                </div>
              </div>
              <RouterLink class="btn btn-secondary btn-small" to="/detail">查看</RouterLink>
            </article>
          </div>
        </div>
        <aside class="panel">
          <p class="eyebrow">搜索建议</p>
          <h3>组合标签更稳定</h3>
          <p class="meta">建议用「主题 + 光线 + 角色姿态」组合，例如：雨景、逆光、侧身。评分排序适合找高质量参考，最新排序适合追社区更新。</p>
          <div class="divider"></div>
          <div class="kicker-row">
            <span class="pill">雨景</span>
            <span class="pill">暖光</span>
            <span class="pill">头像</span>
            <span class="pill">场景</span>
          </div>
        </aside>
      </div>
    </section>
  </main>
</template>