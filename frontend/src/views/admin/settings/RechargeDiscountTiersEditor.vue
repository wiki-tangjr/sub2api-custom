<template>
  <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800/40">
    <div class="flex items-start justify-between gap-3">
      <div>
        <label class="input-label">{{ t('admin.settings.payment.discountTiers') }}</label>
        <p class="input-hint">{{ t('admin.settings.payment.discountTiersHint') }}</p>
      </div>
      <span class="badge badge-primary shrink-0">{{ rows.length }} / {{ MAX_RECHARGE_DISCOUNT_TIERS }}</span>
    </div>

    <div v-if="rows.length" class="mt-3 space-y-2">
      <div
        v-for="(row, index) in rows"
        :key="row.key"
        class="rounded-lg border border-gray-200 bg-gray-50/60 p-2.5 dark:border-dark-600 dark:bg-dark-800/60"
      >
        <div class="flex flex-wrap items-end gap-2">
          <div class="min-w-[7rem] flex-1">
            <label class="mb-1 block text-[11px] font-medium text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.payment.tierMin') }}
            </label>
            <input
              :value="row.min"
              type="text"
              inputmode="decimal"
              class="input py-1.5 text-sm"
              :placeholder="t('admin.settings.payment.tierMinPlaceholder')"
              @input="onRowInput(index, 'min', ($event.target as HTMLInputElement).value)"
            />
          </div>
          <span class="pb-2 text-gray-400 dark:text-dark-500">–</span>
          <div class="min-w-[7rem] flex-1">
            <label class="mb-1 block text-[11px] font-medium text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.payment.tierMax') }}
            </label>
            <input
              :value="row.max"
              type="text"
              inputmode="decimal"
              class="input py-1.5 text-sm"
              :placeholder="t('admin.settings.payment.tierMaxPlaceholder')"
              @input="onRowInput(index, 'max', ($event.target as HTMLInputElement).value)"
            />
          </div>
          <div class="w-[6.5rem]">
            <label class="mb-1 block text-[11px] font-medium text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.payment.tierPercent') }}
            </label>
            <div class="relative">
              <input
                :value="row.percent"
                type="text"
                inputmode="decimal"
                class="input py-1.5 pr-7 text-sm"
                :placeholder="t('admin.settings.payment.tierPercentPlaceholder')"
                @input="onRowInput(index, 'percent', ($event.target as HTMLInputElement).value)"
              />
              <span class="pointer-events-none absolute inset-y-0 right-2.5 flex items-center text-xs text-gray-400 dark:text-dark-500">%</span>
            </div>
          </div>
          <button
            type="button"
            class="btn-ghost btn-icon mb-0.5 shrink-0 text-gray-400 hover:text-red-500 dark:hover:text-red-400"
            :title="t('admin.settings.payment.removeItem')"
            @click="removeRow(index)"
          >
            <Icon name="trash" size="sm" />
          </button>
        </div>
        <p v-if="rowError(row)" class="mt-1.5 text-xs text-red-500">{{ rowError(row) }}</p>
        <p v-else class="mt-1.5 text-xs text-gray-400 dark:text-dark-500">
          {{ describeRow(row) }}
        </p>
      </div>
    </div>
    <p v-else class="mt-3 rounded-lg border border-dashed border-gray-300 px-4 py-5 text-center text-xs text-gray-500 dark:border-dark-600 dark:text-gray-400">
      {{ t('admin.settings.payment.discountTiersEmpty') }}
    </p>

    <div class="mt-3 flex flex-wrap items-center gap-2">
      <button
        type="button"
        class="btn btn-secondary px-4 py-2 text-sm"
        :disabled="rows.length >= MAX_RECHARGE_DISCOUNT_TIERS"
        @click="addRow"
      >
        <Icon name="plus" size="sm" />
        {{ t('admin.settings.payment.addTier') }}
      </button>
      <span class="text-xs text-gray-400 dark:text-dark-500">
        {{ t('admin.settings.payment.discountTiersUnlimitedHint') }}
      </span>
    </div>

    <details v-if="legacyText" class="mt-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
      <summary class="cursor-pointer text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.settings.payment.discountTiersAdvanced') }}
      </summary>
      <p class="mt-2 text-xs text-gray-400 dark:text-dark-500">
        {{ t('admin.settings.payment.discountTiersAdvancedHint') }}
      </p>
      <textarea
        v-model="legacyText"
        rows="2"
        class="input mt-2 w-full font-mono text-xs"
        @change="applyLegacyText"
      ></textarea>
    </details>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import {
  MAX_RECHARGE_DISCOUNT_TIERS,
  formatRechargeDiscountTiersText,
  parseRechargeDiscountTiersText,
  trimNumber,
  type NormalizedRechargeTier,
} from '@/utils/rechargeTiers'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { t } = useI18n()

interface EditableRow {
  key: string
  min: string
  max: string
  percent: string
}

let keySeed = 0

const rows = ref<EditableRow[]>([])
const legacyText = ref('')
let syncing = false

function nextKey(): string {
  keySeed += 1
  return `tier-${keySeed}-${Date.now().toString(36)}`
}

function toRow(tier: NormalizedRechargeTier): EditableRow {
  return {
    key: nextKey(),
    min: trimNumber(tier.min),
    max: tier.max > 0 ? trimNumber(tier.max) : '',
    percent: trimNumber(tier.percent),
  }
}

// 外部值 -> 内部行（含首次加载与保存后的回填）
watch(
  () => props.modelValue,
  (value) => {
    if (syncing) return
    const parsed = parseRechargeDiscountTiersText(value ?? '')
    rows.value = parsed.map(toRow)
    legacyText.value = String(value ?? '')
  },
  { immediate: true },
)

function rowNumber(text: string): number | null {
  const trimmed = text.trim()
  if (trimmed === '') return null
  const value = Number(trimmed)
  if (!Number.isFinite(value) || value < 0) return null
  return Math.round(value * 100) / 100
}

function rowPercent(text: string): number | null {
  const trimmed = text.trim()
  if (trimmed === '') return null
  const value = Number(trimmed)
  if (!Number.isFinite(value) || value <= 0 || value >= 100) return null
  return Math.round(value * 10000) / 10000
}

function rowError(row: EditableRow): string {
  const min = rowNumber(row.min)
  const max = rowNumber(row.max)
  const percent = rowPercent(row.percent)
  if (percent === null) return t('admin.settings.payment.tierPercentError')
  if (max !== null && max <= 0) return t('admin.settings.payment.tierMaxError')
  if (min === null && max === null) return t('admin.settings.payment.tierRangeRequired')
  if (min !== null && max !== null && max < min) return t('admin.settings.payment.tierRangeError')
  return ''
}

function describeRow(row: EditableRow): string {
  const min = rowNumber(row.min)
  const max = rowNumber(row.max)
  const percent = rowPercent(row.percent)
  if (percent === null) return ''
  if (min !== null && max !== null) {
    return t('admin.settings.payment.tierSummaryRange', { min, max, percent })
  }
  if (max !== null) {
    return t('admin.settings.payment.tierSummaryMax', { max, percent })
  }
  return t('admin.settings.payment.tierSummaryMin', { min: min ?? 0, percent })
}

/** 只有全部行都合法时才回写，避免编辑中间态把有效配置清空。 */
function validRows(): NormalizedRechargeTier[] | null {
  const out: NormalizedRechargeTier[] = []
  for (const row of rows.value) {
    if (rowError(row)) return null
    const min = rowNumber(row.min) ?? 0
    const max = rowNumber(row.max) ?? 0
    const percent = rowPercent(row.percent)
    if (percent === null) return null
    out.push({ min, max, percent })
  }
  return out
}

function commit() {
  const tiers = validRows()
  if (!tiers) return
  const text = formatRechargeDiscountTiersText(tiers)
  syncing = true
  legacyText.value = text
  emit('update:modelValue', text)
  queueMicrotask(() => {
    syncing = false
  })
}

function onRowInput(index: number, field: 'min' | 'max' | 'percent', value: string) {
  // 只允许数字、小数点与空白，避免粘贴 Markdown/中文标点破坏配置。
  if (value !== '' && !/^[0-9.\s]*$/.test(value)) return
  const row = rows.value[index]
  if (!row) return
  row[field] = value
  commit()
}

function addRow() {
  if (rows.value.length >= MAX_RECHARGE_DISCOUNT_TIERS) return
  rows.value.push({ key: nextKey(), min: '', max: '', percent: '' })
}

function removeRow(index: number) {
  rows.value.splice(index, 1)
  commit()
}

function applyLegacyText() {
  const parsed = parseRechargeDiscountTiersText(legacyText.value)
  rows.value = parsed.map(toRow)
  const text = formatRechargeDiscountTiersText(parsed)
  legacyText.value = text
  syncing = true
  emit('update:modelValue', text)
  queueMicrotask(() => {
    syncing = false
  })
}
</script>