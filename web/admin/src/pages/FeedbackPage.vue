<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t.feedback.title }}</h1>
        <p class="page-subtitle">{{ t.feedback.subtitle }}</p>
      </div>
      <button type="button" class="btn-refresh" :disabled="loading" @click="loadFeedback">
        <Icon name="refresh-cw" :size="14" />
        {{ t.common.refresh }}
      </button>
    </div>

    <section class="panel feedback-filters">
      <div class="form-grid">
        <div class="form-group">
          <label class="label">{{ t.feedback.status }}</label>
          <select v-model="statusFilter" class="select">
            <option value="">{{ t.feedback.allStatuses }}</option>
            <option value="pending">{{ t.feedback.pending }}</option>
            <option value="processed">{{ t.feedback.processed }}</option>
          </select>
        </div>
        <div class="form-group">
          <label class="label">{{ t.feedback.category }}</label>
          <select v-model="categoryFilter" class="select">
            <option value="">{{ t.feedback.allCategories }}</option>
            <option value="bug">{{ t.feedback.bug }}</option>
            <option value="suggestion">{{ t.feedback.suggestion }}</option>
            <option value="other">{{ t.feedback.other }}</option>
          </select>
        </div>
        <div class="form-group">
          <label class="label">{{ t.feedback.product }}</label>
          <select v-model="productFilter" class="select">
            <option value="">{{ t.feedback.allProducts }}</option>
            <option v-for="product in products" :key="product.id" :value="product.id">
              {{ product.name || product.slug || product.id }}
            </option>
          </select>
        </div>
      </div>
    </section>

    <section class="panel feedback-list-panel">
      <div v-if="loading" class="loading-state">{{ t.common.loading }}</div>
      <div v-else-if="items.length === 0" class="empty-state">{{ t.feedback.empty }}</div>
      <div v-else class="table-container">
        <table class="table feedback-table">
          <thead>
            <tr>
              <th>{{ t.feedback.category }}</th>
              <th>{{ t.feedback.content }}</th>
              <th>{{ t.feedback.context }}</th>
              <th>{{ t.feedback.contact }}</th>
              <th>{{ t.feedback.status }}</th>
              <th>{{ t.feedback.createdAt }}</th>
              <th class="actions">{{ t.feedback.actions }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in items" :key="item.id" class="feedback-row" @click="selected = item">
              <td><span class="badge">{{ categoryLabel(item.category) }}</span></td>
              <td class="feedback-content-cell">{{ preview(item.content) }}</td>
              <td>
                <strong class="feedback-product">{{ item.product_name || item.product_id }}</strong>
                <span class="feedback-context-line">
                  {{ item.variant_platform || '—' }}<span v-if="item.release_version"> · {{ formatVersion(item.release_version) }}</span>
                </span>
              </td>
              <td class="feedback-contact">{{ item.contact || '—' }}</td>
              <td>
                <span class="status-dot" :class="item.status === 'processed' ? 'status-on' : 'status-warning'">
                  <i></i>{{ statusLabel(item.status) }}
                </span>
              </td>
              <td class="date">{{ formatDate(item.created_at) }}</td>
              <td class="actions">
                <div class="table-actions">
                  <button type="button" class="btn-sm" @click.stop="toggleStatus(item)">
                    {{ item.status === 'processed' ? t.feedback.markPending : t.feedback.markProcessed }}
                  </button>
                  <button type="button" class="btn-sm feedback-delete" @click.stop="removeFeedback(item)">
                    {{ t.feedback.delete }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <Pagination
        v-if="total > 0"
        :total="total"
        :page="page"
        :page-size="pageSize"
        @update:page="changePage"
      />
    </section>

    <div v-if="selected" class="feedback-detail-backdrop" @click.self="selected = null">
      <section class="feedback-detail" role="dialog" aria-modal="true">
        <div class="feedback-detail-head">
          <div>
            <h2 class="section-title">{{ t.feedback.detailTitle }}</h2>
            <span class="date">{{ formatDate(selected.created_at) }}</span>
          </div>
          <button type="button" class="btn-icon" :aria-label="t.feedback.closeDetail" @click="selected = null">×</button>
        </div>

        <div class="feedback-detail-meta">
          <span class="badge">{{ categoryLabel(selected.category) }}</span>
          <span class="status-dot" :class="selected.status === 'processed' ? 'status-on' : 'status-warning'">
            <i></i>{{ statusLabel(selected.status) }}
          </span>
        </div>

        <dl class="feedback-detail-context">
          <div><dt>{{ t.feedback.productLabel }}</dt><dd>{{ selected.product_name || selected.product_id }}</dd></div>
          <div><dt>{{ t.feedback.platformLabel }}</dt><dd>{{ selected.variant_platform || '—' }}</dd></div>
          <div><dt>{{ t.feedback.releaseLabel }}</dt><dd>{{ selected.release_version ? formatVersion(selected.release_version) : '—' }}</dd></div>
          <div><dt>{{ t.feedback.contact }}</dt><dd>{{ selected.contact || '—' }}</dd></div>
        </dl>

        <div class="feedback-detail-content">{{ selected.content }}</div>

        <div class="feedback-detail-actions">
          <button type="button" class="btn-secondary" @click="selected = null">{{ t.feedback.closeDetail }}</button>
          <button type="button" class="btn-primary" @click="toggleStatus(selected)">
            {{ selected.status === 'processed' ? t.feedback.markPending : t.feedback.markProcessed }}
          </button>
          <button type="button" class="btn-danger" @click="removeFeedback(selected)">{{ t.feedback.delete }}</button>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useI18n } from '../i18n'
import { useAuth } from '../composables/useAuth'
import { useProducts } from '../composables/useProducts'
import { useToast } from '../composables/useToast'
import { useConfirmDialog } from '../composables/useConfirmDialog'
import { adminFetch } from '../api'
import Icon from '../components/Icon.vue'
import Pagination from '../components/Pagination.vue'

interface FeedbackItem {
  id: string
  product_id: string
  variant_id: string
  release_id: string
  product_name: string
  variant_platform: string
  release_version: string
  category: 'bug' | 'suggestion' | 'other'
  content: string
  contact: string
  status: 'pending' | 'processed'
  created_at: string
  updated_at: string
}

const { t } = useI18n()
const { token, handleAuthError } = useAuth()
const { products, fetchProducts } = useProducts()
const toast = useToast()
const confirmDialog = useConfirmDialog()

const items = ref<FeedbackItem[]>([])
const selected = ref<FeedbackItem | null>(null)
const loading = ref(false)
const page = ref(1)
const pageSize = 20
const total = ref(0)
const statusFilter = ref('')
const categoryFilter = ref('')
const productFilter = ref('')

function categoryLabel(category: FeedbackItem['category']) {
  return t.value.feedback[category]
}

function statusLabel(status: FeedbackItem['status']) {
  return status === 'processed' ? t.value.feedback.processed : t.value.feedback.pending
}

function preview(content: string) {
  return content.length > 100 ? content.slice(0, 100) + '…' : content
}

function formatVersion(version: string) {
  return version.startsWith('v') ? version : `v${version}`
}

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

async function loadFeedback() {
  loading.value = true
  try {
    const query = new URLSearchParams({
      page: String(page.value),
      page_size: String(pageSize)
    })
    if (statusFilter.value) query.set('status', statusFilter.value)
    if (categoryFilter.value) query.set('category', categoryFilter.value)
    if (productFilter.value) query.set('product_id', productFilter.value)

    const response = await adminFetch(`api/feedback?${query.toString()}`, { token: token.value })
    if (handleAuthError(response)) return
    const payload = await response.json()
    if (!response.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.feedback.loadFailed)
      return
    }
    items.value = payload.data?.items || []
    total.value = payload.data?.total || 0
  } catch (error) {
    toast.error(t.value.feedback.loadFailed + ': ' + error)
  } finally {
    loading.value = false
  }
}

function changePage(nextPage: number) {
  page.value = nextPage
  loadFeedback()
}

async function toggleStatus(item: FeedbackItem) {
  const nextStatus = item.status === 'processed' ? 'pending' : 'processed'
  try {
    const response = await adminFetch(`api/feedback/${item.id}`, {
      method: 'PATCH',
      token: token.value,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status: nextStatus })
    })
    if (handleAuthError(response)) return
    const payload = await response.json()
    if (!response.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.feedback.updateFailed)
      return
    }
    item.status = nextStatus
    toast.success(t.value.feedback.updateSuccess)
  } catch (error) {
    toast.error(t.value.feedback.updateFailed + ': ' + error)
  }
}

async function removeFeedback(item: FeedbackItem) {
  const confirmed = await confirmDialog.show({
    title: t.value.feedback.deleteTitle,
    message: t.value.feedback.deleteConfirm,
    confirmText: t.value.feedback.delete,
    cancelText: t.value.common.cancel,
    type: 'danger'
  })
  if (!confirmed) return

  try {
    const response = await adminFetch(`api/feedback/${item.id}`, {
      method: 'DELETE',
      token: token.value
    })
    if (handleAuthError(response)) return
    const payload = await response.json()
    if (!response.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.feedback.deleteFailed)
      return
    }
    if (selected.value?.id === item.id) selected.value = null
    toast.success(t.value.feedback.deleteSuccess)
    await loadFeedback()
  } catch (error) {
    toast.error(t.value.feedback.deleteFailed + ': ' + error)
  }
}

watch([statusFilter, categoryFilter, productFilter], () => {
  page.value = 1
  loadFeedback()
})

onMounted(async () => {
  await fetchProducts()
  await loadFeedback()
})
</script>

<style scoped>
.feedback-filters {
  margin-bottom: var(--space-4);
}

.feedback-list-panel {
  padding: 0;
  overflow: hidden;
}

.feedback-table {
  min-width: 980px;
}

.feedback-row {
  cursor: pointer;
}

.feedback-content-cell {
  max-width: 260px;
  color: var(--text-primary) !important;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.feedback-product,
.feedback-context-line {
  display: block;
}

.feedback-product {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 500;
}

.feedback-context-line,
.feedback-contact {
  max-width: 180px;
  margin-top: 2px;
  color: var(--text-muted);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.feedback-delete {
  color: var(--danger);
}

.feedback-detail-backdrop {
  position: fixed;
  inset: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(15, 23, 42, 0.42);
}

.feedback-detail {
  width: min(100%, 620px);
  max-height: calc(100vh - 40px);
  overflow-y: auto;
  padding: var(--space-5);
  border: 1px solid var(--border);
  border-radius: var(--radius-panel);
  background: var(--surface);
  box-shadow: var(--shadow-popover);
}

.feedback-detail-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-3);
}

.feedback-detail-head .date {
  display: block;
  margin-top: 4px;
}

.feedback-detail-meta {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-top: var(--space-4);
}

.feedback-detail-context {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-3);
  margin: var(--space-4) 0;
  padding: var(--space-3);
  border: 1px solid var(--border-muted);
  border-radius: var(--radius-control);
  background: var(--surface-subtle);
}

.feedback-detail-context div {
  min-width: 0;
}

.feedback-detail-context dt {
  color: var(--text-muted);
  font-size: 11px;
}

.feedback-detail-context dd {
  margin: 2px 0 0;
  color: var(--text-primary);
  font-size: 13px;
  overflow-wrap: anywhere;
}

.feedback-detail-content {
  min-height: 120px;
  padding: var(--space-3);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  color: var(--text-secondary);
  line-height: 1.7;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.feedback-detail-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-4);
}

@media (max-width: 640px) {
  .feedback-detail-context {
    grid-template-columns: 1fr;
  }

  .feedback-detail-actions {
    flex-wrap: wrap;
  }
}
</style>
