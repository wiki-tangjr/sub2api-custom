<template>
  <div :class="variant === 'list' ? 'inline-block' : 'w-full'">
    <div v-if="title || description" class="mb-2">
      <p v-if="title" class="text-xs font-semibold text-gray-500 dark:text-gray-400">{{ title }}</p>
      <p v-if="description" class="mt-1 text-xs text-gray-400 dark:text-gray-500">{{ description }}</p>
    </div>

    <!-- 魔改 #23: 可选分组。没有任何 group 时下面的结构与 #12 完全一致，
         所以老配置的前台表现不会发生变化。 -->
    <template v-for="(group, groupIndex) in groupedEntries" :key="group.key + ':' + groupIndex">
      <div v-if="group.title" :class="groupIndex === 0 ? '' : 'mt-3'">
        <p class="mb-1.5 text-[11px] font-semibold uppercase tracking-wide text-gray-400 dark:text-dark-400">
          {{ group.title }}
        </p>
      </div>
      <div :class="[listClass, groupIndex > 0 ? 'mt-1' : '']">
        <template v-for="(item, index) in group.items" :key="item.id || String(index)">
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
            <button
              type="button"
              :class="entryClass"
              :aria-expanded="hoverShown(item)"
              @click="activate(item)"
            >
              <ContactEntryIcon :entry="item" />
              <span class="truncate">{{ item.label }}</span>
              <Icon
                :name="item.type === 'link' ? 'externalLink' : 'chevronDown'"
                size="sm"
                class="opacity-60"
              />
            </button>

            <!-- 魔改 #23: 悬停卡片加宽并带图标 + 说明，避免长内容被挤成一团 -->
            <div
              v-if="hoverShown(item)"
              class="absolute top-full z-40 mt-1.5 max-w-[calc(100vw-2rem)] rounded-xl border border-gray-200 bg-white p-3.5 text-left shadow-xl dark:border-dark-600 dark:bg-dark-800"
              :class="hoverCardClass"
            >
              <div class="mb-2 flex items-center gap-2">
                <ContactEntryIcon :entry="item" size="md" />
                <p class="min-w-0 flex-1 truncate text-xs font-semibold text-gray-700 dark:text-gray-200">
                  {{ item.label }}
                </p>
              </div>
              <ContactEntryBody :entry="item" @copied="notifyCopied" @failed="notifyCopyFailed" />
            </div>
          </div>
        </template>
      </div>
    </template>

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
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
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
// 触屏设备没有 hover，点击时必须能打开弹窗，否则该条目等于不可用。
const canHover = ref(true)

const safeEntries = computed(() => {
  return (props.entries || []).filter((item) => {
    if (!item || item.enabled === false) return false
    if (item.type === 'link') return !!sanitizeUrl(item.url || '')
    if (item.type === 'qrcode') return !!sanitizeUrl(item.qr_code || '', { allowDataUrl: true })
    if (item.type === 'text') return !!String(item.value || '').trim()
    return false
  })
})

interface ContactEntryGroup {
  key: string
  title: string
  items: ContactEntry[]
}

// 魔改 #23: 同 group 的相邻条目归为一组；没有分组时退化成单个匿名组，
// 渲染结果与 #12 保持完全一致。
const groupedEntries = computed<ContactEntryGroup[]>(() => {
  const items = safeEntries.value
  const hasGroup = items.some((item) => (item.group || '').trim() !== '')
  if (!hasGroup) return [{ key: 'all', title: '', items }]

  const groups: ContactEntryGroup[] = []
  for (const item of items) {
    const title = (item.group || '').trim()
    const last = groups[groups.length - 1]
    if (last && last.title === title) {
      last.items.push(item)
      continue
    }
    groups.push({ key: title || '__ungrouped__', title, items: [item] })
  }
  return groups
})

const listClass = computed(() => {
  if (props.variant === 'list') return 'flex flex-wrap items-center gap-x-4 gap-y-2'
  if (props.variant === 'dropdown') return 'flex flex-col gap-1'
  return 'flex flex-wrap gap-2'
})

// 顶栏下拉菜单在屏幕右侧，悬浮卡片靠右对齐才不会越界；
// 其余场景保持左对齐，与 #12 一致。
const hoverCardClass = computed(() => {
  if (props.variant === 'dropdown') return 'right-0 w-64'
  if (props.variant === 'list') return 'left-0 w-64'
  return 'left-0 w-72'
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
  return canHover.value && displayOf(item) === 'hover' && hoveredId.value !== '' && hoveredId.value === item.id
}

function activate(item: ContactEntry) {
  // 悬停条目在有 hover 能力的设备上由鼠标悬停展示，点击不做事（保持 #12 行为）；
  // 触屏设备无法悬停，点击时退回弹窗，保证内容始终可达。
  if (displayOf(item) === 'hover' && canHover.value) return
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

function updateHoverCapability() {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    canHover.value = true
    return
  }
  canHover.value = window.matchMedia('(hover: hover) and (pointer: fine)').matches
}

let hoverMedia: MediaQueryList | null = null

onMounted(() => {
  updateHoverCapability()
  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    hoverMedia = window.matchMedia('(hover: hover) and (pointer: fine)')
    hoverMedia.addEventListener?.('change', updateHoverCapability)
  }
})

onBeforeUnmount(() => {
  hoverMedia?.removeEventListener?.('change', updateHoverCapability)
})

</script>
