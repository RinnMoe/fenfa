<template>
  <section class="release-details">
    <div class="release-title-row">
      <div>
        <div class="release-platform">{{ platformLabel(variant.platform) }}</div>
        <h2 class="release-name">{{ variant.display_name || productName }}</h2>
      </div>
      <div v-if="variant.arch || variant.installer_type" class="release-meta">
        <span v-if="variant.arch">{{ variant.arch }}</span>
        <span v-if="variant.installer_type">{{ variant.installer_type }}</span>
      </div>
    </div>

    <dl v-if="release" class="release-facts">
      <div>
        <dt>{{ t.variant.version }}</dt>
        <dd class="version">{{ release.version }} <span v-if="release.build">({{ t.variant.build }} {{ release.build }})</span></dd>
      </div>
      <div>
        <dt>{{ t.variant.updatedAt }}</dt>
        <dd>{{ formatTime(release.created_at) }}</dd>
      </div>
      <div v-if="release.min_os || variant.min_os">
        <dt>{{ t.variant.minOS }}</dt>
        <dd>{{ release.min_os || variant.min_os }}</dd>
      </div>
      <div>
        <dt>{{ t.variant.identifier }}</dt>
        <dd class="mono">{{ variant.identifier }}</dd>
      </div>
    </dl>

    <div v-if="release?.changelog" class="changelog">
      <h3>{{ t.variant.changelogTitle }}</h3>
      <p>{{ release.changelog }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from '../i18n'
import { platformLabel } from '../composables/useProduct'
import type { ReleaseItem, VariantItem } from '../types'

defineProps<{
  variant: VariantItem
  release: ReleaseItem | null
  productName: string
  formatTime: (value: string, t: any) => string
}>()

const { t } = useI18n()
</script>
