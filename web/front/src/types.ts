export type Platform = 'ios' | 'android' | 'macos' | 'windows' | 'linux'

export type FeedbackCategory = 'bug' | 'suggestion' | 'other'

export interface ReleaseActions {
  download?: string
  release_page?: string
  ios_manifest?: string
  ios_install?: string
}

export interface ReleaseItem {
  id: string
  version: string
  build?: number
  created_at: string
  download_count?: number
  channel?: string
  min_os?: string
  changelog?: string
  actions: ReleaseActions
}

export interface VariantItem {
  id: string
  platform: Platform
  identifier: string
  display_name?: string
  arch?: string
  installer_type?: string
  min_os?: string
  latest_release?: ReleaseItem | null
  releases: ReleaseItem[]
}

export interface ProductData {
  product: {
    id: string
    slug?: string
    name: string
    icon_url?: string
  }
  variants: VariantItem[]
}

export type VariantActionType = 'disabled' | 'download' | 'install' | 'bind'

export interface VariantAction {
  type: VariantActionType
  href: string
  label: string
}
