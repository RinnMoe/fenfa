<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t.stats.title }}</h1>
        <p class="page-subtitle">{{ t.stats.subtitle }}</p>
      </div>
    </div>

    <div class="panel">
      <div class="form-grid">
        <div class="form-group">
          <label class="label">{{ t.apps.name }}</label>
          <select v-model="selectedProductId" class="select">
            <option value="">{{ t.upload.selectProduct }}</option>
            <option v-for="p in products" :key="p.id" :value="p.id">{{ p.name }}</option>
          </select>
        </div>
        <div class="form-group">
          <label class="label">{{ t.apps.platform }}</label>
          <select v-model="selectedVariantId" class="select" :disabled="!selectedProductId || loadingVariants">
            <option value="">{{ loadingVariants ? t.common.loading : t.upload.selectVariant }}</option>
            <option v-for="v in variantOptions" :key="v.id" :value="v.id">
              {{ v.platform }} · {{ v.display_name || v.identifier }}
            </option>
          </select>
        </div>
      </div>

      <div v-if="loading" class="loading-state">{{ t.common.loading }}</div>

      <div v-else-if="statsData">
        <div class="stats-meta">
          <strong>{{ statsData.variant?.display_name || statsData.product?.name }}</strong>
          <span class="mono">{{ statsData.variant?.identifier }}</span>
          <span>{{ statsData.product?.name }}</span>
        </div>

        <div class="table-container">
          <table class="table">
            <thead>
              <tr>
                <th>{{ t.stats.version }}</th>
                <th>{{ t.stats.build }}</th>
                <th>{{ t.stats.downloadCount }}</th>
                <th>{{ t.stats.createdAt }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in statsData.releases" :key="r.id">
                <td class="version">{{ r.version }}</td>
                <td>{{ r.build }}</td>
                <td class="text-success">{{ r.download_count }}</td>
                <td class="date">{{ new Date(r.created_at).toLocaleString() }}</td>
              </tr>
              <tr v-if="!statsData.releases?.length">
                <td colspan="4" class="empty-state">{{ t.stats.noReleases }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div v-else class="empty-state">
        <p v-if="!selectedProductId">{{ t.upload.selectProductFirst }}</p>
        <p v-else-if="!selectedVariantId">{{ t.upload.selectVariantFirst }}</p>
        <p v-else>{{ t.stats.emptyState }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from '../i18n'
import { useAuth } from '../composables/useAuth'
import { useProducts } from '../composables/useProducts'
import { useProductVariantSelector } from '../composables/useProductVariantSelector'
import { adminURL } from '../api'

const props = defineProps<{
  variantId?: string
}>()

const route = useRoute()
const { t } = useI18n()
const { token, handleAuthError } = useAuth()
const { products } = useProducts()

const {
  selectedProductId, selectedVariantId, loadingVariants,
  variantOptions, initFromVariantId
} = useProductVariantSelector(products, token, handleAuthError)

const loading = ref(false)
const statsData = ref<any>(null)

watch(selectedVariantId, (value) => {
  if (value) fetchStats(value)
  else statsData.value = null
})

async function fetchStats(variantId: string) {
  loading.value = true
  statsData.value = null
  try {
    const res = await fetch(adminURL(`api/variants/${variantId}/stats`), {
      headers: { 'X-Auth-Token': token.value }
    })
    if (handleAuthError(res)) return
    const payload = await res.json()
    if (res.ok && payload.ok) {
      statsData.value = payload.data
    }
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

// Handle route param for variant ID
const initVariantId = props.variantId || (route.params.variantId as string)
if (initVariantId) {
  initFromVariantId(initVariantId).then(data => {
    if (data) statsData.value = data
  })
}
</script>

<style scoped>
.stats-meta {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-height: 32px;
  margin-bottom: var(--space-2);
  color: var(--text-muted);
  font-size: 13px;
}

.stats-meta strong {
  color: var(--text-primary);
}

.stats-meta .mono {
  font-size: 12px;
}

.empty-state p {
  margin: 0;
}
</style>
