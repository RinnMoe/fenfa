import { createRouter, createWebHashHistory } from 'vue-router'

const AppsPage = () => import('./pages/AppsPage.vue')
const UploadPage = () => import('./pages/UploadPage.vue')
const StatsPage = () => import('./pages/StatsPage.vue')
const EventsPage = () => import('./pages/EventsPage.vue')
const UdidPage = () => import('./pages/UdidPage.vue')
const ExportPage = () => import('./pages/ExportPage.vue')
const SettingsPage = () => import('./pages/SettingsPage.vue')

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/apps' },
    { path: '/apps', component: AppsPage },
    { path: '/upload', component: UploadPage },
    { path: '/stats', component: StatsPage },
    { path: '/stats/:variantId', component: StatsPage, props: true },
    { path: '/events', component: EventsPage },
    { path: '/events/:variantId', component: EventsPage, props: true },
    { path: '/udid', component: UdidPage },
    { path: '/export', component: ExportPage },
    { path: '/settings', component: SettingsPage },
  ],
})

export default router
