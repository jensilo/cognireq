<script setup lang="ts">
import BaseLayout from '@/components/Base/BaseLayout.vue'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import IgaSessionSelector from '@/components/IgaSessionSelector.vue'
import { useIgaStore } from '@/stores/igaStore.ts'

const igaStore = useIgaStore()
</script>

<template>
  <BaseLayout>
    <div class="flex flex-col h-full">
      <LoadingSpinner v-if="igaStore.isLoadingSessions" />

      <div v-else-if="igaStore.errorSessions" class="p-4 text-center text-red-500">
        <p>Error loading sessions: {{ igaStore.errorSessions.message }}</p>
      </div>

      <div v-else>
        <IgaSessionSelector />
      </div>
    </div>
  </BaseLayout>
</template>
