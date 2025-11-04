<script setup lang="ts">
import BaseLayout from '@/components/Base/BaseLayout.vue'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import { useIgaStore } from '@/stores/igaStore.ts'
import { computed } from 'vue'
import { useRouter } from 'vue-router'

const igaStore = useIgaStore()
const router = useRouter()

const hasSessions = computed(() => igaStore.igaSessions && igaStore.igaSessions.length > 0)

const selectExistingSession = (sessionId: string) => {
  router.push({ name: 'iga-active', params: { id: sessionId } })
}

const navigateToCreateSession = () => {
  router.push({ name: 'iga-create' })
}
</script>

<template>
  <BaseLayout>
    <div class="p-6 bg-gray-100 min-h-full">
      <h1 class="text-3xl font-extrabold text-gray-900 mb-6">Interview Guide Assist Sessions</h1>

      <div v-if="igaStore.isLoadingSessions" class="flex justify-center items-center h-48">
        <LoadingSpinner />
      </div>

      <div v-else-if="igaStore.errorSessions" class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg relative mb-6">
        <strong class="font-bold">Error! </strong>
        <span class="block sm:inline"> {{ $t(`error.${igaStore.errorSessions.message}`) }}</span>
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
              <span class="font-medium">Created:</span> {{ new Date(session.created_at).toLocaleString('de-DE') }}
            </p>
            <p class="text-sm text-gray-600">
              <span class="font-medium">Updated:</span> {{ session.updated_at ? new Date(session.updated_at).toLocaleString('de-DE') : 'not yet' }}
            </p>
          </div>
        </div>
      </div>

      <div v-else class="mb-8 p-6 bg-white rounded-xl shadow-lg text-center text-gray-600">
        <p class="text-lg">No IGA sessions found. Create a new one to get started!</p>
      </div>

      <button
        @click="navigateToCreateSession"
        class="bg-indigo-600 text-white px-6 py-3 rounded-lg shadow-md hover:bg-indigo-700 transition-colors duration-300 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 cursor-pointer"
      >
        Create New Session
      </button>
    </div>
  </BaseLayout>
</template>
