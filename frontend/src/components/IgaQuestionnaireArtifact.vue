<script setup lang="ts">
import {computed, nextTick, ref} from 'vue'
import {useIgaStore} from '@/stores/igaStore.ts'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import IgaContextEditableField from '@/components/IgaContextEditableField.vue'
import type {
  CreateIgaScriptQuestionPayload,
  IgaChatLogInstructionType,
  IgaScriptQuestion,
  UpdateIgaScriptQuestionPayload
} from '@/api/igaService.ts'
import {useI18n} from 'vue-i18n'
import ErrorMessage from '@/components/Base/ErrorMessage.vue'

const props = defineProps<{
  sessionId: string;
  setChatInput: (text: string, instruction: IgaChatLogInstructionType) => void;
  refineContext: (contextType: string) => void;
  updateSessionContext: (field: 'stakeholder_summary' | 'project_summary' | 'interview_goal' | 'interview_advice', value: string) => void
}>()

const igaStore = useIgaStore()
const {t} = useI18n()

const activeSession = computed(() => igaStore.activeSession)
const igaQuestions = computed(() => {
  if (!igaStore.igaQuestions) return []
  return [...igaStore.igaQuestions].sort((a, b) => a.sort_order - b.sort_order)
})
const isLoadingQuestions = computed(() => igaStore.isLoadingQuestions)
const isIgaLoading = computed(() => igaStore.isAnyActiveSessionStateLoading)

const editingQuestionId = ref<string | null>(null)
const editedQuestionText = ref('')
const editedNoteText = ref('')

const isDeleteConfirmationModalOpen = ref(false)
const questionToDeleteId = ref<string | null>(null)

const questionnaireRoot = ref<HTMLElement | null>(null)

const startEditingQuestion = (question: IgaScriptQuestion) => {
  if (isIgaLoading.value) return // prevent editing while chat is sending
  editingQuestionId.value = question.id
  editedQuestionText.value = question.question_text
  editedNoteText.value = question.note_text
}

const saveQuestionEdit = async (questionId: string) => {
  if (!props.sessionId || !questionId || isIgaLoading.value) return

  const payload: UpdateIgaScriptQuestionPayload = {
    question_text: editedQuestionText.value,
    note_text: editedNoteText.value,
  }

  igaStore.resetUpdateIgaQuestionError() // reset error
  await igaStore.updateIgaQuestion({sessionId: props.sessionId, questionId, payload})
  editingQuestionId.value = null
}

const cancelQuestionEdit = () => {
  editingQuestionId.value = null
}

const addQuestion = async () => {
  if (!props.sessionId || isIgaLoading.value) return

  const sortOrder = igaQuestions.value.length > 0 ? igaQuestions.value[igaQuestions.value.length - 1].sort_order + 1 : 0;

  const payload: CreateIgaScriptQuestionPayload = {
    question_text: t('iga.questionnaire.new-question-placeholder'),
    note_text: '',
    sort_order: sortOrder,
  }

  igaStore.resetCreateIgaQuestionError() // reset error
  await igaStore.createIgaQuestion({sessionId: props.sessionId, payload})
  editingQuestionId.value = null // clear any active editing after adding a new question

  nextTick(() => {
    if (questionnaireRoot.value) {
      questionnaireRoot.value.scrollTop = questionnaireRoot.value.scrollHeight
    }
  })
}

const showDeleteConfirmation = (questionId: string) => {
  questionToDeleteId.value = questionId
  isDeleteConfirmationModalOpen.value = true
}

const cancelDelete = () => {
  isDeleteConfirmationModalOpen.value = false
  questionToDeleteId.value = null
}

const confirmDelete = async () => {
  if (!props.sessionId || !questionToDeleteId.value) return
  igaStore.resetDeleteIgaQuestionError() // reset error
  await igaStore.deleteIgaQuestion({
    sessionId: props.sessionId,
    questionId: questionToDeleteId.value
  })
  cancelDelete()
}

const moveQuestion = async (questionId: string, direction: 'up' | 'down') => {
  if (!props.sessionId || isIgaLoading.value) return

  const questions = igaQuestions.value
  if (!questions) return

  const index = questions.findIndex(q => q.id === questionId)
  if (index === -1) return

  const targetIndex = direction === 'up' ? index - 1 : index + 1

  if (targetIndex < 0 || targetIndex >= questions.length) return

  // create a new array with the questions reordered locally
  const reorderedQuestions = [...questions]
  const [movedQuestion] = reorderedQuestions.splice(index, 1)
  reorderedQuestions.splice(targetIndex, 0, movedQuestion)

  // extract the IDs in the new desired order
  const orderedQuestionIds = reorderedQuestions.map(q => q.id)

  igaStore.resetReorderIgaQuestionsError() // reset error
  // dispatch a single action to update the backend with the new order
  await igaStore.reorderIgaQuestions({ sessionId: props.sessionId, orderedQuestionIds })
}

const moveQuestionUp = (questionId: string) => {
  moveQuestion(questionId, 'up')
}

const moveQuestionDown = (questionId: string) => {
  moveQuestion(questionId, 'down')
}

const citeQuestion = (readableId: string) => {
  if (isIgaLoading.value) return // prevent citing while chat is sending
  props.setChatInput(`Concerning question ${readableId}: `, 'refine-question')
}
</script>

<template>
  <div class="flex flex-col h-full bg-white p-4 overflow-y-auto" ref="questionnaireRoot">
    <ErrorMessage :message="igaStore.errorCreatingQuestion?.message || igaStore.errorUpdatingQuestion?.message || igaStore.errorDeletingQuestion?.message || igaStore.errorReorderingQuestions?.message" />

    <LoadingSpinner v-if="isLoadingQuestions && !activeSession"/>

    <div v-else-if="activeSession">
      <!-- Interview Goal -->
      <IgaContextEditableField
        label="Interview Goal"
        :model-value="activeSession.context_data.interview_goal"
        @updateField="props.updateSessionContext('interview_goal', $event)"
        :on-refine="props.refineContext"
        context-type="interview goal"
        class="mb-6"
        :loading="isIgaLoading"
      />
      <!-- General Interview Advice -->
      <IgaContextEditableField
        label="General Interview Advice"
        :model-value="activeSession.context_data.interview_advice"
        @updateField="props.updateSessionContext('interview_advice', $event)"
        :on-refine="props.refineContext"
        context-type="general interview advice"
        class="mb-6"
        :loading="isIgaLoading"
      />

      <!-- Questions/Notes List -->
      <div v-if="igaQuestions.length < 1 && igaStore.errorQuestions" class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg text-sm" role="alert">
        {{ $t(`error.${igaStore.errorQuestions?.message}`) }}
      </div>

      <div v-else class="mb-6">
        <h2 class="text-lg font-semibold text-gray-800 mb-4">
          {{ $t('iga.questionnaire.questions-notes-heading') }}</h2>
        <button
          @click="addQuestion"
          class="mb-4 px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 cursor-pointer disabled:cursor-progress disabled:bg-indigo-400"
          :disabled="isIgaLoading"
        >
          {{ $t('iga.questionnaire.add-new-question-button') }}
        </button>
        <div v-if="igaQuestions && igaQuestions.length > 0" class="space-y-4">
          <div v-for="question in igaQuestions" :key="question.id"
               class="p-4 border border-gray-200 rounded-lg bg-white shadow-sm">
            <div class="flex justify-between items-start mb-2">
              <h3 class="font-medium text-indigo-700">{{ question.readable_id }}</h3>
              <div class="flex space-x-2">
                <button
                  @click="moveQuestionUp(question.id)"
                  v-if="question.sort_order > 0"
                  class="text-sm text-neutral-600 hover:text-neutral-800 cursor-pointer disabled:cursor-progress"
                  :disabled="isIgaLoading || editingQuestionId === question.id"
                >
                  {{ $t('iga.questionnaire.move-up-button') }}
                </button>
                <button
                  @click="moveQuestionDown(question.id)"
                  v-if="question.sort_order < igaQuestions.length - 1"
                  class="text-sm text-neutral-600 hover:text-neutral-800 cursor-pointer disabled:cursor-progress"
                  :disabled="isIgaLoading || editingQuestionId === question.id"
                >
                  {{ $t('iga.questionnaire.move-down-button') }}
                </button>
                <button
                  @click="citeQuestion(question.readable_id)"
                  class="text-sm text-indigo-600 hover:text-indigo-800 cursor-pointer disabled:cursor-progress"
                  :disabled="isIgaLoading"
                >
                  {{ $t('iga.questionnaire.cite-button') }}
                </button>
                <button
                  v-if="editingQuestionId !== question.id"
                  @click="startEditingQuestion(question)"
                  class="text-sm text-neutral-600 hover:text-neutral-800 cursor-pointer disabled:cursor-progress"
                  :disabled="isIgaLoading"
                >
                  {{ $t('iga.questionnaire.edit-button') }}
                </button>
                <button
                  v-if="editingQuestionId !== question.id"
                  @click="showDeleteConfirmation(question.id)"
                  class="text-sm text-red-600 hover:text-red-800 cursor-pointer disabled:cursor-progress"
                  :disabled="isIgaLoading"
                >
                  {{ $t('iga.questionnaire.delete-button') }}
                </button>
                <button
                  v-else
                  @click="saveQuestionEdit(question.id)"
                  class="text-sm text-green-600 hover:text-green-800 cursor-pointer disabled:cursor-progress"
                  :disabled="isIgaLoading"
                >
                  {{ $t('iga.questionnaire.save-button') }}
                </button>
                <button
                  v-if="editingQuestionId === question.id"
                  @click="cancelQuestionEdit"
                  class="text-sm text-red-600 hover:text-red-800 cursor-pointer disabled:cursor-progress"
                  :disabled="isIgaLoading"
                >
                  {{ $t('iga.questionnaire.cancel-button') }}
                </button>
              </div>
            </div>

            <div v-if="editingQuestionId === question.id">
              <label class="block text-sm font-medium text-neutral-700">{{
                  $t('iga.questionnaire.question-text-label')
                }}</label>
              <textarea
                v-model="editedQuestionText"
                class="w-full p-2 border rounded-md mb-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-700 disabled:cursor-progress disabled:bg-gray-100"
                rows="2"
                :disabled="isIgaLoading"
              ></textarea>
              <label class="block text-sm font-medium text-neutral-700">{{
                  $t('iga.questionnaire.note-text-label')
                }}</label>
              <textarea
                v-model="editedNoteText"
                class="w-full p-2 border rounded-md focus:ring-indigo-500 focus:border-indigo-500 text-gray-700 disabled:cursor-progress disabled:bg-gray-100"
                rows="2"
                :disabled="isIgaLoading"
              ></textarea>
            </div>
            <div v-else>
              <p class="text-neutral-800 mb-1">{{ question.question_text }}</p>
              <p class="text-sm text-neutral-600 italic">Note: {{ question.note_text }}</p>
            </div>
          </div>
        </div>
        <div v-else-if="!igaStore.errorQuestions"
             class="text-neutral-500 italic p-4 border border-gray-200 rounded-lg bg-white shadow-sm">
          {{ $t('iga.questionnaire.no-questions-message') }}
        </div>
      </div>
    </div>
    <div v-else class="text-center text-neutral-500 p-4">
      {{ $t('iga.questionnaire.select-session-message') }}
    </div>

    <!-- Delete Confirmation Modal -->
    <div v-if="isDeleteConfirmationModalOpen"
         class="fixed inset-0 bg-[rgba(75,85,99,0.5)] overflow-y-auto h-full w-full flex items-center justify-center">
      <div class="bg-white p-8 rounded-lg shadow-xl max-w-sm mx-auto">
        <h3 class="text-lg font-bold mb-4">{{ $t('iga.questionnaire.confirm-deletion-title') }}</h3>
        <p class="mb-6">{{ $t('iga.questionnaire.confirm-deletion-message') }}</p>
        <div class="flex justify-end space-x-4">
          <button
            @click="cancelDelete"
            class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
          >
            {{ $t('iga.questionnaire.modal-cancel-button') }}
          </button>
          <button
            @click="confirmDelete"
            class="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2"
          >
            {{ $t('iga.questionnaire.modal-delete-button') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
