<template>
  <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800/40">
    <div class="flex items-start justify-between gap-3">
      <div>
        <label class="input-label">{{ t('admin.settings.payment.quickAmounts') }}</label>
        <p class="input-hint">{{ t('admin.settings.payment.quickAmountsHint') }}</p>
      </div>
      <span class="badge badge-primary shrink-0">{{ amounts.length }} / {{ MAX_RECHARGE_QUICK_AMOUNTS }}</span>
    </div>

    <div v-if="amounts.length" class="mt-3 flex flex-wrap gap-2">
      <div
        v-for="(amount, index) in amounts"
        :key="amount"
        class="inline-flex items-center gap-1 rounded-lg border border-primary-200 bg-primary-50 py-1 pl-2.5 pr-1 dark:border-primary-800 dark:bg-primary-900/20"
      >
        <span class="text-sm font-medium text-primary-700 dark:text-primary-300">
          {{ trimNumber(amount) }}
        </span>
        <button
          type="button"
          class="rounded-md p-1 text-primary-500 transition-colors hover:bg-primary-100 hover:text-primary-700 dark:hover:bg-primary-900/40 dark:hover:text-primary-200"
          :title="t('admin.settings.payment.removeItem')"
          @click="removeAmount(index)"
        >
          <Icon name="x" size="xs" />
        </button>
      </div>
    </div>
    <p v-else class="mt-3 rounded-lg border border-dashed border-gray-300 px-4 py-4 text-center text-xs text-gray-500 dark:border-dark-600 dark:text-gray-400">
      {{ t('admin.settings.payment.quickAmountsEmpty') }}
    </p>

    <div class="mt-3 flex flex-col gap-2 sm:flex-row">
      <div class="relative flex-1">
        <input
          v-model="draft"
          type="text"
          inputmode="decimal"
          class="input w-full py-2 text-sm"
          :placeholder="t('admin.settings.payment.quickAmountsAddPlaceholder')"
          :maxlength="12"
          @keydown.enter.prevent="addDraft"
        />
      </div>
      <button
        type="button"
        class="btn btn-secondary shrink-0 px-4 py-2 text-sm"
        :disabled="!canAddDraft"
        @click="addDraft"
      >
        <Icon name="plus" size="sm" />
        {{ t('admin.settings.payment.addAmount') }}
      </button>
    </div>
    <p v-if="draftInvalid" class="input-error-text">
      {{ t('admin.settings.payment.quickAmountsInvalid') }}
    </p>

    <div v-if="presetAmounts.length" class="mt-3 border-t border-gray-100 pt-3 dark:border-dark-700">
      <p class="text-xs text-gray-400 dark:text-dark-500">{{ t('admin.settings.payment.quickAmountsPresets') }}</p>
      <div class="mt-2 flex flex-wrap gap-1.5">
        <button
          v-for="preset in presetAmounts"
          :key="preset"
          type="button"
          class="rounded-md border border-gray-200 px-2 py-1 text-xs text-gray-600 transition-colors hover:border-primary-300 hover:text-primary-600 dark:border-dark-600 dark:text-gray-300 dark:hover:border-primary-700 dark:hover:text-primary-400"
          @click="addAmount(preset)"
        >
          + {{ trimNumber(preset) }}
        </button>
      </div>
    </div>

    <button
      v-if="amounts.length"
      type="button"
      class="mt-3 text-xs text-gray-400 transition-colors hover:text-red-500 dark:text-dark-500 dark:hover:text-red-400"
      @click="clearAll"
    >
      {{ t('admin.settings.payment.clearAll') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import {
  MAX_RECHARGE_QUICK_AMOUNTS,
  parseRechargeQuickAmountsText,
  formatRechargeQuickAmountsText,
  trimNumber,
} from '@/utils/rechargeTiers'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { t } = useI18n()

// 历史默认按钮，仅作为"快速添加"的候选，不写入设置；留空时前台仍沿用系统默认。
const presetAmounts = [10, 20, 50, 100, 200, 500, 1000, 2000, 5000]

const draft = ref('')
const draftTouched = ref(false)

const amounts = computed(() => parseRechargeQuickAmountsText(props.modelValue ?? ''))

const draftValue = computed(() => Number(draft.value.trim()))
const canAddDraft = computed(
  () =>
    draft.value.trim() !== '' &&
    Number.isFinite(draftValue.value) &&
    draftValue.value > 0 &&
    !amounts.value.includes(Math.round(draftValue.value * 100) / 100),
)
const draftInvalid = computed(
  () => draftTouched.value && draft.value.trim() !== '' && !Number.isFinite(draftValue.value),
)

function commit(next: number[]) {
  emit('update:modelValue', formatRechargeQuickAmountsText(next))
}

function addAmount(value: number) {
  const rounded = Math.round(value * 100) / 100
  if (!Number.isFinite(rounded) || rounded <= 0) return
  if (amounts.value.includes(rounded)) return
  if (amounts.value.length >= MAX_RECHARGE_QUICK_AMOUNTS) return
  commit([...amounts.value, rounded])
}

function addDraft() {
  draftTouched.value = true
  if (!canAddDraft.value) return
  addAmount(draftValue.value)
  draft.value = ''
  draftTouched.value = false
}

function removeAmount(index: number) {
  const next = amounts.value.slice()
  next.splice(index, 1)
  commit(next)
}

function clearAll() {
  commit([])
}
</script>