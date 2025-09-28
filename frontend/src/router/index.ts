import { createRouter, createWebHistory } from 'vue-router'
import LandingPage from "@/views/LandingPage.vue";
import {type Composer, type I18n} from "vue-i18n";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'login',
      component: LandingPage,
      meta: {
        title: 'login.title'
      }
    },
    {
      path: '/about',
      name: 'about',
      component: () => import('../views/AboutView.vue'),
    },
  ],
})

export function setupRouter(i18n: I18n) {
  const { t } = i18n.global as Composer;

  router.beforeEach((to, from, next) => {
    const titleKey = to.meta.title as string | undefined;
    if (titleKey) {
      document.title = t(titleKey);
    } else {
      document.title = t('app.title');
    }

    next();
  });

  return router;
}
