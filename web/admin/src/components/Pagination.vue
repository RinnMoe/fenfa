<template>
  <div v-if="totalPages > 1" class="pagination">
    <span class="pagination-info">
      {{ t.pagination.total.replace('{total}', String(total)) }} · {{ t.pagination.page.replace('{page}', String(page)).replace('{totalPages}', String(totalPages)) }}
    </span>

    <n-pagination
      :item-count="total"
      :page-size="pageSize"
      :page="page"
      @update:page="(p) => $emit('update:page', p)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NPagination } from 'naive-ui'
import { useI18n } from '../i18n'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  total: number
  page: number
  pageSize?: number
}>(), {
  pageSize: 50
})

defineEmits<{
  'update:page': [page: number]
}>()

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
</script>

