<template>
  <section v-if="releases.length" class="release-history">
    <h2>{{ t.variant.allVersions }}</h2>
    <div class="history-table">
      <div class="history-row history-head">
        <span>{{ t.variant.version }}</span>
        <span>{{ t.variant.build }}</span>
        <span>{{ t.variant.updatedAt }}</span>
        <span>{{ t.common.action }}</span>
      </div>
      <button
        v-for="release in releases"
        :key="release.id"
        type="button"
        class="history-row"
        :class="{ active: release.id === selectedReleaseID }"
        @click="$emit('select', release.id)"
      >
        <span class="version">{{ release.version }}</span>
        <span class="muted">{{ release.build || '-' }}</span>
        <span class="muted">{{ formatTime(release.created_at) }}</span>
        <span class="history-action">{{ release.id === selectedReleaseID ? t.variant.current : t.variant.select }}</span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from '../i18n'
import { formatTime as formatTimeRaw } from '../composables/useProduct'
import type { ReleaseItem } from '../types'

defineProps<{
  releases: ReleaseItem[]
  selectedReleaseID: string
}>()

defineEmits<{
  select: [id: string]
}>()

const { t } = useI18n()

function formatTime(value: string) {
  return formatTimeRaw(value, t.value)
}
</script>
