import { defineStore } from 'pinia'
import { useQuery } from '@pinia/colada'
import { fetchUserMe } from '@/api/userService.ts'
import { computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const userMeQuery = useQuery({
    key: ['user', 'me'],
    query: fetchUserMe,
    enabled: false,
  })

  const isAuthenticated = computed(() => {
    return !!userMeQuery.data.value
  })

  return {
    isAuthenticated,
    user: userMeQuery.data,
    ...userMeQuery,
  }
})
