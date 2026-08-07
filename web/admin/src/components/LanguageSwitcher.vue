<template>
  <div class="language-switcher">
    <button 
      @click="toggleLanguage" 
      class="lang-button"
      :title="locale === 'zh' ? 'Switch to English' : '切换到中文'"
    >
      <span class="lang-icon"><Icon name="globe" :size="16" /></span>
      <span class="lang-text">{{ locale === 'zh' ? '中文' : 'EN' }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from '../i18n'
import Icon from './Icon.vue'

const { locale, setLocale } = useI18n()

function toggleLanguage() {
  setLocale(locale.value === 'zh' ? 'en' : 'zh')
}
</script>

<style scoped>
.language-switcher {
  display: flex;
  align-items: center;
}

.lang-button {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 8px;
  background: transparent;
  color: var(--text-muted);
  border: 1px solid transparent;
  border-radius: var(--radius-control);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.12s ease, color 0.12s ease;
}

.lang-button:hover {
  background: var(--surface-subtle);
  color: var(--text-primary);
}

.lang-icon {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.lang-text {
  font-weight: 500;
}
</style>

