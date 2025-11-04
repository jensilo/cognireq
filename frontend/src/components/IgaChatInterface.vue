<script setup lang="ts">
import { ref, nextTick, watch, computed } from 'vue'
import { useIgaStore } from '@/stores/igaStore.ts'
import type { IgaChatLogInstructionType, SendChatPayload } from '@/api/igaService.ts'
import LoadingSpinner from '@/components/LoadingSpinner.vue'

const props = defineProps<{ sessionId: string }>()

const igaStore = useIgaStore()
const chatInput = ref('')
const selectedInstruction = ref<IgaChatLogInstructionType | null>(null)
const chatContainer = ref<HTMLElement | null>(null)

const chatLogs = computed(() => igaStore.igaChatLogs)
const isLoadingChatLogs = computed(() => igaStore.isLoadingChatLogs)
const isSendingChatMessage = computed(() => igaStore.isSendingChatMessage)
const isRetryingChatMessage = computed(() => igaStore.isRetryingChatMessage)

watch(chatLogs, () => {
  nextTick(() => {
    if (chatContainer.value) {
      chatContainer.value.scrollTop = chatContainer.value.scrollHeight
    }
  })
}, { deep: true, immediate: true })

const sendMessage = async () => {
  if (!chatInput.value.trim() || !props.sessionId) return

  const payload: SendChatPayload = {
    message_text: chatInput.value,
    instruction: selectedInstruction.value || 'message',
  }

  await igaStore.sendIgaChatMessage({ sessionId: props.sessionId, payload })
  chatInput.value = ''
  selectedInstruction.value = null
}

const retryMessage = async () => {
  if (!props.sessionId) return
  await igaStore.retryIgaChatMessage(props.sessionId)
}

const formatTimestamp = (timestamp: string) => {
  return new Date(timestamp).toLocaleString()
}

const setChatInput = (text: string, instruction: IgaChatLogInstructionType = 'message') => {
  chatInput.value = text
  selectedInstruction.value = instruction
}

const resetInstruction = () => {
  selectedInstruction.value = null
}

defineExpose({ setChatInput })
</script>

<template>
  <div class="flex flex-col h-full bg-neutral-50">
    <!-- Chat Log Display -->
    <div ref="chatContainer" class="flex-1 overflow-y-auto p-4 space-y-4">
      <LoadingSpinner v-if="isLoadingChatLogs && chatLogs && chatLogs.length < 1" />
      <div v-else-if="chatLogs && chatLogs.length > 0">
        <div
          v-for="log in chatLogs"
          :key="log.id"
          :class="[
            'flex',
            log.role === 'user' ? 'justify-end' : 'justify-start',
            'mt-4 first:mt-0', // Add margin-top to all but the first message
          ]"
        >
          <div
            :class="[
              'max-w-[70%]',
              'p-3',
              'rounded-lg',
              'shadow-md',
              log.role === 'user'
                ? 'bg-indigo-500 text-white'
                : 'bg-white text-neutral-800',
            ]"
          >
            <p class="text-sm">{{ log.message_text }}</p>
          </div>
        </div>
      </div>
      <div v-else class="text-center text-neutral-500">
        Start a conversation with the Interview Guide Assistant!
      </div>
    </div>

    <!-- Chat Input Area -->
    <div class="p-4 border-t border-neutral-200 bg-white">
      <div class="flex space-x-2 mb-3">
        <button
          @click="resetInstruction"
          :class="[
            'px-3 py-1 rounded-full text-sm cursor-pointer',
            selectedInstruction === null
              ? 'bg-indigo-600 text-white'
              : 'bg-neutral-200 text-neutral-700 hover:bg-neutral-300',
          ]"
          :disabled="isSendingChatMessage || isRetryingChatMessage"
        >
          Message
        </button>
        <button
          @click="selectedInstruction = 'refine-question'"
          :class="[
            'px-3 py-1 rounded-full text-sm cursor-pointer',
            selectedInstruction === 'refine-question'
              ? 'bg-indigo-600 text-white'
              : 'bg-neutral-200 text-neutral-700 hover:bg-neutral-300',
          ]"
          :disabled="isSendingChatMessage || isRetryingChatMessage"
        >
          Refine Question
        </button>
        <button
          @click="selectedInstruction = 'refine-context'"
          :class="[
            'px-3 py-1 rounded-full text-sm cursor-pointer',
            selectedInstruction === 'refine-context'
              ? 'bg-indigo-600 text-white'
              : 'bg-neutral-200 text-neutral-700 hover:bg-neutral-300',
          ]"
          :disabled="isSendingChatMessage || isRetryingChatMessage"
        >
          Refine Context
        </button>
      </div>

      <form @submit.prevent="sendMessage" class="flex space-x-2">
        <input
          type="text"
          v-model="chatInput"
          :placeholder="$t('iga.chat-input-placeholder')"
          class="flex-1 p-3 border border-neutral-300 rounded-lg shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
          :disabled="isSendingChatMessage || isRetryingChatMessage"
        />
        <button
          type="submit"
          :disabled="!chatInput.trim() || isSendingChatMessage || isRetryingChatMessage"
          :class="{ 'cursor-progress': (isSendingChatMessage || isRetryingChatMessage) }"
          class="bg-indigo-600 text-white px-4 py-2 rounded-lg shadow-md hover:bg-indigo-700 disabled:bg-indigo-400 cursor-pointer"
        >
          <span v-if="isSendingChatMessage || isRetryingChatMessage">
            <LoadingSpinner class="w-5 h-5" />
          </span>
          <span v-else>Send</span>
        </button>
      </form>
    </div>
  </div>
</template>
