import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './styles/tokens.css'
import './styles/base.css'
import './styles/layout.css'
import './styles/controls.css'
import './styles/forms.css'
import './styles/tables.css'
import './styles/overlays.css'
import './styles/utilities.css'

createApp(App).use(router).mount('#app')
