<template>
  <span
    class="flex flex-shrink-0 items-center justify-center overflow-hidden rounded-full"
    :class="boxClass"
  >
    <img
      v-if="entry.icon_type === 'image' && safeIcon"
      :src="safeIcon"
      alt=""
      class="h-full w-full object-contain"
    />
    <span v-else class="leading-none" :class="emojiClass">{{ resolvedEmoji }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ContactEntry } from '@/types'
import { sanitizeUrl } from '@/utils/url'

const props = withDefaults(defineProps<{ entry: ContactEntry; size?: 'sm' | 'md' | 'lg' }>(), {
  size: 'sm',
})

const safeIcon = computed(() =>
  sanitizeUrl(props.entry.icon || '', { allowDataUrl: true }),
)

const resolvedEmoji = computed(() => (props.entry.icon || '').trim() || '💬')

const boxClass = computed(() => {
  if (props.size === 'lg') return 'h-12 w-12 bg-gray-100 dark:bg-dark-700'
  if (props.size === 'md') return 'h-8 w-8 bg-gray-100 dark:bg-dark-700'
  return 'h-6 w-6'
})

const emojiClass = computed(() => {
  if (props.size === 'lg') return 'text-2xl'
  if (props.size === 'md') return 'text-lg'
  return 'text-sm'
})
</script>