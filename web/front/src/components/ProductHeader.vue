<template>
  <header class="product-header">
    <div class="product-mark">
      <img v-if="product.icon_url" :src="product.icon_url" alt="" />
      <span v-else>{{ product.name.slice(0, 1).toUpperCase() }}</span>
    </div>
    <div class="product-copy">
      <div class="product-kicker">{{ t.hero.productCenter }}</div>
      <h1 class="product-name">{{ product.name }}</h1>
      <p class="product-subtitle">{{ t.hero.subtitle }}</p>
      <div class="platform-labels">
        <span v-for="variant in variants" :key="variant.id">{{ platformLabel(variant.platform) }}</span>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from '../i18n'
import { platformLabel } from '../composables/useProduct'
import type { ProductData } from '../types'

defineProps<{
  product: ProductData['product']
  variants: ProductData['variants']
}>()

const { t } = useI18n()
</script>
