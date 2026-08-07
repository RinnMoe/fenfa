import { computed, reactive, ref } from 'vue'
import type { Platform, ProductData, ReleaseItem, VariantItem } from '../types'

export function detectClientPlatform(): Platform | '' {
  const ua = navigator.userAgent.toLowerCase()
  if (/iphone|ipad|ipod/.test(ua)) return 'ios'
  if (ua.includes('android')) return 'android'
  if (ua.includes('mac os x') || ua.includes('macintosh')) return 'macos'
  if (ua.includes('windows')) return 'windows'
  if (ua.includes('linux')) return 'linux'
  return ''
}

export function platformLabel(platform: string): string {
  const labels: Record<string, string> = {
    ios: 'iOS',
    android: 'Android',
    macos: 'macOS',
    windows: 'Windows',
    linux: 'Linux'
  }
  return labels[platform] || platform
}

export function formatTime(dateStr: string, t: any): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / (1000 * 60))
  const hours = Math.floor(diff / (1000 * 60 * 60))
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (minutes < 1) return t.time.justNow
  if (minutes < 60) return `${minutes} ${t.time.minutesAgo}`
  if (hours < 24) return `${hours} ${t.time.hoursAgo}`
  if (days < 7) return `${days} ${t.time.daysAgo}`
  if (days < 30) return `${Math.floor(days / 7)} ${t.time.weeksAgo}`
  if (days < 365) return `${Math.floor(days / 30)} ${t.time.monthsAgo}`
  return date.toLocaleDateString()
}

export function useProduct() {
  const productData = ref<ProductData | null>(null)
  const loading = ref(true)
  const error = ref('')
  const clientPlatform = detectClientPlatform()
  const selectedReleaseByVariant = reactive<Record<string, ReleaseItem | null>>({})

  const sortedVariants = computed(() => {
    const items = [...(productData.value?.variants || [])]
    return items.sort((left, right) => {
      const leftScore = left.platform === clientPlatform ? 1 : 0
      const rightScore = right.platform === clientPlatform ? 1 : 0
      if (leftScore !== rightScore) return rightScore - leftScore
      return left.platform.localeCompare(right.platform)
    })
  })

  const recommendedVariant = computed(() => {
    return sortedVariants.value.find((variant) => variant.platform === clientPlatform) || sortedVariants.value[0] || null
  })

  function getSelectedRelease(variant: VariantItem): ReleaseItem | null {
    return selectedReleaseByVariant[variant.id] || variant.latest_release || variant.releases[0] || null
  }

  function selectRelease(variantID: string, release: ReleaseItem) {
    selectedReleaseByVariant[variantID] = release
  }

  async function fetchData(resourceID: string, errorMessage: string) {
    const selectedReleaseID = new URLSearchParams(location.search).get('r')
    const endpoint = `/api/products/${resourceID}`
    const res = await fetch(endpoint + (selectedReleaseID ? `?r=${encodeURIComponent(selectedReleaseID)}` : ''))
    const payload = await res.json()

    if (!res.ok || !payload.ok) {
      loading.value = false
      error.value = payload.error?.message || errorMessage
      return
    }

    productData.value = payload.data as ProductData

    for (const variant of productData.value.variants) {
      const preferred = variant.releases.find((release) => release.id === selectedReleaseID) || variant.latest_release || variant.releases[0] || null
      selectedReleaseByVariant[variant.id] = preferred
    }

    loading.value = false
  }

  return {
    clientPlatform,
    error,
    fetchData,
    getSelectedRelease,
    loading,
    platformLabel,
    productData,
    recommendedVariant,
    selectRelease,
    selectedReleaseByVariant,
    sortedVariants
  }
}
