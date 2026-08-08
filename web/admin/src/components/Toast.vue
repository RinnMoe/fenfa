<template>
  <span class="toast-bridge" aria-hidden="true"></span>
</template>

<script setup lang="ts">
import { useMessage } from 'naive-ui'

export interface Toast {
  id: number
  type: 'success' | 'error' | 'warning' | 'info'
  title?: string
  message: string
  duration?: number
}

const message = useMessage()

function addToast(toast: Omit<Toast, 'id'>) {
  const content = toast.title ? `${toast.title} ${toast.message}` : toast.message
  const options = {
    duration: toast.duration || 3000
  }
  if (toast.type === 'success') {
    message.success(content, options)
  } else if (toast.type === 'error') {
    message.error(content, options)
  } else if (toast.type === 'warning') {
    message.warning(content, options)
  } else {
    message.info(content, options)
  }
}

function removeToast(id: number) {
  void id
}

defineExpose({
  addToast,
  removeToast,
  success: (message: string, title?: string) => addToast({ type: 'success', message, title }),
  error: (message: string, title?: string) => addToast({ type: 'error', message, title }),
  warning: (message: string, title?: string) => addToast({ type: 'warning', message, title }),
  info: (message: string, title?: string) => addToast({ type: 'info', message, title })
})
</script>

