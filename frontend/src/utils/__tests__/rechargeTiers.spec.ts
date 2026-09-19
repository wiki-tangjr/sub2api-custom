import { describe, expect, it } from 'vitest'
import type { RechargeDiscountTier } from '@/types/payment'
import {
  formatRechargeDiscountTiersText,
  formatRechargeQuickAmountsText,
  normalizeRechargeDiscountTiers,
  parseRechargeDiscountTiersText,
  parseRechargeQuickAmountsText,
  parseTierRangeText,
  resolveRechargeDiscountPercent,
  tierLowerBound,
} from '@/utils/rechargeTiers'

describe('rechargeTiers', () => {
  it('accepts the legacy threshold alias as a lower bound', () => {
    expect(tierLowerBound({ min: 0, max: 0, percent: 2, threshold: 100 })).toBe(100)
    expect(tierLowerBound({ min: 50, max: 0, percent: 2, threshold: 100 })).toBe(50)
    expect(tierLowerBound(undefined)).toBe(0)
  })

  it('normalizes, filters and sorts tiers like the backend', () => {
    const tiers: RechargeDiscountTier[] = [
      { min: 500, max: 0, percent: 8 },
      { min: 100, max: 500, percent: 5 },
      { min: 100, max: 0, percent: 2 },
      { min: 0, max: 0, percent: 9 },
      { min: 200, max: 0, percent: 0 },
      { min: 300, max: 100, percent: 4 },
    ]
    expect(normalizeRechargeDiscountTiers(tiers)).toEqual([
      { min: 100, max: 500, percent: 5 },
      { min: 100, max: 0, percent: 2 },
      { min: 500, max: 0, percent: 8 },
    ])
  })

  it('returns an empty list for missing or malformed input', () => {
    expect(normalizeRechargeDiscountTiers(undefined)).toEqual([])
    expect(normalizeRechargeDiscountTiers(null)).toEqual([])
  })

  it('resolves ranges inclusively and picks the narrowest match', () => {
    const tiers: RechargeDiscountTier[] = [
      { min: 100, max: 0, percent: 2 },
      { min: 100, max: 500, percent: 5 },
      { min: 500, max: 0, percent: 8 },
      { min: 1000, max: 2000, percent: 12 },
    ]
    expect(resolveRechargeDiscountPercent(99.99, tiers)).toBe(0)
    expect(resolveRechargeDiscountPercent(100, tiers)).toBe(5)
    expect(resolveRechargeDiscountPercent(300, tiers)).toBe(5)
    expect(resolveRechargeDiscountPercent(500, tiers)).toBe(8)
    expect(resolveRechargeDiscountPercent(999, tiers)).toBe(8)
    expect(resolveRechargeDiscountPercent(1000, tiers)).toBe(12)
    expect(resolveRechargeDiscountPercent(1500, tiers)).toBe(12)
    expect(resolveRechargeDiscountPercent(5000, tiers)).toBe(8)
  })

  it('is independent of rule ordering', () => {
    const tiers: RechargeDiscountTier[] = [
      { min: 100, max: 500, percent: 5 },
      { min: 100, max: 0, percent: 2 },
      { min: 500, max: 0, percent: 8 },
    ]
    const reversed = tiers.slice().reverse()
    expect(resolveRechargeDiscountPercent(300, tiers)).toBe(
      resolveRechargeDiscountPercent(300, reversed),
    )
  })

  it('ignores non positive amounts', () => {
    expect(resolveRechargeDiscountPercent(0, [{ min: 1, max: 0, percent: 5 }])).toBe(0)
    expect(resolveRechargeDiscountPercent(-10, [{ min: 1, max: 0, percent: 5 }])).toBe(0)
    expect(resolveRechargeDiscountPercent(Number.NaN, [{ min: 1, max: 0, percent: 5 }])).toBe(0)
  })

  it('parses every supported range notation', () => {
    expect(parseTierRangeText('100')).toEqual({ min: 100, max: 0 })
    expect(parseTierRangeText('100-500')).toEqual({ min: 100, max: 500 })
    expect(parseTierRangeText('100~500')).toEqual({ min: 100, max: 500 })
    expect(parseTierRangeText('100～500')).toEqual({ min: 100, max: 500 })
    expect(parseTierRangeText('100-')).toEqual({ min: 100, max: 0 })
    expect(parseTierRangeText('-500')).toEqual({ min: 0, max: 500 })
    expect(parseTierRangeText(' 100 - 500 ')).toEqual({ min: 100, max: 500 })
  })

  it('rejects illegal range notation', () => {
    expect(parseTierRangeText('')).toBeNull()
    expect(parseTierRangeText('abc')).toBeNull()
    expect(parseTierRangeText('500-100')).toBeNull()
    expect(parseTierRangeText('-')).toBeNull()
    expect(parseTierRangeText('0-0')).toBeNull()
  })

  it('parses backend tier text, legacy and ranged', () => {
    expect(parseRechargeDiscountTiersText('100:2,500:3')).toEqual([
      { min: 100, max: 0, percent: 2 },
      { min: 500, max: 0, percent: 3 },
    ])
    expect(parseRechargeDiscountTiersText('500-1000:5,100-500:2')).toEqual([
      { min: 100, max: 500, percent: 2 },
      { min: 500, max: 1000, percent: 5 },
    ])
    expect(parseRechargeDiscountTiersText('abc:2,100:0,200:150,0:5,400:5')).toEqual([
      { min: 400, max: 0, percent: 5 },
    ])
    expect(parseRechargeDiscountTiersText('   ')).toEqual([])
  })

  it('round trips tier text', () => {
    const tiers = parseRechargeDiscountTiersText('100-500:2,500-:5,1000:7')
    expect(formatRechargeDiscountTiersText(tiers)).toBe('100-500:2,500:5,1000:7')
  })

  it('parses and formats quick amounts', () => {
    expect(parseRechargeQuickAmountsText('10, 50，100；200 500\n10')).toEqual([10, 50, 100, 200, 500])
    expect(parseRechargeQuickAmountsText('abc,0,-5,')).toEqual([])
    expect(formatRechargeQuickAmountsText([10, 50, 100])).toBe('10,50,100')
    expect(formatRechargeQuickAmountsText([])).toBe('')
  })
})