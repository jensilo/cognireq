<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useI18n} from "vue-i18n";

const props = defineProps<{
  message: string | null | undefined;
}>()

const {t, te} = useI18n()

const isVisible = ref(false)
const timeoutId = ref<number | null>(null)

const translatedErrorMessage = computed(() => {
  const m = props.message
  if (!m) {
    return t('error.CLIENT_UNKNOWN_ERROR')
  }

  if (te(m)) {
    return t(m)
  }

  if (te(`error.${m}`)) {
    return t(`error.${m}`)
  }

  return m
})

watch(() => props.message, (newMessage) => {
  if (newMessage) {
    isVisible.value = true
    if (timeoutId.value) {
      clearTimeout(timeoutId.value)
    }

    timeoutId.value = setTimeout(() => {
      isVisible.value = false
      timeoutId.value = null
    }, 5000) // hide after 5 seconds
  } else {
    isVisible.value = false

    if (timeoutId.value) {
      clearTimeout(timeoutId.value)
      timeoutId.value = null
    }
  }
}, { immediate: true })

const dismiss = () => {
  isVisible.value = false
  if (timeoutId.value) {
    clearTimeout(timeoutId.value)
    timeoutId.value = null
  }
}
</script>

<template>
  <transition :appear="true" :duration="300" name="fade">
    <div
      v-if="isVisible && message"
      class="fixed bottom-4 right-4 bg-red-600 text-white p-4 rounded-lg shadow-lg flex items-center justify-between z-50 max-w-sm"
      role="alert"
    >
      <p class="text-sm font-medium mr-4">{{ translatedErrorMessage }}</p>
      <button
        @click="dismiss"
        class="text-white hover:text-red-100 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 cursor-pointer"
        aria-label="Dismiss alert"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
        </svg>
      </button>
    </div>
  </transition>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.5s;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
