<script setup lang="ts">
import { ref } from 'vue'
import { useToast } from '@/composables/useToast'
import ArtCard from '@/components/ArtCard.vue'
import type { ArtItem, Album } from '@/types'

const { show } = useToast()

// Album creation form
const albumName = ref('雨夜角色参考')
const albumTags = ref('雨景, 角色, 暖光')

const handleCreateAlbum = () => {
  show('新相册已创建')
}

// Mock albums
const albums: Album[] = [
  { id: 'a1', name: '雨夜角色参考', tags: ['雨景', '暖光'], count: 24, lastUpdated: '今天' },
  { id: 'a2', name: '头像与表情包', tags: ['头像', '猫耳'], count: 18, lastUpdated: '3天前' },
  { id: 'a3', name: '场景构图库', tags: ['场景', '构图'], count: 31, lastUpdated: '上周' }
]

// Mock collection items
const collectionItems: ArtItem[] = [
  { id: 'c1', title: '樱雨街角的夜间补光', tags: ['#雨夜角色参考', '#4.8'], score: 4.8, favorites: 0, previewVariant: 'default' },
  { id: 'c2', title: '咖啡店窗边回头瞬间', tags: ['#暖光', '#未打标'], score: 0, favorites: 0, previewVariant: 'tall' },
  { id: 'c3', title: '蓝色车站候车室', tags: ['#场景', '#4.6'], score: 4.6, favorites: 0, previewVariant: 'wide' }
]
</script>

<template>
  <main>
    <!-- Hero Section -->
    <section class="section hero" data-od-id="collections-hero">
      <div class="container hero-grid">
        <div>
          <p class="eyebrow">收藏夹 / 相册管理</p>
          <h1>把灵感按用途整理成相册。</h1>
          <p class="lead">创建相册、批量收藏、统一打标签，让社区浏览变成可复用的个人素材库。</p>
        </div>
        <div class="panel panel-raised form-grid">
          <label class="field">
            相册名称
            <input class="input" v-model="albumName" />
          </label>
          <label class="field">
            默认标签
            <input class="input" v-model="albumTags" />
          </label>
          <button class="btn btn-primary" @click="handleCreateAlbum">创建相册</button>
        </div>
      </div>
    </section>

    <!-- Albums Section -->
    <section class="section" data-od-id="album-grid">
      <div class="container">
        <div class="panel-head">
          <div>
            <p class="eyebrow">我的相册</p>
            <h2>管理收藏内容</h2>
          </div>
          <span class="meta">点击作品卡片可进入批量选择</span>
        </div>
        <div class="album-grid">
          <article v-for="album in albums" :key="album.id" class="card album-card">
            <div class="album-cover">
              <div class="thumb"></div>
              <div class="thumb"></div>
              <div class="thumb"></div>
            </div>
            <h3>{{ album.name }}</h3>
            <p class="meta">{{ album.count }} 张作品 · 最近更新：{{ album.lastUpdated }}</p>
            <div class="kicker-row">
              <span v-for="tag in album.tags" :key="tag" class="tag">{{ tag }}</span>
            </div>
          </article>
        </div>
      </div>
    </section>

    <!-- Batch Section -->
    <section class="section" data-od-id="collection-batch">
      <div class="container grid-main">
        <div class="masonry">
          <ArtCard v-for="item in collectionItems" :key="item.id" :item="item" />
        </div>
        <aside class="panel">
          <p class="eyebrow">批量操作面板</p>
          <h3>中等保真交互</h3>
          <p class="meta">选择左侧作品后，底部会出现操作条。这里展示批量收藏、下载与打标签入口，不执行真实文件传输。</p>
          <div class="divider"></div>
          <button class="btn btn-secondary" @click="show('批量下载示意已开始')">批量下载示意</button>
        </aside>
      </div>
    </section>
  </main>
</template>