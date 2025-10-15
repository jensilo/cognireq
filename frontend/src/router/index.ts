import {createRouter, createWebHistory} from 'vue-router'
import LandingPage from "@/views/LandingPage.vue";
import {type Composer, type I18n} from "vue-i18n";
import {useAuthStore} from "@/stores/authStore.ts";

const PUBLIC_AUTH_ROUTES = ['/', '/login', '/register']
const PUBLIC_ROUTES: string[] = ['/about'];

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      alias: '/login',
      name: 'login',
      component: LandingPage,
      meta: {
        title: 'login.title',
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
  const {t} = i18n.global as Composer;
  const authStore = useAuthStore();

  router.beforeEach((to, from, next) => {
    const titleKey = to.meta.title as string | undefined;
    if (titleKey) {
      document.title = t(titleKey);
    } else {
      document.title = t('app.title');
    }

    next();
  });

  router.beforeEach(async (to, from, next) => {
    await authStore.refresh()

    const isPublicAuthRoute = PUBLIC_AUTH_ROUTES.includes(to.path);
    const isPublic = isPublicAuthRoute || PUBLIC_ROUTES.includes(to.path);

    if (authStore.isAuthenticated && isPublicAuthRoute) {
      return next('app/')
    }

    if (!authStore.isAuthenticated && !isPublic) {
      return next('/');
    }

    next()
  })

  return router;
}
