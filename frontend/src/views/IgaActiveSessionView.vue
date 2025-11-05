<script setup lang="ts">
import BaseLayout from '@/components/Base/BaseLayout.vue'
import IgaChatInterface from '@/components/IgaChatInterface.vue'
import IgaQuestionnaireArtifact from '@/components/IgaQuestionnaireArtifact.vue'
import IgaContextEditableField from '@/components/IgaContextEditableField.vue'
import { useIgaStore } from '@/stores/igaStore.ts'
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { IgaChatLogInstructionType, UpdateIgaSessionPayload } from '@/api/igaService.ts'
import LoadingSpinner from "@/components/LoadingSpinner.vue";

const igaStore = useIgaStore()
const route = useRoute()
const router = useRouter()

const sessionId = computed(() => route.params.id as string)
const isIgaLoading = computed(() => igaStore.isAnyActiveSessionStateLoading)

const chatInterfaceRef = ref<InstanceType<typeof IgaChatInterface> | null>(null)
const showChatOnMobile = ref(false)

const setChatInput = (text: string, instruction: IgaChatLogInstructionType) => {
  if (chatInterfaceRef.value) {
    chatInterfaceRef.value.setChatInput(text, instruction)
  }
  showChatOnMobile.value = true // show chat when citing/refining
}

const goBack = () => {
  router.push({ name: 'iga-list' })
}

const updateSessionContext = async (field: 'stakeholder_summary' | 'project_summary' | 'interview_goal' | 'interview_advice', newValue: string) => {
  if (!sessionId.value || !igaStore.activeSession) return

  const currentValue = igaStore.activeSession.context_data[field]

  // only update if the value has actually changed (ignoring leading/trailing whitespace)
  if (newValue.trim() === currentValue.trim()) {
    return // no change, do nothing
  }

  const currentContext = igaStore.activeSession.context_data
  const payload: UpdateIgaSessionPayload = {
    context_data: { ...currentContext, [field]: newValue },
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
      <div v-if="igaStore.isLoadingActiveSession && !igaStore.activeSession">
        <LoadingSpinner />
      </div>

      <div v-else-if="!igaStore.isLoadingActiveSession && igaStore.errorActiveSession" class="p-4 text-center text-red-500">
        <p>{{ $t('iga.active-session.error-loading-session') }} {{ $t(`error.${igaStore.errorActiveSession.message}`) }}</p>
        <button @click="goBack" class="text-md cursor-pointer text-indigo-600 hover:underline mt-2">
          {{ $t('iga.active-session.go-to-session-list') }}
        </button>
      </div>

      <div v-else-if="!igaStore.isLoadingActiveSession && !igaStore.activeSession" class="p-4 text-center text-red-500">
        <p>{{ $t('iga.active-session.session-not-found') }}</p>
        <button @click="goBack" class="text-md cursor-pointer text-indigo-600 hover:underline mt-2">
          {{ $t('iga.active-session.go-to-session-list') }}
        </button>
      </div>

      <div v-if="igaStore.activeSession" class="flex flex-col h-full">
        <!-- Header for IGA Session Context -->
        <div class="bg-white shadow-sm p-4 border-b border-neutral-200">
          <div class="flex items-center mb-2">
            <button @click="goBack" class="text-indigo-600 hover:text-indigo-800 flex items-center mr-4 cursor-pointer">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
              </svg>
              <span class="ml-1">{{ $t('iga.active-session.back-button') }}</span>
            </button>
            <h1 class="text-xl font-bold text-neutral-800">{{ igaStore.activeSession?.project_name }}</h1>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            <IgaContextEditableField
              :label="$t('iga.active-session.stakeholder-summary-label')"
              :model-value="igaStore.activeSession.context_data.stakeholder_summary"
              @updateField="updateSessionContext('stakeholder_summary', $event)"
              :on-refine="refineContext"
              context-type="stakeholder summary"
              :rows="2"
              :loading="isIgaLoading"
            />
            <IgaContextEditableField
              :label="$t('iga.active-session.project-summary-label')"
              :model-value="igaStore.activeSession.context_data.project_summary"
              @updateField="updateSessionContext('project_summary', $event)"
              :on-refine="refineContext"
              context-type="project summary"
              :rows="2"
              :loading="isIgaLoading"
            />
          </div>

          <div class="text-xs text-neutral-400">
            <span>{{ $t('iga.active-session.created-label') }} {{ igaStore.activeSession?.created_at ? new Date(igaStore.activeSession.created_at).toLocaleString('de-DE') : '' }}</span>
            &emsp;
            <span v-if="igaStore.activeSession?.updated_at">{{ $t('iga.active-session.updated-label') }} {{ igaStore.activeSession?.updated_at ? new Date(igaStore.activeSession.updated_at).toLocaleString('de-DE') : '' }}</span>
          </div>
        </div>

        <!-- Mobile Toggle for Chat/Artifact -->
        <div class="md:hidden flex justify-around p-2 bg-white border-b border-neutral-200">
          <button
            @click="showChatOnMobile = true"
            :class="['px-4 py-2 rounded-lg text-sm font-medium', showChatOnMobile ? 'bg-indigo-600 text-white' : 'text-indigo-600']"
          >
            {{ $t('iga.active-session.chat-button') }}
          </button>
          <button
            @click="showChatOnMobile = false"
            :class="['px-4 py-2 rounded-lg text-sm font-medium', !showChatOnMobile ? 'bg-indigo-600 text-white' : 'text-indigo-600']"
          >
            {{ $t('iga.active-session.questionnaire-button') }}
          </button>
        </div>

        <!-- Main Dual-Pane Layout -->
        <div class="flex flex-1 overflow-hidden">
          <!-- Left Pane: Chat Interface -->
          <div
            :class="[
              'w-full md:w-1/2 flex-col bg-neutral-50',
              { 'hidden md:flex': !showChatOnMobile },
              { 'flex': showChatOnMobile },
            ]"
          >
            <IgaChatInterface ref="chatInterfaceRef" :session-id="sessionId" />
          </div>

          <!-- Right Pane: Questionnaire Artifact -->
          <div
            :class="[
              'w-full md:w-1/2 flex-col bg-white border-l border-neutral-200',
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
