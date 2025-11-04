<script setup lang="ts">
import {ref, watch, onMounted, type PropType, computed, watchEffect} from 'vue'
import IHeroiconsSparkles from '~icons/heroicons/sparkles'

const props = defineProps({
  modelValue: { type: String, default: '' },
  label: { type: String, required: true },
  placeholder: { type: String, default: '' },
  rows: { type: Number, default: 2 },
  onRefine: { type: Function as PropType<(contextType: string) => void>, required: true },
  contextType: { type: String, required: true },
  loading: { type: Boolean, required: true }
})

const emit = defineEmits(['updateField'])

const internalValue = ref(props.modelValue)

onMounted(() => {
  internalValue.value = props.modelValue
})

watch(() => props.modelValue, (newValue) => {
  if (newValue !== internalValue.value) {
    internalValue.value = newValue
  }
})

const handleBlur = () => {
  // emit unconditionally on blur. Parent will handle change detection.
  emit('updateField', internalValue.value)
}
</script>

<template>
  <div class="p-3 border border-gray-200 rounded-lg shadow-sm bg-white">
    <h2 class="text-sm font-medium text-gray-700 mb-1 flex items-center justify-between">
      {{ label }}
      <button @click="onRefine(contextType)" class="text-indigo-600 hover:text-indigo-800 text-xs flex items-center cursor-pointer">
        <IHeroiconsSparkles class="w-4 h-4 mr-1"></IHeroiconsSparkles> AI Enhance
      </button>
    </h2>
    <textarea
      v-model="internalValue"
      @blur="handleBlur"
      :placeholder="placeholder"
      :rows="rows"
      :disabled="loading"
      :class="{ 'cursor-progress': loading }"
      class="w-full p-1 border border-gray-300 rounded-md text-sm focus:ring-indigo-500 focus:border-indigo-500 text-gray-700"
    ></textarea>
  </div>
</template>
