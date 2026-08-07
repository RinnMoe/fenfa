<template>
  <div class="release-actions">
    <button
      type="button"
      class="primary-action"
      :class="action.type"
      :disabled="action.type === 'disabled'"
      @click="$emit('primary')"
    >
      {{ action.label }}
    </button>
    <p v-if="variant.platform === 'ios' && !udidBound[variant.id]" class="action-hint">
      {{ t.variant.iosHint }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from '../i18n'
import type { VariantAction, VariantItem } from '../types'

defineProps<{
  action: VariantAction
  variant: VariantItem
  udidBound: Record<string, boolean>
}>()

defineEmits<{
  primary: []
}>()

const { t } = useI18n()
</script>
