import type { RechargeDiscountTier } from '@/types/payment'

/**
 * Customization (#20): 归一化后的充值优惠区间档位。
 * max = 0 表示无上限（即历史兼容的"满 X 减 Y%"）。
 */
export interface NormalizedRechargeTier {
  min: number
  max: number
  percent: number
}

/** 后台配置上限，与后端 maxRechargeDiscountTiers / maxRechargeQuickAmounts 保持一致。 */
export const MAX_RECHARGE_DISCOUNT_TIERS = 24
export const MAX_RECHARGE_QUICK_AMOUNTS = 24

/** 兼容仅带 threshold 的旧数据：min 优先，其次 threshold。 */
export function tierLowerBound(tier: RechargeDiscountTier | null | undefined): number {
  if (!tier) return 0
  const min = Number(tier.min)
  if (Number.isFinite(min) && min > 0) return min
  const threshold = Number(tier.threshold)
  if (Number.isFinite(threshold) && threshold > 0) return threshold
  return 0
}

/**
 * 过滤非法档位并按区间下限升序排列（同下限时无上限排最后）。
 * 与后端 parseRechargeDiscountTiers 的规则严格一致，保证前后台展示一致。
 */
export function normalizeRechargeDiscountTiers(
  tiers?: RechargeDiscountTier[] | null,
): NormalizedRechargeTier[] {
  if (!Array.isArray(tiers)) return []
  const out: NormalizedRechargeTier[] = []
  for (const tier of tiers) {
    if (!tier) continue
    const min = tierLowerBound(tier)
    if (!(min > 0)) continue
    const percent = Number(tier.percent)
    if (!Number.isFinite(percent) || percent <= 0 || percent >= 100) continue
    const rawMax = Number(tier.max)
    const max = Number.isFinite(rawMax) && rawMax > 0 ? rawMax : 0
    if (max > 0 && max < min) continue
    out.push({ min, max, percent })
  }
  out.sort(compareTiers)
  return out
}

function compareTiers(a: NormalizedRechargeTier, b: NormalizedRechargeTier): number {
  if (a.min !== b.min) return a.min - b.min
  if (a.max === 0) return 1
  if (b.max === 0) return -1
  return a.max - b.max
}

/**
 * 按订单真实金额匹配优惠百分比；无命中返回 0。
 * 多区间同时命中时取"下限最高、其次上限最窄"的那条，与后端一致。
 * 注意：最终判定始终由后端完成，前端仅用于展示预览。
 */
export function resolveRechargeDiscountPercent(
  amount: number,
  tiers?: RechargeDiscountTier[] | null,
): number {
  if (!Number.isFinite(amount) || amount <= 0) return 0
  let matched = false
  let bestPercent = 0
  let bestLower = -1
  let bestUpper = -1
  for (const tier of normalizeRechargeDiscountTiers(tiers)) {
    if (amount < tier.min) continue
    if (tier.max > 0 && amount > tier.max) continue
    let narrower = false
    if (!matched) {
      narrower = true
    } else if (tier.min > bestLower) {
      narrower = true
    } else if (tier.min === bestLower && tier.max > 0 && (bestUpper <= 0 || tier.max < bestUpper)) {
      narrower = true
    }
    if (narrower) {
      matched = true
      bestPercent = tier.percent
      bestLower = tier.min
      bestUpper = tier.max
    }
  }
  return matched ? bestPercent : 0
}

// ==================== 后台文本 <-> 结构互转 ====================
// 与后端 parseRechargeTierRange 完全同构，供后台结构化编辑器使用。

/** 解析单段金额区间：`100`、`100-500`、`100~500`、`100-`、`-500`。 */
export function parseTierRangeText(raw: string): { min: number; max: number } | null {
  const text = String(raw ?? '').trim()
  if (!text) return null
  let sepIndex = -1
  for (const sep of ['~', '\uff5e', '\u2014', '\u2013']) {
    const i = text.indexOf(sep)
    if (i >= 0) {
      sepIndex = i
      break
    }
  }
  let minText = ''
  let maxText = ''
  if (sepIndex >= 0) {
    minText = text.slice(0, sepIndex)
    maxText = text.slice(sepIndex + 1)
  } else {
    const dash = text.indexOf('-')
    if (dash >= 0) {
      minText = text.slice(0, dash)
      maxText = text.slice(dash + 1)
    } else {
      minText = text
    }
  }
  minText = minText.trim()
  maxText = maxText.trim()
  let min = 0
  if (minText !== '') {
    const v = Number(minText)
    if (!Number.isFinite(v) || v < 0) return null
    min = round2(v)
  } else if (maxText === '') {
    return null
  }
  let max = 0
  if (maxText !== '') {
    const v = Number(maxText)
    if (!Number.isFinite(v) || v <= 0) return null
    max = round2(v)
    if (max < min) return null
  }
  // 上下限都没有的档位对任何金额都成立，等同于无条件优惠，判为非法。
  if (min <= 0 && max <= 0) return null
  return { min, max }
}

function round2(v: number): number {
  return Math.round(v * 100) / 100
}

function round4(v: number): number {
  return Math.round(v * 10000) / 10000
}

/**
 * 把后台文本框解析为档位结构，兼容历史"满 X"与新的区间写法：
 *   `100:2,500:3`      -> 满 100 减 2%、满 500 减 3%
 *   `100-500:2,500-:5` -> 100~500 减 2%、满 500 减 5%
 * 非法条目直接丢弃，结果按下限升序排列。
 */
export function parseRechargeDiscountTiersText(raw: string): NormalizedRechargeTier[] {
  const text = String(raw ?? '').trim()
  if (!text) return []
  const fields = text.replace(/[\uff0c\uff1b;\n\t ]/g, ',').split(',')
  const byKey = new Map<string, NormalizedRechargeTier>()
  for (const field of fields) {
    const item = field.trim()
    if (!item) continue
    const sep = item.includes(':') ? ':' : item.includes('=') ? '=' : ''
    if (!sep) continue
    const idx = item.indexOf(sep)
    const range = parseTierRangeText(item.slice(0, idx))
    if (!range) continue
    const percent = Number(item.slice(idx + 1).trim())
    if (!Number.isFinite(percent) || percent <= 0 || percent >= 100) continue
    const normalized: NormalizedRechargeTier = {
      min: range.min,
      max: range.max,
      percent: round4(percent),
    }
    if (normalized.percent >= 100) continue
    byKey.set(`${normalized.min}|${normalized.max}`, normalized)
  }
  const out = Array.from(byKey.values())
  out.sort(compareTiers)
  return out.slice(0, MAX_RECHARGE_DISCOUNT_TIERS)
}

/** 序列化档位结构；无上限档位写成 `min:percent`，保持历史配置原文不变。 */
export function formatRechargeDiscountTiersText(tiers: NormalizedRechargeTier[]): string {
  if (!Array.isArray(tiers) || tiers.length === 0) return ''
  return tiers
    .slice()
    .sort(compareTiers)
    .map((tier) => {
      const range = tier.max > 0 ? `${trimNumber(tier.min)}-${trimNumber(tier.max)}` : trimNumber(tier.min)
      return `${range}:${trimNumber(tier.percent)}`
    })
    .join(',')
}

/**
 * 解析快捷充值金额文本框；支持中英文逗号 / 分号 / 空格 / 换行分隔。
 * 非法值丢弃，去重，升序排列，最多 24 个。
 */
export function parseRechargeQuickAmountsText(raw: string): number[] {
  const text = String(raw ?? '').trim()
  if (!text) return []
  const fields = text.replace(/[\uff0c\uff1b;\n\t ]/g, ',').split(',')
  const seen = new Set<string>()
  const out: number[] = []
  for (const field of fields) {
    const item = field.trim()
    if (!item) continue
    const v = Number(item)
    if (!Number.isFinite(v) || v <= 0) continue
    const rounded = round2(v)
    if (rounded <= 0) continue
    const key = trimNumber(rounded)
    if (seen.has(key)) continue
    seen.add(key)
    out.push(rounded)
    if (out.length >= MAX_RECHARGE_QUICK_AMOUNTS) break
  }
  out.sort((a, b) => a - b)
  return out
}

export function formatRechargeQuickAmountsText(amounts: number[]): string {
  if (!Array.isArray(amounts) || amounts.length === 0) return ''
  return amounts.slice(0, MAX_RECHARGE_QUICK_AMOUNTS).map(trimNumber).join(',')
}

/** 去掉浮点尾差与多余的 0，和后端 strconv.FormatFloat(v, 'f', -1, 64) 输出一致。 */
export function trimNumber(value: number): string {
  if (!Number.isFinite(value)) return '0'
  return String(Math.round(value * 10000) / 10000)
}