<template>
  <div class="smart-upload">
    <div v-if="step === 1" class="smart-grid">
      <div class="panel">
        <h3 class="section-title">{{ t.smartUpload.title }}</h3>
        <p class="page-subtitle">{{ t.smartUpload.subtitle }}</p>

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
            accept=".apk,.ipa"
            @change="handleFileSelect"
            class="visually-hidden"
          >

          <div v-if="!selectedFile" class="dropzone-placeholder">
            <Icon name="upload" :size="16" />
            <p class="primary-text">{{ t.smartUpload.dragDropText }}</p>
            <p class="secondary-text">{{ t.smartUpload.orClickToSelect }}</p>
            <p class="file-type-text">{{ t.smartUpload.supportedFiles }}</p>
          </div>

          <div v-else class="file-selected">
            <Icon name="package" :size="20" />
            <div class="file-info">
              <div class="file-name">{{ selectedFile.name }}</div>
              <div class="file-size">{{ formatFileSize(selectedFile.size) }}</div>
            </div>
            <button @click.stop="clearFile" class="remove-file-btn" aria-label="Remove file">×</button>
          </div>
        </div>

        <div v-if="parseError" class="error-message">
          <Icon name="alert" :size="14" />
          {{ t.smartUpload.parseFailed }}: {{ parseError }}
        </div>

        <div v-if="selectedFile" class="form-actions">
          <button @click="parseAppInfo" :disabled="parsing" class="btn-primary">
            <Icon name="search" :size="14" />
            {{ parsing ? t.smartUpload.parsing : t.smartUpload.parseButton }}
          </button>
        </div>

        <div v-if="parsing" class="progress-section">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: parseProgress + '%' }"></div>
          </div>
          <div class="progress-text">{{ t.smartUpload.parsing }} {{ parseProgress }}%</div>
        </div>
      </div>

      <div class="panel target-panel">
        <div class="section-title">{{ t.upload.selectVariant }}</div>
        <div class="target-meta">
          <span>{{ variant?.platform }}</span>
          <span class="mono">{{ variant?.identifier }}</span>
        </div>
      </div>
    </div>

    <div v-if="step === 2 && appInfo" class="panel">
      <div class="parsed-header">
        <img v-if="appInfo.icon_base64" :src="appInfo.icon_base64" class="app-icon" alt="">
        <div v-else class="app-icon-fallback"><Icon name="smartphone" :size="20" /></div>
        <div>
          <h3 class="section-title">{{ appInfo.app_name || t.smartUpload.unnamed }}</h3>
          <div class="version-text">
            {{ appInfo.version || appInfo.version_name }}
            <span v-if="appInfo.build || appInfo.version_code">({{ appInfo.build || appInfo.version_code }})</span>
          </div>
        </div>
      </div>

      <div class="info-grid">
        <div class="info-item">
          <span class="info-label">{{ t.smartUpload.identifier }}</span>
          <span class="mono">{{ appInfo.bundle_id || appInfo.package_name }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">{{ t.smartUpload.minOS }}</span>
          <span>{{ appInfo.min_os || appInfo.min_sdk }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">{{ t.smartUpload.fileSize }}</span>
          <span>{{ formatFileSize(appInfo.file_size) }}</span>
        </div>
      </div>

      <div class="form-grid">
        <div class="form-group">
          <label class="label">{{ t.smartUpload.channel }}</label>
          <input v-model="form.channel" :placeholder="t.smartUpload.channelPlaceholder" class="input">
        </div>
      </div>

      <div class="form-group" style="margin-top: var(--space-3)">
        <label class="label">{{ t.smartUpload.changelog }}</label>
        <textarea v-model="form.changelog" :placeholder="t.smartUpload.changelogPlaceholder" class="textarea" rows="3"></textarea>
      </div>

      <div class="form-group" style="margin-top: var(--space-3)">
        <label class="checkbox-label">
          <input v-model="form.force_update" type="checkbox">
          {{ t.smartUpload.forceUpdate }}
        </label>
      </div>

      <div class="form-actions">
        <button @click="step = 1" class="btn-secondary">{{ t.common.back }}</button>
        <button @click="handleUpload" :disabled="uploading" class="btn-primary">
          {{ uploading ? t.smartUpload.uploading : t.smartUpload.uploadButton }}
        </button>
      </div>

      <div v-if="uploading" class="progress-section">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: uploadProgress + '%' }"></div>
        </div>
        <div class="progress-text">{{ t.smartUpload.uploading }} {{ uploadProgress }}%</div>
      </div>

      <div v-if="uploadResult" class="result-section" :class="uploadResult.ok ? 'success' : 'error'">
        <div v-if="uploadResult.ok">
          <div class="result-title"><Icon name="check" :size="14" /> {{ t.smartUpload.uploadSuccess }}</div>
          <div class="result-content">
            <div><strong>{{ t.smartUpload.downloadPage }}:</strong> <a :href="uploadResult.data.urls.page" target="_blank">{{ uploadResult.data.urls.page }}</a></div>
          </div>
        </div>
        <div v-else>
          {{ t.smartUpload.uploadFailed }}: {{ uploadResult.error?.message || t.smartUpload.unknownError }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from '../i18n'
import { uploadWithProgress } from '../composables/useUpload'
import Icon from './Icon.vue'
import { adminFetch, adminURL } from '../api'

const props = defineProps<{
  token: string
  onAuthError: (res: Response) => boolean
  variant: { id: string; platform: string; identifier: string } | null
}>()

// Dynamic binary upload URL (may point to a different domain)
const uploadBaseURL = ref('')

async function fetchUploadConfig() {
  try {
    const res = await adminFetch('api/upload-config', {
      token: props.token
    })
    if (res.ok) {
      const data = await res.json()
      if (data.ok && data.data?.upload_domain) {
        uploadBaseURL.value = data.data.upload_domain
      }
    }
  } catch (_) { /* fallback to same origin */ }
}

onMounted(fetchUploadConfig)

const emit = defineEmits<{
  success: []
}>()

const { t } = useI18n()

const step = ref(1)
const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)
const isDragging = ref(false)
const parsing = ref(false)
const parseProgress = ref(0)
const parseError = ref('')
const appInfo = ref<any>(null)
const uploading = ref(false)
const uploadProgress = ref(0)
const uploadResult = ref<any>(null)

const form = reactive({
  channel: 'internal',
  changelog: '',
  force_update: false
})

function triggerFileInput() {
  fileInput.value?.click()
}

function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (file) {
    selectedFile.value = file
    parseError.value = ''
  }
}

function handleDrop(event: DragEvent) {
  isDragging.value = false
  const file = event.dataTransfer?.files?.[0]
  if (file) {
    const ext = file.name.split('.').pop()?.toLowerCase()
    if (ext === 'apk' || ext === 'ipa') {
      selectedFile.value = file
      parseError.value = ''
    } else {
      parseError.value = t.value.smartUpload.invalidFileType
    }
  }
}

function clearFile() {
  selectedFile.value = null
  appInfo.value = null
  step.value = 1
  parseProgress.value = 0
  parseError.value = ''
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
}

async function parseAppInfo() {
  if (!selectedFile.value) return
  if (!props.variant) {
    parseError.value = t.value.upload.selectVariantFirst
    return
  }

  parsing.value = true
  parseProgress.value = 0
  parseError.value = ''

  const fd = new FormData()
  fd.set('app_file', selectedFile.value)

  const result = await uploadWithProgress({
    url: adminURL('api/parse-app'),
    token: props.token,
    formData: fd,
    onProgress: (percent) => {
      parseProgress.value = percent
    },
    onAuthError: () => {
      props.onAuthError(new Response(null, { status: 401 }))
    }
  })

  parsing.value = false
  if (result.error?.message === 'AUTH_ERROR') return

  if (result.ok) {
    if (result.data?.platform !== props.variant.platform) {
      parseError.value = t.value.smartUpload.platformMismatch
      return
    }
    appInfo.value = result.data
    step.value = 2
  } else {
    parseError.value = result.error?.message || t.value.smartUpload.unknownError
  }
}

async function handleUpload() {
  if (!selectedFile.value || !appInfo.value || !props.variant) return

  const fd = new FormData()
  fd.set('variant_id', props.variant.id)
  if (appInfo.value.version || appInfo.value.version_name) {
    fd.set('version', appInfo.value.version || appInfo.value.version_name)
  }
  if (appInfo.value.build || appInfo.value.version_code) {
    fd.set('build', String(appInfo.value.build || appInfo.value.version_code))
  }
  if (appInfo.value.min_os || appInfo.value.min_sdk) {
    fd.set('min_os', appInfo.value.min_os || appInfo.value.min_sdk)
  }
  if (form.channel) {
    fd.set('channel', form.channel)
  }
  if (form.changelog) {
    fd.set('changelog', form.changelog)
  }
  if (form.force_update) {
    fd.set('force_update', 'true')
  }
  if (appInfo.value.icon_base64) {
    fd.set('icon_base64', appInfo.value.icon_base64)
  }
  fd.set('app_file', selectedFile.value)

  uploading.value = true
  uploadProgress.value = 0
  uploadResult.value = null

  const response = await uploadWithProgress({
    url: uploadBaseURL.value + '/upload',
    token: props.token,
    formData: fd,
    withCredentials: false,
    onProgress: (percent) => {
      uploadProgress.value = percent
    },
    onAuthError: () => {
      props.onAuthError(new Response(null, { status: 401 }))
    }
  })

  uploading.value = false
  if (response.error?.message === 'AUTH_ERROR') return

  uploadResult.value = response
  if (response.ok) {
    setTimeout(() => {
      emit('success')
      resetForm()
    }, 2000)
  }
}

function resetForm() {
  step.value = 1
  selectedFile.value = null
  appInfo.value = null
  form.channel = 'internal'
  form.changelog = ''
  form.force_update = false
  parseProgress.value = 0
  uploadProgress.value = 0
  uploadResult.value = null
  parseError.value = ''
}
</script>

<style scoped>
.smart-upload {
  width: 100%;
}

.smart-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 260px;
  gap: var(--space-3);
}

.target-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.target-meta {
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: var(--text-secondary);
  font-size: 13px;
}

.parsed-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-3);
}

.app-icon,
.app-icon-fallback {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  object-fit: cover;
  border-radius: var(--radius-sm);
}

.app-icon-fallback {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--surface-subtle);
  border: 1px solid var(--border);
  color: var(--text-muted);
}

.version-text {
  margin-top: 2px;
  color: var(--text-muted);
  font-size: 12px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: var(--space-3);
  margin-bottom: var(--space-3);
  padding: var(--space-3);
  background: var(--surface-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
}

.info-item {
  display: flex;
  flex-direction: column;
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

@media (max-width: 720px) {
  .smart-grid {
    grid-template-columns: 1fr;
  }
}
</style>
