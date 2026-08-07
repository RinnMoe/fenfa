<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <div class="brand-mark">F</div>
      <div class="brand-copy">
        <strong>{{ t.sidebar.title }}</strong>
        <span>{{ t.sidebar.subtitle }}</span>
      </div>
    </div>

    <nav class="sidebar-nav" aria-label="Primary">
      <router-link
        v-for="item in menuItems"
        :key="item.id"
        :to="'/' + item.id"
        class="nav-item"
        active-class="active"
      >
        <span class="nav-icon"><Icon :name="item.icon" :size="16" /></span>
        <span class="nav-text">{{ item.label }}</span>
      </router-link>
    </nav>

    <div class="sidebar-footer">
      <div class="footer-controls">
        <LanguageSwitcher />
        <button
          class="theme-toggle"
          type="button"
          :aria-label="isDark ? t.sidebar.lightMode : t.sidebar.darkMode"
          :title="isDark ? t.sidebar.lightMode : t.sidebar.darkMode"
          @click="toggleTheme"
        >
          <Icon :name="isDark ? 'sun' : 'moon'" :size="16" />
        </button>
      </div>
      <button type="button" class="btn-ghost btn-full" @click="$emit('logout')">
        <Icon name="log-out" :size="14" />
        <span>{{ t.common.logout }}</span>
      </button>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '../i18n'
import { useTheme } from '../theme'
import LanguageSwitcher from './LanguageSwitcher.vue'
import Icon from './Icon.vue'

defineEmits<{
  logout: []
}>()

const { t } = useI18n()
const { isDark, toggleTheme } = useTheme()

const menuItems = computed(() => [
  { id: 'apps', icon: 'smartphone', label: t.value.sidebar.apps },
  { id: 'upload', icon: 'upload', label: t.value.sidebar.upload },
  { id: 'stats', icon: 'bar-chart', label: t.value.sidebar.stats },
  { id: 'events', icon: 'clipboard', label: t.value.sidebar.events },
  { id: 'udid', icon: 'lock', label: t.value.sidebar.udid },
  { id: 'export', icon: 'save', label: t.value.sidebar.export },
  { id: 'settings', icon: 'settings', label: t.value.sidebar.settings }
])
</script>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  width: var(--sidebar-width);
  height: 100vh;
  flex-shrink: 0;
  position: sticky;
  top: 0;
  background: var(--surface);
  border-right: 1px solid var(--border);
}

.sidebar-header {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 52px;
  padding: 0 14px;
  border-bottom: 1px solid var(--border);
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--surface-subtle);
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 700;
}

.brand-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.2;
}

.brand-copy strong {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 600;
}

.brand-copy span {
  overflow: hidden;
  color: var(--text-muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-nav {
  flex: 1;
  padding: 8px;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 36px;
  padding: 0 10px;
  border-radius: var(--radius-control);
  color: var(--text-muted);
  font-size: 13px;
  text-decoration: none;
  transition: background-color 0.1s ease, color 0.1s ease;
}

.nav-item:hover {
  background: var(--surface-subtle);
  color: var(--text-secondary);
}

.nav-item.active {
  background: var(--surface-subtle);
  color: var(--accent);
  font-weight: 500;
}

.nav-icon {
  display: inline-flex;
  flex-shrink: 0;
}

.nav-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-footer {
  padding: 8px;
  border-top: 1px solid var(--border);
}

.footer-controls {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}

.theme-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-control);
  color: var(--text-muted);
  cursor: pointer;
  transition: background-color 0.12s ease, color 0.12s ease;
}

.theme-toggle:hover {
  background: var(--surface-subtle);
  color: var(--text-primary);
}

.sidebar-footer .btn-full {
  justify-content: flex-start;
  height: 32px;
  font-size: 12px;
}

</style>
