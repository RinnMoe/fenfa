<template>
  <div class="page-shell">
    <div v-if="loading" class="state-panel">
      <span class="spinner"></span>
      <p>{{ t.loading }}</p>
    </div>

    <div v-else-if="intro" class="intro-page">
      <div class="intro-card">
        <div class="intro-brand">
          <span class="intro-mark">F</span>
          <span class="intro-name">Fenfa</span>
        </div>
        <p class="intro-tagline">{{ t.intro.tagline }}</p>
        <p class="intro-desc">{{ t.intro.description }}</p>
        <ul class="intro-features">
          <li v-for="feature in t.intro.features" :key="feature">{{ feature }}</li>
        </ul>
        <div class="intro-actions">
          <a
            class="intro-btn-primary"
            href="https://github.com/RinnMoe/fenfa"
            target="_blank"
            rel="noreferrer"
          >
            {{ t.intro.github }}
          </a>
        </div>
      </div>
    </div>

    <div v-else-if="error" class="state-panel state-error">
      <h1>{{ t.error.title }}</h1>
      <p>{{ error }}</p>
    </div>

    <div v-else-if="productData" class="product-page">
      <ProductHeader
        :product="productData.product"
        :variant="activeVariant"
        :release="activeRelease"
      />

      <PlatformTabs
        v-if="activeVariant && sortedVariants.length > 1"
        :variants="sortedVariants"
        v-model="activeVariantId"
      />

      <ReleaseActions
        v-if="activeVariant"
        :action="action"
        :variant="activeVariant"
        :udid-bound="udidBound"
        @primary="handlePrimaryAction"
      />

      <QRCodePanel v-if="activeVariant && activeRelease" :url="pageURL" />

      <ReleaseDetails
        v-if="activeRelease?.changelog"
        :release="activeRelease"
      />

      <ReleaseHistory
        v-if="activeVariant"
        :releases="activeVariant.releases"
        :selected-release-id="activeRelease?.id || ''"
        @select="selectRelease"
      />

      <footer class="page-footer">
        <span v-if="t.footer.brandBy" class="footer-text">{{ t.footer.brandBy }}</span>
        <a
          class="brand-link"
          href="https://github.com/RinnMoe/fenfa"
          target="_blank"
          rel="noreferrer"
          aria-label="Fenfa GitHub repository"
        >
          <svg class="github-icon" viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
            <path fill="currentColor" d="M12 .5C5.65.5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.58.11.79-.25.79-.56 0-.28-.01-1.02-.02-2-3.2.7-3.88-1.54-3.88-1.54-.52-1.33-1.28-1.69-1.28-1.69-1.05-.72.08-.7.08-.7 1.16.08 1.77 1.19 1.77 1.19 1.03 1.77 2.7 1.26 3.36.96.1-.75.4-1.26.73-1.55-2.56-.29-5.25-1.28-5.25-5.7 0-1.26.45-2.29 1.19-3.1-.12-.29-.52-1.46.11-3.05 0 0 .97-.31 3.18 1.18a11.06 11.06 0 0 1 5.78 0c2.21-1.49 3.18-1.18 3.18-1.18.63 1.59.23 2.76.11 3.05.74.81 1.19 1.84 1.19 3.1 0 4.43-2.7 5.41-5.27 5.69.41.36.78 1.06.78 2.14 0 1.55-.01 2.8-.01 3.18 0 .31.21.68.8.56A11.51 11.51 0 0 0 23.5 12C23.5 5.65 18.35.5 12 .5Z"/>
          </svg>
          <span>Fenfa</span>
        </a>
        <span v-if="t.footer.brandService" class="footer-text">{{ t.footer.brandService }}</span>
      </footer>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from './i18n'
import { useProduct } from './composables/useProduct'
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
  error,
  fetchData,
  getSelectedRelease,
  loading,
  platformLabel,
  productData,
  recommendedVariant,
  selectRelease: selectReleaseState,
  sortedVariants
} = useProduct()
const { fetchUDIDStatuses, getVariantAction, handlePrimaryAction: handleUDIDAction, udidBound } = useUDID()
const { logVisit } = useTracking()

const pathParts = location.pathname.split('/').filter(Boolean)
const resourceID = pathParts[1] || ''
const pageURL = location.href
const activeVariantId = ref('')
const intro = ref(false)

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

const pageTitle = computed(() => {
  if (intro.value) return 'Fenfa'
  if (!productData.value) return 'Fenfa Distribution'
  const parts = [productData.value.product.name]
  const variant = activeVariant.value
  if (variant) parts.push(platformLabel(variant.platform))
  const release = activeRelease.value
  if (release?.version) {
    const version = release.version.startsWith('v') ? release.version : ('v' + release.version)
    parts.push(version)
  }
  return parts.join(' · ')
})

watch(pageTitle, (value) => {
  document.title = value
}, { immediate: true })

function selectRelease(id: string) {
  if (!activeVariant.value) return
  const release = activeVariant.value.releases.find((item) => item.id === id)
  if (release) selectReleaseState(activeVariant.value.id, release)
}

function handlePrimaryAction() {
  if (!activeVariant.value || !activeRelease.value) return
  const variant = activeVariant.value
  const release = activeRelease.value
  handleUDIDAction(variant, release, action.value)
}

onMounted(async () => {
  if (!resourceID) {
    intro.value = true
    loading.value = false
    return
  }

  await fetchData(resourceID, t.value.error.loadFailed)
  if (!productData.value) return

  const preferred = recommendedVariant.value
  if (preferred) activeVariantId.value = preferred.id
  await fetchUDIDStatuses(productData.value.variants)

  const variant = activeVariant.value
  const release = activeRelease.value
  if (variant && release) {
    logVisit(variant)
  }
})
</script>
