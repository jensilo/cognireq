<script setup lang="ts">
import { ref } from 'vue'
import { useIgaStore } from '@/stores/igaStore.ts'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import type { CreateIgaSessionPayload } from '@/api/igaService.ts'
import { useRouter } from 'vue-router'
import BaseLayout from '@/components/Base/BaseLayout.vue'

const igaStore = useIgaStore()
const router = useRouter()

const newSessionForm = ref<CreateIgaSessionPayload>({
  project_name: '',
  context_data: {
    stakeholder_summary: '',
    project_summary: '',
    interview_goal: '',
    interview_advice: '',
  },
})

const createSession = async () => {
  const newSession = await igaStore.createIgaSession(newSessionForm.value)
  if (newSession) {
    await router.push(`/iga/${newSession.id}`)
  }
}

const goBack = () => {
  router.push({ name: 'iga-list' })
}
</script>

<template>
  <BaseLayout>
    <div class="p-6 bg-gray-100 min-h-full flex flex-col items-center">
      <div class="w-full max-w-3xl">
        <div class="flex items-center mb-6">
          <button @click="goBack" class="text-indigo-600 hover:text-indigo-800 flex items-center mr-4 cursor-pointer">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
            </svg>
            <span class="ml-1">{{ $t('iga.active-session.back-button') }}</span>
          </button>
          <h1 class="text-3xl font-extrabold text-gray-900">{{ $t('iga.create-session.title') }}</h1>
        </div>

        <div class="p-8 bg-white rounded-xl shadow-lg border border-gray-200">
          <form @submit.prevent="createSession" class="space-y-6">
            <div>
              <label for="projectName" class="block text-sm font-medium text-gray-700 mb-1">{{ $t('iga.create-session.project-name-label') }}</label>
              <input
                type="text"
                id="projectName"
                v-model="newSessionForm.project_name"
                required
                class="mt-1 block w-full p-3 border border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
              />
            </div>
            <div>
              <label for="stakeholderSummary" class="block text-sm font-medium text-gray-700 mb-1">{{ $t('iga.create-session.stakeholder-summary-label') }}</label>
              <textarea
                id="stakeholderSummary"
                v-model="newSessionForm.context_data.stakeholder_summary"
                rows="4"
                class="mt-1 block w-full p-3 border border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
              ></textarea>
            </div>
            <div>
              <label for="projectSummary" class="block text-sm font-medium text-gray-700 mb-1">{{ $t('iga.create-session.project-summary-label') }}</label>
              <textarea
                id="projectSummary"
                v-model="newSessionForm.context_data.project_summary"
                rows="4"
                class="mt-1 block w-full p-3 border border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
              ></textarea>
            </div>
            <button
              type="submit"
              :disabled="igaStore.isCreatingSession"
              class="w-full bg-indigo-600 text-white px-6 py-3 rounded-lg shadow-md hover:bg-indigo-700 disabled:bg-indigo-400 transition-colors duration-300 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 cursor-pointer"
            >
              <span v-if="igaStore.isCreatingSession" class="flex items-center justify-center">
                <LoadingSpinner class="w-5 h-5 mr-2" /> {{ $t('iga.create-session.creating-button') }}
              </span>
              <span v-else>{{ $t('iga.create-session.create-session-button') }}</span>
            </button>
          </form>
        </div>
      </div>
    </div>
  </BaseLayout>
</template>
