// Runtime admin URL suffix, injected by the Go template (admin.html).
// Falls back to "admin" when running under the Vite dev server.
declare global {
  interface Window {
    __FENFA_ADMIN_PATH__?: string
  }
}

let adminPath = (window.__FENFA_ADMIN_PATH__ || 'admin').replace(/^\/+|\/+$/g, '')

/** Updates the runtime admin path after it is changed from the settings page. */
export function setAdminPath(path: string) {
  const normalized = path.replace(/^\/+|\/+$/g, '').trim()
  if (normalized) adminPath = normalized
}

/** Builds a URL under the configured admin path, e.g. adminURL('api/products') -> /admin/api/products */
export function adminURL(path: string): string {
  return `/${adminPath}/${path.replace(/^\/+/, '')}`
}

export interface AdminFetchOptions extends RequestInit {
  token?: string
}

/** Same-origin admin API fetch with the admin path and session cookie. */
export function adminFetch(path: string, options: AdminFetchOptions = {}) {
  const headers = new Headers(options.headers)
  if (options.token) headers.set('X-Auth-Token', options.token)
  const { token: _token, ...fetchOptions } = options
  return fetch(adminURL(path), {
    ...fetchOptions,
    credentials: fetchOptions.credentials || 'include',
    headers
  })
}
