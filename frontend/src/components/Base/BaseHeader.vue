<script setup lang="ts"></script>

<template>
  <header class="h-16 flex items-center justify-end px-6 shadow bg-white border-b-2">
    <span class="text-gray-700 font-medium">
      <router-link to="/profile">
        {{ $t('header.user-profile') }}
      </router-link>
    </span>
  </header>
</template>
