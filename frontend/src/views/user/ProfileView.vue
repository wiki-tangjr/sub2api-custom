<template>
  <AppLayout>
    <div
      data-testid="profile-shell"
      class="mx-auto max-w-[950px] space-y-6"
    >
      <ProfileInfoCard
        :user="user"
        :linuxdo-enabled="linuxdoOAuthEnabled"
        :dingtalk-enabled="dingtalkOAuthEnabled"
        :oidc-enabled="oidcOAuthEnabled"
        :oidc-provider-name="oidcOAuthProviderName"
        :wechat-enabled="wechatOAuthEnabled"
        :wechat-open-enabled="wechatOAuthOpenEnabled"
        :wechat-mp-enabled="wechatOAuthMPEnabled"
      />

      <div v-if="contactEntries.length" class="card p-6 dark:bg-dark-800">
        <div class="flex items-start gap-4">
          <div class="rounded-xl bg-primary-100 p-3 text-primary-600 dark:bg-primary-900/40 dark:text-primary-400">
            <Icon name="chat" size="lg" />
          </div>
          <div class="min-w-0 flex-1">
            <h3 class="font-semibold text-gray-900 dark:text-gray-100">
              {{ contactSectionTitle }}
            </h3>
            <p v-if="contactSectionDescription" class="mt-0.5 text-sm text-gray-500 dark:text-gray-400">
              {{ contactSectionDescription }}
            </p>
            <!-- 魔改 #12: 后台可自定义的联系方式条目 -->
            <ContactEntries
              class="mt-3"
              :variant="contactSectionStyle === 'list' ? 'list' : 'card'"
              :entries="contactEntries"
            />
          </div>
        </div>
      </div>

      <ProfilePasswordForm />

      <ProfileBalanceNotifyCard
        v-if="user && balanceLowNotifyEnabled"
        :enabled="user.balance_notify_enabled ?? true"
        :threshold="user.balance_notify_threshold"
        :extra-emails="user.balance_notify_extra_emails ?? []"
        :system-default-threshold="systemDefaultThreshold"
        :user-email="user.email"
      />

      <ProfileTotpCard />
      <ProfilePasskeyCard :enabled="passkeyEnabled" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@/components/icons'
import AppLayout from '@/components/layout/AppLayout.vue'
import ProfileBalanceNotifyCard from '@/components/user/profile/ProfileBalanceNotifyCard.vue'
import ProfileInfoCard from '@/components/user/profile/ProfileInfoCard.vue'
import ProfilePasswordForm from '@/components/user/profile/ProfilePasswordForm.vue'
import ProfileTotpCard from '@/components/user/profile/ProfileTotpCard.vue'
import ProfilePasskeyCard from '@/components/user/profile/ProfilePasskeyCard.vue'
import ContactEntries from '@/components/common/ContactEntries.vue'
import { isWeChatWebOAuthEnabled } from '@/api/auth'
import { useAppStore } from '@/stores/app'
import type { ContactEntry } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { resolveContactEntries } from '@/utils/contactEntries'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const user = computed(() => authStore.user)

const contactEntries = ref<ContactEntry[]>([])
const contactSectionTitle = ref('')
const contactSectionDescription = ref('')
const contactSectionStyle = ref('card')
const balanceLowNotifyEnabled = ref(false)
const systemDefaultThreshold = ref(0)
const linuxdoOAuthEnabled = ref(false)
const dingtalkOAuthEnabled = ref(false)
const wechatOAuthEnabled = ref(false)
const wechatOAuthOpenEnabled = ref<boolean | undefined>(undefined)
const wechatOAuthMPEnabled = ref<boolean | undefined>(undefined)
const oidcOAuthEnabled = ref(false)
const oidcOAuthProviderName = ref('OIDC')
const passkeyEnabled = ref(false)

onMounted(async () => {
  const profileRefresh = authStore.refreshUser().catch((error) => {
    console.error('Failed to refresh profile:', error)
  })

  const settingsLoad = appStore.fetchPublicSettings()
    .then((settings) => {
      if (!settings) {
        return
      }
      contactEntries.value = resolveContactEntries(settings, t)
      contactSectionTitle.value = settings.contact_section_title || t('common.contactSectionDefaultTitle')
      contactSectionDescription.value = settings.contact_section_description || ''
      contactSectionStyle.value = settings.contact_section_style === 'list' ? 'list' : 'card'
      balanceLowNotifyEnabled.value = settings.balance_low_notify_enabled ?? false
      systemDefaultThreshold.value = settings.balance_low_notify_threshold ?? 0
      linuxdoOAuthEnabled.value = settings.linuxdo_oauth_enabled ?? false
      dingtalkOAuthEnabled.value = settings.dingtalk_oauth_enabled ?? false
      wechatOAuthEnabled.value = isWeChatWebOAuthEnabled(settings)
      wechatOAuthOpenEnabled.value = typeof settings.wechat_oauth_open_enabled === 'boolean'
        ? settings.wechat_oauth_open_enabled
        : undefined
      wechatOAuthMPEnabled.value = typeof settings.wechat_oauth_mp_enabled === 'boolean'
        ? settings.wechat_oauth_mp_enabled
        : undefined
      oidcOAuthEnabled.value = settings.oidc_oauth_enabled ?? false
      oidcOAuthProviderName.value = settings.oidc_oauth_provider_name || 'OIDC'
      passkeyEnabled.value = settings.passkey_enabled === true
    })
    .catch((error) => {
      console.error('Failed to load settings:', error)
    })

  await Promise.all([profileRefresh, settingsLoad])
})
</script>
