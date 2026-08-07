<template>
  <aside class="qr-panel">
    <span class="qr-title">{{ t.actions.scan }}</span>
    <canvas ref="canvas" width="148" height="148"></canvas>
    <span class="qr-url">{{ url }}</span>
  </aside>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import QRCode from 'qrcode'
import { useI18n } from '../i18n'

const props = defineProps<{
  url: string
}>()

const { t } = useI18n()
const canvas = ref<HTMLCanvasElement | null>(null)

async function render() {
  await nextTick()
  if (!canvas.value || !props.url) return
  try {
    await QRCode.toCanvas(canvas.value, props.url, { width: 148, margin: 1 })
  } catch (err) {
    console.warn('qr failed', err)
  }
}

watch(() => props.url, render)
onMounted(render)
</script>
