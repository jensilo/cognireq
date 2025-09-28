import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'
import {createI18n} from "vue-i18n";

import {defaultLocale, languages} from "@/locale";

const i18n = createI18n({
  locale: defaultLocale,
  fallbackLocale: 'en',
  messages: languages,
})
const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(i18n)

app.mount('#app')
