<script setup lang="ts">
import { ref } from 'vue'
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
</script>

<template>
  <div class="p-4">
    <h1 class="text-2xl font-bold mb-4">Interview Guide Assist Sessions</h1>

    <LoadingSpinner v-if="igaStore.isLoadingSessions" />

    <div v-else-if="igaStore.igaSessions && igaStore.igaSessions.length > 0">
      <h2 class="text-xl font-semibold mb-2">Your Sessions</h2>
      <ul class="space-y-2 mb-4">
        <li
          v-for="session in igaStore.igaSessions"
          :key="session.id"
          @click="selectExistingSession(session.id)"
          class="p-3 border rounded-lg cursor-pointer hover:bg-neutral-100"
        >
          <p class="font-medium">{{ session.project_name }}</p>
          <p class="text-sm text-neutral-500">Created: {{ new Date(session.created_at).toLocaleString() }}</p>
          <p class="text-sm text-neutral-500">Updated: {{ new Date(session.updated_at).toLocaleString() }}</p>
        </li>
      </ul>
    </div>

    <div v-else class="mb-4">
      <p class="text-neutral-600">No IGA sessions found. Create a new one to get started!</p>
    </div>

    <button
      @click="showCreateForm = !showCreateForm"
      class="bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 cursor-pointer"
    >
      {{ showCreateForm ? 'Cancel' : 'Create New Session' }}
    </button>

    <div v-if="showCreateForm" class="mt-6 p-4 border rounded-lg bg-neutral-50">
      <h2 class="text-xl font-semibold mb-4">Create New IGA Session</h2>
      <form @submit.prevent="createSession" class="space-y-4">
        <div>
          <label for="projectName" class="block text-sm font-medium text-neutral-700">Project Name</label>
          <input
            type="text"
            id="projectName"
            v-model="newSessionForm.project_name"
            required
            class="mt-1 block w-full p-2 border border-neutral-300 rounded-md shadow-sm"
          />
        </div>
        <div>
          <label for="stakeholderSummary" class="block text-sm font-medium text-neutral-700">Stakeholder Summary</label>
          <textarea
            id="stakeholderSummary"
            v-model="newSessionForm.context_data.stakeholder_summary"
            rows="3"
            class="mt-1 block w-full p-2 border border-neutral-300 rounded-md shadow-sm"
          ></textarea>
        </div>
        <div>
          <label for="projectSummary" class="block text-sm font-medium text-neutral-700">Project Summary</label>
          <textarea
            id="projectSummary"
            v-model="newSessionForm.context_data.project_summary"
            rows="3"
            class="mt-1 block w-full p-2 border border-neutral-300 rounded-md shadow-sm"
          ></textarea>
        </div>
        <button
          type="submit"
          :disabled="igaStore.isCreatingSession"
          class="bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 disabled:bg-green-400 cursor-pointer"
        >
          {{ igaStore.isCreatingSession ? 'Creating...' : 'Create Session' }}
        </button>
      </form>
    </div>
  </div>
</template>
