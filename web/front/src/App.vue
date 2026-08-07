<template>
  <div class="page-shell">
    <div v-if="loading" class="state-panel">
      <span class="spinner"></span>
      <p>{{ t.loading }}</p>
    </div>

    <div v-else-if="error" class="state-panel state-error">
      <h1>{{ t.error.title }}</h1>
      <p>{{ error }}</p>
    </div>

    <div v-else-if="productData" class="product-page">
      <ProductHeader
        :product="productData.product"
        :variants="sortedVariants"
      />

      <div class="content-grid">
        <main class="release-main">
          <PlatformTabs
            v-if="activeVariant"
            :variants="sortedVariants"
            v-model="activeVariantId"
          />

          <ReleaseDetails
            v-if="activeVariant"
            :variant="activeVariant"
            :release="activeRelease"
            :product-name="productData.product.name"
            :format-time="formatTime"
          />

          <ReleaseActions
            v-if="activeVariant"
            :action="action"
            :variant="activeVariant"
            :udid-bound="udidBound"
            @primary="handlePrimaryAction"
          />

          <ReleaseHistory
            v-if="activeVariant"
            :releases="activeVariant.releases"
            :selected-release-id="activeRelease?.id || ''"
            :format-time="formatTime"
            @select="selectRelease"
          />
        </main>

        <QRCodePanel :url="pageURL" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from './i18n'
import { formatTime, useProduct } from './composables/useProduct'
import { useTracking } from './composables/useTracking'
import { useUDID } from './composables/useUDID'
import ProductHeader from './components/ProductHeader.vue'
import PlatformTabs from './components/PlatformTabs.vue'
import ReleaseDetails from './components/ReleaseDetails.vue'
import ReleaseActions from './components/ReleaseActions.vue'
import ReleaseHistory from './components/ReleaseHistory.vue'
import QRCodePanel from './components/QRCodePanel.vue'

const { t } = useI18n()
const {
  clientPlatform,
  error,
  fetchData,
  getSelectedRelease,
  loading,
  productData,
  recommendedVariant,
  selectRelease: selectReleaseState,
  sortedVariants
} = useProduct()
const { fetchUDIDStatuses, getVariantAction, handlePrimaryAction: handleUDIDAction, installing, udidBound } = useUDID()
const { logVisit, trackAction } = useTracking()

const pathParts = location.pathname.split('/').filter(Boolean)
const resourceID = pathParts[1] || ''
const pageURL = location.href
const activeVariantId = ref('')

const activeVariant = computed(() => {
  return sortedVariants.value.find((variant) => variant.id === activeVariantId.value) || null
})

const activeRelease = computed(() => {
  return activeVariant.value ? getSelectedRelease(activeVariant.value) : null
})

const action = computed(() => {
  if (!activeVariant.value || !activeRelease.value) {
    return { type: 'disabled' as const, href: '#', label: t.value.actions.unavailable }
  }
  return getVariantAction(activeVariant.value, activeRelease.value, t.value)
})

function selectVariant(id: string) {
  activeVariantId.value = id
}

function selectRelease(id: string) {
  if (!activeVariant.value) return
  const release = activeVariant.value.releases.find((item) => item.id === id)
  if (release) selectReleaseState(activeVariant.value.id, release)
}

function handlePrimaryAction() {
  if (!activeVariant.value || !activeRelease.value) return
  const variant = activeVariant.value
  const release = activeRelease.value
  handleUDIDAction(variant, release, action.value, (href) => {
    trackAction(variant, release, href, productData.value?.product.id || '', location.pathname)
  })
}

onMounted(async () => {
  await fetchData(resourceID, t.value.error.loadFailed)
  if (!productData.value) return

  const preferred = recommendedVariant.value
  if (preferred) activeVariantId.value = preferred.id
  await fetchUDIDStatuses(productData.value.variants)

  const variant = activeVariant.value
  const release = activeRelease.value
  if (variant && release) {
    await logVisit(variant, release, productData.value.product.id, clientPlatform, location.pathname)
  }
})
</script>
