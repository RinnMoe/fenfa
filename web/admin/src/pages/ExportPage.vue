<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t.exportPage.title }}</h1>
        <p class="page-subtitle">{{ t.exportPage.subtitle }}</p>
      </div>
    </div>

    <div class="panel">
      <div class="filter-grid">
        <div class="form-group">
          <label class="label">{{ t.apps.name }} {{ t.exportPage.optional }}</label>
          <select v-model="selectedProductId" class="select">
            <option value="">{{ t.exportPage.allProducts }}</option>
            <option v-for="p in products" :key="p.id" :value="p.id">{{ p.name }}</option>
          </select>
        </div>
        <div class="form-group">
          <label class="label">{{ t.apps.platform }} {{ t.exportPage.optional }}</label>
          <select v-model="selectedVariantId" class="select" :disabled="!selectedProductId || loadingVariants">
            <option value="">{{ loadingVariants ? t.common.loading : t.exportPage.allVariants }}</option>
            <option v-for="v in variantOptions" :key="v.id" :value="v.id">
              {{ v.platform }} · {{ v.display_name || v.identifier }}
            </option>
          </select>
        </div>
        <div class="form-group">
          <label class="label">{{ t.exportPage.startDate }}</label>
          <input v-model="from" type="date" class="input">
        </div>
        <div class="form-group">
          <label class="label">{{ t.exportPage.endDate }}</label>
          <input v-model="to" type="date" class="input">
        </div>
      </div>

      <div class="export-list">
        <div class="export-row">
          <div>
            <div class="export-title">{{ t.exportPage.releases }}</div>
            <div class="export-desc">{{ t.exportPage.releasesDesc }}</div>
          </div>
          <button class="btn-secondary" @click="handleExport('releases')">
            <Icon name="download" :size="14" />
            {{ t.common.export }}
          </button>
        </div>

        <div class="export-row">
          <div>
            <div class="export-title">{{ t.exportPage.events }}</div>
            <div class="export-desc">{{ t.exportPage.eventsDesc }}</div>
          </div>
          <button class="btn-secondary" @click="handleExport('events')">
            <Icon name="download" :size="14" />
            {{ t.common.export }}
          </button>
        </div>

        <div class="export-row">
          <div>
            <div class="export-title">{{ t.exportPage.iosDevices }}</div>
            <div class="export-desc">{{ t.exportPage.iosDevicesDesc }}</div>
          </div>
          <button class="btn-secondary" @click="handleExport('ios_devices')">
            <Icon name="download" :size="14" />
            {{ t.common.export }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from '../i18n'
import { useAuth } from '../composables/useAuth'
import { useProducts } from '../composables/useProducts'
import { useToast } from '../composables/useToast'
import { useProductVariantSelector } from '../composables/useProductVariantSelector'
import Icon from '../components/Icon.vue'
import { adminFetch } from '../api'

const { t } = useI18n()
const { token, handleAuthError } = useAuth()
const { products } = useProducts()
const toast = useToast()

const {
  selectedProductId, selectedVariantId, loadingVariants, variantOptions
} = useProductVariantSelector(products, token, handleAuthError)

const from = ref('')
const to = ref('')

async function handleExport(type: string) {
  const p = new URLSearchParams()
  if (selectedVariantId.value) p.set('variant_id', selectedVariantId.value)
  if (from.value) p.set('from', from.value)
  if (to.value) p.set('to', to.value)

  const path = `exports/${type}.csv`
  const url = path + (p.toString() ? ('?' + p.toString()) : '')

  try {
    const r = await adminFetch(url, { token: token.value })
    if (handleAuthError(r)) return
    if (!r.ok) throw new Error('HTTP ' + r.status)
    const blob = await r.blob()
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `${type}.csv`
    document.body.appendChild(a)
    a.click()
    a.remove()
    setTimeout(() => URL.revokeObjectURL(a.href), 1000)
  } catch (err) {
    toast.error(t.value.exportPage.exportFailed + ': ' + err)
  }
}
</script>

<style scoped>
.filter-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--space-3);
  margin-bottom: var(--space-3);
}

.export-list {
  border-top: 1px solid var(--border);
}

.export-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  padding: var(--space-3) 0;
  border-bottom: 1px solid var(--border-muted);
}

.export-title {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
}

.export-desc {
  margin-top: 2px;
  color: var(--text-muted);
  font-size: 12px;
}

@media (max-width: 640px) {
  .filter-grid {
    grid-template-columns: 1fr;
  }

  .export-row {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
