<template>
  <div>
    <!-- QR code -->
    <div v-if="entry.type === 'qrcode'" class="text-center">
      <img
        v-if="safeQr"
        :src="safeQr"
        :alt="entry.label"
        class="mx-auto max-h-[min(60vh,360px)] w-auto max-w-full rounded-lg object-contain"
      />
      <p v-else class="text-xs text-red-500">{{ t('common.contactInvalidQrCode') }}</p>
      <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
        {{ entry.description || t('common.contactScanHint') }}
      </p>
    </div>

    <!-- Text value -->
    <div v-else-if="entry.type === 'text'" class="space-y-2">
      <div class="flex items-center gap-2">
        <code
          class="min-w-0 flex-1 select-all truncate rounded-lg bg-gray-100 px-2.5 py-1.5 font-mono text-xs text-gray-800 dark:bg-dark-700 dark:text-gray-200"
        >{{ entry.value }}</code>
        <button type="button" class="btn btn-secondary btn-sm" @click="copyText">
          <Icon name="copy" size="sm" />
          {{ t('common.copy') }}
        </button>
      </div>
      <p v-if="entry.description" class="text-xs text-gray-500 dark:text-gray-400">
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
        class="btn btn-primary btn-sm inline-flex w-full items-center justify-center gap-2"
      >
        <Icon name="externalLink" size="sm" />
        {{ t('common.contactOpenLink') }}
      </a>
      <p v-else class="text-xs text-red-500">{{ t('common.contactInvalidUrl') }}</p>
      <p v-if="entry.description" class="text-xs text-gray-500 dark:text-gray-400">
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

const props = defineProps<{ entry: ContactEntry }>()

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