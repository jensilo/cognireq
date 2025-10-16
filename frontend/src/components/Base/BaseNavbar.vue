<script setup lang="ts">
import { computed } from 'vue'
import LogoutButton from '@/components/LogoutButton.vue'

const props = defineProps<{ isCollapsed: boolean }>()
const emit = defineEmits<{ (e: 'toggleSidebar'): void }>()

const navItems = [
  {
    name: 'Interview Guide Assist',
    to: { name: 'interview-guide-assist' },
    iconComponent: 'i-heroicons-light-bulb',
  },
  { name: 'About', to: { name: 'about' }, iconComponent: 'i-heroicons-information-circle' },
]

const collapseClass = computed(() =>
  props.isCollapsed
    ? 'opacity-0 pointer-events-none w-0 overflow-hidden'
    : 'opacity-100 pointer-events-auto w-auto',
)

const toggleSidebar = () => {
  emit('toggleSidebar')
}
</script>

<template>
  <!-- Main Navigation Container -->
  <nav
    id="sidebar-nav"
    class="flex flex-col h-full bg-gray-900 text-white shadow-xl transition-all duration-300 ease-in-out fixed left-0 top-0 bottom-0 z-20"
    :class="isCollapsed ? 'w-16' : 'w-64'"
  >
    <!-- Logo/Header -->
    <div class="p-4 flex items-center h-16 border-b border-gray-700">
      <!-- Icon/Short Name when collapsed -->
      <span v-if="isCollapsed" class="text-2xl font-bold transition-all duration-300">
        <i-heroicons-bolt-solid class="text-indigo-400" />
      </span>

      <!-- Full App Name when expanded -->
      <h1
        class="text-xl font-semibold whitespace-nowrap overflow-hidden transition-opacity duration-300"
        :class="collapseClass"
      >
        {{ $t('app.name') }}
      </h1>
    </div>

    <!-- Main Navigation Links - Scrollable Area -->
    <div class="flex-grow overflow-y-auto pt-2 space-y-1">
      <RouterLink
        v-for="item in navItems"
        :key="item.name"
        :to="item.to"
        class="flex items-center mx-2 p-3 text-sm font-medium rounded-lg transition-colors duration-150 group"
        :class="isCollapsed ? 'justify-center' : 'justify-start'"
        active-class="bg-indigo-600 text-white shadow-md hover:bg-indigo-500"
        inactive-class="text-gray-300 hover:bg-gray-700"
      >
        <!-- Dynamic Icon Component -->
        <component
          :is="item.iconComponent"
          class="text-xl shrink-0"
          :class="{ 'mr-3': !isCollapsed }"
        />

        <!-- Link Text -->
        <span class="whitespace-nowrap transition-all duration-300" :class="collapseClass">
          {{ $t(`nav.${item.to.name}`) }}
        </span>
      </RouterLink>
    </div>

    <!-- Bottom Section: Toggle and Logout -->
    <div class="p-3 border-t border-gray-700 flex flex-col items-center">
      <!-- Logout Button -->
      <LogoutButton :isCollapsed="isCollapsed" :collapseClass="collapseClass" />

      <!-- Toggle Button at the very bottom -->
      <button
        @click="toggleSidebar"
        class="mt-2 w-full flex justify-center items-center p-2 text-sm text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors duration-150 cursor-pointer"
        :aria-expanded="!isCollapsed"
        aria-label="Toggle Navigation"
      >
        <i-heroicons-chevron-double-left
          class="text-xl transition-transform duration-300"
          :class="{ 'rotate-180': isCollapsed }"
        />
      </button>
    </div>
  </nav>
</template>
