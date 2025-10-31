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
    <div
      class="flex-grow flex flex-col transition-all duration-300 ease-in-out"
      :class="mainContentMargin"
    >
      <!-- Top Bar/Header -->
      <BaseHeader />

      <!-- Page Content Slot -->
      <main class="flex-grow overflow-y-auto">
        <slot />
      </main>
    </div>
  </div>
</template>
