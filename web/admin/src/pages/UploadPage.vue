<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t.upload.title }}</h1>
        <p class="page-subtitle">{{ t.upload.subtitle }}</p>
      </div>
    </div>

    <div class="panel">
      <div class="form-grid">
        <div class="form-group">
          <label class="label" for="upload-product">{{ t.apps.name }}</label>
          <select id="upload-product" v-model="selectedProductId" class="select">
            <option value="">{{ t.upload.selectProduct }}</option>
            <option v-for="product in products" :key="product.id" :value="product.id">
              {{ product.name }}
            </option>
          </select>
        </div>

        <div class="form-group">
          <label class="label" for="upload-variant">{{ t.apps.platform }}</label>
          <select id="upload-variant" v-model="selectedVariantId" class="select" :disabled="!selectedProductId || loadingVariants">
            <option value="">{{ loadingVariants ? t.common.loading : t.upload.selectVariant }}</option>
            <option v-for="variant in variantOptions" :key="variant.id" :value="variant.id">
              {{ variant.platform }} · {{ variant.identifier }}
            </option>
          </select>
        </div>
      </div>
    </div>

    <div v-if="!selectedVariant" class="panel empty-state">
      <p v-if="!selectedProductId">{{ t.upload.selectProductFirst }}</p>
      <p v-else-if="loadingVariants">{{ t.common.loading }}</p>
      <p v-else-if="variantOptions.length === 0">{{ t.upload.noVariantsHint }}</p>
      <p v-else>{{ t.upload.selectVariantFirst }}</p>
    </div>

    <template v-else>
      <div v-if="isMobilePlatform" class="mode-selector">
        <button
          type="button"
          @click="uploadMode = 'smart'"
          class="mode-btn"
          :class="{ active: uploadMode === 'smart' }"
        >
          <span class="mode-title">{{ t.upload.smartMode }}</span>
          <span class="mode-desc">{{ t.upload.smartModeDesc }}</span>
        </button>
        <button
          type="button"
          @click="uploadMode = 'manual'"
          class="mode-btn"
          :class="{ active: uploadMode === 'manual' }"
        >
          <span class="mode-title">{{ t.upload.manualMode }}</span>
          <span class="mode-desc">{{ t.upload.manualModeDesc }}</span>
        </button>
      </div>

      <SmartUpload
        v-if="uploadMode === 'smart' && isMobilePlatform"
        :token="token"
        :on-auth-error="handleAuthError"
        :variant="selectedVariant"
        @success="fetchProducts"
      />

      <div v-else class="panel">
        <div class="form-grid">
          <div class="form-group">
            <label class="label">{{ t.upload.versionRequired }}</label>
            <input v-model="form.version" :placeholder="t.upload.versionPlaceholder" class="input">
          </div>

          <div class="form-group">
            <label class="label">{{ t.upload.build }}</label>
            <input v-model="form.build" :placeholder="t.upload.buildPlaceholder" class="input">
          </div>

          <div class="form-group">
            <label class="label">{{ t.upload.channel }}</label>
            <input v-model="form.channel" :placeholder="t.upload.channelPlaceholder" class="input">
          </div>

          <div class="form-group">
            <label class="label">{{ t.upload.minOS }}</label>
            <input v-model="form.min_os" :placeholder="t.upload.minOSPlaceholder" class="input">
          </div>
        </div>

        <div class="form-group full-width" style="margin-top: var(--space-3)">
          <label class="label">{{ t.upload.changelog }}</label>
          <textarea v-model="form.changelog" :placeholder="t.upload.changelogPlaceholder" class="textarea" rows="3"></textarea>
        </div>

        <div class="form-group full-width" style="margin-top: var(--space-3)">
          <label class="checkbox-label">
            <input v-model="form.force_update" type="checkbox">
            {{ t.upload.forceUpdate }}
          </label>
        </div>

        <div class="form-group full-width" style="margin-top: var(--space-3)">
          <label class="label">{{ t.upload.fileRequired }}</label>
          <input ref="fileInput" type="file" :accept="acceptTypes" class="file-input">
        </div>

        <div class="form-actions">
          <button @click="doUpload" :disabled="uploading" class="btn-primary">
            {{ uploading ? t.upload.uploading : t.upload.uploadButton }}
          </button>
        </div>

        <div v-if="response" class="result-section" :class="response.ok ? 'success' : 'error'">
          <div v-if="response.ok">
            <div class="result-title">{{ t.upload.uploadSuccess }}</div>
            <div class="result-content">
              <div><strong>{{ t.upload.downloadPage }}:</strong> <a :href="response.data.urls.page" target="_blank">{{ response.data.urls.page }}</a></div>
              <div><strong>{{ t.upload.directDownload }}:</strong> <a :href="response.data.urls.download" target="_blank">{{ response.data.urls.download }}</a></div>
            </div>
          </div>
          <div v-else>
            {{ t.upload.uploadFailed }}: {{ response.error?.message || t.upload.unknownError }}
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch, onMounted } from 'vue'
import { useI18n } from '../i18n'
import { useAuth } from '../composables/useAuth'
import { useProducts } from '../composables/useProducts'
import { useToast } from '../composables/useToast'
import { useProductVariantSelector } from '../composables/useProductVariantSelector'
import SmartUpload from '../components/SmartUpload.vue'
import { adminFetch } from '../api'

const { t } = useI18n()
const { token, handleAuthError } = useAuth()
const { products, fetchProducts } = useProducts()
const toast = useToast()

const {
  selectedProductId, selectedVariantId, loadingVariants,
  variantOptions, selectedVariant
} = useProductVariantSelector(products, token, handleAuthError)

const uploadMode = ref<'smart' | 'manual'>('smart')
const uploading = ref(false)
const response = ref<any>(null)
const uploadBaseURL = ref('')

const form = reactive({
  version: '',
  build: '',
  channel: '',
  min_os: '',
  changelog: '',
  force_update: false
})

const fileInput = ref<HTMLInputElement | null>(null)

const isMobilePlatform = computed(() => {
  return selectedVariant.value && ['ios', 'android'].includes(selectedVariant.value.platform)
})

const acceptTypes = computed(() => {
  const platform = selectedVariant.value?.platform || ''
  const map: Record<string, string> = {
    ios: '.ipa',
    android: '.apk',
    macos: '.dmg,.pkg,.zip',
    windows: '.exe,.msi,.zip',
    linux: '.appimage,.deb,.rpm,.zip,.tar.gz'
  }
  return map[platform] || '*'
})

watch(selectedVariant, (variant) => {
  if (!variant) return
  if (['ios', 'android'].includes(variant.platform)) {
    uploadMode.value = 'smart'
  } else {
    uploadMode.value = 'manual'
  }
})

async function fetchUploadConfig() {
  try {
    const res = await adminFetch('api/upload-config', {
      token: token.value
    })
    if (res.ok) {
      const data = await res.json()
      if (data.ok && data.data?.upload_domain) {
        uploadBaseURL.value = data.data.upload_domain
      }
    }
  } catch {}
}

async function doUpload() {
  const file = fileInput.value?.files?.[0]
  if (!selectedVariant.value) {
    toast.error(t.value.upload.selectVariantFirst)
    return
  }
  if (!file) {
    toast.error(t.value.upload.pleaseSelectFile)
    return
  }
  if (!form.version) {
    toast.error(t.value.upload.pleaseEnterVersion)
    return
  }

  const fd = new FormData()
  fd.set('variant_id', selectedVariant.value.id)
  fd.set('version', form.version)
  if (form.build) fd.set('build', form.build)
  if (form.channel) fd.set('channel', form.channel)
  if (form.min_os) fd.set('min_os', form.min_os)
  if (form.changelog) fd.set('changelog', form.changelog)
  if (form.force_update) fd.set('force_update', 'true')
  fd.set('app_file', file)

  uploading.value = true
  try {
    const res = await fetch(uploadBaseURL.value + '/upload', {
      method: 'POST',
      headers: { 'X-Auth-Token': token.value },
      body: fd
    })
    if (handleAuthError(res)) return
    const j = await res.json()
    response.value = j
    if (!j?.ok) {
      toast.error(t.value.upload.uploadFailed + ': ' + (j?.error?.message || res.status))
    } else {
      toast.success(t.value.upload.uploadSuccess)
      fetchProducts()
    }
  } catch (e) {
    toast.error(t.value.upload.uploadFailed + ': ' + e)
  } finally {
    uploading.value = false
  }
}

onMounted(() => {
  fetchUploadConfig()
})
</script>

<style scoped>
.panel + .panel,
.panel + .mode-selector {
  margin-top: var(--space-3);
}

.mode-selector + .panel {
  margin-top: var(--space-3);
}

.panel.empty-state p {
  margin: 0;
}
</style>
