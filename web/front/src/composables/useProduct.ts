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

export function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) return dateStr
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function useProduct() {
  const productData = ref<ProductData | null>(null)
  const loading = ref(true)
  const error = ref('')
  const clientPlatform = detectClientPlatform()
  const selectedReleaseByVariant = reactive<Record<string, ReleaseItem | null>>({})

  const totalDownloadCount = computed(() => {
    return (productData.value?.variants || []).reduce((total, variant) => {
      return total + variant.releases.reduce((sum, release) => sum + (release.download_count || 0), 0)
    }, 0)
  })

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
    let payload: any = null
    try {
      payload = await res.json()
    } catch {
      payload = null
    }

    if (!res.ok || !payload?.ok) {
      loading.value = false
      error.value = payload?.error?.message || errorMessage
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
    sortedVariants,
    totalDownloadCount
  }
}
