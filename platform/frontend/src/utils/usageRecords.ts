import type { UsageLog } from '@/types'
import { DASHBOARD_TIMEZONE, overviewMoney, preciseMoney } from './dashboardOverview'
import { getBillingModeLabel, getDisplayBillingMode } from './billingMode'
import { textInputTokens, textOutputTokens } from './imageUsage'
import { calculateTokenPricePerMillion } from './usagePricing'
import { isUsageRequestType, resolveUsageRequestType } from './usageRequestType'

export const isRecordedNumber = (value: unknown): value is number => typeof value === 'number' && Number.isFinite(value)
export const recordMoney = (value: unknown, precise = false): string =>
  isRecordedNumber(value) ? (precise ? preciseMoney(value) : overviewMoney(value)) : '—'
export const recordNumber = (value: unknown): string => isRecordedNumber(value) ? value.toLocaleString('en-US') : '—'
export const recordDuration = (value: unknown): string => isRecordedNumber(value) && value >= 0 ? `${recordNumber(value)} ms` : '—'

export function recordTime(value: string | null | undefined): string {
  if (!value) return '—'
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return '—'
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone: DASHBOARD_TIMEZONE, year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23',
  }).formatToParts(date)
  const part = (name: string) => parts.find(p => p.type === name)?.value
  return `${part('year')}/${part('month')}/${part('day')} ${part('hour')}:${part('minute')}:${part('second')}`
}

export function recordBillingMode(row: UsageLog, t: (key: string) => string): string {
  const mode = getDisplayBillingMode(row)
  return mode && ['token', 'per_request', 'image', 'video'].includes(mode) ? getBillingModeLabel(mode, t) : '—'
}

export function recordRequestType(row: UsageLog, t: (key: string) => string): string {
  if (!isUsageRequestType(row.request_type) && typeof row.stream !== 'boolean' && !row.openai_ws_mode) return '—'
  return t(`usageRecordDetails.requestTypes.${resolveUsageRequestType(row)}`)
}

export function recordDiscount(row: Pick<UsageLog, 'actual_cost' | 'total_cost'>): number | null {
  return isRecordedNumber(row.actual_cost) && isRecordedNumber(row.total_cost)
    ? Math.round((row.total_cost - row.actual_cost) * 1e10) / 1e10 : null
}

// Image tokens are subsets of input/output tokens; image costs are separate ledger fields.
// Cache TTL counts are subsets of cache creation, with no separate recorded cost allocation.
export function recordTokenParts(row: UsageLog) {
  const input = isRecordedNumber(row.input_tokens) && (row.image_input_tokens == null || isRecordedNumber(row.image_input_tokens))
    ? textInputTokens(row) : null
  const output = isRecordedNumber(row.output_tokens) && (row.image_output_tokens == null || isRecordedNumber(row.image_output_tokens))
    ? textOutputTokens(row) : null
  return [
    { key: 'textInput', tokens: input, cost: row.input_cost },
    { key: 'imageInput', tokens: row.image_input_tokens, cost: row.image_input_cost },
    { key: 'textOutput', tokens: output, cost: row.output_cost },
    { key: 'imageOutput', tokens: row.image_output_tokens, cost: row.image_output_cost },
    { key: 'cacheRead', tokens: row.cache_read_tokens, cost: row.cache_read_cost },
    { key: 'cacheCreation', tokens: row.cache_creation_tokens, cost: row.cache_creation_cost },
  ].map(part => ({ ...part, price: calculateTokenPricePerMillion(part.cost, part.tokens) }))
}
