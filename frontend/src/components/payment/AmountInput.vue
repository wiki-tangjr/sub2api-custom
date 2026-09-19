<template>
  <div class="space-y-4">
    <!-- Quick Amount Buttons -->
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.quickAmounts') }}
      </label>
      <!-- Customization (#16): when the admin configures tiered discounts, each quick
           button shows the credited amount, the discount and, if relevant, the amount
           actually charged, mirroring the common "more credit, more discount" layout.
           With no tiers configured this renders exactly the historic plain buttons. -->
      <div :class="showTierDetails ? 'grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4' : 'grid grid-cols-3 gap-2'">
        <button
          v-for="amt in filteredAmounts"
          :key="amt"
          type="button"
          :class="[
            'rounded-lg border-2 text-center transition-colors',
            showTierDetails ? 'px-3 py-2.5' : 'px-4 py-3 font-medium',
            modelValue === amt
              ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/40 dark:text-primary-300'
              : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-dark-500',
          ]"
          @click="selectAmount(amt)"
        >
          <template v-if="showTierDetails">
            <span class="block text-xs text-gray-400 dark:text-dark-500">{{ t('payment.creditedAmount') }}</span>
            <span class="mt-0.5 block text-base font-semibold">
              {{ formatAmount(creditedFor(amt)) }}
            </span>
            <span
              v-if="tierPercentFor(amt) > 0"
              class="mt-0.5 block text-xs font-medium text-green-600 dark:text-green-400"
            >
              {{ t('payment.discount') }} {{ trimPercent(tierPercentFor(amt)) }}%
            </span>
          </template>
          <template v-else>{{ amt }}</template>
        </button>
      </div>
      <p
        v-if="showTierDetails && hasAnyDiscount"
        class="mt-1.5 text-xs text-gray-400 dark:text-dark-500"
      >
        {{ t('payment.tierDiscountHint') }}
      </p>
    </div>

    <!-- Custom Amount Input -->
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.customAmount') }}
      </label>
      <div class="relative">
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-dark-500">
          $
        </span>
        <input
          type="text"
          inputmode="decimal"
          :value="customText"
          :placeholder="placeholderText"
          class="input w-full py-3 pl-8 pr-4"
          @input="handleInput"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatPaymentAmount } from '@/components/payment/currency'
import type { RechargeDiscountTier } from '@/types/payment'
import { normalizeRechargeDiscountTiers, resolveRechargeDiscountPercent } from '@/utils/rechargeTiers'

const props = withDefaults(defineProps<{
  amounts?: number[]
  modelValue: number | null
  min?: number
  max?: number
  /** Customization (#16): admin-configured tiered discount rules. */
  discountTiers?: RechargeDiscountTier[]
  /** Customization (#16): credit multiplier applied to the chosen amount. */
  multiplier?: number
  /** Customization (#16): currency used to render the credited amount. */
  currency?: string
}>(), {
  amounts: () => [10, 20, 50, 100, 200, 500, 1000, 2000, 5000],
  min: 0,
  max: 0,
  discountTiers: () => [],
  multiplier: 1,
  currency: 'CNY',
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')

// 0 = no limit
const filteredAmounts = computed(() =>
  props.amounts.filter((a) => (props.min <= 0 || a >= props.min) && (props.max <= 0 || a <= props.max))
)

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `≥ ${props.min}`
  if (props.max > 0) return `≤ ${props.max}`
  return t('payment.enterAmount')
})

const AMOUNT_PATTERN = /^\d*(\.\d{0,2})?$/

// Customization (#16/#20): tier metadata for the quick-amount cards. Everything below is
// inert until an admin saves discount tiers in the back office.
// Ranges are matched with the exact same rule as the Go backend
// (resolveRechargeDiscountPercent) so the preview cannot drift from the real charge.
const tiers = computed<RechargeDiscountTier[]>(() => props.discountTiers || [])

const showTierDetails = computed(() => normalizeRechargeDiscountTiers(tiers.value).length > 0)
const hasAnyDiscount = computed(() => normalizeRechargeDiscountTiers(tiers.value).length > 0)

function tierPercentFor(amount: number): number {
  return resolveRechargeDiscountPercent(amount, tiers.value)
}

function creditedFor(amount: number): number {
  const multiplier = Number.isFinite(props.multiplier) && props.multiplier > 0 ? props.multiplier : 1
  return Math.round(amount * multiplier * 100) / 100
}

function formatAmount(value: number): string {
  return formatPaymentAmount(value, props.currency)
}

function trimPercent(value: number): string {
  return String(Math.round(value * 10000) / 10000)
}

function selectAmount(amt: number) {
  customText.value = String(amt)
  emit('update:modelValue', amt)
}

function handleInput(e: Event) {
  const val = (e.target as HTMLInputElement).value
  if (!AMOUNT_PATTERN.test(val)) return
  customText.value = val
  if (val === '') {
    emit('update:modelValue', null)
    return
  }
  const num = parseFloat(val)
  if (!isNaN(num) && num > 0) {
    emit('update:modelValue', num)
  } else {
    emit('update:modelValue', null)
  }
}

watch(() => props.modelValue, (v) => {
  if (v !== null && String(v) !== customText.value) {
    customText.value = String(v)
  }
}, { immediate: true })
</script>
