<script setup lang="ts">
import { ref, computed } from 'vue'
import { useIgaStore } from '@/stores/igaStore.ts'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import type { CreateIgaSessionPayload } from '@/api/igaService.ts'

const igaStore = useIgaStore()

const newSessionForm = ref<CreateIgaSessionPayload>({
  project_name: '',
  context_data: {
    stakeholder_summary: '',
    project_summary: '',
    interview_goal: '',
    interview_advice: '',
  },
})

const showCreateForm = ref(false)

const createSession = async () => {
  await igaStore.createIgaSession(newSessionForm.value)
  showCreateForm.value = false
  newSessionForm.value = {
    project_name: '',
    context_data: {
      stakeholder_summary: '',
      project_summary: '',
      interview_goal: '',
      interview_advice: '',
    },
  }
}

const selectExistingSession = (sessionId: string) => {
  igaStore.selectSession(sessionId)
}

const hasSessions = computed(() => igaStore.igaSessions && igaStore.igaSessions.length > 0)
</script>

<template>
  <div class="p-6 bg-gray-100 min-h-full">
    <h1 class="text-3xl font-extrabold text-gray-900 mb-6">Interview Guide Assist Sessions</h1>

    <div v-if="igaStore.isLoadingSessions" class="flex justify-center items-center h-48">
      <LoadingSpinner />
    </div>

    <div v-else-if="igaStore.errorSessions" class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg relative mb-6">
      <strong class="font-bold">Error!</strong>
      <span class="block sm:inline"> {{ igaStore.errorSessions.message }}</span>
    </div>

    <div v-else-if="hasSessions" class="mb-8">
      <h2 class="text-2xl font-semibold text-gray-800 mb-4">Your Sessions</h2>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <div
          v-for="session in igaStore.igaSessions"
          :key="session.id"
          @click="selectExistingSession(session.id)"
          class="bg-white rounded-xl shadow-lg hover:shadow-xl transition-all duration-300 cursor-pointer p-6 border border-gray-200"
        >
          <h3 class="text-xl font-bold text-indigo-700 mb-2">{{ session.project_name }}</h3>
          <p class="text-sm text-gray-600 mb-1">
            <span class="font-medium">Created:</span> {{ new Date(session.created_at).toLocaleString() }}
          </p>
          <p class="text-sm text-gray-600">
            <span class="font-medium">Updated:</span> {{ new Date(session.updated_at).toLocaleString() }}
          </p>
        </div>
      </div>
    </div>

    <div v-else class="mb-8 p-6 bg-white rounded-xl shadow-lg text-center text-gray-600">
      <p class="text-lg">No IGA sessions found. Create a new one to get started!</p>
    </div>

    <button
      @click="showCreateForm = !showCreateForm"
      class="bg-indigo-600 text-white px-6 py-3 rounded-lg shadow-md hover:bg-indigo-700 transition-colors duration-300 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 cursor-pointer"
    >
      {{ showCreateForm ? 'Cancel Creation' : 'Create New Session' }}
    </button>

    <div v-if="showCreateForm" class="mt-8 p-8 bg-white rounded-xl shadow-lg border border-gray-200">
      <h2 class="text-2xl font-semibold text-gray-800 mb-6">Create New IGA Session</h2>
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
</template>
