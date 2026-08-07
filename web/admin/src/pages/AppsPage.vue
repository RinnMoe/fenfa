<template>
  <div>
    <div class="page-header">
      <div class="page-header-row">
        <div>
          <h1 class="page-title">{{ t.apps.title }}</h1>
          <p class="page-subtitle">{{ t.apps.subtitle }}</p>
        </div>
        <button class="btn-primary" @click="showCreateProduct = !showCreateProduct">
          <Icon :name="showCreateProduct ? 'x' : 'plus'" :size="14" />
          {{ showCreateProduct ? t.common.cancel : t.apps.createProduct }}
        </button>
      </div>
    </div>

    <div v-if="showCreateProduct" class="card create-card">
      <div class="create-grid">
        <input v-model="newProduct.name" class="input" :placeholder="t.apps.productNamePlaceholder" />
        <input v-model="newProduct.slug" class="input" :placeholder="t.apps.productSlugPlaceholder" />
        <input v-model="newProduct.description" class="input" :placeholder="t.apps.productDescriptionPlaceholder" />
      </div>
      <div class="create-actions">
        <button class="btn-primary" :disabled="creatingProduct" @click="createProduct">
          {{ creatingProduct ? t.common.loading : t.apps.createProduct }}
        </button>
      </div>
    </div>

    <div class="panel">
      <div class="search-bar">
        <input
          v-model="localQuery"
          :placeholder="t.apps.searchPlaceholder"
          class="input search-input"
        >
        <button
          @click="fetchProducts()"
          :disabled="loading"
          class="btn-ghost btn-refresh"
        >
          <Icon name="refresh-cw" :size="14" :class="{ spinning: loading }" />
          {{ loading ? t.common.loading : t.apps.refreshButton }}
        </button>
      </div>

      <div class="table-container">
        <table class="table">
          <thead>
            <tr>
              <th>{{ t.apps.name }}</th>
              <th>{{ t.apps.slug }}</th>
              <th>{{ t.apps.variantCount }}</th>
              <th>{{ t.apps.status }}</th>
              <th>{{ t.apps.createdAt }}</th>
              <th class="actions">{{ t.apps.actions }}</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="product in products" :key="product.id">
              <tr
                :class="{ 'row-active': activeProductId === product.id }"
                class="product-row"
                @click="openProduct(product)"
              >
                <td class="product-name">{{ product.name || t.apps.unnamed }}</td>
                <td class="mono">{{ product.slug || product.id }}</td>
                <td>{{ product.variant_count || 0 }}</td>
                <td>
                  <span v-if="product.published" class="status-dot status-on"><i></i>{{ t.apps.published }}</span>
                  <span v-else class="status-dot status-off"><i></i>{{ t.apps.unpublished }}</span>
                </td>
                <td class="date">{{ new Date(product.created_at).toLocaleDateString() }}</td>
                <td class="actions" @click.stop>
                  <div class="dropdown">
                    <button
                      class="btn-icon"
                      type="button"
                      :aria-label="t.apps.actions"
                      @click="toggleProductMenu(product.id)"
                    >
                      <Icon name="more-vertical" :size="16" />
                    </button>
                    <div v-if="openProductMenuId === product.id" class="dropdown-menu">
                      <button class="dropdown-item" @click="openDistribution(product)">
                        <Icon name="external-link" :size="14" />
                        {{ t.apps.viewDistribution }}
                      </button>
                      <button class="dropdown-item" @click="openAnnouncements(product)">
                        <Icon name="info" :size="14" />
                        {{ t.apps.announcements }}
                      </button>
                      <div class="dropdown-divider"></div>
                      <button
                        v-if="product.published"
                        class="dropdown-item"
                        :disabled="updatingProductId === product.id"
                        @click="updateProductPublished(product, false)"
                      >
                        {{ t.apps.unpublish }}
                      </button>
                      <button
                        v-else
                        class="dropdown-item"
                        :disabled="updatingProductId === product.id"
                        @click="updateProductPublished(product, true)"
                      >
                        {{ t.apps.publish }}
                      </button>
                      <div class="dropdown-divider"></div>
                      <button class="dropdown-item danger" @click="deleteProduct(product)">
                        <Icon name="trash" :size="14" />
                        {{ t.apps.deleteProduct }}
                      </button>
                    </div>
                  </div>
                </td>
              </tr>

              <tr v-if="activeProductId === product.id && activeProduct" class="nested-row">
                <td colspan="6">
                  <div class="nested-detail">
                    <div class="flex-between">
                      <div class="section-header">
                        <div>
                          <h2 class="section-title">{{ activeProduct.product.name }}</h2>
                          <p class="page-subtitle">/{{ activeProduct.product.slug || activeProduct.product.id }}</p>
                        </div>
                      </div>
                      <div class="toolbar-actions">
                        <button class="btn-ghost" @click="openAnnouncements(activeProduct.product)">
                          <Icon name="info" :size="14" />
                          {{ t.apps.announcements }}
                        </button>
                        <button class="btn-ghost" @click="closeProduct">
                          <Icon name="x" :size="14" />
                          {{ t.common.cancel }}
                        </button>
                      </div>
                    </div>

                    <div v-if="activeProduct.variants.length === 0" class="empty-state">
                      {{ t.apps.noVariants }}
                    </div>

                    <div v-else class="table-container">
                      <table class="variant-table">
                        <thead>
                          <tr>
                            <th>{{ t.apps.platform }}</th>
                            <th>{{ t.apps.identifier }}</th>
                            <th>{{ t.apps.latestRelease }}</th>
                            <th>{{ t.apps.forceUpdate }}</th>
                            <th>{{ t.apps.status }}</th>
                            <th>{{ t.apps.createdAt }}</th>
                            <th class="actions">{{ t.apps.actions }}</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr v-for="variant in activeProduct.variants" :key="variant.id">
                            <td>
                              <div class="platform-cell">
                                <span class="platform-label">{{ platformLabel(variant.platform) }}</span>
                                <span class="display-name">{{ variant.display_name || activeProduct.product.name }}</span>
                              </div>
                            </td>
                            <td class="mono">{{ variant.identifier || '-' }}</td>
                            <td>
                              <template v-if="variant.releases?.length">
                                <span class="version">{{ variant.releases[0].version }}</span>
                                <span v-if="variant.releases[0].build" class="muted">
                                  ({{ t.upload.build }} {{ variant.releases[0].build }})
                                </span>
                              </template>
                              <span v-else class="muted">-</span>
                            </td>
                            <td>
                              <label v-if="variant.releases?.length" class="switch" :title="t.apps.forceUpdate">
                                <input
                                  type="checkbox"
                                  :checked="!!variant.releases[0].force_update"
                                  :disabled="updatingReleaseId === variant.releases[0].id"
                                  @change="updateReleaseForceUpdate(variant.releases[0], ($event.target as HTMLInputElement).checked)"
                                >
                                <span class="switch-track"></span>
                              </label>
                              <span v-else class="muted">-</span>
                            </td>
                            <td>
                              <span v-if="variant.published" class="status-dot status-on"><i></i>{{ t.apps.published }}</span>
                              <span v-else class="status-dot status-off"><i></i>{{ t.apps.unpublished }}</span>
                            </td>
                            <td class="date">{{ formatDate(variant.releases?.[0]?.created_at) }}</td>
                            <td class="actions">
                              <div class="table-actions">
                                <button class="btn-sm" @click="openUploadModal(activeProduct.product, variant)">
                                  <Icon name="upload" :size="13" />
                                  {{ t.apps.uploadNewVersion }}
                                </button>
                                <div class="dropdown">
                                  <button
                                    class="btn-icon"
                                    type="button"
                                    :aria-label="t.apps.actions"
                                    @click="toggleVariantMenu(variant.id)"
                                  >
                                    <Icon name="more-vertical" :size="16" />
                                  </button>
                                  <div v-if="openVariantMenuId === variant.id" class="dropdown-menu">
                                    <button class="dropdown-item" @click="router.push('/stats/' + variant.id)">
                                      {{ t.apps.viewStats }}
                                    </button>
                                    <button class="dropdown-item" @click="router.push('/events/' + variant.id)">
                                      {{ t.apps.viewEvents }}
                                    </button>
                                    <div class="dropdown-divider"></div>
                                    <button
                                      v-if="variant.published"
                                      class="dropdown-item"
                                      :disabled="updatingVariantId === variant.id"
                                      @click="updateVariantPublished(variant, false)"
                                    >
                                      {{ t.apps.unpublish }}
                                    </button>
                                    <button
                                      v-else
                                      class="dropdown-item"
                                      :disabled="updatingVariantId === variant.id"
                                      @click="updateVariantPublished(variant, true)"
                                    >
                                      {{ t.apps.publish }}
                                    </button>
                                    <div class="dropdown-divider"></div>
                                    <button class="dropdown-item danger" @click="deleteVariant(variant)">
                                      {{ t.apps.deleteVariant }}
                                    </button>
                                  </div>
                                </div>
                              </div>
                            </td>
                          </tr>
                        </tbody>
                      </table>
                    </div>

                    <div class="create-variant">
                      <button v-if="!showCreateVariant" class="btn-dashed" @click="showCreateVariant = true">
                        <Icon name="plus" :size="14" />
                        {{ t.apps.addVariant }}
                      </button>
                      <template v-else>
                        <div class="section-header">
                          <h3 class="section-title">{{ t.apps.addVariant }}</h3>
                          <button class="btn-ghost" @click="showCreateVariant = false">{{ t.common.cancel }}</button>
                        </div>
                        <div class="create-grid">
                          <select v-model="newVariant.platform" class="select">
                            <option value="ios">iOS</option>
                            <option value="android">Android</option>
                            <option value="macos">macOS</option>
                            <option value="windows">Windows</option>
                            <option value="linux">Linux</option>
                          </select>
                          <input v-model="newVariant.identifier" class="input" :placeholder="t.apps.variantIdentifier" />
                          <input v-model="newVariant.display_name" class="input" :placeholder="t.apps.variantDisplayName" />
                        </div>
                        <div class="create-actions">
                          <button class="btn-primary" :disabled="creatingVariant" @click="createVariant">
                            {{ creatingVariant ? t.common.loading : t.apps.createVariant }}
                          </button>
                        </div>
                      </template>
                    </div>
                  </div>
                </td>
              </tr>
            </template>
            <tr v-if="products.length === 0">
              <td colspan="6" class="empty-state">{{ loading ? t.common.loading : t.apps.noData }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <UploadModal
      :show="showUploadModal"
      :product="selectedProduct"
      :variant="selectedVariant"
      @close="showUploadModal = false"
      @success="handleUploadSuccess"
    />

    <AnnouncementModal
      :show="showAnnouncementModal"
      :product="selectedAnnouncementProduct"
      @close="showAnnouncementModal = false"
    />

    <DistributionModal
      :show="showDistributionModal"
      :product="selectedDistributionProduct"
      :token="token"
      @close="showDistributionModal = false"
    />
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from '../i18n'
import { useAuth } from '../composables/useAuth'
import { useProducts } from '../composables/useProducts'
import { useToast } from '../composables/useToast'
import { useConfirmDialog } from '../composables/useConfirmDialog'
import Icon from '../components/Icon.vue'
import UploadModal from '../components/UploadModal.vue'
import DistributionModal from '../components/DistributionModal.vue'
import AnnouncementModal from '../components/AnnouncementModal.vue'

interface ProductSummary {
  id: string
  name: string
  slug?: string
  published: boolean
  variant_count?: number
  created_at: string
}

interface ProductDetail {
  product: ProductSummary
  variants: any[]
}

const router = useRouter()
const { token, handleAuthError } = useAuth()
const { products, loadingProducts: loading, productQuery, fetchProducts } = useProducts()

const { t } = useI18n()
const toast = useToast()
const confirm = useConfirmDialog()
const localQuery = ref(productQuery.value)
const activeProduct = ref<ProductDetail | null>(null)
const activeProductId = ref('')
const updatingProductId = ref('')
const updatingVariantId = ref('')
const updatingReleaseId = ref('')
const creatingProduct = ref(false)
const creatingVariant = ref(false)
const showUploadModal = ref(false)
const showDistributionModal = ref(false)
const showAnnouncementModal = ref(false)
const showCreateProduct = ref(false)
const showCreateVariant = ref(false)
const openProductMenuId = ref('')
const openVariantMenuId = ref('')
const selectedProduct = ref<any>(null)
const selectedVariant = ref<any>(null)
const selectedDistributionProduct = ref<any>(null)
const selectedAnnouncementProduct = ref<any>(null)

const newVariant = reactive({
  platform: 'ios',
  display_name: '',
  identifier: '',
})

const newProduct = reactive({
  name: '',
  slug: '',
  description: ''
})

// Close dropdown on outside click
function handleGlobalClick() {
  openProductMenuId.value = ''
  openVariantMenuId.value = ''
}
onMounted(() => document.addEventListener('click', handleGlobalClick))
onUnmounted(() => document.removeEventListener('click', handleGlobalClick))

function toggleProductMenu(id: string) {
  openProductMenuId.value = openProductMenuId.value === id ? '' : id
  openVariantMenuId.value = ''
}

function toggleVariantMenu(id: string) {
  openVariantMenuId.value = openVariantMenuId.value === id ? '' : id
  openProductMenuId.value = ''
}

function platformLabel(platform: string): string {
  const labels: Record<string, string> = {
    ios: 'iOS',
    android: 'Android',
    macos: 'macOS',
    windows: 'Windows',
    linux: 'Linux'
  }
  return labels[platform] || platform
}

function formatDate(value: string | null | undefined): string {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleDateString()
}

watch(localQuery, (value) => {
  productQuery.value = value
})

async function fetchProductDetail(productId: string) {
  const res = await fetch(`/admin/api/products/${productId}`, {
    headers: { 'X-Auth-Token': token.value }
  })
  if (handleAuthError(res)) return null
  const payload = await res.json()
  if (!res.ok || !payload.ok) {
    throw new Error(payload.error?.message || t.value.apps.fetchFailed)
  }
  return payload.data as ProductDetail
}

function resetProductForm() {
  newProduct.name = ''
  newProduct.slug = ''
  newProduct.description = ''
}

async function openProduct(product: ProductSummary) {
  if (activeProductId.value === product.id) {
    closeProduct()
    return
  }

  try {
    activeProductId.value = product.id
    activeProduct.value = await fetchProductDetail(product.id)
    showCreateVariant.value = false
  } catch (error) {
    activeProductId.value = ''
    toast.error(String(error))
  }
}

function closeProduct() {
  activeProductId.value = ''
  activeProduct.value = null
  showCreateVariant.value = false
  resetVariantForm()
}

function resetVariantForm() {
  newVariant.platform = 'ios'
  newVariant.display_name = ''
  newVariant.identifier = ''
}

function openUploadModal(product: any, variant: any) {
  selectedProduct.value = product
  selectedVariant.value = variant
  showUploadModal.value = true
}

function openDistribution(product: ProductSummary) {
  selectedDistributionProduct.value = product
  showDistributionModal.value = true
}

function openAnnouncements(product: any) {
  selectedAnnouncementProduct.value = product
  showAnnouncementModal.value = true
}

async function createProduct() {
  if (!newProduct.name.trim()) {
    toast.error(t.value.apps.productNameRequired)
    return
  }

  creatingProduct.value = true
  try {
    const res = await fetch('/admin/api/products', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Auth-Token': token.value
      },
      body: JSON.stringify({
        name: newProduct.name,
        slug: newProduct.slug,
        description: newProduct.description
      })
    })

    if (handleAuthError(res)) return

    const payload = await res.json()
    if (!res.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.apps.createProductFailed)
      return
    }

    toast.success(t.value.apps.createProductSuccess)
    resetProductForm()
    showCreateProduct.value = false
    fetchProducts()
    await openProduct(payload.data.product)
  } catch (error) {
    toast.error(t.value.apps.createProductFailed + ': ' + error)
  } finally {
    creatingProduct.value = false
  }
}

async function createVariant() {
  if (!activeProduct.value?.product.id) {
    return
  }

  creatingVariant.value = true
  try {
    const res = await fetch(`/admin/api/products/${activeProduct.value.product.id}/variants`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Auth-Token': token.value
      },
      body: JSON.stringify({
        platform: newVariant.platform,
        display_name: newVariant.display_name,
        identifier: newVariant.identifier,
      })
    })

    if (handleAuthError(res)) return

    const payload = await res.json()
    if (!res.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.apps.createVariantFailed)
      return
    }

    toast.success(t.value.apps.createVariantSuccess)
    resetVariantForm()
    showCreateVariant.value = false
    fetchProducts()
    activeProduct.value = await fetchProductDetail(activeProduct.value.product.id)
  } catch (error) {
    toast.error(t.value.apps.createVariantFailed + ': ' + error)
  } finally {
    creatingVariant.value = false
  }
}

async function updateVariantPublished(variant: any, published: boolean) {
  const confirmed = await confirm.show({
    title: published ? t.value.apps.publish : t.value.apps.unpublish,
    message: published ? t.value.apps.confirmPublishVariant : t.value.apps.confirmUnpublishVariant,
    confirmText: published ? t.value.apps.publish : t.value.apps.unpublish,
    cancelText: t.value.common.cancel,
    type: published ? 'primary' : 'danger'
  })
  if (!confirmed) return

  openVariantMenuId.value = ''
  openProductMenuId.value = ''
  updatingVariantId.value = variant.id
  try {
    const res = await fetch(`/admin/api/variants/${variant.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-Auth-Token': token.value
      },
      body: JSON.stringify({ published })
    })
    if (handleAuthError(res)) return
    const payload = await res.json()
    if (!res.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.apps.updateVariantFailed)
      return
    }

    toast.success(t.value.apps.updateVariantSuccess)
    fetchProducts()
    if (activeProduct.value) {
      activeProduct.value = await fetchProductDetail(activeProduct.value.product.id)
    }
  } catch (error) {
    toast.error(t.value.apps.updateVariantFailed + ': ' + error)
  } finally {
    updatingVariantId.value = ''
  }
}

async function updateReleaseForceUpdate(release: any, forceUpdate: boolean) {
  updatingReleaseId.value = release.id
  try {
    const res = await fetch(`/admin/api/releases/${release.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-Auth-Token': token.value
      },
      body: JSON.stringify({ force_update: forceUpdate })
    })
    if (handleAuthError(res)) return
    const payload = await res.json()
    if (!res.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.apps.updateReleaseFailed)
      return
    }
    toast.success(t.value.apps.updateReleaseSuccess)
    fetchProducts()
    if (activeProduct.value) {
      activeProduct.value = await fetchProductDetail(activeProduct.value.product.id)
    }
  } catch (error) {
    toast.error(t.value.apps.updateReleaseFailed + ': ' + error)
  } finally {
    updatingReleaseId.value = ''
  }
}

async function handleUploadSuccess() {
  showUploadModal.value = false
  fetchProducts()
  if (activeProduct.value) {
    activeProduct.value = await fetchProductDetail(activeProduct.value.product.id)
  }
}

async function deleteProduct(product: ProductSummary) {
  const confirmed = await confirm.show({
    title: t.value.apps.deleteProduct,
    message: t.value.apps.deleteProductConfirm.replace('{name}', product.name),
    confirmText: t.value.common.delete,
    cancelText: t.value.common.cancel,
    type: 'danger'
  })
  if (!confirmed) return

  try {
    const res = await fetch(`/admin/api/products/${product.id}`, {
      method: 'DELETE',
      headers: { 'X-Auth-Token': token.value }
    })
    if (handleAuthError(res)) return
    const payload = await res.json()
    if (!res.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.apps.deleteProductFailed)
      return
    }
    toast.success(t.value.common.deleteSuccess)
    if (activeProductId.value === product.id) closeProduct()
    fetchProducts()
  } catch (error) {
    toast.error(String(error))
  }
}

async function deleteVariant(variant: any) {
  openVariantMenuId.value = ''
  openProductMenuId.value = ''
  const confirmed = await confirm.show({
    title: t.value.apps.deleteVariant,
    message: t.value.apps.deleteVariantConfirm.replace('{name}', variant.display_name || variant.identifier),
    confirmText: t.value.common.delete,
    cancelText: t.value.common.cancel,
    type: 'danger'
  })
  if (!confirmed) return

  try {
    const res = await fetch(`/admin/api/variants/${variant.id}`, {
      method: 'DELETE',
      headers: { 'X-Auth-Token': token.value }
    })
    if (handleAuthError(res)) return
    const payload = await res.json()
    if (!res.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.apps.deleteVariantFailed)
      return
    }
    toast.success(t.value.common.deleteSuccess)
    fetchProducts()
    if (activeProduct.value) {
      activeProduct.value = await fetchProductDetail(activeProduct.value.product.id)
    }
  } catch (error) {
    toast.error(String(error))
  }
}

async function updateProductPublished(product: ProductSummary, published: boolean) {
  const confirmed = await confirm.show({
    title: published ? t.value.apps.publish : t.value.apps.unpublish,
    message: published ? t.value.apps.confirmPublishProduct : t.value.apps.confirmUnpublishProduct,
    confirmText: published ? t.value.apps.publish : t.value.apps.unpublish,
    cancelText: t.value.common.cancel,
    type: published ? 'primary' : 'danger'
  })
  if (!confirmed) return

  updatingProductId.value = product.id
  try {
    const res = await fetch(`/admin/api/products/${product.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-Auth-Token': token.value
      },
      body: JSON.stringify({ published })
    })
    if (handleAuthError(res)) return
    const payload = await res.json()
    if (!res.ok || !payload.ok) {
      toast.error(payload.error?.message || t.value.apps.updateProductFailed)
      return
    }

    toast.success(t.value.apps.updateProductSuccess)
    fetchProducts()
    if (activeProduct.value?.product.id === product.id) {
      activeProduct.value = await fetchProductDetail(product.id)
    }
  } catch (error) {
    toast.error(t.value.apps.updateProductFailed + ': ' + error)
  } finally {
    updatingProductId.value = ''
  }
}
</script>

<style scoped>
.product-name {
  color: var(--text-primary);
  font-weight: 600;
}

.platform-cell {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.3;
}

.platform-label {
  color: var(--text-primary);
  font-size: 12px;
  font-weight: 600;
  text-transform: capitalize;
}

.display-name {
  overflow: hidden;
  color: var(--text-muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nested-detail {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.create-variant {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--border);
}
</style>
