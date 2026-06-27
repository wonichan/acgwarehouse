<script setup lang="ts">
import { ref } from 'vue'
import SegmentedControl from '@/components/SegmentedControl.vue'

const activePeriod = ref('每日')
const periods = ['每日', '每周', '每月']
</script>

<template>
  <main>
    <!-- Hero Section -->
    <section class="section hero" data-od-id="trending-hero">
      <div class="container row-between">
        <div>
          <p class="eyebrow">社区热榜</p>
          <h1>每天、每周、每月的收藏风向。</h1>
          <p class="lead">热榜基于社区收藏、评分和近期互动热度排序，适合快速发现高质量参考图。</p>
        </div>
        <SegmentedControl :options="periods" v-model="activePeriod" />
      </div>
    </section>

    <!-- Ranking Section -->
    <section class="section" data-od-id="trending-list">
      <div class="container grid-main">
        <div class="rank-list">
          <article class="rank-row">
            <div class="thumb"></div>
            <div>
              <h3>樱雨街角的夜间补光</h3>
              <p class="meta">今日新增收藏 42 · 平均评分 4.8</p>
            </div>
            <RouterLink class="btn btn-primary btn-small" to="/detail">查看作品</RouterLink>
          </article>
          <article class="rank-row">
            <div class="thumb"></div>
            <div>
              <h3>橙发猫耳头像设定</h3>
              <p class="meta">角色设定榜上升 · 标签命中 18 次</p>
            </div>
            <RouterLink class="btn btn-secondary btn-small" to="/detail">查看作品</RouterLink>
          </article>
          <article class="rank-row">
            <div class="thumb"></div>
            <div>
              <h3>图书馆靠窗座位</h3>
              <p class="meta">室内场景榜稳定 · 被加入 7 个相册</p>
            </div>
            <RouterLink class="btn btn-secondary btn-small" to="/detail">查看作品</RouterLink>
          </article>
          <article class="rank-row">
            <div class="thumb"></div>
            <div>
              <h3>夏日社团海报底图</h3>
              <p class="meta">海报参考榜新入榜 · 高亮收藏 9</p>
            </div>
            <RouterLink class="btn btn-secondary btn-small" to="/detail">查看作品</RouterLink>
          </article>
        </div>
        <aside class="panel panel-raised">
          <p class="eyebrow">榜单规则</p>
          <h3>社区互动优先</h3>
          <p class="meta">热度由收藏、评分、标签引用和近期浏览共同构成。榜单切换会保留当前内容类别，便于比较不同周期的趋势。</p>
          <div class="divider"></div>
          <div class="stack">
            <div class="row-between">
              <span>收藏权重</span>
              <strong class="num">40%</strong>
            </div>
            <div class="row-between">
              <span>评分权重</span>
              <strong class="num">35%</strong>
            </div>
            <div class="row-between">
              <span>近期互动</span>
              <strong class="num">25%</strong>
            </div>
          </div>
        </aside>
      </div>
    </section>
  </main>
</template>