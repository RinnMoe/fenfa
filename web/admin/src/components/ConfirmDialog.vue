<template>
  <span class="dialog-bridge" aria-hidden="true"></span>
</template>

<script setup lang="ts">
import { useDialog } from 'naive-ui'

interface ConfirmOptions {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  type?: 'danger' | 'primary' | 'warning'
}

const dialog = useDialog()

function show(options: ConfirmOptions): Promise<boolean> {
  return new Promise((resolve) => {
    const resolved = (value: boolean) => {
      resolve(value)
    }
    const common = {
      title: options.title,
      content: options.message,
      positiveText: options.confirmText || 'OK',
      negativeText: options.cancelText || 'Cancel',
      onPositiveClick: () => resolved(true),
      onNegativeClick: () => resolved(false),
      onClose: () => resolved(false)
    }
    if (options.type === 'danger') {
      dialog.error(common)
    } else if (options.type === 'warning') {
      dialog.warning(common)
    } else {
      dialog.success(common)
    }
  })
}

defineExpose({
  show
})
</script>

