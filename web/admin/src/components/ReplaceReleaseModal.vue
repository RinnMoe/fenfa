<template>
  <n-modal
    :show="show && !!release && !!variant"
    preset="card"
    :style="{ width: '640px', maxWidth: 'calc(100vw - 32px)' }"
    :mask-closable="!uploading"
    :close-on-esc="!uploading"
    @update:show="handleShow"
  >
    <template #header>
      <h2 class="modal-title">{{ t.replaceRelease.title }}</h2>
    </template>

    <div class="modal-body">
      <div class="replace-target">
        <div>
          <span class="info-label">{{ t.replaceRelease.currentFile }}</span>
          <span>{{ release?.file_name || '-' }}</span>
        </div>
        <div>
          <span class="info-label">{{ t.apps.latestRelease }}</span>
          <span>{{ release?.version }} <template v-if="release?.build">({{ t.upload.build }} {{ release.build }})</template></span>
        </div>
      </div>

      <div class="form-grid">
        <div class="form-group">
          <label class="label">{{ t.replaceRelease.version }}</label>
          <input v-model="form.version" :placeholder="t.replaceRelease.versionPlaceholder" class="input">
        </div>
        <div class="form-group">
          <label class="label">{{ t.replaceRelease.build }}</label>
          <input v-model="form.build" :placeholder="t.replaceRelease.buildPlaceholder" class="input">
        </div>
        <div class="form-group">
          <label class="label">{{ t.replaceRelease.channel }}</label>
          <input v-model="form.channel" :placeholder="t.replaceRelease.channelPlaceholder" class="input">
        </div>
        <div class="form-group">
          <label class="label">{{ t.replaceRelease.minOS }}</label>
          <input v-model="form.min_os" :placeholder="t.replaceRelease.minOSPlaceholder" class="input">
        </div>
      </div>

      <div class="form-group" style="margin-top: var(--space-3)">
        <label class="label">{{ t.replaceRelease.changelog }}</label>
        <textarea v-model="form.changelog" :placeholder="t.replaceRelease.changelogPlaceholder" class="textarea" rows="3"></textarea>
      </div>

      <div class="form-group" style="margin-top: var(--space-3)">
        <label class="checkbox-label">
          <input v-model="form.force_update" type="checkbox">
          {{ t.replaceRelease.forceUpdate }}
        </label>
      </div>

      <div v-if="iconBase64 && canParse" class="form-group icon-row">
        <img :src="iconBase64" class="icon-preview" alt="">
        <label class="checkbox-label">
          <input v-model="updateIcon" type="checkbox">
          {{ t.replaceRelease.updateIcon }}
        </label>
      </div>

      <div class="form-group" style="margin-top: var(--space-3)">
        <label class="label">{{ t.replaceRelease.fileRequired }}</label>
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
            <p class="primary-text">{{ t.replaceRelease.dragDropText }}</p>
            <p class="secondary-text">{{ t.replaceRelease.orClickToSelect }}</p>
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

        <div v-if="canParse && selectedFile && !parsing" class="form-actions">
          <button @click="parseAppInfo" class="btn-ghost">
            <Icon name="search" :size="14" />
            {{ t.replaceRelease.parseButton }}
          </button>
        </div>

        <div v-if="parsing" class="progress-section">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: parseProgress + '%' }"></div>
          </div>
          <div class="progress-text">{{ t.replaceRelease.parsing }} {{ parseProgress }}%</div>
        </div>

        <div v-if="parseError" class="error-message">
          <Icon name="alert" :size="14" />
          {{ t.replaceRelease.parseFailed }}: {{ parseError }}
        </div>
      </div>

      <div v-if="uploading" class="progress-section">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: uploadProgress + '%' }"></div>
        </div>
        <div class="progress-text">{{ t.replaceRelease.uploading }} {{ uploadProgress }}%</div>
      </div>

      <div v-if="uploadResult && !uploadResult.ok" class="result-section error">
        {{ t.replaceRelease.replaceFailed }}: {{ uploadResult.error?.message || t.replaceRelease.unknownError }}
      </div>
    </div>

    <template #footer>
      <button @click="handleClose" class="btn-secondary">{{ t.common.cancel }}</button>
      <button @click="handleReplace" :disabled="!canReplace" class="btn-primary">
        {{ uploading ? t.replaceRelease.uploading : t.replaceRelease.replaceButton }}
      </button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { NModal } from 'naive-ui'
import { useI18n } from '../i18n'
import { useAuth } from '../composables/useAuth'
import { useToast } from '../composables/useToast'
import { uploadWithProgress } from '../composables/useUpload'
import Icon from './Icon.vue'
import { adminFetch, adminURL } from '../api'

const props = defineProps<{
  show: boolean
  release: any | null
  variant: any | null
}>()

const emit = defineEmits<{
  close: []
  success: []
}>()

const { t } = useI18n()
const { token, handleAuthError } = useAuth()
const toast = useToast()
const uploadBaseURL = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)
const isDragging = ref(false)
const parsing = ref(false)
const parseProgress = ref(0)
const parseError = ref('')
const iconBase64 = ref('')
const updateIcon = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)
const uploadResult = ref<any>(null)

const form = reactive({
  version: '',
  build: '',
  channel: '',
  min_os: '',
  changelog: '',
  force_update: false
})

const canParse = computed(() => {
  const platform = props.variant?.platform || ''
  return platform === 'apk' || platform === 'ipa' || platform === 'android' || platform === 'ios'
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

const canReplace = computed(() => {
  return !!selectedFile.value && form.version.trim().length > 0 && !uploading.value
})

onMounted(async () => {
  try {
    const res = await adminFetch('api/upload-config', { token: token.value })
    if (!res.ok) return
    const payload = await res.json()
    if (payload.ok && payload.data?.upload_domain) {
      uploadBaseURL.value = payload.data.upload_domain
    }
  } catch {}
})

watch(() => props.show, (show) => {
  if (show && props.release) {
    form.version = props.release.version || ''
    form.build = props.release.build != null ? String(props.release.build) : ''
    form.channel = props.release.channel || ''
    form.min_os = props.release.min_os || ''
    form.changelog = props.release.changelog || ''
    form.force_update = !!props.release.force_update
    iconBase64.value = ''
    updateIcon.value = false
    selectedFile.value = null
    parseError.value = ''
    parseProgress.value = 0
    uploadProgress.value = 0
    uploadResult.value = null
  }
})

function handleClose() {
  if (uploading.value) return
  emit('close')
}

function handleShow(next: boolean) {
  if (!next) handleClose()
}

function triggerFileInput() {
  fileInput.value?.click()
}

function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0] || null
  if (file && !fileMatchesVariant(file)) {
    toast.error(t.value.replaceRelease.invalidFileType.replace('{type}', acceptTypes.value))
    if (fileInput.value) fileInput.value.value = ''
    return
  }
  selectedFile.value = file
  parseError.value = ''
  iconBase64.value = ''
  updateIcon.value = false
}

function handleDrop(event: DragEvent) {
  isDragging.value = false
  const file = event.dataTransfer?.files?.[0]
  if (!file) return
  if (!fileMatchesVariant(file)) {
    toast.error(t.value.replaceRelease.invalidFileType.replace('{type}', acceptTypes.value))
    return
  }
  selectedFile.value = file
  parseError.value = ''
  iconBase64.value = ''
  updateIcon.value = false
}

function fileMatchesVariant(file: File): boolean {
  const name = file.name.toLowerCase()
  const allowed = acceptTypes.value.split(',').map((item) => item.trim().toLowerCase()).filter(Boolean)
  return allowed.some((ext) => name.endsWith(ext))
}

function clearFile() {
  selectedFile.value = null
  parseError.value = ''
  iconBase64.value = ''
  updateIcon.value = false
  if (fileInput.value) fileInput.value.value = ''
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
}

async function parseAppInfo() {
  if (!selectedFile.value) return
  parsing.value = true
  parseProgress.value = 0
  parseError.value = ''
  const fd = new FormData()
  fd.set('app_file', selectedFile.value)
  const result = await uploadWithProgress({
    url: uploadBaseURL.value + adminURL('api/parse-app'),
    token: token.value,
    formData: fd,
    onProgress: (percent) => { parseProgress.value = percent },
    onAuthError: () => { handleAuthError(new Response(null, { status: 401 })) }
  })
  parsing.value = false
  if (result.error?.message === 'AUTH_ERROR') return
  if (result.ok && result.data) {
    const info = result.data
    form.version = info.version || info.version_name || form.version
    if (info.build || info.version_code) form.build = String(info.build || info.version_code)
    if (info.min_os || info.min_sdk) form.min_os = info.min_os || info.min_sdk
    iconBase64.value = info.icon_base64 || ''
    updateIcon.value = !!iconBase64.value
  } else {
    parseError.value = result.error?.message || t.value.replaceRelease.parseFailed
  }
}

async function handleReplace() {
  if (!props.release || !selectedFile.value || !canReplace.value) return
  const buildNum = form.build === '' ? undefined : Number(form.build)
  if (buildNum !== undefined && (!Number.isInteger(buildNum) || buildNum < 0)) {
    toast.error(t.value.replaceRelease.invalidBuild)
    return
  }
  const fd = new FormData()
  fd.set('version', form.version.trim())
  if (buildNum !== undefined) fd.set('build', String(buildNum))
  if (form.channel) fd.set('channel', form.channel)
  if (form.min_os) fd.set('min_os', form.min_os)
  if (form.changelog) fd.set('changelog', form.changelog)
  if (form.force_update) fd.set('force_update', 'true')
  if (updateIcon.value && iconBase64.value) fd.set('icon_base64', iconBase64.value)
  fd.set('app_file', selectedFile.value)

  uploading.value = true
  uploadProgress.value = 0
  uploadResult.value = null
  const response = await uploadWithProgress({
    url: uploadBaseURL.value + adminURL(`api/releases/${props.release.id}/replace-file`),
    token: token.value,
    formData: fd,
    onProgress: (percent) => { uploadProgress.value = percent },
    onAuthError: () => { handleAuthError(new Response(null, { status: 401 })) }
  })
  uploading.value = false
  if (response.error?.message === 'AUTH_ERROR') return
  uploadResult.value = response
  if (response.ok) {
    toast.success(t.value.replaceRelease.replaceSuccess)
    emit('success')
  }
}
</script>

<style scoped>
.replace-target {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--space-3);
  margin-bottom: var(--space-4);
  padding: var(--space-3);
  background: var(--surface-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
}

.replace-target div {
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

.icon-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.icon-preview {
  width: 36px;
  height: 36px;
  object-fit: cover;
  border-radius: var(--radius-sm);
}
</style>
