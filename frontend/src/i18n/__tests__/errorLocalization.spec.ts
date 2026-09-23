import { beforeEach, describe, expect, it } from 'vitest'
import {
  ERROR_CODE_ZH,
  ERROR_TEXT_ZH,
  ZH_FALLBACK,
  containsChinese,
  isTranslatableEnglishSentence,
  localizeErrorMessage,
  localizeUnknownError,
} from '@/i18n/errorLocalization'

/**
 * 魔改 #26：错误文案中文化回归测试。
 *
 * isChineseLocale() 刻意不依赖 '@/i18n' 实例（避免 mock vue-i18n 的测试崩溃），
 * 而是与 i18n/index.ts 一样读 localStorage['sub2api_locale']。
 * 因此每个用例前显式写入 zh；英文界面用例改回 en。
 */
describe('errorLocalization (魔改 #26)', () => {
  beforeEach(() => {
    localStorage.setItem('sub2api_locale', 'zh')
  })

  it('按错误码优先映射为中文', () => {
    expect(localizeErrorMessage('', { code: 'INVALID_CREDENTIALS' })).toBe(
      ERROR_CODE_ZH.INVALID_CREDENTIALS,
    )
  })

  it('按英文原文（大小写不敏感）映射为中文', () => {
    expect(localizeErrorMessage('invalid email or password')).toBe('邮箱或密码错误')
    expect(localizeErrorMessage('Invalid Email Or Password')).toBe('邮箱或密码错误')
  })

  it('错误码优先于英文原文', () => {
    const byCode = localizeErrorMessage('invalid email or password', { code: 'TOKEN_EXPIRED' })
    expect(byCode).toBe(ERROR_CODE_ZH.TOKEN_EXPIRED)
    expect(byCode).not.toBe('邮箱或密码错误')
  })

  it('中文文案保持原样', () => {
    expect(localizeErrorMessage('操作成功')).toBe('操作成功')
  })

  it('URL 与纯技术标识符保持原样', () => {
    expect(localizeErrorMessage('https://example.com/path')).toBe('https://example.com/path')
    expect(localizeErrorMessage('SUCCESS')).toBe('SUCCESS')
  })

  it('表内不存在的前端英文错误落到中文兜底，不出现英文', () => {
    const out = localizeErrorMessage('some brand new unexpected failure')
    expect(out).toBe(ZH_FALLBACK.general)
    expect(containsChinese(out)).toBe(true)
  })

  it('网络类英文错误使用网络兜底文案', () => {
    expect(localizeErrorMessage('upstream network unreachable')).toBe(ZH_FALLBACK.network)
  })

  it('显式 fallback 优先于自动兜底', () => {
    expect(localizeErrorMessage('unexpected failure', { fallback: '操作失败，请稍后重试' })).toBe(
      '操作失败，请稍后重试',
    )
  })

  it('isTranslatableEnglishSentence 过滤非句子', () => {
    expect(isTranslatableEnglishSentence('')).toBe(false)
    expect(isTranslatableEnglishSentence('SUCCESS')).toBe(false)
    expect(isTranslatableEnglishSentence('操作成功')).toBe(false)
    expect(isTranslatableEnglishSentence('https://a.b')).toBe(false)
    expect(isTranslatableEnglishSentence('invalid email or password')).toBe(true)
  })

  it('localizeUnknownError 兼容 reason / message 组合对象', () => {
    expect(localizeUnknownError({ reason: 'TOKEN_EXPIRED', message: 'token has expired' })).toBe(
      ERROR_CODE_ZH.TOKEN_EXPIRED,
    )
  })

  it('localizeUnknownError 兼容 Error 实例', () => {
    expect(localizeUnknownError(new Error('invalid email or password'))).toBe('邮箱或密码错误')
  })

  it('英文界面保持原样透传', () => {
    localStorage.setItem('sub2api_locale', 'en')
    expect(localizeErrorMessage('invalid email or password')).toBe('invalid email or password')
  })

  it('新增的前端自有英文键均已收录', () => {
    for (const key of [
      'passkeys are not supported by this browser',
      'passkey sign-in was cancelled',
      'passkey creation was cancelled',
      'no refresh token available',
      'token refresh failed',
      'session changed during token refresh',
      'invalid proxy list response',
      'no response body',
      'not authenticated',
      'validation failed',
      'wechat_jsapi_unavailable',
    ]) {
      expect(ERROR_TEXT_ZH[key], `missing mapping: ${key}`).toBeTruthy()
    }
  })
})
