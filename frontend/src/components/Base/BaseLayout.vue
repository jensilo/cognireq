<script setup lang="ts">
import { ref, computed } from 'vue'
import BaseNavbar from './BaseNavbar.vue'
import BaseHeader from './BaseHeader.vue'

const isSidebarCollapsed = ref(false)

const toggleSidebar = () => {
  isSidebarCollapsed.value = !isSidebarCollapsed.value
}

const mainContentMargin = computed(() => (isSidebarCollapsed.value ? 'ml-16' : 'ml-64'))
</script>

<template>
  <div class="flex h-screen bg-gray-100 antialiased">
    <!-- Navbar Component (Sidebar) -->
    <BaseNavbar :is-collapsed="isSidebarCollapsed" @toggle-sidebar="toggleSidebar" />

    <!-- Main Content Area -->
    <main
      class="flex-grow flex flex-col transition-all duration-300 ease-in-out overflow-y-auto"
      :class="mainContentMargin"
    >
      <!-- Top Bar/Header -->
      <BaseHeader />

      <!-- Page Content Slot -->
      <div class="p-6 flex-grow">
        <slot />
      </div>
    </main>
  </div>
</template>
