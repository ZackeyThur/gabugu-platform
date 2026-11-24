import { describe, expect, it } from 'vitest'
import type { UsageLog } from '@/types'
import { recordBillingMode, recordDiscount, recordDuration, recordMoney, recordTime, recordTokenParts } from '../usageRecords'

describe('usage record presentation', () => {
  it('keeps tiny charges visible, precision and missing values separate from zero', () => {
    expect(recordMoney(0.0000000012)).toBe('<$0.01')
    expect(recordMoney(0.0000000012, true)).toBe('$0.0000000012')
    expect(recordMoney(0)).toBe('$0.00')
    for (const value of [undefined, null, NaN, Infinity]) expect(recordMoney(value, true)).toBe('—')
    expect(recordDuration(null)).toBe('—')
    expect(recordDuration(0)).toBe('0 ms')
    expect(recordDuration(NaN)).toBe('—')
  })
  it('uses Beijing time across UTC day and year boundaries, including seconds', () => {
    expect(recordTime('2025-12-31T16:00:03Z')).toBe('2026/01/01 00:00:03')
    expect(recordTime('invalid')).toBe('—')
  })
  it('preserves a charge above the site standard and ledger decimal precision', () => {
    expect(recordDiscount({ total_cost: 0.1, actual_cost: 0.3 })).toBe(-0.2)
    expect(recordDiscount({ total_cost: 0.0000000013, actual_cost: 0.0000000012 })).toBe(0.0000000001)
    expect(recordDiscount({ total_cost: NaN, actual_cost: 0 })).toBeNull()
  })
  it('splits image tokens but does not subtract separate image cost or double count TTLs', () => {
    const parts = recordTokenParts({ input_tokens: 120, image_input_tokens: 20, output_tokens: 35, image_output_tokens: 30, input_cost: 0.001, image_input_cost: 0.005, output_cost: 0.002, image_output_cost: 0.01, cache_creation_tokens: 50, cache_creation_cost: 0.003, cache_read_tokens: 80, cache_read_cost: 0.004, cache_creation_5m_tokens: 30, cache_creation_1h_tokens: 20 } as UsageLog)
    expect(parts.map(p => p.tokens)).toEqual([100, 20, 5, 30, 80, 50])
    expect(parts[0]?.cost).toBe(0.001)
    expect(parts[0]?.price).toBe(10)
    expect(parts).toHaveLength(6)
    expect(recordTokenParts({} as UsageLog)[0]?.tokens).toBeNull()
  })
  it('keeps explicit token/video modes and does not label unknown modes as token', () => {
    const t = (key: string) => key
    expect(recordBillingMode({ image_count: 2, billing_mode: 'video' } as UsageLog, t)).toBe('admin.usage.billingModeVideo')
    expect(recordBillingMode({ image_count: 2, billing_mode: 'token' } as UsageLog, t)).toBe('admin.usage.billingModeToken')
    expect(recordBillingMode({ image_count: 2 } as UsageLog, t)).toBe('admin.usage.billingModeImage')
    expect(recordBillingMode({ billing_mode: 'unknown' } as UsageLog, t)).toBe('—')
  })
})
