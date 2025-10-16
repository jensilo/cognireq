import { defineStore } from 'pinia'
import {useMutation, useQuery} from '@pinia/colada'
import {fetchUserMe, loginUser} from '@/api/userService.ts'
import { computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const userMeQuery = useQuery({
    key: ['user', 'me'],
    query: fetchUserMe,
    enabled: false,
  })

  const loginMutation = useMutation({
    mutation: (cred: {email: string, password: string}) => loginUser(cred.email, cred.password),
    onSuccess: () => userMeQuery.refresh(),
  })

  const isAuthenticated = computed(() => {
    return !!userMeQuery.data.value
  })

  return {
    isAuthenticated,
    user: userMeQuery.data,
    login: loginMutation.mutateAsync,
    isLoadingLogin: loginMutation.isLoading,
    ...userMeQuery,
  }
})
