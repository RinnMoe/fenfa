<template>
  <Teleport to="body">
    <transition name="dialog">
      <div v-if="show" class="dialog-overlay" @click="$emit('close')">
        <div class="dialog-container distribution-modal" @click.stop>
          <div class="dialog-header">
            <h3 class="dialog-title">{{ t.apps.viewDistribution }}</h3>
            <button class="close-btn" aria-label="Close" @click="$emit('close')">×</button>
          </div>
          <div class="dialog-body">
            <div v-if="loading" class="loading-state">{{ t.common.loading }}</div>
            <div v-else>
              <div class="product-line">
                <strong>{{ product?.name || product?.id }}</strong>
              </div>
              <div class="desc">{{ t.apps.distributionDesc }}</div>
              <div class="link-list">
                <div v-for="(d, idx) in domains" :key="idx" class="link-row">
                  <span class="domain-label">{{ idx === 0 ? t.apps.primaryDomain : t.apps.secondaryDomain }}</span>
                  <span class="url-text">{{ buildUrl(d) }}</span>
                  <div class="row-actions">
                    <button class="btn-sm" @click="copy(buildUrl(d))">
                      <Icon name="copy" :size="13" />
                      {{ copiedUrl === buildUrl(d) ? t.apps.copied : t.apps.copyUrl }}
                    </button>
                    <button class="btn-sm" @click="showQR(buildUrl(d))">
                      <Icon name="qr-code" :size="13" />
                      {{ t.apps.qrCode }}
                    </button>
                  </div>
                </div>
                <div v-if="qrUrl" class="qr-area">
                  <canvas ref="qrCanvas" width="140" height="140"></canvas>
                  <div class="qr-title">{{ t.apps.qrFor }}</div>
                  <div class="qr-url">{{ qrUrl }}</div>
                  <div class="qr-hint">{{ t.apps.qrHint }}</div>
                </div>
              </div>
            </div>
          </div>
          <div class="dialog-footer">
            <button class="btn-secondary" @click="$emit('close')">{{ t.common.close }}</button>
          </div>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, nextTick, computed } from 'vue'
import { useI18n } from '../i18n'
import QRCode from 'qrcode'
import Icon from './Icon.vue'
import { adminURL } from '../api'

interface ProductLite { id: string; name?: string; slug?: string }

const props = defineProps<{ show: boolean; product: ProductLite | null; token: string }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()

const settings = ref<{ primary_domain: string; secondary_domains: string[] } | null>(null)
const loading = ref(false)
const qrUrl = ref('')

const copiedUrl = ref('')

const domains = computed(() => {
  if (!settings.value) return [] as string[]
  const arr = [settings.value.primary_domain || '']
  const secs = Array.isArray(settings.value.secondary_domains) ? settings.value.secondary_domains : []
  return arr.concat(secs).filter(Boolean)
})

function buildUrl(domain: string) {
  const base = domain.replace(/\/$/, '')
  const key = props.product?.slug || props.product?.id || ''
  return `${base}/products/${key}`
}

const qrCanvas = ref<HTMLCanvasElement | null>(null)

async function fetchSettings() {
  loading.value = true
  try {
    const res = await fetch(adminURL('api/settings'), { headers: { 'X-Auth-Token': props.token } })
    const j = await res.json()
    if (res.ok && j.ok) {
      settings.value = j.data
    }
  } catch (e) {
    // ignore
  } finally {
    loading.value = false
  }
}

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    copiedUrl.value = text
    setTimeout(() => { if (copiedUrl.value === text) copiedUrl.value = '' }, 1500)
  } catch {}
}

async function showQR(url: string) {
  qrUrl.value = url
  await nextTick()
  if (qrCanvas.value) {
    QRCode.toCanvas(qrCanvas.value, url, { width: 180, margin: 1 })
  }
}

watch(() => props.show, (v) => {
  if (v) {
    if (!settings.value) fetchSettings()
    qrUrl.value = ''
  }
})

onMounted(() => { if (props.show) fetchSettings() })
</script>

<style scoped>
.distribution-modal {
  max-width: 680px;
}

.product-line {
  margin-bottom: 4px;
  color: var(--text-primary);
  font-size: 14px;
}

.desc {
  margin-bottom: var(--space-3);
  color: var(--text-muted);
  font-size: 12px;
}

.link-list {
  display: flex;
  flex-direction: column;
}

.link-row {
  display: grid;
  grid-template-columns: 110px minmax(0, 1fr) auto;
  gap: var(--space-3);
  align-items: center;
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--border-muted);
}

.domain-label {
  color: var(--text-muted);
  font-size: 12px;
}

.url-text {
  overflow: auto;
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: 12px;
  white-space: nowrap;
}

.row-actions {
  display: flex;
  gap: var(--space-2);
}

.qr-area {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--border);
}

.qr-title {
  color: var(--text-secondary);
  font-size: 12px;
}

.qr-url {
  overflow: hidden;
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.qr-hint {
  color: var(--text-muted);
  font-size: 12px;
}

@media (max-width: 640px) {
  .link-row {
    grid-template-columns: 1fr;
    gap: 6px;
    padding: var(--space-3) 0;
  }

  .url-text {
    white-space: normal;
    word-break: break-all;
  }
}
</style>
