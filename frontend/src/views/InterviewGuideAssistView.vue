<script setup lang="ts">
import BaseLayout from '@/components/Base/BaseLayout.vue'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import IgaSessionSelector from '@/components/IgaSessionSelector.vue'
import IgaChatInterface from '@/components/IgaChatInterface.vue'
import IgaQuestionnaireArtifact from '@/components/IgaQuestionnaireArtifact.vue'
import { useIgaStore } from '@/stores/igaStore.ts'
import { computed, ref } from 'vue'
import type {IgaChatLogInstructionType} from "@/api/igaService.ts";

const igaStore = useIgaStore()

const activeSession = computed(() => igaStore.activeSession)
const chatInterfaceRef = ref<InstanceType<typeof IgaChatInterface> | null>(null)
const showChatOnMobile = ref(false)

const setChatInput = (text: string, instruction: IgaChatLogInstructionType) => {
  if (chatInterfaceRef.value) {
    chatInterfaceRef.value.setChatInput(text, instruction)
  }
  showChatOnMobile.value = true // show chat when citing/refining
}
</script>

<template>
  <BaseLayout>
    <div class="flex flex-col h-full">
      <LoadingSpinner v-if="igaStore.isLoadingSessions" />

      <div v-else-if="!igaStore.activeSessionId">
        <IgaSessionSelector />
      </div>

      <div v-else class="flex flex-col h-full">
        <!-- Header for IGA Session Context -->
        <div class="bg-white shadow-sm p-4 border-b border-neutral-200">
          <div class="flex justify-between items-center mb-2">
            <h1 class="text-xl font-bold text-neutral-800">{{ activeSession?.project_name }}</h1>
            <button @click="igaStore.clearActiveSession" class="text-md cursor-pointer text-indigo-600 hover:underline">
              Change Session
            </button>
          </div>
          <div class="text-sm text-neutral-600 mb-2">
            <p><strong>Stakeholder Summary:</strong> {{ activeSession?.context_data.stakeholder_summary }}</p>
            <p><strong>Project Summary:</strong> {{ activeSession?.context_data.project_summary }}</p>
          </div>
          <div class="text-xs text-neutral-400">
            <p>Created: {{ activeSession?.created_at ? new Date(activeSession.created_at).toLocaleString() : '' }}</p>
            <p>Updated: {{ activeSession?.updated_at ? new Date(activeSession.updated_at).toLocaleString() : '' }}</p>
          </div>
        </div>

        <!-- Mobile Toggle for Chat/Artifact -->
        <div class="md:hidden flex justify-around p-2 bg-white border-b border-neutral-200">
          <button
            @click="showChatOnMobile = true"
            :class="['px-4 py-2 rounded-lg text-sm font-medium', showChatOnMobile ? 'bg-indigo-600 text-white' : 'text-indigo-600']"
          >
            Chat
          </button>
          <button
            @click="showChatOnMobile = false"
            :class="['px-4 py-2 rounded-lg text-sm font-medium', !showChatOnMobile ? 'bg-indigo-600 text-white' : 'text-indigo-600']"
          >
            Questionnaire
          </button>
        </div>

        <!-- Main Dual-Pane Layout -->
        <div class="flex flex-1 overflow-hidden">
          <!-- Left Pane: Chat Interface -->
          <div
            :class="[
              'w-full md:w-1/3 flex-col bg-neutral-50',
              { 'hidden md:flex': !showChatOnMobile },
              { 'flex': showChatOnMobile },
            ]"
          >
            <IgaChatInterface ref="chatInterfaceRef" :session-id="igaStore.activeSessionId" />
          </div>

          <!-- Right Pane: Questionnaire Artifact -->
          <div
            :class="[
              'w-full md:w-2/3 flex-col bg-white',
              { 'hidden md:flex': showChatOnMobile },
              { 'flex': !showChatOnMobile },
            ]"
          >
            <IgaQuestionnaireArtifact :session-id="igaStore.activeSessionId" :set-chat-input="setChatInput" />
          </div>
        </div>
      </div>
    </div>
  </BaseLayout>
</template>
