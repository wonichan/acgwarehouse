<script setup lang="ts">
type DetailStatusVariant = 'missing-id' | 'error'

const props = defineProps<{
  readonly variant: DetailStatusVariant
  readonly message?: string
}>()

const emit = defineEmits<{
  retry: []
}>()
</script>

<template>
  <div class="container">
    <article class="panel panel-raised">
      <template v-if="props.variant === 'missing-id'">
        <p class="eyebrow">作品详情</p>
        <h1>请选择一张作品</h1>
        <p class="lead">详情页需要有效的图片 ID，例如从图库、搜索结果或热榜进入 /detail?id=5149。</p>
        <div class="hero-actions">
          <RouterLink class="btn btn-primary" to="/">返回图库</RouterLink>
          <RouterLink class="btn btn-secondary" to="/search">去搜索作品</RouterLink>
        </div>
      </template>

      <template v-else>
        <p class="eyebrow">加载失败</p>
        <h1>无法展示作品</h1>
        <p class="lead">{{ props.message }}</p>
        <div class="hero-actions">
          <button class="btn btn-primary" type="button" @click="emit('retry')">重试</button>
          <RouterLink class="btn btn-secondary" to="/">返回图库</RouterLink>
        </div>
      </template>
    </article>
  </div>
</template>
