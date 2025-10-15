import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import {createI18n, type I18n} from "vue-i18n";

import {defaultLocale, languages} from "@/locale";
import {setupRouter} from "@/router";

const i18n = createI18n({
  locale: defaultLocale,
  fallbackLocale: 'en',
  messages: languages,
}) as I18n
const app = createApp(App)
app.use(createPinia())
const router = setupRouter(i18n);
app.use(router)
app.use(i18n)

app.mount('#app')
