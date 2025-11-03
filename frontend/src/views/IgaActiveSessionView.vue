<script setup lang="ts">
import BaseLayout from '@/components/Base/BaseLayout.vue'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import IgaChatInterface from '@/components/IgaChatInterface.vue'
import IgaQuestionnaireArtifact from '@/components/IgaQuestionnaireArtifact.vue'
import { useIgaStore } from '@/stores/igaStore.ts'
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { IgaChatLogInstructionType, UpdateIgaSessionPayload } from '@/api/igaService.ts'

const igaStore = useIgaStore()
const route = useRoute()
const router = useRouter()

const sessionId = computed(() => route.params.id as string)

const chatInterfaceRef = ref<InstanceType<typeof IgaChatInterface> | null>(null)
const showChatOnMobile = ref(false)

const setChatInput = (text: string, instruction: IgaChatLogInstructionType) => {
  if (chatInterfaceRef.value) {
    chatInterfaceRef.value.setChatInput(text, instruction)
  }
  showChatOnMobile.value = true // show chat when citing/refining
}

const handleClearActiveSession = () => {
  router.push('/iga')
}

const updateSessionContext = async (field: 'stakeholder_summary' | 'project_summary' | 'interview_goal' | 'interview_advice', value: string) => {
  if (!sessionId.value || !igaStore.activeSession) return

  const currentContext = igaStore.activeSession.context_data
  const payload: UpdateIgaSessionPayload = {
    context_data: { ...currentContext, [field]: value },
  }

  await igaStore.updateIgaSession({ id: sessionId.value, payload })
}

const refineContext = (contextType: string) => {
  setChatInput(`Refine the ${contextType}: `, 'refine-context')
}
</script>

<template>
  <BaseLayout>
    <div class="flex flex-col h-full">
      <LoadingSpinner v-if="igaStore.isLoadingActiveSession" />

      <div v-else-if="igaStore.errorActiveSession" class="p-4 text-center text-red-500">
        <p>Error loading session: {{ igaStore.errorActiveSession.message }}</p>
        <button @click="handleClearActiveSession" class="text-md cursor-pointer text-indigo-600 hover:underline mt-2">
          Go to Session List
        </button>
      </div>

      <div v-else-if="!igaStore.activeSession" class="p-4 text-center text-red-500">
        <p>Session not found.</p>
        <button @click="handleClearActiveSession" class="text-md cursor-pointer text-indigo-600 hover:underline mt-2">
          Go to Session List
        </button>
      </div>

      <div v-else class="flex flex-col h-full">
        <!-- Header for IGA Session Context -->
        <div class="bg-white shadow-sm p-4 border-b border-neutral-200">
          <div class="flex justify-between items-center mb-2">
            <h1 class="text-xl font-bold text-neutral-800">{{ igaStore.activeSession?.project_name }}</h1>
            <button @click="handleClearActiveSession" class="text-md cursor-pointer text-indigo-600 hover:underline">
              Change Session
            </button>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            <div class="p-3 border rounded-lg bg-neutral-50">
              <h2 class="text-sm font-medium text-neutral-700 mb-1 flex items-center justify-between">
                Stakeholder Summary
                <button @click="refineContext('stakeholder summary')" class="text-indigo-600 hover:text-indigo-800 text-xs cursor-pointer">
                  AI Enhance
                </button>
              </h2>
              <textarea
                v-model="igaStore.activeSession.context_data.stakeholder_summary"
                @blur="updateSessionContext('stakeholder_summary', igaStore.activeSession.context_data.stakeholder_summary)"
                class="w-full p-1 border rounded-md text-sm focus:ring-indigo-500 focus:border-indigo-500"
                rows="2"
              ></textarea>
            </div>
            <div class="p-3 border rounded-lg bg-neutral-50">
              <h2 class="text-sm font-medium text-neutral-700 mb-1 flex items-center justify-between">
                Project Summary
                <button @click="refineContext('project summary')" class="text-indigo-600 hover:text-indigo-800 text-xs cursor-pointer">
                  AI Enhance
                </button>
              </h2>
              <textarea
                v-model="igaStore.activeSession.context_data.project_summary"
                @blur="updateSessionContext('project_summary', igaStore.activeSession.context_data.project_summary)"
                class="w-full p-1 border rounded-md text-sm focus:ring-indigo-500 focus:border-indigo-500"
                rows="2"
              ></textarea>
            </div>
          </div>

          <div class="text-xs text-neutral-400">
            <p>Created: {{ igaStore.activeSession?.created_at ? new Date(igaStore.activeSession.created_at).toLocaleString() : '' }}</p>
            <p>Updated: {{ igaStore.activeSession?.updated_at ? new Date(igaStore.activeSession.updated_at).toLocaleString() : '' }}</p>
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
            <IgaChatInterface ref="chatInterfaceRef" :session-id="sessionId" />
          </div>

          <!-- Right Pane: Questionnaire Artifact -->
          <div
            :class="[
              'w-full md:w-2/3 flex-col bg-white',
              { 'hidden md:flex': showChatOnMobile },
              { 'flex': !showChatOnMobile },
            ]"
          >
            <IgaQuestionnaireArtifact :session-id="sessionId" :set-chat-input="setChatInput" :refine-context="refineContext" :update-session-context="updateSessionContext" />
          </div>
        </div>
      </div>
    </div>
  </BaseLayout>
</template>
