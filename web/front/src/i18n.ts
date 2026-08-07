import { computed } from 'vue'

function detectLanguage(): 'zh' | 'en' {
  return navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

const translations = {
  zh: {
    loading: '加载中...',
    error: {
      title: '页面暂时不可用',
      loadFailed: '加载产品信息失败'
    },
    actions: {
      install: '安装',
      installing: '正在安装...',
      download: '下载',
      downloadAndroid: '下载 Android 版',
      downloadWindows: '下载 Windows 版',
      downloadMacos: '下载 macOS 版',
      downloadLinux: '下载 Linux 版',
      bindDevice: '绑定设备',
      scan: '手机扫码安装',
      unavailable: '暂不可用'
    },
    footer: {
      brandBy: '由',
      brandService: '提供分发服务'
    },
    variant: {
      otherVersions: '其他版本',
      hideVersions: '收起版本',
      changelogTitle: '更新说明',
      current: '当前版本',
      iosHint: 'iOS 首次安装前需要先绑定当前设备 UDID。'
    }
  },
  en: {
    loading: 'Loading...',
    error: {
      title: 'This page is temporarily unavailable',
      loadFailed: 'Failed to load product information'
    },
    actions: {
      install: 'Install',
      installing: 'Installing...',
      download: 'Download',
      downloadAndroid: 'Download Android',
      downloadWindows: 'Download Windows',
      downloadMacos: 'Download macOS',
      downloadLinux: 'Download Linux',
      bindDevice: 'Bind Device',
      scan: 'Scan with phone',
      unavailable: 'Unavailable'
    },
    footer: {
      brandBy: 'Distribution service by',
      brandService: ''
    },
    variant: {
      otherVersions: 'Other versions',
      hideVersions: 'Hide versions',
      changelogTitle: 'Changelog',
      current: 'Current',
      iosHint: 'On iOS, the device must be registered before the first install.'
    }
  }
}

export type Language = 'zh' | 'en'

let currentLanguage: Language = detectLanguage()

export function useI18n() {
  const t = computed(() => translations[currentLanguage])

  const setLanguage = (lang: Language) => {
    currentLanguage = lang
  }

  const getLanguage = () => currentLanguage

  return { t, setLanguage, getLanguage }
}
