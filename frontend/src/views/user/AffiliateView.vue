<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
        ></div>
      </div>

      <template v-else-if="detail">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="card p-5">
            <p class="flex items-center gap-1.5 text-sm text-gray-500 dark:text-dark-400">
              <Icon name="dollar" size="sm" class="text-primary-500" />
              {{ t('affiliate.stats.rebateRate') }}
            </p>
            <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">
              {{ formattedRebateRate }}<span class="ml-0.5 text-base font-medium">%</span>
            </p>
            <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ t('affiliate.stats.rebateRateHint') }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.invitedUsers') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCount(detail.aff_count) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.availableQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">
              {{ formatCurrency(detail.aff_quota) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.totalQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCurrency(detail.aff_history_quota) }}
            </p>
            <p v-if="detail.aff_frozen_quota > 0" class="mt-1 text-xs text-amber-600 dark:text-amber-400">
              {{ t('affiliate.stats.frozenQuota') }}: {{ formatCurrency(detail.aff_frozen_quota) }}
            </p>
          </div>
        </div>

        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.title') }}</h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.description') }}</p>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.yourCode') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm font-semibold text-gray-900 dark:text-white sm:flex-1 sm:truncate">{{ detail.aff_code }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyCode">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyCode') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.inviteLink') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm text-gray-700 dark:text-gray-300 sm:flex-1 sm:truncate">{{ inviteLink }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyInviteLink">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="mt-5 rounded-xl border border-primary-200 bg-primary-50 p-4 dark:border-primary-900/40 dark:bg-primary-900/20">
            <p class="text-sm font-medium text-primary-800 dark:text-primary-200">{{ t('affiliate.tips.title') }}</p>
            <ul class="mt-2 space-y-1 text-sm text-primary-700 dark:text-primary-300">
              <li>1. {{ t('affiliate.tips.line1') }}</li>
              <li>2. {{ t('affiliate.tips.line2', { rate: `${formattedRebateRate}%` }) }}</li>
              <li>3. {{ t('affiliate.tips.line3') }}</li>
              <li v-if="detail.aff_frozen_quota > 0">4. {{ t('affiliate.tips.line4') }}</li>
            </ul>
          </div>
        </div>

        <div class="card p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.transfer.title') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.transfer.description') }}</p>
            </div>
            <button
              class="btn btn-primary"
              :disabled="transferring || detail.aff_quota <= 0"
              @click="transferQuota"
            >
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
            </button>
          </div>
          <p v-if="detail.aff_quota <= 0" class="mt-3 text-sm text-amber-600 dark:text-amber-400">
            {{ t('affiliate.transfer.empty') }}
          </p>
        </div>

        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.invitees.title') }}</h3>
          <div v-if="detail.invitees.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('affiliate.invitees.empty') }}
          </div>
          <!-- 桌面端（>=768px）：表格 -->
          <div v-else class="mt-4">
           <div class="hidden overflow-x-auto md:block">
            <table class="w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.email') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="px-3 py-2 font-medium text-right">{{ t('affiliate.invitees.columns.rebate') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in detail.invitees"
                  :key="item.user_id"
                  class="border-b border-gray-100 last:border-b-0 dark:border-dark-800"
                >
                  <td class="px-3 py-3 text-gray-900 dark:text-white">{{ item.email || '-' }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ item.username || '-' }}</td>
                  <td class="px-3 py-3 text-right font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
           </div>

           <!-- 移动端（<768px）：卡片，与全站表格移动端体验一致 -->
           <div class="space-y-3 md:hidden">
            <div
              v-for="item in detail.invitees"
              :key="item.user_id"
              class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900"
            >
              <div class="flex items-start justify-between gap-4">
                <span class="text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('affiliate.invitees.columns.email') }}</span>
                <span class="text-right text-sm text-gray-900 dark:text-white break-all">{{ item.email || '-' }}</span>
              </div>
              <div class="mt-2 flex items-start justify-between gap-4">
                <span class="text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('affiliate.invitees.columns.username') }}</span>
                <span class="text-right text-sm text-gray-700 dark:text-gray-300">{{ item.username || '-' }}</span>
              </div>
              <div class="mt-2 flex items-start justify-between gap-4">
                <span class="text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('affiliate.invitees.columns.rebate') }}</span>
                <span class="text-right text-sm font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(item.total_rebate) }}</span>
              </div>
              <div class="mt-2 flex items-start justify-between gap-4">
                <span class="text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('affiliate.invitees.columns.joinedAt') }}</span>
                <span class="text-right text-sm text-gray-700 dark:text-gray-300">{{ formatDateTime(item.created_at) || '-' }}</span>
              </div>
            </div>
           </div>
          </div>
        </div>

        <!-- 一级代理：下级代理与邀请返利管理 -->
        <div v-if="detail.is_level_one_agent" class="card p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.agents.title') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.agents.description') }}</p>
            </div>
            <span class="inline-flex w-fit items-center rounded-full bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
              {{ t('affiliate.agents.rateHint', { rate: `${formattedRebateRate}%` }) }}
            </span>
          </div>

          <div
            v-if="detail.invitees.length === 0"
            class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400"
          >
            {{ t('affiliate.agents.empty') }}
          </div>

          <div v-else class="mt-4 space-y-3">
            <div
              v-for="item in detail.invitees"
              :key="`agent-${item.user_id}`"
              class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900"
            >
              <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                <div class="min-w-0 space-y-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="text-sm font-medium text-gray-900 dark:text-white break-all">{{ item.email || '-' }}</span>
                    <span
                      v-if="item.agent_level === 2"
                      class="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
                    >{{ t('affiliate.agents.levelTwo') }}</span>
                    <span
                      v-else
                      class="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-dark-300"
                    >{{ t('affiliate.agents.levelNormal') }}</span>
                    <span
                      v-if="item.affiliate_hidden"
                      class="inline-flex items-center rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
                    >{{ t('affiliate.agents.hidden') }}</span>
                  </div>
                  <p class="text-xs text-gray-500 dark:text-dark-400">
                    {{ item.username || '-' }} · {{ t('affiliate.invitees.columns.joinedAt') }}: {{ formatDateTime(item.created_at) || '-' }}
                  </p>
                </div>

                <div class="flex flex-wrap items-center gap-4">
                  <div class="text-right">
                    <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('affiliate.invitees.columns.rebate') }}</p>
                    <p class="text-sm font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(item.total_rebate) }}</p>
                  </div>
                  <div class="text-right">
                    <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('affiliate.agents.rateLabel') }}</p>
                    <p class="text-sm font-medium text-gray-900 dark:text-white">{{ formatRate(item.rebate_rate_percent) }}</p>
                  </div>
                  <div class="flex flex-wrap gap-2">
                    <button class="btn btn-secondary btn-sm" @click="openSubAgentModal(item)">
                      {{ item.agent_level === 2 ? t('affiliate.agents.editRate') : t('affiliate.agents.setSubAgent') }}
                    </button>
                    <button
                      v-if="item.agent_level === 2"
                      class="btn btn-secondary btn-sm"
                      :disabled="subAgentBusyId === item.user_id"
                      @click="removeSubAgent(item)"
                    >
                      {{ t('affiliate.agents.cancelSubAgent') }}
                    </button>
                    <button
                      class="btn btn-secondary btn-sm"
                      :disabled="subAgentBusyId === item.user_id"
                      @click="toggleInviteeAffiliate(item)"
                    >
                      {{ item.affiliate_hidden ? t('affiliate.agents.enableAffiliate') : t('affiliate.agents.disableAffiliate') }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 设置二级代理弹窗 -->
        <div
          v-if="subAgentModal.open"
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
          @click.self="closeSubAgentModal"
        >
          <div class="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-dark-900">
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.agents.modalTitle') }}</h3>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ t('affiliate.agents.modalDescription', { email: subAgentModal.email }) }}
            </p>
            <div class="mt-4 space-y-2">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.agents.modalRateLabel') }}</label>
              <input
                v-model="subAgentModal.rate"
                type="number"
                min="0"
                :max="maxSubAgentRate"
                step="0.01"
                class="input w-full"
                :placeholder="String(maxSubAgentRate)"
              />
              <p class="text-xs text-gray-500 dark:text-dark-400">
                {{ t('affiliate.agents.modalRateHint', { max: `${formattedRebateRate}%` }) }}
              </p>
            </div>
            <div class="mt-6 flex justify-end gap-3">
              <button class="btn btn-secondary" @click="closeSubAgentModal">{{ t('affiliate.agents.modalCancel') }}</button>
              <button class="btn btn-primary" :disabled="subAgentSaving" @click="saveSubAgent">
                {{ subAgentSaving ? t('affiliate.agents.modalSaving') : t('affiliate.agents.modalSave') }}
              </button>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import userAPI from '@/api/user'
import type { AffiliateInvitee, UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const transferring = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)
const subAgentBusyId = ref<number | null>(null)
const subAgentSaving = ref(false)
const subAgentModal = ref<{ open: boolean; userId: number; email: string; rate: string }>({
  open: false,
  userId: 0,
  email: '',
  rate: '',
})

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

// Rebate rate is a percentage in the range [0, 100]; backend already clamps it.
// We trim trailing zeros (e.g. 20.00 → "20", 12.50 → "12.5") for a cleaner UI.
const formattedRebateRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(v * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

function formatCount(value: number): string {
  return value.toLocaleString()
}

const maxSubAgentRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  return Math.round(v * 100) / 100
})

function formatRate(value?: number | null): string {
  if (value === null || value === undefined) return t('affiliate.agents.rateInherited')
  const rounded = Math.round(value * 100) / 100
  return `${Number.isInteger(rounded) ? rounded : rounded}%`
}

function openSubAgentModal(item: AffiliateInvitee): void {
  subAgentModal.value = {
    open: true,
    userId: item.user_id,
    email: item.email || item.username || String(item.user_id),
    rate: item.rebate_rate_percent != null ? String(item.rebate_rate_percent) : String(maxSubAgentRate.value),
  }
}

function closeSubAgentModal(): void {
  subAgentModal.value.open = false
}

async function saveSubAgent(): Promise<void> {
  if (!detail.value || subAgentSaving.value) return
  const rate = Number(subAgentModal.value.rate)
  if (!Number.isFinite(rate) || rate < 0 || rate > maxSubAgentRate.value) {
    appStore.showError(t('affiliate.agents.rateInvalid', { max: `${formattedRebateRate.value}%` }))
    return
  }
  subAgentSaving.value = true
  try {
    await userAPI.setAffiliateSubAgent({
      user_id: subAgentModal.value.userId,
      level: 2,
      rate_percent: rate,
    })
    appStore.showSuccess(t('affiliate.agents.saveSuccess'))
    closeSubAgentModal()
    await loadAffiliateDetail(true)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.agents.saveFailed')))
  } finally {
    subAgentSaving.value = false
  }
}

async function removeSubAgent(item: AffiliateInvitee): Promise<void> {
  if (!detail.value || subAgentBusyId.value !== null) return
  subAgentBusyId.value = item.user_id
  try {
    await userAPI.setAffiliateSubAgent({ user_id: item.user_id, level: 0, clear_rate: true })
    appStore.showSuccess(t('affiliate.agents.cancelSuccess'))
    await loadAffiliateDetail(true)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.agents.saveFailed')))
  } finally {
    subAgentBusyId.value = null
  }
}

async function toggleInviteeAffiliate(item: AffiliateInvitee): Promise<void> {
  if (!detail.value || subAgentBusyId.value !== null) return
  subAgentBusyId.value = item.user_id
  try {
    await userAPI.setInviteeAffiliateHidden(item.user_id, !item.affiliate_hidden)
    appStore.showSuccess(t(item.affiliate_hidden ? 'affiliate.agents.enableSuccess' : 'affiliate.agents.disableSuccess'))
    await loadAffiliateDetail(true)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.agents.saveFailed')))
  } finally {
    subAgentBusyId.value = null
  }
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) {
    loading.value = true
  }
  try {
    detail.value = await userAPI.getAffiliateDetail()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.loadFailed')))
  } finally {
    if (!silent) {
      loading.value = false
    }
  }
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, t('affiliate.codeCopied'))
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('affiliate.linkCopied'))
}

async function transferQuota(): Promise<void> {
  if (!detail.value || detail.value.aff_quota <= 0 || transferring.value) return
  transferring.value = true
  try {
    const resp = await userAPI.transferAffiliateQuota()
    appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCurrency(resp.transferred_quota) }))
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  } finally {
    transferring.value = false
  }
}

onMounted(() => {
  void loadAffiliateDetail()
})
</script>
