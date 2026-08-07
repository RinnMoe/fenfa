import { reactive, ref } from 'vue'
import type { ReleaseItem, VariantAction, VariantItem } from '../types'

function getCookie(name: string): string | null {
  const match = document.cookie.match(new RegExp('(^| )' + name + '=([^;]+)'))
  return match ? match[2] : null
}

export function useUDID() {
  const udidBound = reactive<Record<string, boolean>>({})
  const installing = ref(false)

  async function fetchUDIDStatuses(variants: VariantItem[]) {
    const iosVariants = variants.filter((variant) => variant.platform === 'ios')
    if (iosVariants.length === 0) return

    const urlParams = new URLSearchParams(location.search)
    if (urlParams.get('udid_bound') === '1') {
      const boundVariantID = urlParams.get('variant')
      if (boundVariantID) {
        udidBound[boundVariantID] = true
      } else {
        iosVariants.forEach((variant) => {
          udidBound[variant.id] = true
        })
      }
      urlParams.delete('udid_bound')
      urlParams.delete('variant')
      const nextURL = urlParams.toString() ? `${location.pathname}?${urlParams}` : location.pathname
      history.replaceState(null, '', nextURL)
      return
    }

    const nonce = getCookie('udid_nonce')
    await Promise.all(iosVariants.map(async (variant) => {
      try {
        const params = new URLSearchParams()
        if (nonce) params.set('nonce', nonce)
        params.set('variant', variant.id)
        const res = await fetch(`/udid/status?${params}`)
        const payload = await res.json()
        udidBound[variant.id] = !!payload?.data?.bound
      } catch {
        udidBound[variant.id] = false
      }
    }))
  }

  function getVariantAction(variant: VariantItem, release: ReleaseItem | null, t: any): VariantAction {
    if (!release) return { type: 'disabled', href: '#', label: t.actions.unavailable }
    if (variant.platform === 'ios') {
      if (udidBound[variant.id]) {
        return {
          type: 'install',
          href: release.actions?.ios_install || '',
          label: installing.value ? t.actions.installing : t.actions.install
        }
      }
      return {
        type: 'bind',
        href: `/udid/profile.mobileconfig?variant=${encodeURIComponent(variant.id)}`,
        label: t.actions.bindDevice
      }
    }
    const downloadLabels: Record<string, string> = {
      android: t.actions.downloadAndroid,
      windows: t.actions.downloadWindows,
      macos: t.actions.downloadMacos,
      linux: t.actions.downloadLinux
    }

    return {
      type: 'download',
      href: release.actions?.download || '',
      label: downloadLabels[variant.platform] || t.actions.download
    }
  }

  function handlePrimaryAction(
    variant: VariantItem,
    release: ReleaseItem | null,
    action: VariantAction,
    track: (href: string) => void
  ) {
    if (!release || !action.href) return
    if (action.type === 'install') {
      installing.value = true
      track(action.href)
      location.href = action.href
      setTimeout(() => { installing.value = false }, 10000)
      return
    }
    track(action.href)
    location.href = action.href
  }

  return {
    fetchUDIDStatuses,
    getVariantAction,
    handlePrimaryAction,
    installing,
    udidBound
  }
}
