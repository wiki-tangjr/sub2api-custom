import type { ContactEntry, PublicSettings } from '@/types'
import { sanitizeUrl } from '@/utils/url'

type Translate = (key: string) => string

/**
 * 魔改 #12: 解析前台要展示的客服联系方式条目。
 *
 * 后端 /settings/public 已经会把旧的 telegram_group_url / wechat_group_qr_code /
 * contact_info 转换成 contact_entries 返回；这里再做一次兜底，保证旧缓存或
 * 接口降级（stores/app.ts 的合成 PublicSettings）时前台仍然能显示联系方式。
 */
export function resolveContactEntries(
  settings: PublicSettings | null | undefined,
  translate: Translate,
  fallbackContactInfo = '',
): ContactEntry[] {
  const configured = settings?.contact_entries
  if (Array.isArray(configured) && configured.length > 0) {
    return configured.filter((entry) => !!entry && entry.enabled !== false)
  }

  const legacy: ContactEntry[] = []
  const push = (entry: Omit<ContactEntry, 'sort_order'>) => {
    legacy.push({ ...entry, sort_order: legacy.length })
  }

  const telegramUrl = sanitizeUrl(settings?.telegram_group_url || '')
  if (telegramUrl) {
    push({
      id: 'legacy-telegram',
      enabled: true,
      label: settings?.telegram_entry_label || translate('common.telegramGroup'),
      icon_type: 'emoji',
      icon: '✈️',
      type: 'link',
      url: telegramUrl,
      display: 'modal',
      open_target: 'new_tab',
    })
  }

  const qrCode = sanitizeUrl(settings?.wechat_group_qr_code || '', { allowDataUrl: true })
  if (qrCode) {
    push({
      id: 'legacy-wechat-group',
      enabled: true,
      label: settings?.wechat_group_entry_label || translate('common.wechatGroup'),
      icon_type: 'emoji',
      icon: '💬',
      type: 'qrcode',
      qr_code: qrCode,
      display: 'modal',
      open_target: 'new_tab',
    })
  }

  const info = (fallbackContactInfo || settings?.contact_info || '').trim()
  if (info) {
    push({
      id: 'legacy-wechat-contact',
      enabled: true,
      label: settings?.wechat_contact_entry_label || translate('common.wechatContactDefault'),
      icon_type: 'emoji',
      icon: '💬',
      type: 'text',
      value: info,
      display: 'hover',
      open_target: 'new_tab',
    })
  }

  return legacy
}
