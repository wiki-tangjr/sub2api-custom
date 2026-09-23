/**
 * 全局错误文案中文化（魔改 #26）
 *
 * 背景：后端 456 个错误码的 message 均为英文硬编码，前端只做 reason 透传时
 * 用户会看到 invalid email or password 这类英文提示。
 *
 * 策略（仅在中文界面生效，英文界面保持原文）：
 * 1. 优先按稳定的 reason 机器码查表（errorMessagesZh.ERROR_CODE_ZH）；
 * 2. 其次按英文原文小写精确匹配（errorMessagesZh.ERROR_TEXT_ZH）；
 * 3. 仍然拿不到中文时，若整条消息是纯 ASCII 英文（无中文），按类型回退到中文兜底，
 *    避免任何英文句子直接暴露给用户；已经含中文、或纯符号/数字/URL 的消息保持原样。
 *
 * 该模块不改变任何请求/业务逻辑，只在最终展示层做翻译。
 */

import { ERROR_CODE_ZH, ERROR_TEXT_ZH } from '@/utils/errorMessagesZh'

export { ERROR_CODE_ZH, ERROR_TEXT_ZH }

/** 中文兜底文案（按错误提示类型区分，避免全部落到同一句） */
export const ZH_FALLBACK = {
  /** 通用兜底 */
  general: '操作失败，请稍后重试',
  /** 网络类 */
  network: '网络连接异常，请检查网络后重试',
  /** 登录/会话类 */
  auth: '登录状态异常，请重新登录',
  /** 权限类 */
  permission: '权限不足，无法执行此操作',
  /** 支付类 */
  payment: '支付请求处理失败，请稍后重试',
} as const

export type ZhFallbackKind = keyof typeof ZH_FALLBACK

/**
 * 是否处于中文界面。
 *
 * 这里刻意**不 import '@/i18n'**：那会让本模块对 i18n 实例产生模块级依赖，
 * 而大量既有单测会 mock 'vue-i18n'，一旦被间接引入就会整文件崩溃
 * （"No createI18n export is defined on the vue-i18n mock"）。
 * 因此改为读取与 i18n 完全相同的判定来源：
 *   1. localStorage['sub2api_locale']（用户在站内切换语言时由 setLocale 写入）；
 *   2. navigator.language 是否以 zh 开头；
 *   3. 非浏览器环境（单测）回退 true，保证用户可见文案始终为中文。
 * 该判定与 i18n/index.ts 的 getDefaultLocale() 逻辑保持一致。
 */
export function isChineseLocale(): boolean {
  try {
    if (typeof window === 'undefined') return true
    const saved = window.localStorage?.getItem('sub2api_locale')
    if (saved === 'zh') return true
    if (saved === 'en') return false
    const lang = typeof navigator !== 'undefined' ? navigator.language || '' : ''
    return lang.toLowerCase().startsWith('zh')
  } catch {
    return true
  }
}

/** 判断字符串是否含有中文字符 */
export function containsChinese(text: string): boolean {
  return /[\u4e00-\u9fff]/.test(text)
}

/**
 * 判断是否为需要汉化的英文句子。
 * 排除：空串、纯数字、纯符号/emoji、URL、以及不含空格的稳定标识符（如 SUCCESS、wxpay_error）。
 */
export function isTranslatableEnglishSentence(text: string): boolean {
  const value = (text || '').trim()
  if (!value) return false
  if (containsChinese(value)) return false
  // 纯 ASCII 检查（含常见英文标点）
  if (!/^[\x20-\x7E\s]*$/.test(value)) return false
  // URL 保持原样
  if (/^(https?:\/\/|www\.)/i.test(value)) return false
  // 纯数字 / 金额 / 百分比等
  if (/^[\d\s.,:%+\-*/=()[\]{}]+$/.test(value)) return false
  // 不含空格、且不含句读的稳定标识符（如错误枚举、字段名）保持原样
  if (!/[\s.!?:;,'"]/.test(value) && /^[A-Za-z0-9_.\-]+$/.test(value)) return false
  // 至少包含 2 个连续英文字母，避免误伤诸如 N/A 的缩写
  return /[A-Za-z]{2}/.test(value)
}

/** 依据英文原文语义挑选更贴切的兜底文案 */
function pickFallbackKind(text: string, kind?: ZhFallbackKind): ZhFallbackKind {
  if (kind) return kind
  const lower = text.toLowerCase()
  if (/network|connection|timeout|socket|failed to fetch|offline/.test(lower)) return 'network'
  if (/unauthorized|authentication|session|login|token|credential/.test(lower)) return 'auth'
  if (/permission|forbidden|not allowed|denied/.test(lower)) return 'permission'
  if (/payment|order|refund|balance|charge/.test(lower)) return 'payment'
  return 'general'
}

/**
 * 将可能的英文错误文案转换为中文。
 *
 * @param message 原始错误文案
 * @param options.code   稳定的错误码（reason），优先按码查表
 * @param options.kind   指定兜底类型（可选）
 * @param options.fallback 完全无法判断时的兜底文案（可选）
 */
export function localizeErrorMessage(
  message: unknown,
  options: { code?: unknown; kind?: ZhFallbackKind; fallback?: string } = {},
): string {
  const raw = typeof message === 'string' ? message : message == null ? '' : String(message)
  const text = raw.trim()
  const { code, kind, fallback } = options

  if (!isChineseLocale()) return raw

  // 1) 稳定错误码 -> 中文
  if (code != null) {
    const mapped = ERROR_CODE_ZH[String(code)]
    if (mapped) return mapped
  }

  // 2) 英文原文精确匹配（大小写/首尾空白不敏感）
  if (text) {
    const mapped = ERROR_TEXT_ZH[text.toLowerCase()]
    if (mapped) return mapped
  }

  // 3) 已是中文，或无需翻译的技术串
  if (!isTranslatableEnglishSentence(text)) {
    if (!text) {
      return fallback && containsChinese(fallback) ? fallback : ZH_FALLBACK.general
    }
    return raw
  }

  // 4) 英文兜底
  if (fallback && containsChinese(fallback)) return fallback
  return ZH_FALLBACK[pickFallbackKind(text, kind)]
}

/**
 * 从任意错误对象中提取中文错误文案。
 * 与 utils/apiError 的提取顺序保持一致，但最终统一做中文化处理。
 */
export function localizeUnknownError(
  err: unknown,
  options: { fallback?: string; kind?: ZhFallbackKind } = {},
): string {
  const { fallback, kind } = options

  if (err && typeof err === 'object') {
    const e = err as {
      reason?: unknown
      code?: unknown
      message?: unknown
      error?: unknown
      response?: { data?: { detail?: unknown; message?: unknown; reason?: unknown; code?: unknown } }
    }
    const code =
      e.reason ??
      e.response?.data?.reason ??
      (typeof e.code === 'string' ? e.code : undefined) ??
      e.response?.data?.code
    const msg = e.message ?? e.error ?? e.response?.data?.detail ?? e.response?.data?.message
    return localizeErrorMessage(msg, { code, kind, fallback })
  }

  if (err instanceof Error) {
    return localizeErrorMessage(err.message, { kind, fallback })
  }

  if (err == null) return localizeErrorMessage('', { kind, fallback })
  return localizeErrorMessage(String(err), { kind, fallback })
}

export default localizeErrorMessage
