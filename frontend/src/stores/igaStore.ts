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
  type IgaChatLog,
  reorderIgaQuestions,
  retryIgaChatMessage,
  type SendChatPayload,
  sendIgaChatMessage,
  updateIgaQuestion,
  type UpdateIgaScriptQuestionPayload,
  updateIgaSession,
  type UpdateIgaSessionPayload,
} from '@/api/igaService.ts'
import {useRoute} from "vue-router";
import {computed} from "vue";

export const useIgaStore = defineStore('iga', () => {
  const cache = useQueryCache()
  const route = useRoute()

  // --- IGA Sessions ---
  const igaSessionsQuery = useQuery({
    key: ['iga', 'sessions'],
    query: fetchIgaSessions,
    refetchOnMount: true,
  })

  const createIgaSessionMutation = useMutation({
    mutation: (payload: CreateIgaSessionPayload) => createIgaSession(payload),
    onSuccess: () => cache.invalidateQueries({key: ['iga', 'sessions']}),
  })

  const activeIgaSessionQuery = useQuery({
    key: () => ['iga', 'session', route.params.id],
    query: () => fetchIgaSession(route.params.id as string),
    enabled: computed(() => !!route.params.id),
    refetchOnMount: true,
    staleTime: 0,
  })

  const updateIgaSessionMutation = useMutation({
    mutation: ({id, payload}: {
      id: string;
      payload: UpdateIgaSessionPayload
    }) => updateIgaSession(id, payload),
    onSuccess: () => {
      cache.invalidateQueries({key: ['iga', 'sessions']})
      cache.invalidateQueries({key: ['iga', 'session', route.params.id]})
    },
  })

  // --- IGA Chat Logs ---
  const igaChatLogsQuery = useQuery({
    key: ['iga', 'session', route.params.id, 'chat'],
    query: () => fetchIgaChatLog(route.params.id as string),
    enabled: computed(() => !!route.params.id),
    staleTime: 0,
  })

  const sendIgaChatMessageMutation = useMutation({
    mutation: ({sessionId, payload}: {
      sessionId: string;
      payload: SendChatPayload
    }) => sendIgaChatMessage(sessionId, payload),
    onMutate: async ({sessionId, payload}) => {
      const queryKey = ['iga', 'session', sessionId, 'chat']
      const oldLogs = cache.getQueryData<IgaChatLog[]>(queryKey)
      if (oldLogs === undefined) {
        return
      }

      // optimistically update the chat logs
      const newMessage: IgaChatLog = {
        id: crypto.randomUUID(),
        iga_session_id: sessionId,
        role: 'user',
        message_text: payload.message_text,
        metadata: {type: payload.instruction},
        created_at: new Date().toISOString(),
        is_optimistic: true,
      }

      const newLogs: IgaChatLog[] = [
        ...(oldLogs || []),
        newMessage
      ]
      cache.setQueryData(queryKey, newLogs)
      cache.cancelQueries({key: queryKey})

      return {newLogs, oldLogs, newMessage}
    },
    onSettled: (_data, _err, _var, {oldLogs, newMessage}) => {
      if (newMessage) {
        const queryKey = ['iga', 'session', newMessage.iga_session_id, 'chat']
        cache.invalidateQueries({key: queryKey})
      }

      if (oldLogs === undefined) {
        igaChatLogsQuery.refetch()
      }
    },
  })

  const retryIgaChatMessageMutation = useMutation({
    mutation: (sessionId: string) => retryIgaChatMessage(sessionId),
    onSuccess: () => {
      cache.invalidateQueries({key: ['iga', 'session', route.params.id]})
    },
  })

  // --- IGA Questions ---
  const igaQuestionsQuery = useQuery({
    key: ['iga', 'session', route.params.id, 'questions'],
    query: () => fetchIgaQuestions(route.params.id as string),
    enabled: computed(() => !!route.params.id),
    staleTime: 0,
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

  const reorderIgaQuestionsMutation = useMutation({
    mutation: ({sessionId, orderedQuestionIds}: {
      sessionId: string;
      orderedQuestionIds: string[]
    }) => reorderIgaQuestions(sessionId, orderedQuestionIds),
    onSuccess: () => igaQuestionsQuery.refetch(),
  })

  return {
    // State
    activeSession: activeIgaSessionQuery.data,
    igaSessions: igaSessionsQuery.data,
    igaChatLogs: igaChatLogsQuery.data,
    igaQuestions: igaQuestionsQuery.data,

    // Actions
    createIgaSession: createIgaSessionMutation.mutateAsync,
    updateIgaSession: updateIgaSessionMutation.mutateAsync,
    sendIgaChatMessage: sendIgaChatMessageMutation.mutateAsync,
    retryIgaChatMessage: retryIgaChatMessageMutation.mutateAsync,
    createIgaQuestion: createIgaQuestionMutation.mutateAsync,
    updateIgaQuestion: updateIgaQuestionMutation.mutateAsync,
    deleteIgaQuestion: deleteIgaQuestionMutation.mutateAsync,
    reorderIgaQuestions: reorderIgaQuestionsMutation.mutateAsync,

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
    isReorderingQuestions: reorderIgaQuestionsMutation.isLoading,
    isAnyActiveSessionStateLoading: computed(() => {
      return activeIgaSessionQuery.isLoading.value ||
        igaChatLogsQuery.isLoading.value ||
        igaQuestionsQuery.isLoading.value ||
        createIgaSessionMutation.isLoading.value ||
        sendIgaChatMessageMutation.isLoading.value ||
        retryIgaChatMessageMutation.isLoading.value ||
        updateIgaSessionMutation.isLoading.value ||
        updateIgaQuestionMutation.isLoading.value ||
        createIgaQuestionMutation.isLoading.value ||
        deleteIgaQuestionMutation.isLoading.value ||
        reorderIgaQuestionsMutation.isLoading.value
    }),

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
    errorReorderingQuestions: reorderIgaQuestionsMutation.error,
  }
})
