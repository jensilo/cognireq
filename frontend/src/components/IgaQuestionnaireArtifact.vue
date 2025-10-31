<script setup lang="ts">
import { ref, computed } from 'vue'
import { useIgaStore } from '@/stores/igaStore.ts'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import type {
  IgaChatLogInstructionType,
  IgaScriptQuestion,
  UpdateIgaScriptQuestionPayload,
  UpdateIgaSessionPayload
} from '@/api/igaService.ts'

const props = defineProps<{ sessionId: string; setChatInput: (text: string, instruction: IgaChatLogInstructionType) => void }>()

const igaStore = useIgaStore()

const activeSession = computed(() => igaStore.activeSession)
const igaQuestions = computed(() => igaStore.igaQuestions)
const isLoadingQuestions = computed(() => igaStore.isLoadingQuestions)

const editingQuestionId = ref<string | null>(null)
const editedQuestionText = ref('')
const editedNoteText = ref('')

const startEditingQuestion = (question: IgaScriptQuestion) => {
  editingQuestionId.value = question.id
  editedQuestionText.value = question.question_text
  editedNoteText.value = question.note_text
}

const saveQuestionEdit = async (questionId: string) => {
  if (!props.sessionId || !questionId) return

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
  props.setChatInput(`Concerning question ${readableId}: `, 'refine-question')
}

const refineContext = (contextType: string) => {
  props.setChatInput(`Refine the ${contextType}: `, 'refine-context')
}

const updateSessionContext = async (field: 'interview_goal' | 'interview_advice', value: string) => {
  if (!props.sessionId || !activeSession.value) return

  const currentContext = activeSession.value.context_data
  const payload: UpdateIgaSessionPayload = {
    context_data: { ...currentContext, [field]: value },
  }
  await igaStore.updateIgaSession({ id: props.sessionId, payload })
}
</script>

<template>
  <div class="flex flex-col h-full bg-white p-4 overflow-y-auto">
    <LoadingSpinner v-if="isLoadingQuestions" />

    <div v-else-if="activeSession">
      <!-- Interview Goal -->
      <div class="mb-6 p-4 border rounded-lg bg-neutral-50">
        <h2 class="text-lg font-semibold mb-2 flex items-center justify-between">
          Interview Goal
          <button @click="refineContext('interview goal')" class="text-indigo-600 hover:text-indigo-800 text-sm">
            AI Enhance
          </button>
        </h2>
        <textarea
          v-model="activeSession.context_data.interview_goal"
          @blur="updateSessionContext('interview_goal', activeSession.context_data.interview_goal)"
          class="w-full p-2 border rounded-md focus:ring-indigo-500 focus:border-indigo-500"
          rows="3"
        ></textarea>
      </div>

      <!-- General Interview Advice -->
      <div class="mb-6 p-4 border rounded-lg bg-neutral-50">
        <h2 class="text-lg font-semibold mb-2 flex items-center justify-between">
          General Interview Advice
          <button @click="refineContext('general interview advice')" class="text-indigo-600 hover:text-indigo-800 text-sm">
            AI Enhance
          </button>
        </h2>
        <textarea
          v-model="activeSession.context_data.interview_advice"
          @blur="updateSessionContext('interview_advice', activeSession.context_data.interview_advice)"
          class="w-full p-2 border rounded-md focus:ring-indigo-500 focus:border-indigo-500"
          rows="3"
        ></textarea>
      </div>

      <!-- Questions/Notes List -->
      <div class="mb-6">
        <h2 class="text-lg font-semibold mb-4">Questions & Notes</h2>
        <div v-if="igaQuestions && igaQuestions.length > 0" class="space-y-4">
          <div v-for="question in igaQuestions" :key="question.id" class="p-4 border rounded-lg bg-white shadow-sm">
            <div class="flex justify-between items-start mb-2">
              <h3 class="font-medium text-indigo-700">{{ question.readable_id }}</h3>
              <div class="flex space-x-2">
                <button
                  @click="citeQuestion(question.readable_id)"
                  class="text-sm text-indigo-600 hover:text-indigo-800"
                >
                  Cite
                </button>
                <button
                  v-if="editingQuestionId !== question.id"
                  @click="startEditingQuestion(question)"
                  class="text-sm text-neutral-600 hover:text-neutral-800"
                >
                  Edit
                </button>
                <button
                  v-else
                  @click="saveQuestionEdit(question.id)"
                  class="text-sm text-green-600 hover:text-green-800"
                >
                  Save
                </button>
                <button
                  v-if="editingQuestionId === question.id"
                  @click="cancelQuestionEdit"
                  class="text-sm text-red-600 hover:text-red-800"
                >
                  Cancel
                </button>
              </div>
            </div>

            <div v-if="editingQuestionId === question.id">
              <label class="block text-sm font-medium text-neutral-700">Question Text</label>
              <textarea
                v-model="editedQuestionText"
                class="w-full p-2 border rounded-md mb-2 focus:ring-indigo-500 focus:border-indigo-500"
                rows="2"
              ></textarea>
              <label class="block text-sm font-medium text-neutral-700">Note Text</label>
              <textarea
                v-model="editedNoteText"
                class="w-full p-2 border rounded-md focus:ring-indigo-500 focus:border-indigo-500"
                rows="2"
              ></textarea>
            </div>
            <div v-else>
              <p class="text-neutral-800 mb-1">{{ question.question_text }}</p>
              <p class="text-sm text-neutral-600 italic">Note: {{ question.note_text }}</p>
            </div>
          </div>
        </div>
        <div v-else class="text-neutral-500 italic">
          No questions generated yet. Start a conversation in the chat to generate some!
        </div>
      </div>
    </div>
    <div v-else class="text-center text-neutral-500">
      Select or create an IGA session to view the questionnaire artifact.
    </div>
  </div>
</template>
