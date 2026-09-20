<template>
  <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
    <div class="mb-3 flex items-start justify-between gap-3">
      <div>
        <p class="text-sm font-semibold text-gray-700 dark:text-gray-300">
          {{ t('admin.settings.site.contactEntries.title') }}
        </p>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.site.contactEntries.hint') }}
        </p>
      </div>
      <span class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300">
        {{ entries.length }} / {{ maxEntries }}
      </span>
    </div>

    <div v-if="entries.length === 0" class="rounded-lg border border-dashed border-gray-300 px-4 py-6 text-center text-xs text-gray-500 dark:border-dark-600 dark:text-gray-400">
      {{ t('admin.settings.site.contactEntries.empty') }}
    </div>

    <div v-else class="space-y-3">
      <div
        v-for="(item, index) in entries"
        :key="item.id"
        class="rounded-lg border border-gray-200 bg-gray-50/60 p-3 dark:border-dark-700 dark:bg-dark-800/40"
      >
        <div class="mb-3 flex items-center justify-between gap-2">
          <div class="flex min-w-0 items-center gap-2">
            <Toggle v-model="item.enabled" />
            <ContactEntryIcon :entry="item" />
            <span class="truncate text-sm font-medium text-gray-800 dark:text-gray-200">
              {{ item.label || t('admin.settings.site.contactEntries.untitled') }}
            </span>
            <span class="rounded bg-gray-200 px-1.5 py-0.5 text-[10px] uppercase text-gray-600 dark:bg-dark-700 dark:text-gray-300">
              {{ item.type }}
            </span>
            <span
              v-if="(item.group || '').trim()"
              class="rounded bg-primary-100 px-1.5 py-0.5 text-[10px] text-primary-700 dark:bg-primary-900/40 dark:text-primary-300"
            >
              {{ (item.group || '').trim() }}
            </span>
          </div>
          <div class="flex flex-shrink-0 items-center gap-1">
            <button type="button" class="btn-ghost btn-icon" :disabled="index === 0" :title="t('admin.settings.site.contactEntries.moveUp')" @click="move(index, -1)">
              <Icon name="arrowUp" size="sm" />
            </button>
            <button type="button" class="btn-ghost btn-icon" :disabled="index === entries.length - 1" :title="t('admin.settings.site.contactEntries.moveDown')" @click="move(index, 1)">
              <Icon name="arrowDown" size="sm" />
            </button>
            <button type="button" class="btn-ghost btn-icon" :title="t('admin.settings.site.contactEntries.preview')" @click="previewEntry = item">
              <Icon name="eye" size="sm" />
            </button>
            <button type="button" class="btn-ghost btn-icon text-red-600 dark:text-red-400" :title="t('common.remove')" @click="remove(index)">
              <Icon name="trash" size="sm" />
            </button>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.settings.site.contactEntries.label') }}
            </label>
            <input v-model="item.label" type="text" class="input text-sm" :maxlength="50" :placeholder="t('admin.settings.site.contactEntries.labelPlaceholder')" />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.settings.site.contactEntries.type') }}
            </label>
            <select v-model="item.type" class="input text-sm" @change="onTypeChange(item)">
              <option value="link">{{ t('admin.settings.site.contactEntries.typeLink') }}</option>
              <option value="qrcode">{{ t('admin.settings.site.contactEntries.typeQrcode') }}</option>
              <option value="text">{{ t('admin.settings.site.contactEntries.typeText') }}</option>
            </select>
          </div>
        </div>

        <!-- 魔改 #23: 分组名。同名 + 相邻的条目会在前台聚成一组并显示小标题。 -->
        <div class="mt-3">
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
            {{ t('admin.settings.site.contactEntries.group') }}
          </label>
          <input
            v-model="item.group"
            type="text"
            class="input text-sm"
            :maxlength="30"
            :list="groupDatalistId"
            :placeholder="t('admin.settings.site.contactEntries.groupPlaceholder')"
            @change="commit(entries.slice())"
          />
          <p class="mt-1 text-xs text-gray-400 dark:text-dark-400">
            {{ t('admin.settings.site.contactEntries.groupHint') }}
          </p>
        </div>

        <div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.settings.site.contactEntries.iconType') }}
            </label>
            <div class="flex items-center gap-2">
              <select v-model="item.icon_type" class="input text-sm" @change="commit(entries.slice())">
                <option value="emoji">{{ t('admin.settings.site.contactEntries.iconTypeEmoji') }}</option>
                <option value="image">{{ t('admin.settings.site.contactEntries.iconTypeImage') }}</option>
              </select>
              <ContactEntryIcon :entry="item" size="md" />
            </div>
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.settings.site.contactEntries.display') }}
            </label>
            <select v-model="item.display" class="input text-sm" @change="commit(entries.slice())">
              <option value="modal">{{ t('admin.settings.site.contactEntries.displayModal') }}</option>
              <option value="hover">{{ t('admin.settings.site.contactEntries.displayHover') }}</option>
              <option value="inline">{{ t('admin.settings.site.contactEntries.displayInline') }}</option>
            </select>
            <p class="mt-1 text-xs text-gray-400 dark:text-dark-400">
              {{ displayHint(item) }}
            </p>
          </div>
        </div>

        <div class="mt-3">
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
            {{ t('admin.settings.site.contactEntries.icon') }}
          </label>
          <input
            v-if="item.icon_type !== 'image'"
            v-model="item.icon"
            type="text"
            class="input text-sm"
            :maxlength="8"
            :placeholder="t('admin.settings.site.contactEntries.iconPlaceholder')"
          />
          <ImageUpload
            v-else
            v-model="item.icon"
            mode="image"
            size="sm"
            :upload-label="t('admin.settings.site.uploadImage')"
            :remove-label="t('admin.settings.site.remove')"
            :max-size="500 * 1024"
          />
        </div>

        <div v-if="item.type === 'link'" class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.settings.site.contactEntries.url') }}
            </label>
            <input v-model="item.url" type="url" class="input font-mono text-xs" :maxlength="2048" placeholder="https://t.me/your_group" />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.settings.site.contactEntries.openTarget') }}
            </label>
            <select v-model="item.open_target" class="input text-sm" @change="commit(entries.slice())">
              <option value="new_tab">{{ t('admin.settings.site.contactEntries.openNewTab') }}</option>
              <option value="current_tab">{{ t('admin.settings.site.contactEntries.openCurrentTab') }}</option>
            </select>
            <p class="mt-1 text-xs text-gray-400 dark:text-dark-400">
              {{ t('admin.settings.site.contactEntries.openTargetHint') }}
            </p>
          </div>
        </div>

        <div v-else-if="item.type === 'qrcode'" class="mt-3">
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
            {{ t('admin.settings.site.contactEntries.qrCode') }}
          </label>
          <ImageUpload
            :model-value="item.qr_code || ''"
            mode="image"
            @update:model-value="(value: string) => (item.qr_code = value)"
            size="sm"
            :upload-label="t('admin.settings.site.uploadImage')"
            :remove-label="t('admin.settings.site.remove')"
            :max-size="2 * 1024 * 1024"
          />
          <p class="mt-1 text-xs text-gray-400 dark:text-dark-400">
            {{ t('admin.settings.site.contactEntries.qrHint') }}
          </p>
        </div>

        <div v-else class="mt-3">
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
            {{ t('admin.settings.site.contactEntries.value') }}
          </label>
          <input v-model="item.value" type="text" class="input text-sm" :maxlength="200" :placeholder="t('admin.settings.site.contactEntries.valuePlaceholder')" />
          <p class="mt-1 text-xs text-gray-400 dark:text-dark-400">
            {{ t('admin.settings.site.contactEntries.valueHint') }}
          </p>
        </div>

        <div class="mt-3">
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
            {{ t('admin.settings.site.contactEntries.description') }}
          </label>
          <input v-model="item.description" type="text" class="input text-sm" :maxlength="200" :placeholder="t('admin.settings.site.contactEntries.descriptionPlaceholder')" />
        </div>
      </div>
    </div>

    <button
      type="button"
      class="mt-3 flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed border-gray-300 px-4 py-2.5 text-sm text-gray-500 transition-colors hover:border-primary-400 hover:text-primary-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:text-gray-400 dark:hover:border-primary-500 dark:hover:text-primary-400"
      :disabled="entries.length >= maxEntries"
      @click="add"
    >
      <Icon name="plus" size="sm" />
      {{ t('admin.settings.site.contactEntries.add') }}
    </button>

    <!-- 魔改 #23: 前台效果实时预览，管理员改完即可对照，不用反复切换到前台确认。 -->
    <div v-if="entries.length > 0" class="mt-4 rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800/60">
      <div class="mb-3 flex items-center justify-between gap-2">
        <p class="text-xs font-semibold text-gray-600 dark:text-gray-300">
          {{ t('admin.settings.site.contactEntries.previewTitle') }}
        </p>
        <span class="text-[11px] text-gray-400 dark:text-dark-400">
          {{ t('admin.settings.site.contactEntries.previewHint') }}
        </span>
      </div>
      <ContactEntries :entries="previewEntries" :title="previewSectionTitle" variant="card" />
    </div>

    <datalist :id="groupDatalistId">
      <option v-for="name in groupOptions" :key="name" :value="name" />
    </datalist>

    <BaseDialog :show="previewEntry !== null" :title="previewEntry?.label || ''" width="narrow" @close="previewEntry = null">
      <ContactEntryBody
        v-if="previewEntry"
        :entry="previewEntry"
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
import { useAppStore } from '@/stores'
import Icon from '@/components/icons/Icon.vue'
import ImageUpload from '@/components/common/ImageUpload.vue'
import Toggle from '@/components/common/Toggle.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ContactEntries from '@/components/common/ContactEntries.vue'
import ContactEntryIcon from '@/components/common/ContactEntryIcon.vue'
import ContactEntryBody from '@/components/common/ContactEntryBody.vue'

const props = defineProps<{
  modelValue: ContactEntry[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: ContactEntry[]]
}>()

const { t } = useI18n()
const appStore = useAppStore()

const maxEntries = 20

const entries = computed<ContactEntry[]>(() => props.modelValue || [])

// 预览里不显示停用条目，与前台真实表现保持一致。
const previewEntries = computed(() => entries.value.filter((item) => item.enabled !== false))
const previewSectionTitle = computed(() => t('common.contactSectionDefaultTitle'))

const previewEntry = ref<ContactEntry | null>(null)

const groupDatalistId = 'contact-entry-groups'

const groupOptions = computed(() => {
  const seen = new Set<string>()
  for (const item of entries.value) {
    const name = (item.group || '').trim()
    if (name) seen.add(name)
  }
  return Array.from(seen)
})

function displayHint(item: ContactEntry): string {
  if (item.display === 'hover') return t('admin.settings.site.contactEntries.displayHoverHint')
  if (item.display === 'inline') return t('admin.settings.site.contactEntries.displayInlineHint')
  return t('admin.settings.site.contactEntries.displayModalHint')
}

function commit(next: ContactEntry[]) {
  emit('update:modelValue', next.map((item, index) => ({ ...item, sort_order: index })))
}

function generateId(): string {
  const cryptoObj = globalThis.crypto
  if (cryptoObj?.getRandomValues) {
    const bytes = new Uint8Array(8)
    cryptoObj.getRandomValues(bytes)
    return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
  }
  return `c${Date.now().toString(36)}${Math.random().toString(36).slice(2, 10)}`
}

function add() {
  if (entries.value.length >= maxEntries) return
  commit([
    ...entries.value,
    {
      id: generateId(),
      enabled: true,
      label: t('admin.settings.site.contactEntries.newLabel'),
      icon_type: 'emoji',
      icon: '💬',
      type: 'link',
      url: '',
      qr_code: '',
      value: '',
      description: '',
      group: '',
      display: 'modal',
      open_target: 'new_tab',
      sort_order: entries.value.length,
    },
  ])
}

function remove(index: number) {
  const next = entries.value.slice()
  next.splice(index, 1)
  commit(next)
}

function move(index: number, direction: -1 | 1) {
  const target = index + direction
  if (target < 0 || target >= entries.value.length) return
  const next = entries.value.slice()
  const [item] = next.splice(index, 1)
  next.splice(target, 0, item)
  commit(next)
}

function onTypeChange(item: ContactEntry) {
  if (item.type === 'link' && !item.open_target) item.open_target = 'new_tab'
  commit(entries.value.slice())
}

function notifyCopied() {
  appStore.showSuccess(t('common.copiedToClipboard'))
}

function notifyCopyFailed() {
  appStore.showError(t('common.copyFailed'))
}
</script>
