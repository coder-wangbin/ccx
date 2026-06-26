import { createApp } from 'vue'
import App from './App.vue'
import i18n from './i18n/vue-i18n'
import { installFrontendErrorReporter } from './lib/frontend-error-reporter'
import './assets/index.css'

installFrontendErrorReporter()

const app = createApp(App)
app.use(i18n)
app.mount('#app')
