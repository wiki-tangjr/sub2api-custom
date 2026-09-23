<template>
  <div :class="rootClass">
    <div v-if="title || description" :class="variant === 'dropdown' ? 'mb-2 px-1' : 'mb-3'">
      <p v-if="title" class="text-xs font-semibold text-gray-600 dark:text-gray-300">{{ title }}</p>
      <p v-if="description" class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">
        {{ description }}
      </p>
    </div>

    <!-- 魔改 #29: 所有使用场景共用同一套分组与条目骨架，只有密度不同。 -->
    <template v-for="(group, groupIndex) in groupedEntries" :key="group.key + ':' + groupIndex">
      <div
        v-if="group.title"
        class="flex items-center gap-2"
        :class="groupIndex === 0 ? 'mb-2' : 'mb-2 mt-4'"
      >
        <p class="shrink-0 text-xs font-medium text-gray-500 dark:text-gray-400">
          {{ group.title }}
        </p>
        <span v-if="variant !== 'list'" class="h-px flex-1 bg-gray-200 dark:bg-dark-700" />
      </div>

      <div :class="[listClass, groupIndex > 0 && !group.title ? 'mt-2' : '']">
        <template v-for="(item, index) in group.items" :key="item.id || String(index)">
          <!-- 联系方式总览：每个条目都是结构一致、宽度稳定的详情行。 -->
          <div
            v-if="variant === 'sheet'"
            data-testid="contact-sheet-entry"
            class="rounded-lg border border-gray-200 bg-gray-50/70 p-3 dark:border-dark-700 dark:bg-dark-900/30"
          >
            <div class="flex min-w-0 items-center gap-3">
              <ContactEntryIcon :entry="item" size="md" />
              <p class="min-w-0 flex-1 truncate text-sm font-semibold text-gray-900 dark:text-gray-100">
                {{ item.label }}
              </p>
            </div>
            <ContactEntryBody
              class="mt-3"
              :entry="item"
              compact
              @copied="notifyCopied"
              @failed="notifyCopyFailed"
            />
          </div>

          <!-- 直接打开的链接。 -->
          <a
            v-else-if="displayOf(item) === 'inline' && item.type === 'link'"
            :href="safeLink(item)"
            :target="targetOf(item)"
            :rel="targetOf(item) === '_blank' ? 'noopener noreferrer' : undefined"
            :class="entryClass"
          >
            <ContactEntryIcon :entry="item" :size="variant === 'list' ? 'sm' : 'md'" />
            <span class="min-w-0 flex-1">
              <span class="block truncate">{{ item.label }}</span>
              <span
                v-if="item.description && variant !== 'list'"
                class="mt-0.5 block truncate text-xs font-normal text-gray-500 dark:text-gray-400"
              >
                {{ item.description }}
              </span>
            </span>
            <Icon name="externalLink" size="sm" class="shrink-0 opacity-55" />
          </a>

          <!-- 可复制文本：整行都是明确的复制动作，不再嵌套输入框形按钮。 -->
          <button
            v-else-if="displayOf(item) === 'inline' && item.type === 'text'"
            type="button"
            :class="inlineTextClass"
            :title="t('common.copy')"
            @click="copyInline(item)"
          >
            <ContactEntryIcon :entry="item" :size="variant === 'list' ? 'sm' : 'md'" />
            <span class="min-w-0 flex-1 text-left">
              <span class="block truncate font-medium">{{ item.label }}</span>
              <code class="mt-0.5 block truncate font-mono text-xs font-normal text-gray-500 dark:text-gray-400">
                {{ item.value }}
              </code>
            </span>
            <Icon name="copy" size="sm" class="shrink-0 opacity-60" />
          </button>

          <!-- 内联二维码仅用于管理员明确选择内联的卡片场景。 -->
          <div
            v-else-if="displayOf(item) === 'inline' && item.type === 'qrcode' && variant !== 'list'"
            class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800"
          >
            <div class="mb-3 flex items-center gap-3">
              <ContactEntryIcon :entry="item" size="md" />
              <p class="min-w-0 flex-1 truncate text-sm font-semibold text-gray-900 dark:text-gray-100">
                {{ item.label }}
              </p>
            </div>
            <ContactEntryBody :entry="item" compact />
          </div>

          <!-- 弹窗 / 悬停入口。 -->
          <div
            v-else
            class="relative min-w-0"
            @mouseenter="hoverIn(item)"
            @mouseleave="hoverOut(item)"
            @focusin="hoverIn(item)"
            @focusout="hoverOut(item)"
          >
            <button
              type="button"
              :class="entryClass"
              :aria-expanded="displayOf(item) === 'hover' ? hoverShown(item) : undefined"
              :aria-haspopup="displayOf(item) === 'modal' || !canHover ? 'dialog' : undefined"
              @click="activate(item)"
            >
              <ContactEntryIcon :entry="item" :size="variant === 'list' ? 'sm' : 'md'" />
              <span class="min-w-0 flex-1">
                <span class="block truncate">{{ item.label }}</span>
                <span
                  v-if="item.description && variant !== 'list'"
                  class="mt-0.5 block truncate text-xs font-normal text-gray-500 dark:text-gray-400"
                >
                  {{ item.description }}
                </span>
              </span>
              <Icon name="chevronRight" size="sm" class="shrink-0 opacity-45" />
            </button>

            <!-- pt-2 仍是可悬停区域的一部分，避免按钮和浮层之间产生鼠标死区。 -->
            <div
              v-if="hoverShown(item)"
              data-testid="contact-hover-panel"
              class="contact-hover-panel absolute top-full z-[60] pt-2"
              :class="hoverCardClass"
            >
              <div class="overflow-hidden rounded-lg border border-gray-200 bg-white text-left shadow-xl dark:border-dark-600 dark:bg-dark-800">
                <div class="flex min-w-0 items-center gap-3 border-b border-gray-100 px-3 py-2.5 dark:border-dark-700">
                  <ContactEntryIcon :entry="item" size="md" />
                  <p class="min-w-0 flex-1 truncate text-sm font-semibold text-gray-900 dark:text-gray-100">
                    {{ item.label }}
                  </p>
                </div>
                <div class="p-3">
                  <ContactEntryBody
                    :entry="item"
                    compact
                    @copied="notifyCopied"
                    @failed="notifyCopyFailed"
                  />
                </div>
              </div>
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
    variant?: 'card' | 'list' | 'dropdown' | 'sheet'
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
  if (props.variant === 'dropdown') return 'flex flex-col gap-1.5'
  if (props.variant === 'sheet') return 'flex flex-col gap-2'
  return 'grid grid-cols-1 gap-2 sm:grid-cols-2'
})

const rootClass = computed(() => props.variant === 'list' ? 'inline-block max-w-full' : 'w-full')

// 魔改 #29: 浮层宽度由稳定的 CSS min() 控制，不再被内部文本最小宽度撑破。
// 顶栏下拉靠右对齐，其余使用场景靠左对齐。
const hoverCardClass = computed(() => {
  if (props.variant === 'dropdown') return 'contact-hover-dropdown'
  if (props.variant === 'list') return 'contact-hover-list'
  return 'contact-hover-card'
})

const entryClass = computed(() => {
  if (props.variant === 'dropdown') {
    return 'group flex w-full min-w-0 items-center gap-2.5 rounded-lg border border-transparent px-2.5 py-2 text-left text-sm font-medium text-gray-700 transition-colors hover:border-gray-200 hover:bg-gray-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-gray-200 dark:hover:border-dark-600 dark:hover:bg-dark-700/70'
  }
  if (props.variant === 'list') {
    return 'inline-flex max-w-full items-center gap-1.5 rounded-md text-sm font-medium text-primary-600 transition-colors hover:text-primary-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-primary-400 dark:hover:text-primary-300'
  }
  return 'group flex w-full min-w-0 items-center gap-3 rounded-lg border border-gray-200 bg-white px-3 py-2.5 text-left text-sm font-medium text-gray-800 shadow-sm transition-colors hover:border-primary-300 hover:bg-primary-50/40 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100 dark:hover:border-primary-700 dark:hover:bg-primary-900/15'
})

const inlineTextClass = computed(() => {
  if (props.variant === 'list') {
    return 'inline-flex max-w-full items-center gap-1.5 rounded-md text-sm text-gray-700 transition-colors hover:text-primary-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-gray-200 dark:hover:text-primary-300'
  }
  return entryClass.value
})

function displayOf(item: ContactEntry): 'modal' | 'hover' | 'inline' {
  if (props.forceInline) return 'inline'
  if (item.display === 'hover' || item.display === 'inline') return item.display
  return 'modal'
}

function safeLink(item: ContactEntry): string {
  return sanitizeUrl(item.url || '')
}

function targetOf(item: ContactEntry): string {
  return item.open_target === 'current_tab' ? '_self' : '_blank'
}

// 魔改 #24: 悬停卡片延时关闭。
// 鼠标从按钮移向卡片时，即使中间出现短暂的 mouseleave（快速移动、越界抖动），
// 只要在 HOVER_CLOSE_DELAY 内重新进入 wrapper 就会取消关闭，卡片不会消失，
// 里面的复制按钮因此可以稳定点到。
const HOVER_CLOSE_DELAY = 180
let hoverCloseTimer: ReturnType<typeof setTimeout> | null = null

function clearHoverCloseTimer() {
  if (hoverCloseTimer !== null) {
    clearTimeout(hoverCloseTimer)
    hoverCloseTimer = null
  }
}

function hoverIn(item: ContactEntry) {
  clearHoverCloseTimer()
  if (displayOf(item) === 'hover') hoveredId.value = item.id
}

function hoverOut(item: ContactEntry) {
  clearHoverCloseTimer()
  hoverCloseTimer = setTimeout(() => {
    hoverCloseTimer = null
    if (hoveredId.value === item.id) hoveredId.value = ''
  }, HOVER_CLOSE_DELAY)
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
  clearHoverCloseTimer()
  hoverMedia?.removeEventListener?.('change', updateHoverCapability)
})

</script>

<style scoped>
.contact-hover-panel {
  width: min(20rem, calc(100vw - 1rem));
  max-width: calc(100vw - 1rem);
  box-sizing: border-box;
}

.contact-hover-dropdown {
  right: 0;
}

.contact-hover-list,
.contact-hover-card {
  left: 0;
}
</style>
