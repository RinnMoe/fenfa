<template>
  <nav v-if="variants.length > 1" class="platform-tabs" aria-label="Platform">
    <button
      v-for="variant in variants"
      :key="variant.id"
      type="button"
      :class="{ active: variant.id === modelValue }"
      @click="$emit('update:modelValue', variant.id)"
    >
      {{ platformLabel(variant.platform) }}
    </button>
  </nav>
</template>

<script setup lang="ts">
import { platformLabel } from '../composables/useProduct'
import type { VariantItem } from '../types'

defineProps<{
  variants: VariantItem[]
  modelValue: string
}>()

defineEmits<{
  'update:modelValue': [id: string]
}>()
</script>
