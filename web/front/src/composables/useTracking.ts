import type { VariantItem } from '../types'

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
  function logVisit(variant: VariantItem) {
    sendBeaconJSON('/events', {
      type: 'visit',
      variant_id: variant.id
    })
  }

  return { logVisit }
}
