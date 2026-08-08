<template>
  <n-modal
    :show="show && !!release"
    preset="card"
    :style="{ width: '560px', maxWidth: 'calc(100vw - 32px)' }"
    :mask-closable="!saving"
    :close-on-esc="!saving"
    @update:show="handleShow"
  >
    <template #header>
      <h2 class="modal-title">{{ t.editRelease.title }}</h2>
    </template>

    <div class="modal-body">
      <div class="form-grid">
        <div class="form-group">
          <label class="label">{{ t.editRelease.version }}</label>
          <input v-model="form.version" :placeholder="t.editRelease.versionPlaceholder" class="input">
        </div>
        <div class="form-group">
          <label class="label">{{ t.editRelease.build }}</label>
          <input v-model="form.build" :placeholder="t.editRelease.buildPlaceholder" class="input">
        </div>
        <div class="form-group">
          <label class="label">{{ t.editRelease.channel }}</label>
          <input v-model="form.channel" :placeholder="t.editRelease.channelPlaceholder" class="input">
        </div>
        <div class="form-group">
          <label class="label">{{ t.editRelease.minOS }}</label>
          <input v-model="form.min_os" :placeholder="t.editRelease.minOSPlaceholder" class="input">
        </div>
      </div>

      <div class="form-group" style="margin-top: var(--space-3)">
        <label class="label">{{ t.editRelease.changelog }}</label>
        <textarea v-model="form.changelog" :placeholder="t.editRelease.changelogPlaceholder" class="textarea" rows="4"></textarea>
      </div>

      <div class="form-group" style="margin-top: var(--space-3)">
        <label class="checkbox-label">
          <input v-model="form.force_update" type="checkbox">
          {{ t.editRelease.forceUpdate }}
        </label>
      </div>
    </div>

    <template #footer>
      <button @click="handleClose" class="btn-secondary">{{ t.common.cancel }}</button>
      <button @click="handleSave" :disabled="saving || !canSave" class="btn-primary">
        {{ saving ? t.editRelease.saving : t.editRelease.saveButton }}
      </button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { NModal } from 'naive-ui'
import { useI18n } from '../i18n'
import { useAuth } from '../composables/useAuth'
import { useToast } from '../composables/useToast'
import { adminFetch } from '../api'

const props = defineProps<{
  show: boolean
  release: any | null
}>()

const emit = defineEmits<{
  close: []
  success: []
}>()

const { t } = useI18n()
const { token, handleAuthError } = useAuth()
const toast = useToast()
const saving = ref(false)

const form = reactive({
  version: '',
  build: '',
  channel: '',
  min_os: '',
  changelog: '',
  force_update: false
})

const canSave = computed(() => form.version.trim().length > 0)

watch(() => props.show, (show) => {
  if (show && props.release) {
    form.version = props.release.version || ''
    form.build = props.release.build != null ? String(props.release.build) : ''
    form.channel = props.release.channel || ''
    form.min_os = props.release.min_os || ''
    form.changelog = props.release.changelog || ''
    form.force_update = !!props.release.force_update
  }
})

function handleClose() {
  if (saving.value) return
  emit('close')
}

function handleShow(next: boolean) {
  if (!next) handleClose()
}

async function handleSave() {
  if (!props.release || !canSave.value) return
  const buildNum = form.build === '' ? undefined : Number(form.build)
  if (buildNum !== undefined && (!Number.isInteger(buildNum) || buildNum < 0)) {
    toast.error(t.value.editRelease.invalidBuild)
    return
  }
  saving.value = true
  try {
    const res = await adminFetch(`api/releases/${props.release.id}`, {
      method: 'PUT',
      token: token.value,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        version: form.version.trim(),
        build: buildNum,
        channel: form.channel,
        min_os: form.min_os,
        changelog: form.changelog,
        force_update: form.force_update
      })
    })
    if (handleAuthError(res)) return
    const payload = await res.json()
    if (!res.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.editRelease.saveFailed)
      return
    }
    toast.success(t.value.editRelease.saveSuccess)
    emit('success')
  } catch (error) {
    toast.error(t.value.editRelease.saveFailed + ': ' + error)
  } finally {
    saving.value = false
  }
}
</script>
