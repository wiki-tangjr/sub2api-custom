<template>
  <div :class="variant === 'list' ? 'inline-block' : 'w-full'">
    <div v-if="title || description" class="mb-2">
      <p v-if="title" class="text-xs font-semibold text-gray-500 dark:text-gray-400">{{ title }}</p>
      <p v-if="description" class="mt-1 text-xs text-gray-400 dark:text-gray-500">{{ description }}</p>
    </div>

    <div :class="listClass">
      <template v-for="(item, index) in safeEntries" :key="item.id || String(index)">
        <!-- inline link -->
        <a
          v-if="displayOf(item) === 'inline' && item.type === 'link'"
          :href="safeLink(item)"
          :target="targetOf(item)"
          :rel="targetOf(item) === '_blank' ? 'noopener noreferrer' : undefined"
          :class="entryClass"
        >
          <ContactEntryIcon :entry="item" />
          <span class="truncate">{{ item.label }}</span>
        </a>

        <!-- inline text -->
        <span
          v-else-if="displayOf(item) === 'inline' && item.type === 'text'"
          class="inline-flex max-w-full items-center gap-2 rounded-lg bg-gray-100 px-2 py-1 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200"
        >
          <ContactEntryIcon :entry="item" />
          <span class="font-medium">{{ item.label }}：</span>
          <code class="select-all truncate font-mono">{{ item.value }}</code>
          <button type="button" class="text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="copyInline(item)">
            <Icon name="copy" size="sm" />
          </button>
        </span>

        <!-- inline qrcode -->
        <span
          v-else-if="displayOf(item) === 'inline' && item.type === 'qrcode'"
          class="inline-flex flex-col items-center gap-1 rounded-lg border border-gray-200 p-2 dark:border-dark-600"
        >
          <ContactEntryIcon :entry="item" size="md" />
          <span class="text-xs font-medium text-gray-700 dark:text-gray-200">{{ item.label }}</span>
          <img
            v-if="safeQrOf(item)"
            :src="safeQrOf(item)"
            :alt="item.label"
            class="h-32 w-32 object-contain"
          />
        </span>

        <!-- modal / hover -->
        <div
          v-else
          class="relative"
          @mouseenter="hoverIn(item)"
          @mouseleave="hoverOut(item)"
        >
          <button type="button" :class="entryClass" @click="activate(item)">
            <ContactEntryIcon :entry="item" />
            <span class="truncate">{{ item.label }}</span>
            <Icon
              :name="item.type === 'link' ? 'externalLink' : 'chevronDown'"
              size="sm"
              class="opacity-60"
            />
          </button>

          <div
            v-if="hoverShown(item)"
            class="absolute left-0 top-full z-30 mt-1 w-64 rounded-xl border border-gray-200 bg-white p-3 text-left shadow-xl dark:border-dark-600 dark:bg-dark-800"
          >
            <p class="mb-2 text-xs font-semibold text-gray-700 dark:text-gray-200">{{ item.label }}</p>
            <ContactEntryBody :entry="item" @copied="notifyCopied" @failed="notifyCopyFailed" />
          </div>
        </div>
      </template>
    </div>

    <BaseDialog :show="modalOpen" :title="modalTitle" width="narrow" @close="closeModal">
      <ContactEntryBody
        v-if="modalEntry"
        :entry="modalEntry"
        @copied="notifyCopied"
        @failed="notifyCopyFailed"
      />
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ContactEntry } from '@/types'
import { sanitizeUrl } from '@/utils/url'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import ContactEntryIcon from '@/components/common/ContactEntryIcon.vue'
import ContactEntryBody from '@/components/common/ContactEntryBody.vue'
import { useAppStore } from '@/stores/app'

const props = withDefaults(
  defineProps<{
    entries?: ContactEntry[]
    title?: string
    description?: string
    variant?: 'card' | 'list' | 'dropdown'
    forceInline?: boolean
  }>(),
  {
    entries: () => [],
    title: '',
    description: '',
    variant: 'card',
    forceInline: false,
  },
)

const { t } = useI18n()
const appStore = useAppStore()

const hoveredId = ref('')
const modalEntry = ref<ContactEntry | null>(null)

const safeEntries = computed(() => {
  return (props.entries || []).filter((item) => {
    if (!item || item.enabled === false) return false
    if (item.type === 'link') return !!sanitizeUrl(item.url || '')
    if (item.type === 'qrcode') return !!sanitizeUrl(item.qr_code || '', { allowDataUrl: true })
    if (item.type === 'text') return !!String(item.value || '').trim()
    return false
  })
})

const listClass = computed(() => {
  if (props.variant === 'list') return 'flex flex-wrap items-center gap-x-4 gap-y-2'
  if (props.variant === 'dropdown') return 'flex flex-col gap-1'
  return 'flex flex-wrap gap-2'
})

const entryClass = computed(() => {
  if (props.variant === 'dropdown') {
    return 'flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-xs font-medium text-primary-600 transition-colors hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/30'
  }
  if (props.variant === 'list') {
    return 'inline-flex items-center gap-1.5 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300'
  }
  return 'btn btn-secondary btn-sm inline-flex items-center gap-2'
})

function displayOf(item: ContactEntry): 'modal' | 'hover' | 'inline' {
  if (props.forceInline) return 'inline'
  if (item.display === 'hover' || item.display === 'inline') return item.display
  return 'modal'
}

function safeLink(item: ContactEntry): string {
  return sanitizeUrl(item.url || '')
}

function safeQrOf(item: ContactEntry): string {
  return sanitizeUrl(item.qr_code || '', { allowDataUrl: true })
}

function targetOf(item: ContactEntry): string {
  return item.open_target === 'current_tab' ? '_self' : '_blank'
}

function hoverIn(item: ContactEntry) {
  if (displayOf(item) === 'hover') hoveredId.value = item.id
}

function hoverOut(item: ContactEntry) {
  if (hoveredId.value === item.id) hoveredId.value = ''
}

function hoverShown(item: ContactEntry): boolean {
  return displayOf(item) === 'hover' && hoveredId.value !== '' && hoveredId.value === item.id
}

function activate(item: ContactEntry) {
  // Hover entries are revealed by mouse; inline entries render directly.
  // Everything else (including links) opens the shared modal so the link
  // button / QR image / copy action stays reachable.
  if (displayOf(item) === 'hover') return
  modalEntry.value = item
}

const modalOpen = computed(() => modalEntry.value !== null)
const modalTitle = computed(() => modalEntry.value?.label || t('common.contactSupport'))

function closeModal() {
  modalEntry.value = null
}

async function copyInline(item: ContactEntry) {
  const value = String(item.value || '').trim()
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
    notifyCopied()
  } catch (error) {
    console.error('Copy contact entry failed:', error)
    notifyCopyFailed()
  }
}

function notifyCopied() {
  appStore.showSuccess(t('common.copiedToClipboard'))
}

function notifyCopyFailed() {
  appStore.showError(t('common.copyFailed'))
}
</script>