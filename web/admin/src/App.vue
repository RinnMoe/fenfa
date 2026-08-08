<template>
  <n-config-provider
    :locale="naiveLocale"
    :date-locale="naiveDateLocale"
    :theme="isDark ? darkTheme : undefined"
    :theme-overrides="naiveThemeOverrides"
  >
    <n-message-provider>
      <n-dialog-provider>
        <!-- Login Page -->
        <LoginPage
          v-if="!isLoggedIn"
          @login="handleLogin"
        />

        <!-- Main Layout -->
        <div v-else class="main-layout">
          <Sidebar
            @logout="handleLogout"
          />

          <main class="main-content">
            <div class="content-wrapper">
              <div v-if="isDev" class="dev-banner">
                <Icon name="wrench" :size="16" style="margin-right: 6px" /> {{ t.common.devMode }}
              </div>

              <router-view />
            </div>
          </main>

          <Toast ref="toastRef" />
          <ConfirmDialog ref="confirmRef" />
        </div>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import {
  NConfigProvider,
  NMessageProvider,
  NDialogProvider,
  darkTheme,
  zhCN,
  dateZhCN,
  enUS,
  dateEnUS
} from 'naive-ui'
import { useI18n } from './i18n'
import { useAuth } from './composables/useAuth'
import { useProducts } from './composables/useProducts'
import { provideToast } from './composables/useToast'
import { provideConfirmDialog } from './composables/useConfirmDialog'
import { useTheme } from './theme'
import Icon from './components/Icon.vue'
import Toast from './components/Toast.vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
import LoginPage from './pages/LoginPage.vue'
import Sidebar from './components/Sidebar.vue'

const { t, locale } = useI18n()
const { isDark } = useTheme()
const toastRef = ref<InstanceType<typeof Toast>>()
const confirmRef = ref<InstanceType<typeof ConfirmDialog>>()
provideToast(toastRef as any)
provideConfirmDialog(confirmRef as any)

const naiveLocale = computed(() => (locale.value === 'en' ? enUS : zhCN))
const naiveDateLocale = computed(() => (locale.value === 'en' ? dateEnUS : dateZhCN))
const naiveThemeOverrides = computed(() => ({
  common: {
    primaryColor: isDark.value ? '#60a5fa' : '#2563eb',
    primaryColorHover: isDark.value ? '#93c5fd' : '#1d4ed8',
    primaryColorPressed: isDark.value ? '#60a5fa' : '#1d4ed8',
    primaryColorSuppl: isDark.value ? '#60a5fa' : '#2563eb',
    borderRadius: '6px'
  }
}))

const { token, isLoggedIn, handleLogin: authLogin, doLogout, handleAuthError } = useAuth()
const { fetchProducts, clearProducts } = useProducts()

const isDev = import.meta.env.DEV

function handleLogin(newToken: string) {
  authLogin(newToken)
}

async function handleLogout() {
  const confirmed = await confirmRef.value?.show({
    title: t.value.common.logout,
    message: t.value.messages.confirmLogout,
    confirmText: t.value.common.logout,
    cancelText: t.value.common.cancel,
    type: 'danger'
  })
  if (!confirmed) return
  doLogout()
}

watch(token, (v) => {
  if (v) fetchProducts()
  else clearProducts()
})

onMounted(() => {
  if (token.value) {
    fetchProducts()
  }
})
</script>

