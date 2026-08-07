<template>
  <div v-if="show && product" class="modal-overlay" @click.self="handleClose">
    <div class="modal-container announcement-modal">
      <div class="modal-header">
        <h2 class="modal-title">{{ t.announcements.title }}</h2>
        <button @click="handleClose" class="close-btn">✕</button>
      </div>

      <div class="modal-body">
        <div class="announcement-product">
          {{ t.announcements.product }}: <strong>{{ product.name }}</strong>
        </div>

        <div v-if="loading" class="announcement-state">{{ t.common.loading }}</div>

        <template v-else>
          <div class="announcement-toolbar">
            <button class="btn-primary" @click="openCreate">
              <Icon name="plus" :size="14" />
              {{ t.announcements.create }}
            </button>
          </div>

          <div v-if="showForm" class="announcement-form">
            <div class="form-grid">
              <div class="form-group">
                <label class="label">{{ t.announcements.title }} *</label>
                <input v-model="form.title" class="input" :placeholder="t.announcements.titlePlaceholder">
              </div>

              <div class="form-group">
                <label class="label">{{ t.announcements.enabled }}</label>
                <label class="checkbox-label">
                  <input v-model="form.enabled" type="checkbox">
                  {{ form.enabled ? t.announcements.enabledOn : t.announcements.enabledOff }}
                </label>
              </div>
            </div>

            <div class="form-group">
              <label class="label">{{ t.announcements.content }} *</label>
              <textarea v-model="form.content" class="input textarea" rows="4" :placeholder="t.announcements.contentPlaceholder"></textarea>
            </div>

            <div class="form-grid">
              <div class="form-group">
                <label class="label">{{ t.announcements.startAt }}</label>
                <input v-model="form.start_at" type="datetime-local" class="input">
              </div>
              <div class="form-group">
                <label class="label">{{ t.announcements.endAt }}</label>
                <input v-model="form.end_at" type="datetime-local" class="input">
              </div>
            </div>

            <div class="form-actions">
              <button class="btn-secondary" @click="showForm = false">{{ t.common.cancel }}</button>
              <button class="btn-primary" :disabled="saving" @click="save">
                {{ saving ? t.common.loading : (editingID ? t.common.save : t.announcements.create) }}
              </button>
            </div>
          </div>

          <div class="table-container">
            <table class="table">
              <thead>
                <tr>
                  <th>{{ t.announcements.title }}</th>
                  <th>{{ t.announcements.enabled }}</th>
                  <th>{{ t.announcements.startAt }}</th>
                  <th>{{ t.announcements.endAt }}</th>
                  <th>{{ t.announcements.createdAt }}</th>
                  <th style="text-align:right">{{ t.common.actions }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in items" :key="item.id">
                  <td>{{ item.title }}</td>
                  <td>
                    <span :class="item.enabled ? 'status-on' : 'status-off'">
                      {{ item.enabled ? t.announcements.enabledOn : t.announcements.enabledOff }}
                    </span>
                  </td>
                  <td>{{ formatDate(item.start_at) || '-' }}</td>
                  <td>{{ formatDate(item.end_at) || '-' }}</td>
                  <td>{{ formatDate(item.created_at) }}</td>
                  <td class="actions" style="text-align:right">
                    <button class="btn-text" @click="openEdit(item)">{{ t.common.edit }}</button>
                    <button class="btn-text" @click="toggleEnabled(item)">
                      {{ item.enabled ? t.announcements.disable : t.announcements.enable }}
                    </button>
                    <button class="btn-text btn-text-danger" @click="remove(item)">{{ t.common.delete }}</button>
                  </td>
                </tr>
                <tr v-if="items.length === 0">
                  <td colspan="6" class="empty-state">{{ t.announcements.empty }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>
      </div>

      <div class="modal-footer">
        <button class="btn-secondary" @click="handleClose">{{ t.common.close }}</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useAuth } from '../composables/useAuth'
import { useI18n } from '../i18n'
import { useToast } from '../composables/useToast'
import { useConfirmDialog } from '../composables/useConfirmDialog'
import Icon from './Icon.vue'
import { adminURL } from '../api'

const props = defineProps<{
  show: boolean
  product: { id: string; name: string } | null
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const { token, handleAuthError } = useAuth()
const { t } = useI18n()
const toast = useToast()
const confirm = useConfirmDialog()

const loading = ref(false)
const saving = ref(false)
const items = ref<any[]>([])
const showForm = ref(false)
const editingID = ref('')
const form = reactive({
  title: '',
  content: '',
  enabled: true,
  start_at: '',
  end_at: ''
})

watch(() => props.show, async (visible) => {
  if (visible && props.product) {
    items.value = []
    showForm.value = false
    editingID.value = ''
    await fetchAnnouncements()
  }
})

async function fetchAnnouncements() {
  if (!props.product) return
  loading.value = true
  try {
    const res = await fetch(adminURL(`api/products/${props.product.id}/announcements`), {
      headers: { 'X-Auth-Token': token.value }
    })
    if (handleAuthError(res)) return
    const payload = await res.json()
    if (!res.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.announcements.loadFailed)
      return
    }
    items.value = payload.data?.items || []
  } catch (error) {
    toast.error(t.value.announcements.loadFailed + ': ' + error)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingID.value = ''
  form.title = ''
  form.content = ''
  form.enabled = true
  form.start_at = ''
  form.end_at = ''
  showForm.value = true
}

function openEdit(item: any) {
  editingID.value = item.id
  form.title = item.title || ''
  form.content = item.content || ''
  form.enabled = !!item.enabled
  form.start_at = toDateTimeLocal(item.start_at)
  form.end_at = toDateTimeLocal(item.end_at)
  showForm.value = true
}

function toDateTimeLocal(value: string | null): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function fromDateTimeLocal(value: string): string | null {
  if (!value) return null
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? null : date.toISOString()
}

async function save() {
  if (!props.product) return
  if (!form.title.trim() || !form.content.trim()) {
    toast.error(t.value.announcements.required)
    return
  }
  const startAt = fromDateTimeLocal(form.start_at)
  const endAt = fromDateTimeLocal(form.end_at)
  if (startAt && endAt && new Date(startAt) > new Date(endAt)) {
    toast.error(t.value.announcements.invalidRange)
    return
  }

  saving.value = true
  const body = {
    title: form.title,
    content: form.content,
    enabled: form.enabled,
    start_at: startAt,
    end_at: endAt
  }
  try {
    const url = editingID.value
      ? adminURL(`api/announcements/${editingID.value}`)
      : adminURL(`api/products/${props.product.id}/announcements`)
    const res = await fetch(url, {
      method: editingID.value ? 'PUT' : 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Auth-Token': token.value
      },
      body: JSON.stringify(body)
    })
    if (handleAuthError(res)) return
    const payload = await res.json()
    if (!res.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.announcements.saveFailed)
      return
    }
    toast.success(t.value.announcements.saveSuccess)
    showForm.value = false
    await fetchAnnouncements()
    emit('saved')
  } catch (error) {
    toast.error(t.value.announcements.saveFailed + ': ' + error)
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(item: any) {
  const body = {
    title: item.title,
    content: item.content,
    enabled: !item.enabled,
    start_at: item.start_at,
    end_at: item.end_at
  }
  try {
    const res = await fetch(adminURL(`api/announcements/${item.id}`), {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-Auth-Token': token.value
      },
      body: JSON.stringify(body)
    })
    if (handleAuthError(res)) return
    const payload = await res.json()
    if (!res.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.announcements.saveFailed)
      return
    }
    toast.success(t.value.announcements.saveSuccess)
    await fetchAnnouncements()
    emit('saved')
  } catch (error) {
    toast.error(t.value.announcements.saveFailed + ': ' + error)
  }
}

async function remove(item: any) {
  const confirmed = await confirm.show({
    title: t.value.announcements.deleteTitle,
    message: t.value.announcements.deleteConfirm,
    confirmText: t.value.common.delete,
    cancelText: t.value.common.cancel,
    type: 'danger'
  })
  if (!confirmed) return
  try {
    const res = await fetch(adminURL(`api/announcements/${item.id}`), {
      method: 'DELETE',
      headers: { 'X-Auth-Token': token.value }
    })
    if (handleAuthError(res)) return
    const payload = await res.json()
    if (!res.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.announcements.deleteFailed)
      return
    }
    toast.success(t.value.common.deleteSuccess)
    await fetchAnnouncements()
    emit('saved')
  } catch (error) {
    toast.error(t.value.announcements.deleteFailed + ': ' + error)
  }
}

function formatDate(value: string | null): string {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : date.toLocaleString()
}

function handleClose() {
  if (saving.value) return
  emit('close')
}
</script>

<style scoped>
.announcement-modal {
  max-width: 640px;
}

.announcement-product {
  margin-bottom: var(--space-3);
  color: var(--text-muted);
  font-size: 13px;
}

.announcement-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: var(--space-3);
}

.announcement-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin-bottom: var(--space-3);
  padding: var(--space-3);
  background: var(--surface-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-panel);
}

.announcement-state,
.empty-state {
  padding: var(--space-4);
  color: var(--text-muted);
  text-align: center;
}

.table-container {
  max-height: 340px;
  overflow-y: auto;
}

@media (max-width: 640px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
