<template>
  <header class="product-header">
    <div class="product-mark">
      <img v-if="product.icon_url" :src="product.icon_url" alt="" />
      <span v-else>{{ product.name.slice(0, 1).toUpperCase() }}</span>
    </div>
    <div class="product-copy">
      <h1 class="product-name">{{ product.name }}</h1>
      <p v-if="variant" class="product-meta">
        <span class="product-platform">{{ platformLabel(variant.platform) }}</span>
        <span v-if="release" class="product-version">{{ formatVersion(release.version) }}</span>
        <span v-if="release?.created_at" class="product-date">{{ formatDate(release.created_at) }}</span>
        <span v-if="typeof downloadCount === 'number'" class="product-downloads">{{ formatDownloads(downloadCount) }}</span>
      </p>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from '../i18n'
import { formatDate, platformLabel } from '../composables/useProduct'
import type { ProductData, ReleaseItem, VariantItem } from '../types'

defineProps<{
  product: ProductData['product']
  variant: VariantItem | null
  release: ReleaseItem | null
  downloadCount?: number
}>()

const { t } = useI18n()

function formatVersion(version: string) {
  return version.startsWith('v') ? version : `v${version}`
}

function formatDownloads(count: number) {
  return t.value.variant.downloadCount.replace('{{ count }}', count.toLocaleString())
}
</script>
