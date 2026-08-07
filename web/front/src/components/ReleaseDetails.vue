<template>
  <section v-if="release?.changelog" class="release-notes">
    <h2>{{ t.variant.changelogTitle }}</h2>
    <p>{{ release.changelog }}</p>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from '../i18n'
import type { ReleaseItem } from '../types'

defineProps<{
  release: ReleaseItem | null
}>()

const { t } = useI18n()
</script>
