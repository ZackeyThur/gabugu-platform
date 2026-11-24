import { describe, it, expect } from 'vitest'
import {
  buildQuickRechargeOptions,
  buildTrendBuckets,
  calculateBalanceRunway,
  calculateCacheEfficiency,
  calculateYesterdayDelta,
  compactQuotaMoney,
  dailySpending,
  dashboardPlatforms,
  dashboardToday,
  isSubscriptionPrimaryUser,
  normalizeActiveSubscriptions,
  overviewMoney,
  preciseMoney,
  quotaState,
  rankedModels,
  splitFormattedDecimal,
  splitMetricUnit,
} from '../dashboardOverview'
import type { UserDashboardStats } from '@/api/usage'
import type { TrendDataPoint, ModelStat, PlatformQuotaItem, UserSubscription } from '@/types'
import type { CheckoutInfoResponse } from '@/types/payment'
describe('费用总览日期与金额', () => {
  it('北京时间午夜不依赖浏览器日期', () => {
    expect(dashboardToday(new Date('2026-11-08T15:59:59Z'))).toBe('2026-11-08')
    expect(dashboardToday(new Date('2026-11-08T16:00:00Z'))).toBe('2026-11-09')
  })
  it('小额与负金额保留且精确值可查', () => {
    expect(overviewMoney(.000123)).toBe('<$0.01')
    expect(overviewMoney(-.000123)).toBe('-<$0.01')
    expect(preciseMoney(.000123)).toBe('$0.000123')
    expect(preciseMoney(.0000000001)).toBe('$0.0000000001')
    expect(overviewMoney(0)).toBe('$0.00')
    expect(overviewMoney(NaN)).toBe('—')
  })
  it('平台未归属微小差额不会被0.0001阈值吞掉', () => {
    const cards = dashboardPlatforms({ total_actual_cost: .00005, today_actual_cost: .0000000001, by_platform: [] } as unknown as UserDashboardStats, [])
    expect(cards).toEqual([{ platform: '__other__', total_actual_cost: .00005, today_actual_cost: .0000000001, total_requests: null, total_tokens: null }])
  })
  it('按日期补零且仅统计实际费用', () => {
    const result = dailySpending([{ date: '2026-11-08', cost: 100, actual_cost: .02 } as TrendDataPoint], '2026-11-07', '2026-11-09')
    expect(result).toEqual([{ date: '2026-11-07', amount: 0 }, { date: '2026-11-08', amount: .02 }, { date: '2026-11-09', amount: 0 }])
    expect(dailySpending([], 'invalid', '2026-11-09')).toEqual([])
  })
  it('模型按实扣前5与其他，份额分母包含完整列表，展开不丢金额', () => {
    const input = [1, 7, 3, 6, 2, 5, 4].map(n => ({ model: `m${n}`, actual_cost: n, cost: 100 - n }) as ModelStat)
    const rows = rankedModels(input, false)
    expect(rows.map(r => r.name)).toEqual(['m7', 'm6', 'm5', 'm4', 'm3', '__other__'])
    expect(rows.at(-1)?.amount).toBe(3)
    expect(rows.reduce((sum, r) => sum + r.share, 0)).toBeCloseTo(100)
    expect(rankedModels(input, true)).toHaveLength(7)
    expect(input[0].model).toBe('m1')
  })
  it('多窗口取最严格状态', () => {
    expect(quotaState({ daily_limit_usd: 10, daily_usage_usd: 1, weekly_limit_usd: 10, weekly_usage_usd: 10 } as PlatformQuotaItem)).toBe('exhausted')
  })

  it('buildTrendBuckets 按小时补齐24桶且未来小时为null，按天聚合指标与缓存命中率', () => {
    const now = new Date('2026-11-08T02:30:00Z') // 10:30 UTC+8
    const hourly = buildTrendBuckets(
      [{
        date: '2026-11-08 09:00',
        requests: 5,
        input_tokens: 100,
        output_tokens: 50,
        cache_creation_tokens: 0,
        cache_read_tokens: 300,
        total_tokens: 450,
        cost: 1,
        actual_cost: 0.6,
      }],
      '2026-11-08',
      '2026-11-08',
      'hour',
      '2026-11-08',
      now
    )
    expect(hourly).toHaveLength(24)
    expect(hourly[9].actualCost).toBeCloseTo(0.6)
    expect(hourly[9].cacheHitRate).toBeCloseTo(75)
    expect(hourly[10].future).toBe(false)
    expect(hourly[11].future).toBe(true)
    expect(hourly[11].actualCost).toBeNull()
    expect(hourly[11].requests).toBeNull()
  })

  it('calculateBalanceRunway 严格隔离现金余额与订阅额度，纯订阅用户不误报余额危机', () => {
    const pureSub = calculateBalanceRunway(10, {
      recent_7d_balance_actual_cost: 0,
      recent_7d_subscription_actual_cost: 700,
      today_balance_actual_cost: 0,
      today_actual_cost: 100,
    } as UserDashboardStats)
    expect(pureSub.daysLeft).toBeNull()
    expect(pureSub.status).toBe('idle')

    const critical = calculateBalanceRunway(4, {
      recent_7d_balance_actual_cost: 14,
      recent_7d_subscription_actual_cost: 0,
      today_balance_actual_cost: 2,
    } as UserDashboardStats)
    expect(critical.dailyBurn).toBeCloseTo(2)
    expect(critical.daysLeft).toBeCloseTo(2)
    expect(critical.status).toBe('critical')

    const healthy = calculateBalanceRunway(100, {
      recent_7d_balance_actual_cost: 14,
    } as UserDashboardStats)
    expect(healthy.status).toBe('healthy')
  })

  it('calculateYesterdayDelta 精准计算昨日同期同比与无基线新消费', () => {
    expect(calculateYesterdayDelta({
      today_actual_cost: 12,
      yesterday_same_time_actual_cost: 10,
    } as UserDashboardStats)).toMatchObject({ direction: 'up', percent: 20 })

    expect(calculateYesterdayDelta({
      today_actual_cost: 5,
      yesterday_same_time_actual_cost: 10,
    } as UserDashboardStats)).toMatchObject({ direction: 'down', percent: -50 })

    expect(calculateYesterdayDelta({
      today_actual_cost: 1,
      yesterday_same_time_actual_cost: 0,
    } as UserDashboardStats)).toMatchObject({ direction: 'new', percent: null })

    expect(calculateYesterdayDelta({
      today_actual_cost: 0,
      yesterday_same_time_actual_cost: 0,
    } as UserDashboardStats)).toBeNull()
  })

  it('calculateCacheEfficiency 正确计算提示词缓存命中率', () => {
    const res = calculateCacheEfficiency(200, 100, 700)
    expect(res.promptTokens).toBe(1000)
    expect(res.rate).toBeCloseTo(70)
    expect(calculateCacheEfficiency(0, 0, 0).rate).toBe(0)
  })

  it('buildQuickRechargeOptions 联动后台满赠与限额，禁用充值时返回空', () => {
    const baseCheckout = {
      methods: { alipay: { single_min: 10, single_max: 500, daily_limit: 0, available: true, fee_rate: 0, currency: 'CNY' } },
      global_min: 10,
      global_max: 500,
      plans: [],
      balance_disabled: false,
      balance_recharge_multiplier: 1,
      recharge_fee_rate: 0,
      help_text: '',
      help_image_url: '',
      stripe_publishable_key: '',
      recharge_bonus_mode: 'bonus',
      recharge_bonus_tiers: [
        { min_amount: 50, bonus_percent: 5 },
        { min_amount: 100, bonus_percent: 10 },
        { min_amount: 300, bonus_percent: 15 },
        { min_amount: 1000, bonus_percent: 20 }, // exceeds global_max 500 -> excluded
      ],
    } as CheckoutInfoResponse

    const opts = buildQuickRechargeOptions(baseCheckout, 5)
    expect(opts.map(o => o.amount)).toEqual([50, 100, 300])
    expect(opts[1]).toMatchObject({ amount: 100, percent: 10, bonus: 10, credited: 110, recommended: true })

    expect(buildQuickRechargeOptions({ ...baseCheckout, balance_disabled: true }, 5)).toEqual([])

    // Disjoint payment method ranges: 50 falls in gap between [1..20] and [100..500] -> must be excluded
    const disjointCheckout: CheckoutInfoResponse = {
      ...baseCheckout,
      methods: {
        small: { single_min: 1, single_max: 20, daily_limit: 0, available: true, fee_rate: 0, currency: 'CNY' },
        large: { single_min: 100, single_max: 500, daily_limit: 0, available: true, fee_rate: 0, currency: 'CNY' },
      },
      global_min: 1,
      global_max: 500,
    }
    expect(buildQuickRechargeOptions(disjointCheckout, 5).map(o => o.amount)).toEqual([100, 300])

    // Real backend /payment/checkout-info payload omits `available` on MethodLimits (returns enabled methods only)
    // and may have empty recharge_bonus_tiers and global_min/max = 0 (unlimited).
    const liveShapedCheckout: CheckoutInfoResponse = {
      ...baseCheckout,
      methods: {
        alipay: { single_min: 0, single_max: 0, daily_limit: 0, daily_used: 0, daily_remaining: 0, fee_rate: 0, currency: 'CNY' },
        wxpay: { single_min: 0, single_max: 0, daily_limit: 0, daily_used: 0, daily_remaining: 0, fee_rate: 0, currency: 'CNY' },
      },
      global_min: 0,
      global_max: 0,
      recharge_bonus_tiers: [],
    }
    const liveHeavy = buildQuickRechargeOptions(liveShapedCheckout, 65.4)
    expect(liveHeavy.map(o => o.amount)).toEqual([50, 100, 300])
    expect(liveHeavy.every(o => o.currencySymbol === '¥' && o.credited === o.amount)).toBe(true)
    expect(buildQuickRechargeOptions(liveShapedCheckout, 3).map(o => o.amount)).toEqual([20, 50, 100])
    // Explicitly disabled methods (available: false) must still be excluded
    const disabledMethodsCheckout: CheckoutInfoResponse = {
      ...liveShapedCheckout,
      methods: {
        alipay: { single_min: 0, single_max: 0, daily_limit: 0, daily_used: 0, daily_remaining: 0, fee_rate: 0, currency: 'CNY', available: false },
      },
    }
    expect(buildQuickRechargeOptions(disabledMethodsCheckout, 65.4)).toEqual([])

    // ISO 'T' hourly timestamp normalization
    const isoBuckets = buildTrendBuckets(
      [{ date: '2026-11-09T09:00:00+08:00', requests: 2, input_tokens: 10, output_tokens: 5, cache_creation_tokens: 0, cache_read_tokens: 0, total_tokens: 15, cost: 0.5, actual_cost: 0.4 }],
      '2026-11-09',
      '2026-11-09',
      'hour',
      '2026-11-09',
      new Date('2026-11-09T04:00:00Z') // 12:00 UTC+8
    )
    expect(isoBuckets[9].actualCost).toBe(0.4)
  })

  it('splitFormattedDecimal 正确拆分瑞士金融小数并保护微额与单位哨兵', () => {
    expect(splitFormattedDecimal('$86.50')).toEqual({ major: '$86', cents: '.50', suffix: '' })
    expect(splitFormattedDecimal('$1,234.56')).toEqual({ major: '$1,234', cents: '.56', suffix: '' })
    expect(splitFormattedDecimal('<$0.01')).toBeNull()
    expect(splitFormattedDecimal('-<$0.01')).toBeNull()
    expect(splitFormattedDecimal('44.03M')).toEqual({ major: '44', cents: '.03', suffix: 'M' })
    expect(splitFormattedDecimal('—')).toBeNull()
  })

  it('splitMetricUnit 精准拆分紧凑量级和单位后缀', () => {
    expect(splitMetricUnit('6.87M')).toEqual({ value: '6.87', unit: 'M' })
    expect(splitMetricUnit('1.64s')).toEqual({ value: '1.64', unit: 's' })
    expect(splitMetricUnit('77.4%')).toEqual({ value: '77.4', unit: '%' })
    expect(splitMetricUnit('420ms')).toEqual({ value: '420', unit: 'ms' })
    expect(splitMetricUnit('286')).toBeNull()
    expect(splitMetricUnit('—')).toBeNull()
  })

  it('compactQuotaMoney 与 normalizeActiveSubscriptions 准确计算绑定窗口、重置倒计时与不限额套餐', () => {
    expect(compactQuotaMoney(25)).toBe('$25')
    expect(compactQuotaMoney(8.2)).toBe('$8.20')
    expect(compactQuotaMoney(0.004)).toBe('<$0.01')

    const now = new Date('2026-11-24T08:00:00Z')
    const healthyPlan = {
      id: 1,
      user_id: 1,
      group_id: 10,
      status: 'active',
      starts_at: '2026-11-01T00:00:00Z',
      expires_at: '2026-12-16T08:00:00Z',
      daily_usage_usd: 8.2,
      weekly_usage_usd: 61.4,
      monthly_usage_usd: 213.5,
      daily_window_start: '2026-11-23T22:00:00Z', // 14h left
      weekly_window_start: '2026-11-21T02:00:00Z', // 3d 18h left
      monthly_window_start: '2026-11-01T00:00:00Z',
      group: {
        id: 10,
        name: 'Claude 极速专享月卡',
        platform: 'anthropic',
        rate_multiplier: 0.65,
        daily_limit_usd: 25,
        weekly_limit_usd: 140,
        monthly_limit_usd: 500,
      },
    } as unknown as UserSubscription

    const exhaustedPlan = {
      id: 2,
      user_id: 1,
      group_id: 20,
      status: 'active',
      starts_at: '2026-11-01T00:00:00Z',
      expires_at: '2026-11-26T08:00:00Z', // 2d -> critical
      daily_usage_usd: 25,
      weekly_usage_usd: 100,
      monthly_usage_usd: 300,
      daily_window_start: '2026-11-24T00:00:00Z',
      weekly_window_start: null,
      monthly_window_start: null,
      group: {
        id: 20,
        name: 'OpenAI Codex Pro',
        platform: 'openai',
        rate_multiplier: 1,
        daily_limit_usd: 25,
        weekly_limit_usd: 140,
        monthly_limit_usd: null,
      },
    } as unknown as UserSubscription

    const unlimitedPlan = {
      id: 3,
      user_id: 1,
      group_id: 30,
      status: 'active',
      starts_at: '2026-11-01T00:00:00Z',
      expires_at: null,
      daily_usage_usd: 0,
      weekly_usage_usd: 0,
      monthly_usage_usd: 0,
      daily_window_start: null,
      weekly_window_start: null,
      monthly_window_start: null,
      group: { id: 30, name: '内部不限额专线', platform: 'gemini' },
    } as unknown as UserSubscription

    const normalized = normalizeActiveSubscriptions([healthyPlan, exhaustedPlan, unlimitedPlan], now)
    // Exhausted plan rises to priority 0
    expect(normalized[0].id).toBe(2)
    expect(normalized[0].barState).toBe('exhausted')
    expect(normalized[0].expiryStatus).toBe('critical')
    expect(normalized[0].bindingWindow?.remaining).toBe(0)
    expect(normalized[0].needsRenew).toBe(true)

    // Healthy plan preserves multi-window remaining & reset countdowns
    expect(normalized[1].id).toBe(1)
    expect(normalized[1].barState).toBe('active')
    expect(normalized[1].bindingWindow?.key).toBe('daily')
    expect(normalized[1].bindingWindow?.remaining).toBeCloseTo(16.8)
    expect(normalized[1].bindingWindow?.resetDurationText).toBe('14h 0m')
    expect(normalized[1].windows).toHaveLength(3)
    expect(normalized[1].needsRenew).toBe(false)

    // Unlimited plan gracefully flagged with no binding window
    expect(normalized[2].isUnlimited).toBe(true)
    expect(normalized[2].bindingWindow).toBeNull()
    expect(normalized[2].expiryStatus).toBe('healthy')
  })

  it('isSubscriptionPrimaryUser 准确识别纯订阅、混合与按量用户画像', () => {
    const sub = [{ id: 1, group_id: 1, status: 'active' }] as unknown as UserSubscription[]
    // No active subscriptions -> always false
    expect(isSubscriptionPrimaryUser({ balance: 0, activeSubscriptions: [], stats: null })).toBe(false)
    // Zero cash balance with active subscription -> true
    expect(isSubscriptionPrimaryUser({ balance: 0, activeSubscriptions: sub, stats: null })).toBe(true)
    // Subscription-only site -> true even with cash balance
    expect(isSubscriptionPrimaryUser({ balance: 100, activeSubscriptions: sub, stats: null, billingMode: 'subscription_only' })).toBe(true)
    // Hybrid user with meaningful cash balance & cash spend -> false (keeps Balance-first + Hybrid ledger)
    expect(
      isSubscriptionPrimaryUser({
        balance: 45,
        activeSubscriptions: sub,
        stats: { recent_7d_balance_actual_cost: 21, recent_7d_subscription_actual_cost: 65 } as UserDashboardStats,
        billingMode: 'recharge_and_subscription',
        paymentEnabled: true,
      })
    ).toBe(false)
  })
})
