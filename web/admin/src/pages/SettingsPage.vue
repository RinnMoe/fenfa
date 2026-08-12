<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t.settings.title }}</h1>
        <p class="page-subtitle">{{ t.settings.subtitle }}</p>
      </div>
    </div>

    <div v-if="loading" class="loading-state">
      {{ t.common.loading }}
    </div>

    <form v-else class="settings-form" @submit.prevent="handleSave">
      <section class="panel settings-section">
        <div class="settings-head">
          <h2 class="section-title">{{ t.settings.generalTitle }}</h2>
        </div>

        <div class="settings-body">
          <div class="settings-row">
            <div class="settings-copy">
              <label class="label required" for="primary-domain">{{ t.settings.primaryDomain }}</label>
              <p class="help-text">{{ t.settings.primaryDomainHelp }}</p>
            </div>
            <div class="settings-field">
              <input
                id="primary-domain"
                v-model="form.primaryDomain"
                type="text"
                class="input"
                :placeholder="t.settings.primaryDomainPlaceholder"
                required
              >
              <p v-if="errors.primaryDomain" class="error-text">{{ errors.primaryDomain }}</p>
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="secondary-domains">{{ t.settings.secondaryDomains }}</label>
              <p class="help-text">{{ t.settings.secondaryDomainsHelp }}</p>
            </div>
            <div class="settings-field">
              <textarea
                id="secondary-domains"
                v-model="secondaryDomainsText"
                class="textarea"
                :placeholder="t.settings.secondaryDomainsPlaceholder"
                rows="4"
              ></textarea>
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label required" for="organization">{{ t.settings.organization }}</label>
              <p class="help-text">{{ t.settings.organizationHelp }}</p>
            </div>
            <div class="settings-field">
              <input
                id="organization"
                v-model="form.organization"
                type="text"
                class="input"
                :placeholder="t.settings.organizationPlaceholder"
                required
              >
              <p v-if="errors.organization" class="error-text">{{ errors.organization }}</p>
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label required" for="admin-path">{{ t.settings.adminPath }}</label>
              <p class="help-text">{{ t.settings.adminPathHelp }}</p>
            </div>
            <div class="settings-field">
              <div class="input-prefix">
                <span>/</span>
                <input
                  id="admin-path"
                  v-model="form.adminPath"
                  type="text"
                  class="input"
                  :placeholder="t.settings.adminPathPlaceholder"
                  required
                >
              </div>
              <p v-if="errors.adminPath" class="error-text">{{ errors.adminPath }}</p>
            </div>
          </div>
        </div>
      </section>

      <section class="panel settings-section">
        <div class="settings-head">
          <h2 class="section-title">{{ t.email.title }}</h2>
          <p class="settings-desc">{{ t.email.subtitle }}</p>
        </div>

        <div class="settings-body">
          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="email-enabled">{{ t.email.enabled }}</label>
              <p class="help-text">{{ t.email.enabledHelp }}</p>
            </div>
            <div class="settings-field checkbox-group">
              <label class="checkbox-label">
                <input id="email-enabled" v-model="emailForm.enabled" type="checkbox">
                <span>{{ emailForm.enabled ? t.email.enabledOn : t.email.enabledOff }}</span>
              </label>
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="smtp-host">{{ t.email.smtpHost }}</label>
              <p class="help-text">{{ t.email.smtpHostHelp }}</p>
            </div>
            <div class="settings-field">
              <input id="smtp-host" v-model="emailForm.smtpHost" type="text" class="input" :placeholder="t.email.smtpHostPlaceholder">
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="smtp-port">{{ t.email.smtpPort }}</label>
              <p class="help-text">{{ t.email.smtpPortHelp }}</p>
            </div>
            <div class="settings-field">
              <input id="smtp-port" v-model.number="emailForm.smtpPort" type="number" min="1" max="65535" class="input" placeholder="587">
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="smtp-encryption">{{ t.email.encryption }}</label>
              <p class="help-text">{{ t.email.encryptionHelp }}</p>
            </div>
            <div class="settings-field">
              <select id="smtp-encryption" v-model="emailForm.encryption" class="select">
                <option value="starttls">{{ t.email.starttls }}</option>
                <option value="ssl">{{ t.email.ssl }}</option>
                <option value="none">{{ t.email.none }}</option>
              </select>
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="smtp-username">{{ t.email.username }}</label>
              <p class="help-text">{{ t.email.usernameHelp }}</p>
            </div>
            <div class="settings-field">
              <input id="smtp-username" v-model="emailForm.username" type="text" class="input" :placeholder="t.email.usernamePlaceholder">
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="smtp-password">{{ t.email.password }}</label>
              <p class="help-text">{{ t.email.passwordHelp }}</p>
            </div>
            <div class="settings-field">
              <input id="smtp-password" v-model="emailForm.password" type="password" class="input" :placeholder="emailForm.passwordSet ? t.email.passwordConfigured : t.email.passwordPlaceholder" autocomplete="new-password">
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="smtp-from">{{ t.email.from }}</label>
              <p class="help-text">{{ t.email.fromHelp }}</p>
            </div>
            <div class="settings-field">
              <input id="smtp-from" v-model="emailForm.from" type="email" class="input" :placeholder="t.email.fromPlaceholder">
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="smtp-to">{{ t.email.to }}</label>
              <p class="help-text">{{ t.email.toHelp }}</p>
            </div>
            <div class="settings-field">
              <input id="smtp-to" v-model="emailForm.to" type="email" class="input" :placeholder="t.email.toPlaceholder">
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <span class="label">{{ t.email.testTitle }}</span>
              <p class="help-text">{{ t.email.testHelp }}</p>
            </div>
            <div class="settings-field">
              <button type="button" class="btn-test" :disabled="testingEmail" @click="testEmailConnection">
                {{ testingEmail ? t.email.testing : t.email.testButton }}
              </button>
            </div>
          </div>
        </div>
      </section>

      <section class="panel settings-section">
        <div class="settings-head">
          <h2 class="section-title">{{ t.storage.title }}</h2>
          <p class="settings-desc">{{ t.storage.subtitle }}</p>
        </div>

        <div class="settings-body">
          <div class="settings-row">
            <div class="settings-copy">
              <label class="label">{{ t.storage.storageType }}</label>
            </div>
            <div class="settings-field">
              <div class="radio-group">
                <label class="radio-label" :class="{ active: storageForm.storageType === 'local' }">
                  <input type="radio" v-model="storageForm.storageType" value="local">
                  <span class="radio-title">{{ t.storage.local }}</span>
                  <span class="radio-desc">{{ t.storage.localDesc }}</span>
                </label>
                <label class="radio-label" :class="{ active: storageForm.storageType === 's3' }">
                  <input type="radio" v-model="storageForm.storageType" value="s3">
                  <span class="radio-title">{{ t.storage.s3 }}</span>
                  <span class="radio-desc">{{ t.storage.s3Desc }}</span>
                </label>
              </div>
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="upload-domain">{{ t.storage.uploadDomain }}</label>
              <p class="help-text">{{ t.storage.uploadDomainHelp }}</p>
            </div>
            <div class="settings-field">
              <input
                id="upload-domain"
                v-model="storageForm.uploadDomain"
                type="text"
                class="input"
                :placeholder="t.storage.uploadDomainPlaceholder"
              >
            </div>
          </div>

          <template v-if="storageForm.storageType === 's3'">
            <div class="settings-row">
              <div class="settings-copy">
                <label class="label" for="s3-endpoint">{{ t.storage.s3Endpoint }}</label>
                <p class="help-text">{{ t.storage.s3EndpointHelp }}</p>
              </div>
              <div class="settings-field">
                <input id="s3-endpoint" v-model="storageForm.s3Endpoint" type="text" class="input" :placeholder="t.storage.s3EndpointPlaceholder">
              </div>
            </div>

            <div class="settings-row">
              <div class="settings-copy">
                <label class="label" for="s3-bucket">{{ t.storage.s3Bucket }}</label>
                <p class="help-text">{{ t.storage.s3BucketHelp }}</p>
              </div>
              <div class="settings-field">
                <input id="s3-bucket" v-model="storageForm.s3Bucket" type="text" class="input" :placeholder="t.storage.s3BucketPlaceholder">
              </div>
            </div>

            <div class="settings-row">
              <div class="settings-copy">
                <label class="label" for="s3-access-key">{{ t.storage.s3AccessKey }}</label>
              </div>
              <div class="settings-field">
                <input id="s3-access-key" v-model="storageForm.s3AccessKey" type="text" class="input" :placeholder="t.storage.s3AccessKeyPlaceholder">
              </div>
            </div>

            <div class="settings-row">
              <div class="settings-copy">
                <label class="label" for="s3-secret-key">{{ t.storage.s3SecretKey }}</label>
              </div>
              <div class="settings-field">
                <input id="s3-secret-key" v-model="storageForm.s3SecretKey" type="password" class="input" :placeholder="storageForm.s3SecretKeySet ? t.storage.s3SecretKeyConfigured : t.storage.s3SecretKeyPlaceholder">
              </div>
            </div>

            <div class="settings-row">
              <div class="settings-copy">
                <label class="label" for="s3-public-url">{{ t.storage.s3PublicURL }}</label>
                <p class="help-text">{{ t.storage.s3PublicURLHelp }}</p>
              </div>
              <div class="settings-field">
                <input id="s3-public-url" v-model="storageForm.s3PublicURL" type="text" class="input" :placeholder="t.storage.s3PublicURLPlaceholder">
              </div>
            </div>
          </template>
        </div>
      </section>

      <section class="panel settings-section">
        <div class="settings-head">
          <h2 class="section-title">{{ t.apple.title }}</h2>
          <p class="settings-desc">{{ t.apple.subtitle }}</p>
        </div>

        <div class="settings-body">
          <div class="apple-status" :class="appleStatus.connected ? 'connected' : (appleStatus.configured ? 'configured' : 'not-configured')">
            <Icon v-if="appleStatus.connected" name="check-circle" :size="16" />
            <Icon v-else-if="appleStatus.configured" name="alert-triangle" :size="16" />
            <Icon v-else name="x-circle" :size="16" />
            <span class="status-text">{{ appleStatus.message }}</span>
            <button type="button" @click="testAppleConnection" :disabled="testingConnection" class="btn-sm btn-test">
              {{ testingConnection ? t.apple.testing : t.apple.testConnection }}
            </button>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="apple-key-id">{{ t.apple.keyId }}</label>
              <p class="help-text">{{ t.apple.keyIdHelp }}</p>
            </div>
            <div class="settings-field">
              <input id="apple-key-id" v-model="appleForm.keyId" type="text" class="input" :placeholder="t.apple.keyIdPlaceholder">
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="apple-issuer-id">{{ t.apple.issuerId }}</label>
              <p class="help-text">{{ t.apple.issuerIdHelp }}</p>
            </div>
            <div class="settings-field">
              <input id="apple-issuer-id" v-model="appleForm.issuerId" type="text" class="input" :placeholder="t.apple.issuerIdPlaceholder">
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="apple-team-id">{{ t.apple.teamId }}</label>
              <p class="help-text">{{ t.apple.teamIdHelp }}</p>
            </div>
            <div class="settings-field">
              <input id="apple-team-id" v-model="appleForm.teamId" type="text" class="input" :placeholder="t.apple.teamIdPlaceholder">
            </div>
          </div>

          <div class="settings-row">
            <div class="settings-copy">
              <label class="label" for="apple-private-key">{{ t.apple.privateKey }}</label>
              <p class="help-text">{{ t.apple.privateKeyHelp }}</p>
            </div>
            <div class="settings-field">
              <div class="file-input-wrapper">
                <input id="apple-private-key" type="file" accept=".p8,.pem" @change="handlePrivateKeyFile" class="file-input">
                <span v-if="appleForm.privateKeySet" class="file-status configured">{{ t.apple.privateKeyConfigured }}</span>
                <span v-else-if="appleForm.privateKey" class="file-status selected">{{ t.apple.privateKeySelected }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <div class="save-bar">
        <div v-if="lastUpdated" class="info-text">
          {{ t.settings.lastUpdated }}: {{ lastUpdated }}
        </div>
        <button type="submit" :disabled="saving" class="btn-primary">
          {{ saving ? t.settings.saving : t.settings.saveButton }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from '../i18n'
import { useAuth } from '../composables/useAuth'
import { useToast } from '../composables/useToast'
import Icon from '../components/Icon.vue'
import { adminFetch, setAdminPath } from '../api'

const { token, handleAuthError: onAuthError } = useAuth()

const { t } = useI18n()
const toast = useToast()

const loading = ref(true)
const saving = ref(false)
const lastUpdated = ref('')

const form = reactive({
  primaryDomain: '',
  organization: '',
  adminPath: 'admin'
})

const secondaryDomainsText = ref('')

const errors = reactive({
  primaryDomain: '',
  organization: '',
  adminPath: ''
})

const storageForm = reactive({
  storageType: 'local',
  uploadDomain: '',
  s3Endpoint: '',
  s3Bucket: '',
  s3AccessKey: '',
  s3SecretKey: '',
  s3SecretKeySet: false,
  s3PublicURL: ''
})

const emailForm = reactive({
  enabled: false,
  smtpHost: '',
  smtpPort: 587,
  encryption: 'starttls',
  username: '',
  password: '',
  passwordSet: false,
  from: '',
  to: ''
})

const appleForm = reactive({
  keyId: '',
  issuerId: '',
  teamId: '',
  privateKey: '',
  privateKeySet: false
})

const appleStatus = reactive({
  configured: false,
  connected: false,
  message: ''
})

const testingConnection = ref(false)
const testingEmail = ref(false)

function isValidDomain(domain: string): boolean {
  if (!domain) return false
  return domain.startsWith('http://') || domain.startsWith('https://')
}

function applyEmailSettings(data: Record<string, any>) {
  emailForm.enabled = Boolean(data.email_enabled)
  emailForm.smtpHost = data.smtp_host || ''
  emailForm.smtpPort = Number(data.smtp_port) || 587
  emailForm.encryption = data.smtp_encryption || 'starttls'
  emailForm.username = data.smtp_username || ''
  emailForm.password = ''
  emailForm.passwordSet = Boolean(data.smtp_password_set)
  emailForm.from = data.smtp_from || ''
  emailForm.to = data.smtp_to || ''
}

async function loadSettings() {
  loading.value = true
  try {
    const res = await adminFetch('api/settings', {
      token: token.value
    })
    if (onAuthError(res)) return
    if (!res.ok) {
      toast.error(t.value.settings.loadFailed)
      return
    }

    const data = await res.json()
    if (data.ok && data.data) {
      form.primaryDomain = data.data.primary_domain || ''
      form.organization = data.data.organization || ''
      form.adminPath = data.data.admin_path || 'admin'

      if (data.data.secondary_domains && Array.isArray(data.data.secondary_domains)) {
        secondaryDomainsText.value = data.data.secondary_domains.join('\n')
      }

      if (data.data.updated_at) {
        lastUpdated.value = new Date(data.data.updated_at).toLocaleString()
      }

      storageForm.storageType = data.data.storage_type || 'local'
      storageForm.uploadDomain = data.data.upload_domain || ''
      storageForm.s3Endpoint = data.data.s3_endpoint || ''
      storageForm.s3Bucket = data.data.s3_bucket || ''
      storageForm.s3PublicURL = data.data.s3_public_url || ''
      storageForm.s3SecretKeySet = data.data.s3_configured || false

      applyEmailSettings(data.data)

      if (data.data.apple_configured !== undefined) {
        appleStatus.configured = data.data.apple_configured
        appleForm.keyId = data.data.apple_key_id || ''
        appleForm.teamId = data.data.apple_team_id || ''
        appleForm.privateKeySet = data.data.apple_configured
      }
    }
  } catch (e) {
    toast.error(t.value.settings.loadFailed + ': ' + e)
  } finally {
    loading.value = false
  }

  await checkAppleStatus()
}

function validateForm(): boolean {
  errors.primaryDomain = ''
  errors.organization = ''
  errors.adminPath = ''
  let isValid = true

  if (!form.primaryDomain) {
    errors.primaryDomain = t.value.settings.requiredField
    isValid = false
  } else if (!isValidDomain(form.primaryDomain)) {
    errors.primaryDomain = t.value.settings.invalidDomain
    isValid = false
  }

  if (!form.organization) {
    errors.organization = t.value.settings.requiredField
    isValid = false
  }

  if (!/^[A-Za-z0-9_-]{1,64}$/.test(form.adminPath)) {
    errors.adminPath = t.value.settings.invalidAdminPath
    isValid = false
  }

  return isValid
}

async function handleSave() {
  if (!validateForm()) return

  saving.value = true
  try {
    const secondaryDomains = secondaryDomainsText.value
      .split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0)

    const payload: Record<string, any> = {
      primary_domain: form.primaryDomain,
      secondary_domains: secondaryDomains,
      organization: form.organization,
      admin_path: form.adminPath,
      storage_type: storageForm.storageType,
      upload_domain: storageForm.uploadDomain,
      email_enabled: emailForm.enabled,
      smtp_host: emailForm.smtpHost,
      smtp_port: emailForm.smtpPort,
      smtp_encryption: emailForm.encryption,
      smtp_username: emailForm.username,
      smtp_from: emailForm.from,
      smtp_to: emailForm.to,
    }

    if (emailForm.password) payload.smtp_password = emailForm.password

    if (storageForm.storageType === 's3') {
      payload.s3_endpoint = storageForm.s3Endpoint
      payload.s3_bucket = storageForm.s3Bucket
      payload.s3_public_url = storageForm.s3PublicURL
      if (storageForm.s3AccessKey) payload.s3_access_key = storageForm.s3AccessKey
      if (storageForm.s3SecretKey) payload.s3_secret_key = storageForm.s3SecretKey
    }

    if (appleForm.keyId) payload.apple_key_id = appleForm.keyId
    if (appleForm.issuerId) payload.apple_issuer_id = appleForm.issuerId
    if (appleForm.teamId) payload.apple_team_id = appleForm.teamId
    if (appleForm.privateKey) payload.apple_private_key = appleForm.privateKey

    const res = await adminFetch('api/settings', {
      method: 'PUT',
      token: token.value,
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(payload)
    })

    if (onAuthError(res)) return

    const data = await res.json()
    if (!res.ok || !data.ok) {
      toast.error(data.error?.message || t.value.settings.saveFailed)
      return
    }

    const previousAdminPath = form.adminPath
    toast.success(t.value.settings.saveSuccess)

    if (data.data) {
      if (data.data.updated_at) {
        lastUpdated.value = new Date(data.data.updated_at).toLocaleString()
      }
      form.adminPath = data.data.admin_path || form.adminPath
      storageForm.storageType = data.data.storage_type || 'local'
      storageForm.uploadDomain = data.data.upload_domain || ''
      storageForm.s3Endpoint = data.data.s3_endpoint || ''
      storageForm.s3Bucket = data.data.s3_bucket || ''
      storageForm.s3PublicURL = data.data.s3_public_url || ''
      storageForm.s3SecretKeySet = data.data.s3_configured || false
      storageForm.s3SecretKey = ''
      storageForm.s3AccessKey = ''

      applyEmailSettings(data.data)

      if (data.data.apple_configured !== undefined) {
        appleStatus.configured = data.data.apple_configured
        appleForm.keyId = data.data.apple_key_id || ''
        appleForm.teamId = data.data.apple_team_id || ''
        appleForm.privateKeySet = data.data.apple_configured
        appleForm.privateKey = ''
      }
    }

    if (form.adminPath !== previousAdminPath) {
      setAdminPath(form.adminPath)
      window.location.assign('/' + form.adminPath)
      return
    }

    await checkAppleStatus()
  } catch (e) {
    toast.error(t.value.settings.saveFailed + ': ' + e)
  } finally {
    saving.value = false
  }
}

async function testEmailConnection() {
  testingEmail.value = true
  try {
    const res = await adminFetch('api/settings/email/test', {
      method: 'POST',
      token: token.value,
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        smtp_host: emailForm.smtpHost,
        smtp_port: emailForm.smtpPort,
        smtp_encryption: emailForm.encryption,
        smtp_username: emailForm.username,
        smtp_password: emailForm.password,
        smtp_from: emailForm.from,
        smtp_to: emailForm.to
      })
    })
    if (onAuthError(res)) return
    const data = await res.json()
    if (!res.ok || !data.ok) {
      toast.error(data.error?.message || t.value.email.testFailed)
      return
    }
    toast.success(t.value.email.testSuccess)
  } catch (e) {
    toast.error(t.value.email.testFailed + ': ' + e)
  } finally {
    testingEmail.value = false
  }
}

async function checkAppleStatus() {
  try {
    const res = await adminFetch('api/apple/status', {
      token: token.value
    })
    if (!res.ok) return
    const data = await res.json()
    if (data.ok && data.data) {
      appleStatus.configured = data.data.configured
      appleStatus.connected = data.data.connected
      appleStatus.message = data.data.message
    }
  } catch {
    appleStatus.message = 'Failed to check status'
  }
}

function handlePrivateKeyFile(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = (e) => {
    appleForm.privateKey = e.target?.result as string
  }
  reader.readAsText(file)
}

async function testAppleConnection() {
  testingConnection.value = true
  try {
    const res = await adminFetch('api/apple/status', {
      token: token.value
    })
    if (onAuthError(res)) return
    const data = await res.json()
    if (data.ok && data.data) {
      appleStatus.configured = data.data.configured
      appleStatus.connected = data.data.connected
      appleStatus.message = data.data.message
      if (data.data.connected) {
        toast.success(t.value.apple.connectionSuccess)
      } else {
        toast.error(data.data.message || t.value.apple.connectionFailed)
      }
    }
  } catch (e) {
    toast.error(t.value.apple.connectionFailed + ': ' + e)
  } finally {
    testingConnection.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.settings-section {
  padding: 0;
  overflow: hidden;
}

.settings-head {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--space-4);
  border-bottom: 1px solid var(--border);
}

.settings-desc {
  margin: 0;
  color: var(--text-muted);
  font-size: 12px;
}

.settings-body {
  display: flex;
  flex-direction: column;
}

.settings-row {
  display: grid;
  grid-template-columns: minmax(220px, 320px) minmax(0, 1fr);
  gap: var(--space-4);
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--border-muted);
}

.settings-row:last-child {
  border-bottom: none;
}

.settings-copy .label {
  display: block;
  margin-bottom: 4px;
}

.settings-field {
  min-width: 0;
}

.input-prefix {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-muted);
}

.input-prefix .input {
  flex: 1;
}

.apple-status {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin: var(--space-4);
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  background: var(--surface-subtle);
  color: var(--text-secondary);
}

.apple-status.connected {
  border-color: var(--success);
  background: var(--success-subtle);
  color: var(--success);
}

.apple-status.configured {
  border-color: var(--warning);
  background: var(--warning-subtle);
  color: var(--warning);
}

.status-text {
  flex: 1;
  font-size: 13px;
}

.file-input-wrapper {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

@media (max-width: 720px) {
  .settings-row {
    grid-template-columns: 1fr;
    gap: var(--space-2);
  }

  .apple-status {
    align-items: flex-start;
    flex-wrap: wrap;
  }
}
</style>
