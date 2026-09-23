<template>
  <div class="min-w-0">
    <!-- QR code -->
    <div v-if="entry.type === 'qrcode'" class="text-center">
      <div
        v-if="safeQr"
        class="mx-auto w-fit max-w-full rounded-lg border border-gray-200 bg-white p-2 shadow-sm dark:border-dark-600"
      >
        <img
          :src="safeQr"
          :alt="entry.label"
          :class="qrClass"
        />
      </div>
      <p v-else class="text-xs text-red-500">{{ t('common.contactInvalidQrCode') }}</p>
      <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">
        {{ entry.description || t('common.contactScanHint') }}
      </p>
    </div>

    <!-- Text value -->
    <div v-else-if="entry.type === 'text'" class="space-y-2">
      <div class="flex min-w-0 items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 p-1.5 pl-3 dark:border-dark-700 dark:bg-dark-900/40">
        <code
          class="min-w-0 flex-1 select-all break-all font-mono text-sm text-gray-800 dark:text-gray-100"
        >{{ entry.value }}</code>
        <button
          type="button"
          data-testid="contact-copy-button"
          class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-gray-200 bg-white text-gray-500 transition-colors hover:border-primary-300 hover:bg-primary-50 hover:text-primary-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-primary-700 dark:hover:bg-primary-900/30 dark:hover:text-primary-300"
          :title="t('common.copy')"
          :aria-label="t('common.copy')"
          @click="copyText"
        >
          <Icon name="copy" size="sm" />
        </button>
      </div>
      <p v-if="entry.description" class="text-xs leading-5 text-gray-500 dark:text-gray-400">
        {{ entry.description }}
      </p>
    </div>

    <!-- Link -->
    <div v-else-if="entry.type === 'link'" class="space-y-2">
      <a
        v-if="safeUrl"
        :href="safeUrl"
        :target="targetAttr"
        :rel="targetAttr === '_blank' ? 'noopener noreferrer' : undefined"
        :class="linkClass"
      >
        <span class="min-w-0 flex-1 text-left">{{ t('common.contactOpenLink') }}</span>
        <Icon name="externalLink" size="sm" class="shrink-0 opacity-70" />
      </a>
      <p v-else class="text-xs text-red-500">{{ t('common.contactInvalidUrl') }}</p>
      <p v-if="entry.description" class="text-xs leading-5 text-gray-500 dark:text-gray-400">
        {{ entry.description }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ContactEntry } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'

const props = withDefaults(defineProps<{ entry: ContactEntry; compact?: boolean }>(), {
  compact: false,
})

const emit = defineEmits<{
  copied: []
  failed: []
}>()

const { t } = useI18n()

const safeQr = computed(() => sanitizeUrl(props.entry.qr_code || '', { allowDataUrl: true }))
const safeUrl = computed(() => sanitizeUrl(props.entry.url || ''))
const targetAttr = computed(() =>
  props.entry.open_target === 'current_tab' ? '_self' : '_blank',
)
const qrClass = computed(() => props.compact
  ? 'h-36 w-36 max-w-full object-contain'
  : 'max-h-[min(60vh,320px)] w-auto max-w-full object-contain',
)
const linkClass = computed(() => [
  'inline-flex w-full items-center gap-3 rounded-lg border border-primary-200 bg-primary-50 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:bg-primary-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300 dark:hover:border-primary-700 dark:hover:bg-primary-900/35',
  props.compact ? 'px-3 py-2' : 'px-3.5 py-2.5',
])

async function copyText() {
  const value = String(props.entry.value || '').trim()
  if (!value) return
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(value)
    } else {
      const textarea = document.createElement('textarea')
      textarea.value = value
      textarea.setAttribute('readonly', '')
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
    }
    emit('copied')
  } catch (error) {
    console.error('Copy contact entry failed:', error)
    emit('failed')
  }
}
</script>
