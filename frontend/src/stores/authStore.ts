import { defineStore } from 'pinia'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import { fetchUserMe, loginUser, logoutUser } from '@/api/userService.ts'
import { computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const cache = useQueryCache()

  const userMeQuery = useQuery({
    key: ['user', 'me'],
    query: fetchUserMe,
    staleTime: Infinity,
    enabled: false,
  })

  const loginMutation = useMutation({
    mutation: (cred: { email: string; password: string }) => loginUser(cred.email, cred.password),
    onSuccess: () => userMeQuery.refetch(),
  })

  const logoutMutation = useMutation({
    mutation: () => logoutUser(),
    onSuccess: () => cache.setQueryData(['user', 'me'], undefined),
  })

  const isAuthenticated = computed(() => {
    return !!userMeQuery.data.value
  })

  return {
    isAuthenticated,
    user: userMeQuery.data,
    login: loginMutation.mutateAsync,
    loginMutation: loginMutation,
    logout: logoutMutation.mutateAsync,
    logoutMutation: logoutMutation,
    ...userMeQuery,
  }
})
