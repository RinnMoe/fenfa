<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-top">
        <div class="login-brand">
          <span class="brand-mark">F</span>
          <span>Fenfa</span>
        </div>
        <LanguageSwitcher />
      </div>

      <h1 class="login-title">{{ t.login.title }}</h1>
      <p class="login-subtitle">{{ t.login.subtitle }}</p>

      <p v-if="setupRequired" class="setup-hint">{{ t.login.setupHint }}</p>

      <form class="login-form" @submit.prevent="handleLogin">
        <div class="form-group">
          <label class="label" for="admin-username">{{ t.login.usernameLabel }}</label>
          <input
            id="admin-username"
            v-model="username"
            type="text"
            :placeholder="t.login.usernamePlaceholder"
            autocomplete="username"
            class="input"
          >
        </div>

        <div class="form-group">
          <label class="label" for="admin-password">{{ t.login.passwordLabel }}</label>
          <input
            id="admin-password"
            v-model="password"
            type="password"
            :placeholder="t.login.passwordPlaceholder"
            :autocomplete="setupRequired ? 'new-password' : 'current-password'"
            class="input"
          >
        </div>

        <div v-if="setupRequired" class="form-group">
          <label class="label" for="admin-password-confirm">{{ t.login.confirmPasswordLabel }}</label>
          <input
            id="admin-password-confirm"
            v-model="passwordConfirmation"
            type="password"
            :placeholder="t.login.confirmPasswordPlaceholder"
            autocomplete="new-password"
            class="input"
          >
        </div>

        <div v-if="errorMessage" class="error-message" role="alert">
          <Icon name="alert" :size="14" />
          {{ errorMessage }}
        </div>

        <button type="submit" class="btn-primary btn-full" :disabled="isLoggingIn || checkingStatus">
          {{ isLoggingIn ? t.login.loggingIn : (setupRequired ? t.login.setupButton : t.login.loginButton) }}
        </button>
      </form>

      <p class="login-footer">{{ t.login.brand }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from '../i18n'
import { useAuth } from '../composables/useAuth'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'
import Icon from '../components/Icon.vue'
import { adminFetch } from '../api'

const emit = defineEmits<{ login: [uploadToken: string] }>()

const { t } = useI18n()
const { handleLogin: markLoggedIn } = useAuth()
const username = ref('')
const password = ref('')
const passwordConfirmation = ref('')
const setupRequired = ref(false)
const checkingStatus = ref(true)
const isLoggingIn = ref(false)
const errorMessage = ref('')

async function loadAuthStatus() {
  checkingStatus.value = true
  try {
    const res = await adminFetch('api/auth/status')
    if (res.ok) {
      const data = await res.json()
      setupRequired.value = Boolean(data.ok && data.data?.setup_required)
    }
  } catch {
    setupRequired.value = false
  } finally {
    checkingStatus.value = false
  }
}

async function handleLogin() {
  errorMessage.value = ''
  if (!username.value.trim() || !password.value) {
    errorMessage.value = t.value.login.pleaseEnterCredentials
    return
  }
  if (setupRequired.value && passwordConfirmation.value !== password.value) {
    errorMessage.value = t.value.login.passwordsDoNotMatch
    return
  }

  isLoggingIn.value = true
  try {
    const res = await adminFetch(setupRequired.value ? 'api/auth/setup' : 'api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: username.value,
        password: password.value,
        ...(setupRequired.value ? { password_confirmation: passwordConfirmation.value } : {})
      })
    })

    if (res.status === 401) {
      errorMessage.value = t.value.login.invalidCredentials
      return
    }

    if (res.status === 409) {
      await loadAuthStatus()
      errorMessage.value = t.value.login.setupComplete
      return
    }

    if (!res.ok) {
      const data = await res.json().catch(() => null)
      errorMessage.value = data?.error?.message || t.value.login.loginFailed
      return
    }

    const data = await res.json()
    if (!data.ok) {
      errorMessage.value = data.error?.message || t.value.login.loginFailed
      return
    }
    const uploadToken = data.data?.upload_token || ''
    markLoggedIn(uploadToken)
    emit('login', uploadToken)
  } catch {
    errorMessage.value = t.value.login.loginFailed
  } finally {
    isLoggingIn.value = false
  }
}

onMounted(loadAuthStatus)
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-4);
  background: var(--canvas);
}

.login-card {
  width: 360px;
  max-width: 100%;
  padding: var(--space-6);
  background: var(--surface);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-panel);
  box-shadow: var(--shadow-popover);
}

.login-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-6);
}

.login-brand {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--surface-subtle);
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 700;
}

.login-title {
  margin: 0 0 4px;
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 600;
}

.login-subtitle {
  margin: 0 0 var(--space-5);
  color: var(--text-muted);
  font-size: 13px;
}

.setup-hint {
  margin: 0 0 var(--space-4);
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  background: var(--surface-subtle);
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.login-form .btn-full {
  height: var(--control-height);
}

.login-footer {
  margin: var(--space-5) 0 0;
  color: var(--text-faint);
  font-size: 11px;
  text-align: center;
}
</style>
