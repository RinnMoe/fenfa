<template>
  <section v-if="releases.length > 1" class="release-history">
    <button
      type="button"
      class="history-toggle"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      <span>{{ expanded ? t.variant.hideVersions : t.variant.otherVersions }}</span>
      <span class="history-chevron" aria-hidden="true"></span>
    </button>

    <div v-if="expanded" class="history-list">
      <button
        v-for="release in releases"
        :key="release.id"
        type="button"
        class="history-row"
        :class="{ active: release.id === selectedReleaseID }"
        :aria-current="release.id === selectedReleaseID ? 'true' : undefined"
        @click="$emit('select', release.id)"
      >
        <span class="history-version">{{ formatVersion(release.version) }}</span>
        <span class="history-date">{{ formatDate(release.created_at) }}</span>
        <span v-if="release.id === selectedReleaseID" class="history-current">{{ t.variant.current }}</span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from '../i18n'
import { formatDate } from '../composables/useProduct'
import type { ReleaseItem } from '../types'

defineProps<{
  releases: ReleaseItem[]
  selectedReleaseID: string
}>()

defineEmits<{
  select: [id: string]
}>()

const { t } = useI18n()
const expanded = ref(false)

function formatVersion(version: string) {
  return version.startsWith('v') ? version : `v${version}`
}
</script>
