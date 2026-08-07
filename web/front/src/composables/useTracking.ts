import type { ReleaseItem, VariantItem } from '../types'

function sendBeaconJSON(url: string, payload: any) {
  try {
    if (navigator.sendBeacon) {
      const blob = new Blob([JSON.stringify(payload)], { type: 'application/json' })
      navigator.sendBeacon(url, blob)
      return
    }
  } catch {}
  fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  }).catch(() => {})
}

export function useTracking() {
  async function logVisit(
    variant: VariantItem,
    release: ReleaseItem,
    productID: string,
    clientPlatform: string,
    path: string
  ) {
    try {
      const mod: any = await import('ua-parser-js')
      const UAParser = mod.UAParser || mod.default
      const parser = new UAParser(navigator.userAgent)
      sendBeaconJSON('/events', {
        type: 'visit',
        variant_id: variant.id,
        release_id: release.id,
        extra: {
          product_id: productID,
          variant_id: variant.id,
          ua: parser.getResult(),
          client_platform: clientPlatform,
          path
        }
      })
    } catch {}
  }

  function trackAction(
    variant: VariantItem,
    release: ReleaseItem | null,
    href: string,
    productID: string,
    path: string
  ) {
    if (!release || !href) return
    sendBeaconJSON('/events', {
      type: 'click',
      variant_id: variant.id,
      release_id: release.id,
      extra: {
        product_id: productID,
        variant_id: variant.id,
        href,
        path
      }
    })
  }

  return { logVisit, trackAction }
}
