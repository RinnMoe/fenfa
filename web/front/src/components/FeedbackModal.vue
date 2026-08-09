<template>
  <div
    class="feedback-backdrop"
    role="dialog"
    aria-modal="true"
    :aria-labelledby="titleID"
    @click.self="close"
    @keydown.esc="close"
  >
    <section class="feedback-modal">
      <header class="feedback-header">
        <div>
          <h2 :id="titleID">{{ t.feedback.title }}</h2>
          <p>{{ t.feedback.intro }}</p>
        </div>
        <button type="button" class="feedback-close" :aria-label="t.feedback.close" @click="close">×</button>
      </header>

      <div class="feedback-context">
        <span class="feedback-context-label">{{ t.feedback.context }}</span>
        <strong>{{ product.name }}</strong>
        <span v-if="variant">{{ t.feedback.platform }}: {{ platformLabel(variant.platform) }}</span>
        <span v-if="release">{{ t.feedback.release }}: {{ formatVersion(release.version) }}</span>
      </div>

      <form v-if="!success" class="feedback-form" @submit.prevent="submit">
        <label class="feedback-field">
          <span>{{ t.feedback.category }}</span>
          <select v-model="form.category" required>
            <option value="bug">{{ t.feedback.categoryBug }}</option>
            <option value="suggestion">{{ t.feedback.categorySuggestion }}</option>
            <option value="other">{{ t.feedback.categoryOther }}</option>
          </select>
        </label>

        <label class="feedback-field">
          <span>{{ t.feedback.content }}</span>
          <textarea
            v-model="form.content"
            rows="5"
            maxlength="4000"
            required
            :placeholder="t.feedback.contentPlaceholder"
          ></textarea>
        </label>

        <label class="feedback-field">
          <span>{{ t.feedback.contact }}</span>
          <input v-model="form.contact" maxlength="255" type="text" :placeholder="t.feedback.contactPlaceholder">
        </label>

        <label class="feedback-honeypot" aria-hidden="true">
          Website
          <input v-model="form.website" tabindex="-1" autocomplete="off" type="text">
        </label>

        <p v-if="errorMessage" class="feedback-message feedback-error">{{ errorMessage }}</p>

        <div class="feedback-actions">
          <button type="button" class="feedback-secondary" :disabled="submitting" @click="close">
            {{ t.feedback.close }}
          </button>
          <button type="submit" class="feedback-primary" :disabled="submitting">
            {{ submitting ? t.feedback.submitting : t.feedback.submit }}
          </button>
        </div>
      </form>

      <div v-else class="feedback-success">
        <p>{{ t.feedback.success }}</p>
        <button type="button" class="feedback-primary" @click="close">{{ t.feedback.close }}</button>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useI18n } from '../i18n'
import { platformLabel } from '../composables/useProduct'
import type { ProductData, ReleaseItem, VariantItem } from '../types'

const props = defineProps<{
  product: ProductData['product']
  variant: VariantItem | null
  release: ReleaseItem | null
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const titleID = 'feedback-modal-title'
const submitting = ref(false)
const success = ref(false)
const errorMessage = ref('')
const form = reactive({
  category: 'bug',
  content: '',
  contact: '',
  website: ''
})

function close() {
  if (!submitting.value) emit('close')
}

function formatVersion(version: string) {
  return version.startsWith('v') ? version : `v${version}`
}

async function submit() {
  if (!props.product.id || !form.content.trim()) return

  submitting.value = true
  errorMessage.value = ''
  try {
    const response = await fetch('/api/v1/feedback', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        product_id: props.product.id,
        variant_id: props.variant?.id || '',
        release_id: props.release?.id || '',
        category: form.category,
        content: form.content,
        contact: form.contact,
        website: form.website
      })
    })
    const payload = await response.json().catch(() => null)
    if (response.status === 429) {
      errorMessage.value = t.value.feedback.rateLimited
      return
    }
    if (!response.ok || !payload?.ok) {
      errorMessage.value = t.value.feedback.error
      return
    }

    form.content = ''
    form.contact = ''
    form.website = ''
    success.value = true
  } catch {
    errorMessage.value = t.value.feedback.error
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.feedback-backdrop {
  position: fixed;
  inset: 0;
  z-index: 40;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(15, 23, 42, 0.42);
}

.feedback-modal {
  width: min(100%, 480px);
  max-height: min(700px, calc(100vh - 32px));
  overflow-y: auto;
  padding: 24px;
  border: 1px solid var(--border);
  border-radius: var(--radius-panel);
  background: var(--surface);
  box-shadow: var(--shadow-popover);
}

.feedback-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.feedback-header h2 {
  margin: 0;
  color: var(--text-primary);
  font-size: 18px;
}

.feedback-header p {
  margin: 6px 0 0;
  color: var(--text-muted);
  font-size: 13px;
}

.feedback-close {
  width: 28px;
  height: 28px;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  background: transparent;
  color: var(--text-muted);
  font-size: 20px;
  line-height: 1;
  cursor: pointer;
}

.feedback-close:hover {
  color: var(--text-primary);
  border-color: var(--border-strong);
}

.feedback-context {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 8px;
  margin-top: 20px;
  padding: 10px 12px;
  border: 1px solid var(--border-muted);
  border-radius: var(--radius-control);
  background: var(--surface-subtle);
  color: var(--text-muted);
  font-size: 12px;
}

.feedback-context strong {
  color: var(--text-primary);
}

.feedback-context-label {
  width: 100%;
  color: var(--text-faint);
}

.feedback-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-top: 18px;
}

.feedback-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 500;
}

.feedback-field select,
.feedback-field input,
.feedback-field textarea {
  width: 100%;
  padding: 9px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  outline: none;
  background: var(--canvas);
  color: var(--text-primary);
  font: inherit;
  font-weight: 400;
}

.feedback-field textarea {
  min-height: 110px;
  resize: vertical;
}

.feedback-field select:focus,
.feedback-field input:focus,
.feedback-field textarea:focus {
  border-color: var(--accent);
  background: var(--surface);
}

.feedback-honeypot {
  position: absolute;
  left: -10000px;
  width: 1px;
  height: 1px;
  overflow: hidden;
}

.feedback-message {
  margin: 0;
  padding: 9px 10px;
  border-radius: var(--radius-control);
  font-size: 13px;
}

.feedback-error {
  border: 1px solid var(--danger);
  background: var(--danger-subtle);
  color: var(--danger);
}

.feedback-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 2px;
}

.feedback-primary,
.feedback-secondary {
  min-height: 38px;
  padding: 0 16px;
  border-radius: var(--radius-control);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.feedback-primary {
  border: 1px solid var(--accent);
  background: var(--accent);
  color: #fff;
}

.feedback-primary:hover:not(:disabled) {
  border-color: var(--accent-hover);
  background: var(--accent-hover);
}

.feedback-secondary {
  border: 1px solid var(--border-strong);
  background: transparent;
  color: var(--text-secondary);
}

.feedback-secondary:hover:not(:disabled) {
  color: var(--text-primary);
}

.feedback-primary:disabled,
.feedback-secondary:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.feedback-success {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  padding: 36px 0 12px;
  color: var(--text-secondary);
  text-align: center;
}

.feedback-success p {
  margin: 0;
}
</style>
