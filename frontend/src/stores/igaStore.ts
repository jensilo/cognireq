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
      // optimistically update the chat logs
      const queryKey = ['iga', 'session', sessionId, 'chat']
      const newMessage: IgaChatLog = {
        id: crypto.randomUUID(),
        iga_session_id: sessionId,
        role: 'user',
        message_text: payload.message_text,
        metadata: {type: payload.instruction},
        created_at: new Date().toISOString(),
        is_optimistic: true,
      }

      const oldLogs = cache.getQueryData<IgaChatLog[]>(queryKey) || []
      console.log(queryKey, oldLogs)
      const newLogs: IgaChatLog[] = [
        ...(oldLogs),
        newMessage
      ]
      cache.setQueryData(queryKey, newLogs)
      cache.cancelQueries({key: queryKey})

      return {newLogs, oldLogs, newMessage}
    },
    onSettled: (_data, _err, _var, {newMessage}) => {
      if (!newMessage) {
        return
      }

      const queryKey = ['iga', 'session', newMessage.iga_session_id, 'chat']
      cache.invalidateQueries({key: queryKey})
    },
    /*onSuccess: (newLog, vars, context) => {
      const queryKey = ['iga', 'session', vars.sessionId, 'chat']
      const newLogs: IgaChatLog[] = [
        ...(context.oldLogs || []),
        newLog
      ]
      cache.setQueryData(queryKey, newLogs)

      return {newLogs, newLog}
    }*/
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
    onSuccess: () => cache.invalidateQueries({key: ['iga', 'session', route.params.id, 'questions']}),
  })

  const updateIgaQuestionMutation = useMutation({
    mutation:
      ({sessionId, questionId, payload}: {
        sessionId: string;
        questionId: string;
        payload: UpdateIgaScriptQuestionPayload
      }) => updateIgaQuestion(sessionId, questionId, payload),
    onSuccess: () => cache.invalidateQueries({key: ['iga', 'session', route.params.id, 'questions']}),
  })

  const deleteIgaQuestionMutation = useMutation({
    mutation: ({sessionId, questionId}: {
      sessionId: string;
      questionId: string
    }) => deleteIgaQuestion(sessionId, questionId),
    onSuccess: () => cache.invalidateQueries({key: ['iga', 'session', route.params.id, 'questions']}),
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
