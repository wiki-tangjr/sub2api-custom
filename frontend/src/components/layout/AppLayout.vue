<template>
  <div class="flex min-h-screen bg-gray-50 dark:bg-dark-950">
    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="relative flex min-h-screen min-w-0 flex-1 flex-col"
      :class="[sidebarCollapsed ? 'lg:ml-[72px]' : 'lg:ml-64']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main class="flex-1 p-4 md:p-6 lg:p-8">
        <slot />
      </main>

      <!-- ICP & Public Security Filing Footer (custom) -->
      <footer
        class="mt-auto shrink-0 border-t border-gray-200/80 bg-white/40 px-4 py-4 pb-6 backdrop-blur-sm sm:py-5 dark:border-dark-800/80 dark:bg-dark-900/30"
      >
        <div class="flex flex-wrap items-center justify-center gap-x-4 gap-y-2 text-center text-xs text-gray-400 dark:text-dark-500">
          <a
            href="https://beian.miit.gov.cn"
            target="_blank"
            rel="noopener noreferrer"
            class="group inline-flex items-center justify-center gap-1.5 whitespace-nowrap py-1 transition-colors duration-200 hover:text-primary-600 dark:hover:text-primary-400"
          >
            <svg
              class="h-3.5 w-3.5 opacity-70 transition-opacity group-hover:opacity-100"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
            </svg>
            <span>滇ICP备2026013786号-1</span>
          </a>
          <a
            href="https://beian.mps.gov.cn/#/query/webSearch?code=53011102001665"
            target="_blank"
            rel="noreferrer"
            class="group inline-flex items-center justify-center gap-1.5 whitespace-nowrap py-1 transition-colors duration-200 hover:text-primary-600 dark:hover:text-primary-400"
          >
            <img
              src="/assets/image/gongan-beian.png"
              alt="公安备案图标"
              width="20"
              height="20"
              class="h-4 w-4 opacity-80 transition-opacity group-hover:opacity-100"
            />
            <span>滇公网安备53011102001665号</span>
          </a>
        </div>
      </footer>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>
