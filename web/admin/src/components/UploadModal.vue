<template>
  <div v-if="show && product && variant" class="modal-overlay" @click.self="handleClose">
    <div class="modal-container upload-modal">
      <div class="modal-header">
        <h2 class="modal-title">{{ t.uploadModal.title }}</h2>
        <button @click="handleClose" class="close-btn" aria-label="Close">×</button>
      </div>

      <div class="modal-body">
        <div class="upload-target">
          <div>
            <span class="info-label">{{ t.uploadModal.appName }}</span>
            <span>{{ product.name }}</span>
          </div>
          <div>
            <span class="info-label">{{ t.uploadModal.platform }}</span>
            <span>{{ variant.platform }}</span>
          </div>
          <div>
            <span class="info-label">{{ t.uploadModal.identifier }}</span>
            <span class="mono">{{ variant.identifier }}</span>
          </div>
        </div>

        <div class="form-grid">
          <div class="form-group">
            <label class="label">{{ t.uploadModal.versionRequired }}</label>
            <input v-model="form.version" :placeholder="t.uploadModal.versionPlaceholder" class="input">
          </div>
          <div class="form-group">
            <label class="label">{{ t.uploadModal.build }}</label>
            <input v-model="form.build" :placeholder="t.uploadModal.buildPlaceholder" class="input">
          </div>
          <div class="form-group">
            <label class="label">{{ t.uploadModal.channel }}</label>
            <input v-model="form.channel" :placeholder="t.uploadModal.channelPlaceholder" class="input">
          </div>
          <div class="form-group">
            <label class="label">{{ t.uploadModal.minOS }}</label>
            <input v-model="form.min_os" :placeholder="t.uploadModal.minOSPlaceholder" class="input">
          </div>
        </div>

        <div class="form-group" style="margin-top: var(--space-3)">
          <label class="label">{{ t.uploadModal.changelog }}</label>
          <textarea v-model="form.changelog" :placeholder="t.uploadModal.changelogPlaceholder" class="textarea" rows="3"></textarea>
        </div>

        <div class="form-group" style="margin-top: var(--space-3)">
          <label class="checkbox-label">
            <input v-model="form.force_update" type="checkbox">
            {{ t.uploadModal.forceUpdate }}
          </label>
        </div>

        <div class="form-group" style="margin-top: var(--space-3)">
          <label class="label">{{ t.uploadModal.fileRequired }}</label>
          <div
            class="dropzone"
            :class="{ dragover: isDragging, 'has-file': selectedFile }"
            @drop.prevent="handleDrop"
            @dragover.prevent="isDragging = true"
            @dragleave.prevent="isDragging = false"
            @click="triggerFileInput"
          >
            <input
              ref="fileInput"
              type="file"
              :accept="acceptTypes"
              @change="handleFileSelect"
              class="visually-hidden"
            >

            <div v-if="!selectedFile" class="dropzone-placeholder">
              <Icon name="upload" :size="16" />
              <p class="primary-text">{{ t.uploadModal.dragDropText }}</p>
              <p class="secondary-text">{{ t.uploadModal.orClickToSelect }}</p>
              <p class="file-type-text">{{ acceptTypes }}</p>
            </div>

            <div v-else class="file-selected">
              <div class="file-info">
                <div class="file-name">{{ selectedFile.name }}</div>
                <div class="file-size">{{ formatFileSize(selectedFile.size) }}</div>
              </div>
              <button @click.stop="clearFile" class="remove-file-btn" aria-label="Remove file">×</button>
            </div>
          </div>
        </div>

        <div v-if="uploading" class="progress-section">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: uploadProgress + '%' }"></div>
          </div>
          <div class="progress-text">{{ t.uploadModal.uploading }} {{ uploadProgress }}%</div>
        </div>

        <div v-if="uploadResult" class="result-section" :class="uploadResult.ok ? 'success' : 'error'">
          <div v-if="uploadResult.ok">
            <div class="result-title">{{ t.uploadModal.uploadSuccess }}</div>
            <div class="result-content">
              <div><strong>{{ t.uploadModal.version }}:</strong> {{ uploadResult.data.release.version }}</div>
              <div><strong>{{ t.uploadModal.downloadPage }}:</strong> <a :href="uploadResult.data.urls.page" target="_blank">{{ uploadResult.data.urls.page }}</a></div>
            </div>
          </div>
          <div v-else>
            {{ t.uploadModal.uploadFailed }}: {{ uploadResult.error?.message || t.uploadModal.unknownError }}
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button @click="handleClose" class="btn-secondary">{{ t.common.cancel }}</button>
        <button @click="handleUpload" :disabled="!canUpload" class="btn-primary">
          {{ uploading ? t.uploadModal.uploading : t.uploadModal.uploadButton }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from '../i18n'
import Icon from './Icon.vue'
import { adminURL } from '../api'

const props = defineProps<{
  show: boolean
  product: { id: string; name: string } | null
  variant: { id: string; platform: string; identifier: string } | null
}>()

const emit = defineEmits<{
  close: []
  success: []
}>()

const { t } = useI18n()
const uploadBaseURL = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)
const isDragging = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)
const uploadResult = ref<any>(null)

const form = reactive({
  version: '',
  build: '',
  channel: 'internal',
  min_os: '',
  changelog: '',
  force_update: false
})

const acceptTypes = computed(() => {
  const platform = props.variant?.platform || ''
  const map: Record<string, string> = {
    ios: '.ipa',
    android: '.apk',
    macos: '.dmg,.pkg,.zip',
    windows: '.exe,.msi,.zip',
    linux: '.appimage,.deb,.rpm,.zip,.tar.gz'
  }
  return map[platform] || '*'
})

const canUpload = computed(() => {
  return !!selectedFile.value && !!form.version && !uploading.value
})

onMounted(async () => {
  try {
    const token = localStorage.getItem('fenfa_admin_token') || ''
    const res = await fetch(adminURL('api/upload-config'), {
      headers: { 'X-Auth-Token': token }
    })
    if (!res.ok) return
    const payload = await res.json()
    if (payload.ok && payload.data?.upload_domain) {
      uploadBaseURL.value = payload.data.upload_domain
    }
  } catch {}
})

function resetForm() {
  form.version = ''
  form.build = ''
  form.channel = 'internal'
  form.min_os = ''
  form.changelog = ''
  form.force_update = false
  selectedFile.value = null
  uploadProgress.value = 0
  uploadResult.value = null
}

function handleClose() {
  if (uploading.value) return
  emit('close')
  resetForm()
}

function triggerFileInput() {
  fileInput.value?.click()
}

function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement
  selectedFile.value = target.files?.[0] || null
}

function lowerFileName(file: File): string {
  return file.name.toLowerCase()
}

function fileMatchesVariant(file: File): boolean {
  const name = lowerFileName(file)
  const allowed = acceptTypes.value.split(',').map((item) => item.trim().toLowerCase()).filter(Boolean)
  return allowed.some((ext) => name.endsWith(ext))
}

function handleDrop(event: DragEvent) {
  isDragging.value = false
  const file = event.dataTransfer?.files?.[0]
  if (!file) return
  if (!fileMatchesVariant(file)) {
    alert(t.value.uploadModal.invalidFileType.replace('{type}', acceptTypes.value))
    return
  }
  selectedFile.value = file
}

function clearFile() {
  selectedFile.value = null
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
}

async function handleUpload() {
  if (!props.variant || !selectedFile.value || !form.version) return

  const fd = new FormData()
  fd.set('variant_id', props.variant.id)
  fd.set('version', form.version)
  if (form.build) fd.set('build', form.build)
  if (form.channel) fd.set('channel', form.channel)
  if (form.min_os) fd.set('min_os', form.min_os)
  if (form.changelog) fd.set('changelog', form.changelog)
  if (form.force_update) fd.set('force_update', 'true')
  fd.set('app_file', selectedFile.value)

  uploading.value = true
  uploadProgress.value = 0
  uploadResult.value = null

  try {
    const token = localStorage.getItem('fenfa_admin_token') || ''
    const xhr = new XMLHttpRequest()

    xhr.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable) {
        uploadProgress.value = Math.round((event.loaded / event.total) * 100)
      }
    })

    const response = await new Promise<any>((resolve, reject) => {
      xhr.addEventListener('load', () => {
        if (xhr.status === 200 || xhr.status === 201) {
          resolve(JSON.parse(xhr.responseText))
          return
        }
        reject(new Error(`HTTP ${xhr.status}`))
      })
      xhr.addEventListener('error', () => reject(new Error('Network error')))
      xhr.open('POST', uploadBaseURL.value + '/upload')
      xhr.setRequestHeader('X-Auth-Token', token)
      xhr.send(fd)
    })

    uploadResult.value = response
    if (response.ok) {
      setTimeout(() => {
        emit('success')
        resetForm()
      }, 1200)
    }
  } catch (error: any) {
    uploadResult.value = {
      ok: false,
      error: { message: error.message }
    }
  } finally {
    uploading.value = false
  }
}
</script>

<style scoped>
.upload-modal {
  max-width: 560px;
}

.upload-target {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: var(--space-3);
  margin-bottom: var(--space-4);
  padding: var(--space-3);
  background: var(--surface-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
}

.upload-target div {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 2px;
  color: var(--text-secondary);
  font-size: 13px;
}

.info-label {
  color: var(--text-muted);
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
</style>
