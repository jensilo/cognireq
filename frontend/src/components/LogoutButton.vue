<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore.ts'

defineProps<{ isCollapsed: boolean; collapseClass: string }>()
const router = useRouter()
const { logout } = useAuthStore()

async function handleLogout() {
  await logout()
  await router.push('/')
}
</script>

<template>
  <button
    @click="handleLogout"
    class="w-full flex items-center p-3 text-sm font-medium text-red-400 hover:bg-gray-700 rounded-lg transition-colors duration-150 cursor-pointer"
    :class="isCollapsed ? 'justify-center' : 'justify-start'"
    aria-label="Logout"
  >
    <i-heroicons-arrow-right-start-on-rectangle
      class="text-xl shrink-0"
      :class="{ 'mr-3': !isCollapsed }"
    />
    <span class="whitespace-nowrap transition-all duration-300" :class="collapseClass">
      {{ $t('nav.logout') }}
    </span>
  </button>
</template>
