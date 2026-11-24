import type { UserDashboardStats } from '@/api/usage'
import type { GroupPlatform, ModelStat, PlatformQuotaItem, TrendDataPoint, UserSubscription } from '@/types'
import type { CheckoutInfoResponse } from '@/types/payment'
import type { SiteBillingMode } from '@/utils/siteBillingMode'
import {
  normalizeRechargeBonusMode,
  normalizeRechargeBonusTiers,
  quoteRechargeBonus,
  type RechargeBonusMode,
} from '@/utils/rechargeBonus'
import {
  getRemainingDurationParts,
  getRemainingExpiryDuration,
  isOneTimeDailyQuota,
  type RemainingDurationParts,
  type RemainingExpiryDuration,
} from '@/utils/subscriptionQuota'

export const DASHBOARD_TIMEZONE = 'Asia/Shanghai'
const dayMs = 86_400_000

export function dashboardToday(now = new Date()): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: DASHBOARD_TIMEZONE, year: 'numeric', month: '2-digit', day: '2-digit',
  }).format(now)
}

export function dashboardCurrentHour(now = new Date()): number {
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone: DASHBOARD_TIMEZONE,
    hour: '2-digit',
    hour12: false,
  }).formatToParts(now)
  const raw = Number(parts.find(p => p.type === 'hour')?.value ?? '0')
  if (!Number.isFinite(raw)) return 0
  return raw === 24 ? 0 : Math.max(0, Math.min(23, raw))
}

export function shiftDate(date: string, days: number): string {
  return new Date(Date.parse(date + 'T00:00:00Z') + days * dayMs).toISOString().slice(0, 10)
}

export function overviewMoney(value: number): string {
  if (!Number.isFinite(value)) return '—'
  if (value > 0 && value < 0.01) return '<$0.01'
  if (value < 0 && value > -0.01) return '-<$0.01'
  return new Intl.NumberFormat('en-US', {
    style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 2,
  }).format(value)
}

export function preciseMoney(value: number): string {
  if (!Number.isFinite(value)) return '—'
  return new Intl.NumberFormat('en-US', {
    style: 'currency', currency: 'USD', minimumFractionDigits: 4, maximumFractionDigits: 10,
  }).format(value)
}

export interface FormattedDecimalParts {
  major: string
  cents: string
  suffix: string
}

/**
 * Splits a formatted currency string (e.g. "$86.50") into major and decimal parts for Swiss-banking
 * optical typography, skipping "<$0.01" sentinel strings so micro-spend badges remain intact.
 */
export function splitFormattedDecimal(text: string): FormattedDecimalParts | null {
  if (!text || text.includes('<')) return null
  const match = /^([^\d]*\d[\d,]*)(\.\d+)(.*)$/.exec(text)
  if (!match) return null
  return { major: match[1], cents: match[2], suffix: match[3] || '' }
}

/**
 * Splits compact telemetry figures with a trailing unit letter or symbol (e.g. "6.87M", "1.64s", "77.4%")
 * so the numeric value remains bold while the trailing unit is subtly recessed without changing textContent.
 */
export function splitMetricUnit(text: string): { value: string; unit: string } | null {
  if (!text) return null
  const match = /^(-?\d[\d,]*(?:\.\d+)?)(K|M|B|%|ms|s)$/.exec(text.trim())
  if (!match) return null
  return { value: match[1], unit: match[2] }
}

export function formatCompactTokens(value: number): string {
  if (!Number.isFinite(value)) return '0'
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(2)}B`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return Math.round(value).toLocaleString()
}

export function formatDurationCompact(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return '—'
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`
  return `${Math.round(ms)}ms`
}

export const quotaWindows = ['daily', 'weekly', 'monthly'] as const
export type QuotaWindow = typeof quotaWindows[number]
export type QuotaState = 'disabled' | 'exhausted' | 'near' | 'active'

export function hasQuota(q?: PlatformQuotaItem): boolean {
  return !!q && quotaWindows.some(w => q[`${w}_limit_usd`] != null)
}

export function quotaState(q?: PlatformQuotaItem): QuotaState | null {
  if (!hasQuota(q)) return null
  if (quotaWindows.some(w => q?.[`${w}_limit_usd`] === 0)) return 'disabled'
  const ratios = quotaWindows.flatMap(w => {
    const limit = q?.[`${w}_limit_usd`]
    return limit != null && limit > 0 ? [(q?.[`${w}_usage_usd`] ?? 0) / limit] : []
  })
  if (ratios.some(r => r >= 1)) return 'exhausted'
  if (ratios.some(r => r >= 0.75)) return 'near'
  return 'active'
}

export interface UrgentQuotaInfo {
  window: QuotaWindow
  used: number
  limit: number
  ratio: number
  state: QuotaState
}

export function mostUrgentQuota(q?: PlatformQuotaItem): UrgentQuotaInfo | null {
  const state = quotaState(q)
  if (!state || !q) return null
  for (const w of quotaWindows) {
    if (q[`${w}_limit_usd`] === 0) {
      return { window: w, used: q[`${w}_usage_usd`] ?? 0, limit: 0, ratio: 1, state: 'disabled' }
    }
  }
  let best: UrgentQuotaInfo | null = null
  for (const w of quotaWindows) {
    const limit = q[`${w}_limit_usd`]
    if (limit == null || limit <= 0) continue
    const used = q[`${w}_usage_usd`] ?? 0
    const ratio = Math.max(0, used / limit)
    if (!best || ratio > best.ratio) {
      best = { window: w, used, limit, ratio, state }
    }
  }
  return best
}

export interface DashboardPlatform {
  platform: string
  total_actual_cost: number
  today_actual_cost: number
  total_requests: number | null
  total_tokens: number | null
  quota?: PlatformQuotaItem
}

const platformOrder = [
  'anthropic',
  'openai',
  'gemini',
  'antigravity',
  'grok',
  'typesafe',
  'kimi',
  'zhipu',
  'deepseek',
  'minimax',
  'opencode_go',
  'command_code',
  'cline',
]

const platformNames: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
  typesafe: 'TypeSafe / Jev',
  kimi: 'Kimi',
  zhipu: 'Zhipu GLM',
  deepseek: 'DeepSeek',
  minimax: 'MiniMax',
  opencode_go: 'OpenCode Go',
  command_code: 'Command Code',
  cline: 'Cline',
}

export const CONCRETE_PLATFORM_OPTIONS = platformOrder.map(id => ({
  value: id,
  label: platformNames[id] ?? id,
}))

export const platformName = (platform: string): string => platformNames[platform] ?? platform

export function dashboardPlatforms(stats: UserDashboardStats, quotas: PlatformQuotaItem[]): DashboardPlatform[] {
  const usage = new Map((stats.by_platform ?? []).map(p => [p.platform, p]))
  const limits = new Map(quotas.map(q => [q.platform, q]))
  const names = new Set([...usage.keys(), ...quotas.filter(hasQuota).map(q => q.platform)])
  const cards = [...names].map(platform => ({
    platform,
    total_actual_cost: usage.get(platform)?.total_actual_cost ?? 0,
    today_actual_cost: usage.get(platform)?.today_actual_cost ?? 0,
    total_requests: usage.get(platform)?.total_requests ?? 0,
    total_tokens: usage.get(platform)?.total_tokens ?? 0,
    quota: limits.get(platform),
  } as DashboardPlatform)).sort((a, b) => {
    const ai = platformOrder.indexOf(a.platform), bi = platformOrder.indexOf(b.platform)
    return (ai < 0 ? 99 : ai) - (bi < 0 ? 99 : bi) || a.platform.localeCompare(b.platform)
  })
  const total = Math.max(0, stats.total_actual_cost - cards.reduce((s, c) => s + c.total_actual_cost, 0))
  const today = Math.max(0, stats.today_actual_cost - cards.reduce((s, c) => s + c.today_actual_cost, 0))
  // Match the ledger's ten decimal places while ignoring floating-point dust.
  if (Math.round(total * 1e10) > 0 || Math.round(today * 1e10) > 0) {
    cards.push({ platform: '__other__', total_actual_cost: total, today_actual_cost: today, total_requests: null, total_tokens: null })
  }
  return cards
}

export function dailySpending(trend: TrendDataPoint[], start: string, end: string) {
  const byDate = new Map(trend.map(point => [point.date.slice(0, 10), point.actual_cost]))
  const days = Math.round((Date.parse(end) - Date.parse(start)) / dayMs) + 1
  if (!Number.isFinite(days) || days < 1 || days > 3660) return []
  return Array.from({ length: days }, (_, i) => {
    const date = shiftDate(start, i)
    return { date, amount: byDate.get(date) ?? 0 }
  })
}

export interface DashboardTrendBucket {
  key: string
  label: string
  date: string
  future: boolean
  requests: number | null
  inputTokens: number | null
  outputTokens: number | null
  cacheCreationTokens: number | null
  cacheReadTokens: number | null
  totalTokens: number | null
  cost: number | null
  actualCost: number | null
  cacheHitRate: number | null
}

function normalizeHourlyKey(rawDate: string): string {
  const trimmed = rawDate.trim().replace('T', ' ')
  if (trimmed.length >= 13) return trimmed.slice(0, 13) + ':00'
  return trimmed
}

export function buildTrendBuckets(
  trend: TrendDataPoint[],
  start: string,
  end: string,
  granularity: 'day' | 'hour',
  today = dashboardToday(),
  now = new Date()
): DashboardTrendBucket[] {
  if (granularity === 'hour') {
    if (!/^\d{4}-\d{2}-\d{2}$/.test(start)) return []
    const byHour = new Map<string, TrendDataPoint>()
    for (const point of trend) {
      const key = normalizeHourlyKey(point.date)
      const prev = byHour.get(key)
      if (!prev) {
        byHour.set(key, { ...point })
      } else {
        byHour.set(key, {
          date: key,
          requests: prev.requests + point.requests,
          input_tokens: prev.input_tokens + point.input_tokens,
          output_tokens: prev.output_tokens + point.output_tokens,
          cache_creation_tokens: prev.cache_creation_tokens + point.cache_creation_tokens,
          cache_read_tokens: prev.cache_read_tokens + point.cache_read_tokens,
          total_tokens: prev.total_tokens + point.total_tokens,
          cost: prev.cost + point.cost,
          actual_cost: prev.actual_cost + point.actual_cost,
        })
      }
    }
    const isToday = start === today
    const isFutureDay = start > today
    const currentHour = dashboardCurrentHour(now)
    return Array.from({ length: 24 }, (_, hour) => {
      const hh = String(hour).padStart(2, '0')
      const key = `${start} ${hh}:00`
      const future = isFutureDay || (isToday && hour > currentHour)
      if (future) {
        return {
          key,
          label: `${hh}:00`,
          date: start,
          future: true,
          requests: null,
          inputTokens: null,
          outputTokens: null,
          cacheCreationTokens: null,
          cacheReadTokens: null,
          totalTokens: null,
          cost: null,
          actualCost: null,
          cacheHitRate: null,
        }
      }
      const p = byHour.get(key)
      const inputTokens = p?.input_tokens ?? 0
      const outputTokens = p?.output_tokens ?? 0
      const cacheCreationTokens = p?.cache_creation_tokens ?? 0
      const cacheReadTokens = p?.cache_read_tokens ?? 0
      const promptTokens = inputTokens + cacheCreationTokens + cacheReadTokens
      return {
        key,
        label: `${hh}:00`,
        date: start,
        future: false,
        requests: p?.requests ?? 0,
        inputTokens,
        outputTokens,
        cacheCreationTokens,
        cacheReadTokens,
        totalTokens: p?.total_tokens ?? (inputTokens + outputTokens + cacheCreationTokens + cacheReadTokens),
        cost: p?.cost ?? 0,
        actualCost: p?.actual_cost ?? 0,
        cacheHitRate: promptTokens > 0 ? (cacheReadTokens / promptTokens) * 100 : null,
      }
    })
  }

  const days = Math.round((Date.parse(end) - Date.parse(start)) / dayMs) + 1
  if (!Number.isFinite(days) || days < 1 || days > 3660) return []
  const byDate = new Map<string, TrendDataPoint>()
  for (const point of trend) {
    const key = point.date.slice(0, 10)
    const prev = byDate.get(key)
    if (!prev) {
      byDate.set(key, { ...point })
    } else {
      byDate.set(key, {
        date: key,
        requests: prev.requests + point.requests,
        input_tokens: prev.input_tokens + point.input_tokens,
        output_tokens: prev.output_tokens + point.output_tokens,
        cache_creation_tokens: prev.cache_creation_tokens + point.cache_creation_tokens,
        cache_read_tokens: prev.cache_read_tokens + point.cache_read_tokens,
        total_tokens: prev.total_tokens + point.total_tokens,
        cost: prev.cost + point.cost,
        actual_cost: prev.actual_cost + point.actual_cost,
      })
    }
  }
  return Array.from({ length: days }, (_, i) => {
    const date = shiftDate(start, i)
    const future = date > today
    if (future) {
      return {
        key: date,
        label: date.slice(5).replace('-', '/'),
        date,
        future: true,
        requests: null,
        inputTokens: null,
        outputTokens: null,
        cacheCreationTokens: null,
        cacheReadTokens: null,
        totalTokens: null,
        cost: null,
        actualCost: null,
        cacheHitRate: null,
      }
    }
    const p = byDate.get(date)
    const inputTokens = p?.input_tokens ?? 0
    const outputTokens = p?.output_tokens ?? 0
    const cacheCreationTokens = p?.cache_creation_tokens ?? 0
    const cacheReadTokens = p?.cache_read_tokens ?? 0
    const promptTokens = inputTokens + cacheCreationTokens + cacheReadTokens
    return {
      key: date,
      label: date.slice(5).replace('-', '/'),
      date,
      future: false,
      requests: p?.requests ?? 0,
      inputTokens,
      outputTokens,
      cacheCreationTokens,
      cacheReadTokens,
      totalTokens: p?.total_tokens ?? (inputTokens + outputTokens + cacheCreationTokens + cacheReadTokens),
      cost: p?.cost ?? 0,
      actualCost: p?.actual_cost ?? 0,
      cacheHitRate: promptTokens > 0 ? (cacheReadTokens / promptTokens) * 100 : null,
    }
  })
}

export function rankedModels(models: ModelStat[], expanded: boolean) {
  const sorted = [...models].sort((a, b) => b.actual_cost - a.actual_cost || a.model.localeCompare(b.model))
  const total = sorted.reduce((sum, m) => sum + m.actual_cost, 0)
  const rows = (expanded ? sorted : sorted.slice(0, 5)).map(m => ({
    name: m.model, amount: m.actual_cost, share: total > 0 ? m.actual_cost / total * 100 : 0, other: false,
  }))
  if (!expanded && sorted.length > 5) {
    const amount = sorted.slice(5).reduce((sum, m) => sum + m.actual_cost, 0)
    rows.push({ name: '__other__', amount, share: total > 0 ? amount / total * 100 : 0, other: true })
  }
  return rows
}

export interface BalanceRunway {
  dailyBurn: number
  daysLeft: number | null
  status: 'healthy' | 'warning' | 'critical' | 'idle'
  basis: '7d' | 'today' | 'none'
}

/**
 * Computes cash runway strictly from billing_type=0 (balance) actual spend.
 * Subscription spend (billing_type=1) is never mixed in, preventing false low-balance alarms for hybrid/subscription users.
 */
export function calculateBalanceRunway(
  balance: number | null,
  stats: UserDashboardStats | null
): BalanceRunway {
  if (balance == null || !Number.isFinite(balance) || !stats) {
    return { dailyBurn: 0, daysLeft: null, status: 'idle', basis: 'none' }
  }
  const avg7dCash = (stats.recent_7d_balance_actual_cost ?? 0) / 7
  const todayCash = stats.today_balance_actual_cost ?? 0
  let dailyBurn = 0
  let basis: '7d' | 'today' | 'none' = 'none'
  if (avg7dCash > 0.0001) {
    dailyBurn = avg7dCash
    basis = '7d'
  } else if (todayCash > 0.0001) {
    dailyBurn = todayCash
    basis = 'today'
  }
  if (dailyBurn <= 0) {
    return { dailyBurn: 0, daysLeft: null, status: 'idle', basis: 'none' }
  }
  const daysLeft = Math.max(0, balance / dailyBurn)
  const status: BalanceRunway['status'] =
    daysLeft < 3 ? 'critical' : daysLeft < 10 ? 'warning' : 'healthy'
  return { dailyBurn, daysLeft, status, basis }
}

export interface YesterdayDelta {
  todayCost: number
  baselineCost: number
  percent: number | null
  direction: 'up' | 'down' | 'flat' | 'new'
}

export function calculateYesterdayDelta(stats: UserDashboardStats | null): YesterdayDelta | null {
  if (!stats) return null
  if (stats.yesterday_same_time_actual_cost === undefined && stats.yesterday_actual_cost === undefined) {
    return null
  }
  const todayCost = stats.today_actual_cost ?? 0
  const baselineCost = stats.yesterday_same_time_actual_cost ?? 0
  if (baselineCost <= 1e-6) {
    if (todayCost > 1e-6) {
      return { todayCost, baselineCost: 0, percent: null, direction: 'new' }
    }
    return null
  }
  const percent = ((todayCost - baselineCost) / baselineCost) * 100
  const direction: YesterdayDelta['direction'] =
    Math.abs(percent) < 1 ? 'flat' : percent > 0 ? 'up' : 'down'
  return { todayCost, baselineCost, percent, direction }
}

export interface CacheEfficiency {
  promptTokens: number
  cacheReadTokens: number
  cacheCreationTokens: number
  rate: number
}

export function calculateCacheEfficiency(
  inputTokens: number,
  cacheCreationTokens: number,
  cacheReadTokens: number
): CacheEfficiency {
  const safeInput = Math.max(0, Number(inputTokens) || 0)
  const safeCreation = Math.max(0, Number(cacheCreationTokens) || 0)
  const safeRead = Math.max(0, Number(cacheReadTokens) || 0)
  const promptTokens = safeInput + safeCreation + safeRead
  const rate = promptTokens > 0 ? (safeRead / promptTokens) * 100 : 0
  return {
    promptTokens,
    cacheReadTokens: safeRead,
    cacheCreationTokens: safeCreation,
    rate,
  }
}

export interface QuickRechargeOption {
  amount: number
  currencySymbol: string
  mode: RechargeBonusMode
  percent: number
  payBase: number
  credited: number
  bonus: number
  recommended: boolean
}

function availableCheckoutMethods(checkout: CheckoutInfoResponse) {
  // Backend /payment/checkout-info omits `available` on MethodLimits (returns only enabled methods).
  // Match PaymentView.vue (`ml?.available !== false`) so real responses are never filtered out.
  return Object.values(checkout.methods ?? {}).filter(m => m && m.available !== false)
}

function resolveCurrencySymbol(checkout: CheckoutInfoResponse): string {
  const methods = availableCheckoutMethods(checkout)
  if (methods.length > 0) {
    const currencies = new Set(methods.map(m => (m.currency || 'CNY').toUpperCase()))
    if (currencies.has('CNY') || currencies.has('RMB')) return '¥'
    if (currencies.has('USD')) return '$'
    const first = methods[0]?.currency?.toUpperCase()
    if (first === 'EUR') return '€'
    if (first === 'GBP') return '£'
    if (first === 'JPY') return '¥'
  }
  return '¥'
}

function resolveCheckoutAmountBounds(checkout: CheckoutInfoResponse): { min: number; max: number } {
  const methods = availableCheckoutMethods(checkout)
  let min = checkout.global_min > 0 ? checkout.global_min : 1
  let max = checkout.global_max > 0 ? checkout.global_max : Number.POSITIVE_INFINITY
  if (methods.length > 0) {
    const methodMins = methods.map(m => m.single_min).filter(v => Number.isFinite(v) && v > 0)
    const methodMaxs = methods.map(m => m.single_max).filter(v => Number.isFinite(v) && v > 0)
    if (methodMins.length > 0) {
      min = Math.max(min, Math.min(...methodMins))
    }
    if (methodMaxs.length > 0) {
      max = Math.min(max, Math.max(...methodMaxs))
    }
  }
  return { min, max }
}

/**
 * Builds 2-3 restrained quick-recharge pill options synced with backend bonus/discount tiers
 * and [global_min, global_max] payment bounds (honoring ¥1 = $1 multiplier=1 default).
 */
export function buildQuickRechargeOptions(
  checkout: CheckoutInfoResponse | null | undefined,
  dailyCashBurn = 0
): QuickRechargeOption[] {
  if (!checkout || checkout.balance_disabled) return []
  const methods = availableCheckoutMethods(checkout)
  if (Object.keys(checkout.methods ?? {}).length > 0 && methods.length === 0) return []

  const { min, max } = resolveCheckoutAmountBounds(checkout)
  const currencySymbol = resolveCurrencySymbol(checkout)
  const mode = normalizeRechargeBonusMode(checkout.recharge_bonus_mode)
  const tiers = normalizeRechargeBonusTiers(checkout.recharge_bonus_tiers)
  const multiplier =
    typeof checkout.balance_recharge_multiplier === 'number' &&
    Number.isFinite(checkout.balance_recharge_multiplier) &&
    checkout.balance_recharge_multiplier > 0
      ? checkout.balance_recharge_multiplier
      : 1

  const isMethodEligible = (amount: number): boolean => {
    if (amount < min || amount > max) return false
    if (methods.length === 0) return true
    const quote = quoteRechargeBonus(tiers, amount, { mode, multiplier })
    const payable = quote.payBase > 0 ? quote.payBase : amount
    return methods.some((m) => {
      const mMin = Number.isFinite(m.single_min) && m.single_min > 0 ? m.single_min : 0
      const mMax = Number.isFinite(m.single_max) && m.single_max > 0 ? m.single_max : Number.POSITIVE_INFINITY
      return payable >= mMin && payable <= mMax
    })
  }

  const validTierAmounts = tiers
    .filter(t => t.bonus_percent > 0 && isMethodEligible(t.min_amount))
    .map(t => t.min_amount)

  let chosenAmounts: number[] = []
  if (validTierAmounts.length > 0) {
    if (validTierAmounts.length <= 3) {
      chosenAmounts = validTierAmounts
    } else {
      const target = dailyCashBurn > 0 ? Math.max(min, dailyCashBurn * 21) : validTierAmounts[1]!
      let nearestIdx = 0
      let bestDist = Infinity
      validTierAmounts.forEach((amt, idx) => {
        const dist = Math.abs(amt - target)
        if (dist < bestDist) {
          bestDist = dist
          nearestIdx = idx
        }
      })
      const startIdx = Math.max(0, Math.min(validTierAmounts.length - 3, nearestIdx - 1))
      chosenAmounts = validTierAmounts.slice(startIdx, startIdx + 3)
    }
  } else {
    const defaults = dailyCashBurn >= 10 ? [50, 100, 300] : [20, 50, 100]
    chosenAmounts = defaults.filter(a => isMethodEligible(a)).slice(0, 3)
    if (chosenAmounts.length === 0 && Number.isFinite(min) && isMethodEligible(Math.ceil(min))) {
      chosenAmounts = [Math.ceil(min)]
    }
  }

  const targetMonthly = dailyCashBurn > 0 ? dailyCashBurn * 20 : 0
  let recommendedIdx = chosenAmounts.length > 1 ? 1 : 0
  if (targetMonthly > 0 && chosenAmounts.length > 0) {
    const idx = chosenAmounts.findIndex(a => a >= targetMonthly)
    recommendedIdx = idx >= 0 ? idx : chosenAmounts.length - 1
  }

  return chosenAmounts.map((amount, idx) => {
    const quote = quoteRechargeBonus(tiers, amount, { mode, multiplier })
    return {
      amount,
      currencySymbol,
      mode: quote.mode,
      percent: quote.percent,
      payBase: quote.payBase,
      credited: quote.credited,
      bonus: quote.bonus,
      recommended: idx === recommendedIdx && chosenAmounts.length >= 2,
    }
  })
}

export function compactQuotaMoney(value: number): string {
  if (!Number.isFinite(value)) return '—'
  if (value > 0 && value < 0.01) return '<$0.01'
  const rounded = Math.round(value * 100) / 100
  const isWhole = Math.abs(rounded - Math.round(rounded)) < 1e-6
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: isWhole ? 0 : 2,
    maximumFractionDigits: isWhole ? 0 : 2,
  }).format(rounded)
}

export function formatShortDurationParts(parts: RemainingDurationParts | null): string {
  if (!parts) return ''
  if (parts.days > 0) return `${parts.days}d ${parts.hours}h`
  if (parts.hours > 0) return `${parts.hours}h ${parts.minutes}m`
  return `${Math.max(1, parts.minutes)}m`
}

export interface SubscriptionWindowSummary {
  key: QuotaWindow
  used: number
  limit: number
  remaining: number
  ratio: number
  percent: number
  state: 'active' | 'near' | 'exhausted'
  isOneTime: boolean
  windowActive: boolean
  resetParts: RemainingDurationParts | null
  resetDurationText: string
}

export interface NormalizedSubscriptionSummary {
  id: number
  groupId: number
  groupName: string
  platform: GroupPlatform | null
  rateMultiplier: number | null
  expiresAt: string | null
  daysRemaining: number | null
  expiryDuration: RemainingExpiryDuration | null
  expiryStatus: 'healthy' | 'warning' | 'critical'
  isUnlimited: boolean
  windows: SubscriptionWindowSummary[]
  bindingWindow: SubscriptionWindowSummary | null
  tightestWindow: SubscriptionWindowSummary | null
  barState: 'active' | 'near' | 'exhausted'
  needsRenew: boolean
}

const windowDurationMs: Record<QuotaWindow, number> = {
  daily: dayMs,
  weekly: 7 * dayMs,
  monthly: 30 * dayMs,
}

export function normalizeActiveSubscriptions(
  subscriptions: UserSubscription[] | null | undefined,
  now = new Date()
): NormalizedSubscriptionSummary[] {
  if (!subscriptions?.length) return []

  const items = subscriptions.map((sub): NormalizedSubscriptionSummary => {
    const expiryDuration = sub.expires_at ? getRemainingExpiryDuration(sub.expires_at, now) : null
    const daysRemaining = !sub.expires_at
      ? null
      : expiryDuration?.unit === 'days'
        ? expiryDuration.days
        : 0

    const expiryStatus: NormalizedSubscriptionSummary['expiryStatus'] = !sub.expires_at
      ? 'healthy'
      : !expiryDuration || expiryDuration.unit === 'hoursMinutes' || expiryDuration.days <= 3
        ? 'critical'
        : expiryDuration.days <= 7
          ? 'warning'
          : 'healthy'

    const oneTimeDaily = isOneTimeDailyQuota(sub)
    const windows: SubscriptionWindowSummary[] = []

    for (const key of quotaWindows) {
      const limit = sub.group?.[`${key}_limit_usd`]
      if (limit == null || !Number.isFinite(limit) || limit <= 0) continue
      const used = Math.max(0, Number(sub[`${key}_usage_usd`]) || 0)
      const remaining = Math.max(0, limit - used)
      const ratio = used / limit
      const percent = Math.min(100, Math.round(ratio * 100))
      const state: SubscriptionWindowSummary['state'] =
        ratio >= 1 ? 'exhausted' : ratio >= 0.75 ? 'near' : 'active'

      let resetParts: RemainingDurationParts | null = null
      let windowActive = false
      const isOneTime = key === 'daily' && oneTimeDaily

      if (isOneTime && sub.expires_at) {
        windowActive = true
        resetParts = getRemainingDurationParts(sub.expires_at, now)
      } else {
        const startStr = sub[`${key}_window_start`]
        if (startStr) {
          const startMs = Date.parse(startStr)
          if (Number.isFinite(startMs)) {
            windowActive = true
            resetParts = getRemainingDurationParts(new Date(startMs + windowDurationMs[key]), now)
          }
        }
      }

      windows.push({
        key,
        used,
        limit,
        remaining,
        ratio,
        percent,
        state,
        isOneTime,
        windowActive,
        resetParts,
        resetDurationText: formatShortDurationParts(resetParts),
      })
    }

    const isUnlimited = windows.length === 0
    // Binding window: lowest remaining USD headroom right now (tie-broken daily > weekly > monthly)
    const bindingWindow = windows.reduce<SubscriptionWindowSummary | null>((best, cur) => {
      if (!best) return cur
      return cur.remaining < best.remaining - 1e-6 ? cur : best
    }, null)

    const tightestWindow = windows.reduce<SubscriptionWindowSummary | null>((best, cur) => {
      if (!best) return cur
      return cur.ratio > best.ratio ? cur : best
    }, null)

    const barState: NormalizedSubscriptionSummary['barState'] = tightestWindow?.state ?? 'active'
    const needsRenew =
      (daysRemaining !== null && daysRemaining <= 7) ||
      barState === 'near' ||
      barState === 'exhausted'

    const rawMultiplier = sub.group?.rate_multiplier
    const rateMultiplier =
      typeof rawMultiplier === 'number' && Number.isFinite(rawMultiplier) && rawMultiplier > 0
        ? rawMultiplier
        : null

    return {
      id: sub.id,
      groupId: sub.group_id,
      groupName: sub.group?.name || `#${sub.group_id}`,
      platform: sub.group?.platform ?? null,
      rateMultiplier,
      expiresAt: sub.expires_at ?? null,
      daysRemaining,
      expiryDuration,
      expiryStatus,
      isUnlimited,
      windows,
      bindingWindow,
      tightestWindow,
      barState,
      needsRenew,
    }
  })

  // Surface exhausted or critically expiring (<=3d) plans first while preserving original order otherwise
  return items
    .map((item, index) => ({
      item,
      index,
      priority:
        item.barState === 'exhausted'
          ? 2
          : item.expiryStatus === 'critical'
            ? 1
            : 0,
    }))
    .sort((a, b) => b.priority - a.priority || a.index - b.index)
    .map(entry => entry.item)
}

export function isSubscriptionPrimaryUser(params: {
  balance: number | null
  activeSubscriptions: UserSubscription[] | null | undefined
  stats: UserDashboardStats | null
  billingMode?: SiteBillingMode
  paymentEnabled?: boolean
}): boolean {
  const subs = params.activeSubscriptions ?? []
  if (subs.length === 0) return false
  if (params.billingMode === 'subscription_only') return true
  if (params.paymentEnabled === false) return true
  const bal = params.balance ?? 0
  if (bal <= 0.01) return true
  const sub7d = params.stats?.recent_7d_subscription_actual_cost ?? 0
  const cash7d = params.stats?.recent_7d_balance_actual_cost ?? 0
  if (sub7d > 0.01 && cash7d <= 0.01 && bal < 5) return true
  return false
}
