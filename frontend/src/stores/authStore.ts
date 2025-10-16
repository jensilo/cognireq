import { defineStore } from 'pinia'
import {useMutation, useQuery, useQueryCache, type UseQueryEntry} from '@pinia/colada'
import {fetchUserMe, loginUser, logoutUser} from '@/api/userService.ts'
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
    onSuccess: () => userMeQuery.refresh(),
  })

  const logoutMutation = useMutation({
    mutation: () => logoutUser(),
  })

  const isAuthenticated = computed(() => {
    return !!userMeQuery.data.value
  })

  const logout = async () => {
    await logoutMutation.mutateAsync()
    if (logoutMutation.status.value === 'error') {
      console.error('logout should never error')
      return
    }

    cache.setQueryData(['user', 'me'], undefined)
  }

  return {
    isAuthenticated,
    user: userMeQuery.data,
    login: loginMutation.mutateAsync,
    loginMutation: loginMutation,
    logout: logout,
    logoutMutation: logoutMutation,
    ...userMeQuery,
  }
})
