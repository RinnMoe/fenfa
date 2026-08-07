// Runtime admin URL suffix, injected by the Go template (admin.html).
// Falls back to "admin" when running under the Vite dev server.
declare global {
  interface Window {
    __FENFA_ADMIN_PATH__?: string
  }
}

const adminPath = (window.__FENFA_ADMIN_PATH__ || 'admin').replace(/^\/+|\/+$/g, '')

/** Builds a URL under the configured admin path, e.g. adminURL('api/products') -> /admin/api/products */
export function adminURL(path: string): string {
  return `/${adminPath}/${path.replace(/^\/+/, '')}`
}