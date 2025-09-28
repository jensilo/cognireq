<template>
  <div class="min-h-screen bg-neutral-50 flex items-center justify-center p-4">
    <div class="max-w-4xl w-full bg-white shadow-xl rounded-lg overflow-hidden grid md:grid-cols-2">

      <div class="p-8 lg:p-12 flex flex-col justify-center">
        <h1 class="text-4xl font-black text-indigo-700 mb-4">
          {{ $t('app.name') }}
        </h1>

        <p class="text-lg text-neutral-600 mb-6">
          {{ $t('login.subtitle') }}
        </p>

        <div class="space-y-3">
          <div class="flex items-start">
            <svg class="h-5 w-5 text-indigo-500 mt-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
            </svg>
            <p class="ml-3 text-base text-neutral-700" v-html="$t('login.feature-1')"></p>
          </div>
          <div class="flex items-start">
            <svg class="h-5 w-5 text-indigo-500 mt-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
            </svg>
            <p class="ml-3 text-base text-neutral-700" v-html="$t('login.feature-2')"></p>
          </div>
          <div class="flex items-start">
            <svg class="h-5 w-5 text-indigo-500 mt-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
            </svg>
            <p class="ml-3 text-base text-neutral-700" v-html="$t('login.feature-3')"></p>
          </div>
        </div>
      </div>

      <div class="p-8 lg:p-12 bg-neutral-50 border-t md:border-t-0 md:border-l border-neutral-200 flex flex-col justify-center">
        <h2 class="text-2xl font-semibold text-neutral-800 mb-6 text-center">
          {{ $t('login.sign-in-header') }}
        </h2>

        <form @submit.prevent="handleSubmit" class="space-y-6">
          <!-- Error Display Area -->
          <div v-if="error" class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg text-sm" role="alert">
            {{ $t(error) }}
          </div>

          <div>
            <label for="email" class="text-sm font-medium text-neutral-700 sr-only">{{ $t('login.placeholder-email') }}</label>
            <input
              id="email"
              name="email"
              type="email"
              :placeholder="$t('login.placeholder-email')"
              required
              v-model="email"
              :disabled="isLoading"
              class="w-full p-3 border border-neutral-300 rounded-lg placeholder-neutral-400 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-neutral-900 disabled:bg-neutral-100 disabled:cursor-not-allowed"
            />
          </div>
          <div>
            <label for="password" class="text-sm font-medium text-neutral-700 sr-only">{{ $t('login.placeholder-password') }}</label>
            <input
              id="password"
              name="password"
              type="password"
              :placeholder="$t('login.placeholder-password')"
              required
              v-model="password"
              :disabled="isLoading"
              class="w-full p-3 border border-neutral-300 rounded-lg placeholder-neutral-400 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-neutral-900 disabled:bg-neutral-100 disabled:cursor-not-allowed"
            />
          </div>

          <div>
            <button
              type="submit"
              :disabled="isLoading"
              class="w-full flex justify-center py-3 px-4 border border-transparent text-base font-medium rounded-lg text-white bg-indigo-600 hover:bg-indigo-700 shadow-md focus:outline-none focus:ring-3 focus:ring-offset-2 focus:ring-indigo-500 transition duration-150 ease-in-out disabled:bg-indigo-400 disabled:cursor-not-allowed"
            >
              <span v-if="isLoading" class="flex items-center">
                <!-- Loading Spinner -->
                <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-white" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                {{ $t('login.button-loading') }}
              </span>
              <span v-else>
                {{ $t('login.button-default') }}
              </span>
            </button>
          </div>
        </form>

        <div class="mt-6 text-center">
          <p class="text-sm text-neutral-600">
            {{ $t('login.register-prompt') }}
            <router-link to="/register" class="font-medium text-indigo-600 hover:text-indigo-500">
              {{ $t('login.register-link') }}
            </router-link>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {ref} from 'vue';
import {useRouter} from 'vue-router';
import {loginUser} from "@/api/userService.ts";
import {ApiError} from "@/api/client.ts";

const router = useRouter();

const email = ref('');
const password = ref('');
const isLoading = ref(false);
const error = ref<string | null>(null);

async function handleSubmit(): Promise<void> {
  error.value = null;
  isLoading.value = true;

  try {
    await loginUser(email.value, password.value);
    await router.push('/app');
  } catch (err) {
    console.error('Login attempt failed:', err);

    if (err instanceof ApiError) {
      error.value = `error.${err.webError.code}`;
    } else if (err instanceof Error) {
      error.value = 'error.CLIENT_NETWORK_CONNECTION_ERROR';
    } else {
      error.value = 'error.CLIENT_UNKNOWN_ERROR';
    }
  } finally {
    isLoading.value = false;
  }
}
</script>
