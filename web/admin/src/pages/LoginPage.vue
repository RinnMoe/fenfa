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

      <form class="login-form" @submit.prevent="handleLogin">
        <div class="form-group">
          <label class="label" for="admin-token">{{ t.login.tokenLabel }}</label>
          <input
            id="admin-token"
            v-model="localToken"
            type="password"
            :placeholder="t.login.tokenPlaceholder"
            autocomplete="current-password"
            class="input"
          >
        </div>

        <div v-if="errorMessage" class="error-message" role="alert">
          <Icon name="alert" :size="14" />
          {{ errorMessage }}
        </div>

        <button type="submit" class="btn-primary btn-full" :disabled="isLoggingIn">
          {{ isLoggingIn ? t.login.loggingIn : t.login.loginButton }}
        </button>
      </form>

      <p class="login-footer">{{ t.login.brand }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from '../i18n'
import { useAuth } from '../composables/useAuth'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'
import Icon from '../components/Icon.vue'
import { adminURL } from '../api'

const emit = defineEmits<{
  login: [token: string]
}>()

const { t } = useI18n()
const { token } = useAuth()
const localToken = ref(token.value)
const isLoggingIn = ref(false)
const errorMessage = ref('')

async function handleLogin() {
  errorMessage.value = ''
  if (!localToken.value.trim()) {
    errorMessage.value = t.value.login.pleaseEnterToken
    return
  }

  isLoggingIn.value = true
  try {
    const res = await fetch(adminURL('api/products?q='), {
      headers: { 'X-Auth-Token': localToken.value }
    })

    if (res.status === 401 || res.status === 403) {
      errorMessage.value = t.value.login.invalidToken
      return
    }

    if (!res.ok) {
      errorMessage.value = t.value.login.loginFailed
      return
    }

    emit('login', localToken.value)
  } catch (e) {
    errorMessage.value = t.value.login.loginFailed + ': ' + e
  } finally {
    isLoggingIn.value = false
  }
}
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
