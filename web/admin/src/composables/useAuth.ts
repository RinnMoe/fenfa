import { ref, computed, type Ref } from 'vue'
import { adminFetch } from '../api'

// The admin session is kept in an HttpOnly cookie. This separate in-memory
// upload token is only used when an upload domain is configured.
const token = ref<string>('')
const authenticated = ref(false)
const sessionReady = ref(false)
const isLoggedIn = computed(() => authenticated.value)

export function useAuth() {
  function handleLogin(uploadToken = '') {
    token.value = uploadToken
    authenticated.value = true
  }

  function doLogout() {
    void adminFetch('api/auth/logout', { method: 'POST' }).catch(() => {})
    authenticated.value = false
    token.value = ''
    window.location.hash = '#/'
  }

  async function restoreSession() {
    try {
      const res = await adminFetch('api/auth/me')
      authenticated.value = res.ok
      if (res.ok) {
        const data = await res.json().catch(() => null)
        token.value = data?.data?.upload_token || ''
      } else {
        token.value = ''
      }
    } catch {
      authenticated.value = false
      token.value = ''
    } finally {
      sessionReady.value = true
    }
  }

  function handleAuthError(res: Response): boolean {
    if (res.status === 401 || res.status === 403) {
      doLogout()
      return true
    }
    return false
  }

  return {
    token: token as Ref<string>,
    isLoggedIn,
    sessionReady,
    handleLogin,
    doLogout,
    handleAuthError,
    restoreSession,
  }
}
