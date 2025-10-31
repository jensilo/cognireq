import {defineStore} from 'pinia'
import {useMutation, useQuery, useQueryCache} from '@pinia/colada'
import {
  createIgaQuestion,
  type CreateIgaScriptQuestionPayload,
  createIgaSession,
  type CreateIgaSessionPayload,
  deleteIgaQuestion,
  fetchIgaChatLog,
  fetchIgaQuestions,
  fetchIgaSession,
  fetchIgaSessions,
  retryIgaChatMessage,
  type SendChatPayload,
  sendIgaChatMessage,
  updateIgaQuestion,
  type UpdateIgaScriptQuestionPayload,
  updateIgaSession,
  type UpdateIgaSessionPayload,
} from '@/api/igaService.ts'
import {computed, ref} from 'vue'

export const useIgaStore = defineStore('iga', () => {
  const cache = useQueryCache()
  const activeSessionId = ref<string | null>(null)

  // --- IGA Sessions ---
  const igaSessionsQuery = useQuery({
    key: ['iga', 'sessions'],
    query: fetchIgaSessions,
  })

  const createIgaSessionMutation = useMutation({
    mutation: (payload: CreateIgaSessionPayload) => createIgaSession(payload),
    onSuccess: (newSession) => {
      igaSessionsQuery.refetch().then()
      activeSessionId.value = newSession.id
    },
  })

  const activeIgaSessionQuery = useQuery({
    key: () => ['iga', 'session', activeSessionId.value],
    query: () => fetchIgaSession(activeSessionId.value ?? ''),
    enabled: computed(() => !!activeSessionId.value),
  })

  const updateIgaSessionMutation = useMutation({
    mutation: ({id, payload}: {
      id: string;
      payload: UpdateIgaSessionPayload
    }) => updateIgaSession(id, payload),
    onSuccess: () => {
      activeIgaSessionQuery.refetch().then()
      igaSessionsQuery.refetch().then()
    },
  })

  // --- IGA Chat Logs ---
  const igaChatLogsQuery = useQuery({
    key: ['iga', 'session', activeSessionId.value, 'chat'],
    query: () => fetchIgaChatLog(activeSessionId.value ?? ''),
    enabled: computed(() => !!activeSessionId.value),
  })

  const sendIgaChatMessageMutation = useMutation({
    mutation: ({sessionId, payload}: {
      sessionId: string;
      payload: SendChatPayload
    }) => sendIgaChatMessage(sessionId, payload),
    onSuccess: () => {
      igaChatLogsQuery.refetch().then()
      activeIgaSessionQuery.refetch().then() // session might update after chat message
      igaQuestionsQuery.refetch().then() // questions might update after chat message
    },
  })

  const retryIgaChatMessageMutation = useMutation({
    mutation: (sessionId: string) => retryIgaChatMessage(sessionId),
    onSuccess: () => {
      igaChatLogsQuery.refetch().then()
      activeIgaSessionQuery.refetch().then()
      igaQuestionsQuery.refetch().then()
    },
  })

  // --- IGA Questions ---
  const igaQuestionsQuery = useQuery({
    key: ['iga', 'session', activeSessionId.value, 'questions'],
    query: () => fetchIgaQuestions(activeSessionId.value ?? ''),
    enabled: computed(() => !!activeSessionId.value),
  })

  const createIgaQuestionMutation = useMutation({
    mutation: ({sessionId, payload}: {
      sessionId: string;
      payload: CreateIgaScriptQuestionPayload
    }) => createIgaQuestion(sessionId, payload),
    onSuccess: () => igaQuestionsQuery.refetch(),
  })

  const updateIgaQuestionMutation = useMutation({
    mutation:
      ({sessionId, questionId, payload}: {
        sessionId: string;
        questionId: string;
        payload: UpdateIgaScriptQuestionPayload
      }) => updateIgaQuestion(sessionId, questionId, payload),
    onSuccess: () => igaQuestionsQuery.refetch(),
  })

  const deleteIgaQuestionMutation = useMutation({
    mutation: ({sessionId, questionId}: {
      sessionId: string;
      questionId: string
    }) => deleteIgaQuestion(sessionId, questionId),
    onSuccess: () => igaQuestionsQuery.refetch(),
  })

  const selectSession = (id: string) => {
    activeSessionId.value = id
  }

  const clearActiveSession = () => {
    cache.invalidateQueries({ key: ['iga', 'session', activeSessionId.value] }).then()
    activeSessionId.value = null
  }

  return {
    // State
    activeSessionId,
    activeSession: activeIgaSessionQuery.data,
    igaSessions: igaSessionsQuery.data,
    igaChatLogs: igaChatLogsQuery.data,
    igaQuestions: igaQuestionsQuery.data,

    // Actions
    selectSession,
    clearActiveSession,
    createIgaSession: createIgaSessionMutation.mutateAsync,
    updateIgaSession: updateIgaSessionMutation.mutateAsync,
    sendIgaChatMessage: sendIgaChatMessageMutation.mutateAsync,
    retryIgaChatMessage: retryIgaChatMessageMutation.mutateAsync,
    createIgaQuestion: createIgaQuestionMutation.mutateAsync,
    updateIgaQuestion: updateIgaQuestionMutation.mutateAsync,
    deleteIgaQuestion: deleteIgaQuestionMutation.mutateAsync,

    // Loading states
    isLoadingSessions: igaSessionsQuery.isLoading,
    isLoadingActiveSession: activeIgaSessionQuery.isLoading,
    isLoadingChatLogs: igaChatLogsQuery.isLoading,
    isLoadingQuestions: igaQuestionsQuery.isLoading,
    isCreatingSession: createIgaSessionMutation.isLoading,
    isSendingChatMessage: sendIgaChatMessageMutation.isLoading,
    isRetryingChatMessage: retryIgaChatMessageMutation.isLoading,
    isUpdatingSession: updateIgaSessionMutation.isLoading,
    isUpdatingQuestion: updateIgaQuestionMutation.isLoading,
    isCreatingQuestion: createIgaQuestionMutation.isLoading,
    isDeletingQuestion: deleteIgaQuestionMutation.isLoading,

    // Errors
    errorSessions: igaSessionsQuery.error,
    errorActiveSession: activeIgaSessionQuery.error,
    errorChatLogs: igaChatLogsQuery.error,
    errorQuestions: igaQuestionsQuery.error,
    errorCreatingSession: createIgaSessionMutation.error,
    errorSendingChatMessage: sendIgaChatMessageMutation.error,
    errorRetryingChatMessage: retryIgaChatMessageMutation.error,
    errorUpdatingSession: updateIgaSessionMutation.error,
    errorUpdatingQuestion: updateIgaQuestionMutation.error,
    errorCreatingQuestion: createIgaQuestionMutation.error,
    errorDeletingQuestion: deleteIgaQuestionMutation.error,
  }
})
