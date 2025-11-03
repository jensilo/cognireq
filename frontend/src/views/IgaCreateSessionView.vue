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
</script>

<template>
  <BaseLayout>
    <div class="p-6 bg-gray-100 min-h-full">
      <h1 class="text-3xl font-extrabold text-gray-900 mb-6">Create New IGA Session</h1>

      <div class="p-8 bg-white rounded-xl shadow-lg border border-gray-200">
        <form @submit.prevent="createSession" class="space-y-6">
          <div>
            <label for="projectName" class="block text-sm font-medium text-gray-700 mb-1">Project Name</label>
            <input
              type="text"
              id="projectName"
              v-model="newSessionForm.project_name"
              required
              class="mt-1 block w-full p-3 border border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>
          <div>
            <label for="stakeholderSummary" class="block text-sm font-medium text-gray-700 mb-1">Stakeholder Summary</label>
            <textarea
              id="stakeholderSummary"
              v-model="newSessionForm.context_data.stakeholder_summary"
              rows="4"
              class="mt-1 block w-full p-3 border border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
            ></textarea>
          </div>
          <div>
            <label for="projectSummary" class="block text-sm font-medium text-gray-700 mb-1">Project Summary</label>
            <textarea
              id="projectSummary"
              v-model="newSessionForm.context_data.project_summary"
              rows="4"
              class="mt-1 block w-full p-3 border border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
            ></textarea>
          </div>
          <div>
            <label for="interviewGoal" class="block text-sm font-medium text-gray-700 mb-1">Interview Goal</label>
            <textarea
              id="interviewGoal"
              v-model="newSessionForm.context_data.interview_goal"
              rows="4"
              class="mt-1 block w-full p-3 border border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
            ></textarea>
          </div>
          <div>
            <label for="interviewAdvice" class="block text-sm font-medium text-gray-700 mb-1">Interview Advice</label>
            <textarea
              id="interviewAdvice"
              v-model="newSessionForm.context_data.interview_advice"
              rows="4"
              class="mt-1 block w-full p-3 border border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
            ></textarea>
          </div>
          <button
            type="submit"
            :disabled="igaStore.isCreatingSession"
            class="bg-green-600 text-white px-6 py-3 rounded-lg shadow-md hover:bg-green-700 disabled:bg-green-400 transition-colors duration-300 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2 cursor-pointer"
          >
            <span v-if="igaStore.isCreatingSession" class="flex items-center justify-center">
              <LoadingSpinner class="w-5 h-5 mr-2" /> Creating...
            </span>
            <span v-else>Create Session</span>
          </button>
        </form>
      </div>
    </div>
  </BaseLayout>
</template>
