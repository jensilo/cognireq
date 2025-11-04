<script setup lang="ts">
import { ref, computed } from 'vue'
import { useIgaStore } from '@/stores/igaStore.ts'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import IgaContextEditableField from '@/components/IgaContextEditableField.vue'
import type { IgaScriptQuestion, UpdateIgaScriptQuestionPayload } from '@/api/igaService.ts'

const props = defineProps<{ sessionId: string; setChatInput: (text: string, instruction: string) => void; refineContext: (contextType: string) => void; updateSessionContext: (field: 'stakeholder_summary' | 'project_summary' | 'interview_goal' | 'interview_advice', value: string) => void }>()

const igaStore = useIgaStore()

const activeSession = computed(() => igaStore.activeSession)
const igaQuestions = computed(() => igaStore.igaQuestions)
const isLoadingActiveSession = computed(() => igaStore.isLoadingActiveSession)
const isLoadingQuestions = computed(() => igaStore.isLoadingQuestions)
const isSendingMessage = computed(() => igaStore.isSendingChatMessage || igaStore.isRetryingChatMessage)

const editingQuestionId = ref<string | null>(null)
const editedQuestionText = ref('')
const editedNoteText = ref('')

const startEditingQuestion = (question: IgaScriptQuestion) => {
  if (igaStore.isSendingChatMessage) return // prevent editing while chat is sending
  editingQuestionId.value = question.id
  editedQuestionText.value = question.question_text
  editedNoteText.value = question.note_text
}

const saveQuestionEdit = async (questionId: string) => {
  if (!props.sessionId || !questionId || igaStore.isSendingChatMessage) return

  const payload: UpdateIgaScriptQuestionPayload = {
    question_text: editedQuestionText.value,
    note_text: editedNoteText.value,
  }

  await igaStore.updateIgaQuestion({ sessionId: props.sessionId, questionId, payload })
  editingQuestionId.value = null
}

const cancelQuestionEdit = () => {
  editingQuestionId.value = null
}

const citeQuestion = (readableId: string) => {
  if (igaStore.isSendingChatMessage) return // prevent citing while chat is sending
  props.setChatInput(`Concerning question ${readableId}: `, 'refine-question')
}
</script>

<template>
  <div class="flex flex-col h-full bg-white p-4 overflow-y-auto">
    <LoadingSpinner v-if="isLoadingQuestions && !activeSession" />

    <div v-else-if="activeSession">
      <!-- Interview Goal -->
      <IgaContextEditableField
        label="Interview Goal"
        :model-value="activeSession.context_data.interview_goal"
        @updateField="props.updateSessionContext('interview_goal', $event)"
        :on-refine="props.refineContext"
        context-type="interview goal"
        class="mb-6"
        :loading="isLoadingActiveSession || isSendingMessage"
      />

      <!-- General Interview Advice -->
      <IgaContextEditableField
        label="General Interview Advice"
        :model-value="activeSession.context_data.interview_advice"
        @updateField="props.updateSessionContext('interview_advice', $event)"
        :on-refine="props.refineContext"
        context-type="general interview advice"
        class="mb-6"
        :loading="isLoadingActiveSession || isSendingMessage"
      />

      <!-- Questions/Notes List -->
      <div class="mb-6">
        <h2 class="text-lg font-semibold text-gray-800 mb-4">Questions & Notes</h2>
        <div v-if="igaQuestions && igaQuestions.length > 0" class="space-y-4">
          <div v-for="question in igaQuestions" :key="question.id" class="p-4 border border-gray-200 rounded-lg bg-white shadow-sm">
            <div class="flex justify-between items-start mb-2">
              <h3 class="font-medium text-indigo-700">{{ question.readable_id }}</h3>
              <div class="flex space-x-2">
                <button
                  @click="citeQuestion(question.readable_id)"
                  class="text-sm text-indigo-600 hover:text-indigo-800 cursor-pointer"
                  :disabled="igaStore.isSendingChatMessage"
                >
                  Cite
                </button>
                <button
                  v-if="editingQuestionId !== question.id"
                  @click="startEditingQuestion(question)"
                  class="text-sm text-neutral-600 hover:text-neutral-800 cursor-pointer"
                  :disabled="igaStore.isSendingChatMessage"
                >
                  Edit
                </button>
                <button
                  v-else
                  @click="saveQuestionEdit(question.id)"
                  class="text-sm text-green-600 hover:text-green-800 cursor-pointer"
                  :disabled="igaStore.isSendingChatMessage"
                >
                  Save
                </button>
                <button
                  v-if="editingQuestionId === question.id"
                  @click="cancelQuestionEdit"
                  class="text-sm text-red-600 hover:text-red-800 cursor-pointer"
                  :disabled="igaStore.isSendingChatMessage"
                >
                  Cancel
                </button>
              </div>
            </div>

            <div v-if="editingQuestionId === question.id">
              <label class="block text-sm font-medium text-neutral-700">Question Text</label>
              <textarea
                v-model="editedQuestionText"
                class="w-full p-2 border rounded-md mb-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-700"
                rows="2"
                :disabled="igaStore.isSendingChatMessage"
              ></textarea>
              <label class="block text-sm font-medium text-neutral-700">Note Text</label>
              <textarea
                v-model="editedNoteText"
                class="w-full p-2 border rounded-md focus:ring-indigo-500 focus:border-indigo-500 text-gray-700"
                rows="2"
                :disabled="igaStore.isSendingChatMessage"
              ></textarea>
            </div>
            <div v-else>
              <p class="text-neutral-800 mb-1">{{ question.question_text }}</p>
              <p class="text-sm text-neutral-600 italic">Note: {{ question.note_text }}</p>
            </div>
          </div>
        </div>
        <div v-else class="text-neutral-500 italic p-4 border border-gray-200 rounded-lg bg-white shadow-sm">
          No questions generated yet. Start a conversation in the chat to generate some!
        </div>
      </div>
    </div>
    <div v-else class="text-center text-neutral-500 p-4">
      Select or create an IGA session to view the questionnaire artifact.
    </div>
  </div>
</template>
