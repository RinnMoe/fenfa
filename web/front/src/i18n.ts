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
    intro: {
      tagline: '自托管应用分发平台',
      description: '面向 iOS、Android、macOS、Windows 与 Linux 的多平台应用分发与版本管理平台，支持智能上传、iOS UDID 绑定、强制更新与产品公告。',
      features: ['多平台分发', '智能上传', 'iOS UDID 绑定', '强制更新', '产品公告', 'S3 / R2 存储'],
      github: '查看 GitHub'
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
    feedback: {
      button: '用户反馈',
      title: '用户反馈',
      intro: '告诉我们你遇到的问题或想要的改进。',
      context: '反馈对象',
      product: '产品',
      platform: '平台',
      release: '版本',
      category: '反馈类型',
      categoryBug: '问题反馈',
      categorySuggestion: '功能建议',
      categoryOther: '其他',
      content: '反馈内容',
      contentPlaceholder: '请描述你遇到的问题或建议',
      contact: '联系方式（可选）',
      contactPlaceholder: '邮箱、Telegram、微信或其他联系方式',
      submit: '提交反馈',
      submitting: '提交中...',
      close: '关闭',
      success: '感谢你的反馈，我们会尽快查看。',
      error: '提交失败，请稍后再试。',
      rateLimited: '提交过于频繁，请稍后再试。'
    },
    variant: {
      otherVersions: '其他版本',
      hideVersions: '收起版本',
      changelogTitle: '更新说明',
      current: '当前版本',
      iosHint: 'iOS 首次安装前需要先绑定当前设备 UDID。',
      downloadCount: '{{ count }} 次下载'
    }
  },
  en: {
    loading: 'Loading...',
    error: {
      title: 'This page is temporarily unavailable',
      loadFailed: 'Failed to load product information'
    },
    intro: {
      tagline: 'Self-hosted app distribution platform',
      description: 'A multi-platform app distribution and release management platform for iOS, Android, macOS, Windows and Linux, with smart upload, iOS UDID binding, force updates and product announcements.',
      features: ['Multi-platform', 'Smart upload', 'iOS UDID binding', 'Force updates', 'Announcements', 'S3 / R2 storage'],
      github: 'View on GitHub'
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
    feedback: {
      button: 'Feedback',
      title: 'Send feedback',
      intro: 'Tell us about a problem or an improvement you would like to see.',
      context: 'Feedback context',
      product: 'Product',
      platform: 'Platform',
      release: 'Version',
      category: 'Type',
      categoryBug: 'Problem',
      categorySuggestion: 'Suggestion',
      categoryOther: 'Other',
      content: 'Feedback',
      contentPlaceholder: 'Describe the problem or suggestion',
      contact: 'Contact (optional)',
      contactPlaceholder: 'Email, Telegram, or another way to reach you',
      submit: 'Send feedback',
      submitting: 'Sending...',
      close: 'Close',
      success: 'Thank you. We will review your feedback soon.',
      error: 'Unable to send feedback. Please try again later.',
      rateLimited: 'Too many submissions. Please try again later.'
    },
    variant: {
      otherVersions: 'Other versions',
      hideVersions: 'Hide versions',
      changelogTitle: 'Changelog',
      current: 'Current',
      iosHint: 'On iOS, the device must be registered before the first install.',
      downloadCount: '{{ count }} downloads'
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
