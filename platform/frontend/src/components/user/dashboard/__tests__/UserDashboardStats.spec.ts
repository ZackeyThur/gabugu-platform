import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import type { UserDashboardStats as Stats, PlatformDashboardStats } from '@/api/usage'
import type { PlatformQuotaItem, UserSubscription } from '@/types'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, params?: object) => params ? `${key}:${JSON.stringify(params)}` : key }) }))
import UserDashboardStats from '../UserDashboardStats.vue'
enableAutoUnmount(afterEach)
const base = { total_api_keys: 1, active_api_keys: 1, total_requests: 3, total_input_tokens: 0, total_output_tokens: 0, total_cache_creation_tokens: 0, total_cache_read_tokens: 0, total_tokens: 1234, total_cost: 2, total_actual_cost: 1, today_requests: 2, today_input_tokens: 0, today_output_tokens: 0, today_cache_creation_tokens: 0, today_cache_read_tokens: 0, today_tokens: 100, today_cost: 0.01, today_actual_cost: 0.000123, average_duration_ms: 1000, rpm: 1, tpm: 10, by_platform: [] } satisfies Stats
const usage = (platform: string, cost: number): PlatformDashboardStats => ({ platform, total_requests: 3, total_tokens: 1234, total_actual_cost: cost, today_requests: 2, today_tokens: 100, today_actual_cost: 0 })
const quota = (platform: string, daily: number | null = null, used = 0): PlatformQuotaItem => ({ platform, daily_limit_usd: daily, weekly_limit_usd: null, monthly_limit_usd: null, daily_usage_usd: used, weekly_usage_usd: 0, monthly_usage_usd: 0 }) as PlatformQuotaItem
function render(over: Partial<InstanceType<typeof UserDashboardStats>['$props']> = {}) {
  return mount(UserDashboardStats, { props: { stats: { ...base, total_actual_cost: 0, today_actual_cost: 0 }, balance: 10, isSimple: false, ...over }, global: { stubs: { Icon: true, PlatformIcon: true, teleport: true, 'router-link': { props: ['to'], template: '<a :href="to"><slot /></a>' } } } })
}
const platforms = (w: ReturnType<typeof render>) => w.findAll('[data-testid="platform-card"]').map(c => c.attributes('data-platform'))
describe('费用总览摘要与平台', () => {
  it('顶部仅余额和今日费用两卡，支付开关和简易模式生效', async () => {
    const w = render()
    expect(w.findAll('.overview-account-card')).toHaveLength(2)
    expect(w.find('a[href="/purchase"]').exists()).toBe(true)
    await w.setProps({ paymentEnabled: false })
    expect(w.find('a[href="/purchase"]').exists()).toBe(false)
    expect(w.find('a[href="/redeem"]').exists()).toBe(true)
    await w.setProps({ isSimple: true })
    expect(w.findAll('.overview-account-card')).toHaveLength(1)
    expect(w.find('.overview-platform-section').exists()).toBe(false)
  })
  it('余额失败不显示伪零，错误独立且可重试', async () => {
    const w = render({ balance: null, balanceError: true, stats: base })
    const cards = w.findAll('.overview-account-card')
    expect(cards[0].find('.overview-amount').exists()).toBe(false)
    expect(cards[0].text()).toContain('balanceError')
    expect(cards[1].find('.overview-amount').text()).toBe('<$0.01')
    expect(cards[1].find('.overview-amount').attributes('title')).toBe('$0.000123')
    await cards[0].get('button').trigger('click')
    expect(w.emitted('retryBalance')).toHaveLength(1)
  })
  it('固定排序、平台融合、有限额无用量与其他费用补差', () => {
    const w = render({ stats: { ...base, by_platform: [usage('kimi', .1), usage('grok', .2), usage('anthropic', .3)] }, platformQuotas: [quota('openai', 10), quota('grok', 20), quota('gemini')] })
    expect(platforms(w)).toEqual(['anthropic', 'openai', 'grok', 'kimi', '__other__'])
    expect(w.text()).toContain('dashboard.platformCount:{"count":4}')
    expect(w.get('[data-platform="__other__"]').text()).toContain('$0.40')
  })
  it('零额度禁用、75%临界及100%耗尽', () => {
    const w = render({ platformQuotas: [quota('openai', 0), quota('gemini', 10, 7.5), quota('grok', 10, 10)] })
    expect(w.get('[data-platform="openai"] [data-state]').attributes('data-state')).toBe('disabled')
    expect(w.get('[data-platform="gemini"] [data-state]').attributes('data-state')).toBe('near')
    expect(w.get('[data-platform="grok"] [data-state]').attributes('data-state')).toBe('exhausted')
  })
  it('平台抽屉显示请求Token和额度精确值', async () => {
    const w = render({ stats: { ...base, total_actual_cost: .3, by_platform: [usage('openai', .3)] }, platformQuotas: [quota('openai', 10, 7.5)] })
    await w.get('[data-platform="openai"] button').trigger('click')
    expect(w.get('[role="dialog"]').text()).toContain('1,234')
    expect(w.get('[role="dialog"]').text()).toContain('$7.5000 / $10.0000')
    expect(w.get('progress').attributes('value')).toBe('7.5')
  })
  it('费用明细保留实扣、标准计费和订阅费用解释', async () => {
    const w = render({ stats: base, subscriptionEnabled: false })
    await w.findAll('.overview-account-card')[1].get('button').trigger('click')
    const dialog = w.get('[role="dialog"]')
    expect(dialog.text()).toContain('$0.000123')
    expect(dialog.text()).toContain('$0.0100')
    expect(dialog.text()).toContain('1,234')
    expect(dialog.text()).toContain('subscriptionExplanation')
  })
  it('其他费用保留微小补差并可在抽屉查看数据库精度', async () => {
    const w = render({ stats: { ...base, total_actual_cost: .00005, today_actual_cost: .0000000001 } })
    expect(platforms(w)).toEqual(['__other__'])
    expect(w.get('[data-platform="__other__"] .overview-platform-amount').attributes('title')).toBe('$0.0000000001')
    await w.get('[data-platform="__other__"] button').trigger('click')
    expect(w.get('[role="dialog"]').text()).toContain('$0.00005')
    expect(w.get('[role="dialog"]').text()).toContain('$0.0000000001')
  })
  it('TypeSafe / Jev在未知平台前，未知平台费用不丢失', () => {
    const w = render({ stats: { ...base, today_actual_cost: 0, total_actual_cost: .5, by_platform: [usage('unknown-provider', .3), usage('typesafe', .2)] } })
    expect(platforms(w)).toEqual(['typesafe', 'unknown-provider'])
    expect(w.get('[data-platform="typesafe"]').text()).toContain('TypeSafe / Jev')
    expect(w.get('[data-platform="unknown-provider"]').text()).toContain('$0.30')
  })
  it('无使用无额度为空态，额度错误不掩盖费用', () => {
    expect(platforms(render({ platformQuotas: [quota('openai')] }))).toEqual([])
    const w = render({ stats: { ...base, by_platform: [usage('typesafe', 1)] }, quotasError: true })
    expect(w.text()).toContain('TypeSafe / Jev')
    expect(w.text()).toContain('quotasError')
    expect(w.get('[data-platform="typesafe"]').text()).toContain('$1.00')
  })

  it('现金续航严格排除订阅额度消耗，且支持快捷满赠胶囊与 <1 RPM 活跃指示', () => {
    // Pure subscription burn must NOT show a critical cash runway warning
    const subOnly = render({
      balance: 5,
      stats: {
        ...base,
        recent_7d_balance_actual_cost: 0,
        recent_7d_subscription_actual_cost: 700,
        today_actual_cost: 100,
        rpm: 0,
        rpm_exact: 0.4,
        recent_5m_requests: 2,
      },
    })
    expect(subOnly.find('.overview-runway-pill').exists()).toBe(false)
    expect(subOnly.get('.overview-live-label').text()).toContain('<1 RPM')
    expect(subOnly.get('.overview-live-label').classes()).toContain('is-live')

    // Cash burn shows accurate runway + quick recharge bonus pills
    const cashUser = render({
      balance: 20,
      stats: {
        ...base,
        recent_7d_balance_actual_cost: 70,
        today_actual_cost: 12,
        yesterday_same_time_actual_cost: 10,
      },
      checkoutInfo: {
        methods: { alipay: { single_min: 10, single_max: 1000, daily_limit: 0, available: true, fee_rate: 0, currency: 'CNY' } },
        global_min: 10,
        global_max: 1000,
        plans: [],
        balance_disabled: false,
        balance_recharge_multiplier: 1,
        recharge_fee_rate: 0,
        help_text: '',
        help_image_url: '',
        stripe_publishable_key: '',
        recharge_bonus_mode: 'bonus',
        recharge_bonus_tiers: [{ min_amount: 100, bonus_percent: 10 }, { min_amount: 300, bonus_percent: 15 }],
      },
    })
    expect(cashUser.get('.overview-runway-pill').attributes('data-status')).toBe('critical')
    expect(cashUser.findAll('.overview-quick-tile')).toHaveLength(2)
    expect(cashUser.get('.overview-delta-badge').attributes('data-direction')).toBe('up')
  })

  it('平台卡展示累计占比，抽屉提供按平台直钻 /usage 入口', async () => {
    const w = render({
      stats: { ...base, total_actual_cost: 10, by_platform: [usage('openai', 6), usage('anthropic', 4)] },
    })
    expect(w.get('[data-platform="openai"] .overview-platform-share').text()).toBe('60.0%')
    await w.get('[data-platform="openai"] button').trigger('click')
    const drillLink = w.get('[role="dialog"] .overview-drawer-drill')
    expect(drillLink.text()).toContain('viewPlatformUsage')
  })

  it('支持 7 日迷你走势键盘擦洗、点击下钻以及快捷充值档位悬停续航预演', async () => {
    const w = render({
      balance: 20,
      today: '2026-11-24',
      stats: {
        ...base,
        total_actual_cost: 10,
        recent_7d_balance_actual_cost: 70,
        by_platform: [usage('anthropic', 6), usage('openai', 4)],
      },
      baseline7d: [
        { date: '2026-11-22', actual_cost: 8, cost: 12, requests: 10, input_tokens: 0, output_tokens: 0, cache_creation_tokens: 0, cache_read_tokens: 0, total_tokens: 100 },
        { date: '2026-11-23', actual_cost: 16.5, cost: 24, requests: 20, input_tokens: 0, output_tokens: 0, cache_creation_tokens: 0, cache_read_tokens: 0, total_tokens: 200 },
        { date: '2026-11-24', actual_cost: 13.4, cost: 19, requests: 15, input_tokens: 0, output_tokens: 0, cache_creation_tokens: 0, cache_read_tokens: 0, total_tokens: 150 },
      ],
      checkoutInfo: {
        methods: { alipay: { single_min: 10, single_max: 1000, daily_limit: 0, available: true, fee_rate: 0, currency: 'CNY' } },
        global_min: 10,
        global_max: 1000,
        plans: [],
        balance_disabled: false,
        balance_recharge_multiplier: 1,
        recharge_fee_rate: 0,
        help_text: '',
        help_image_url: '',
        stripe_publishable_key: '',
        recharge_bonus_mode: 'bonus',
        recharge_bonus_tiers: [{ min_amount: 100, bonus_percent: 10 }],
      },
    })

    // Multi-brand 3px share spectrum renders 2 segments
    expect(w.findAll('.overview-platform-spectrum-seg')).toHaveLength(2)

    // Keyboard scrub on 7d sparkline (ArrowLeft -> yesterday 11/23 $16.50) + Enter emits drillDay
    const stage = w.get('.overview-sparkline-stage')
    await stage.trigger('keydown', { key: 'ArrowLeft' })
    expect(w.get('.overview-spark-callout').text()).toContain('$16.50')
    await stage.trigger('keydown', { key: 'Enter' })
    expect(w.emitted('drillDay')).toEqual([['2026-11-23']])

    // Home jumps to first day of 7d window (2026-11-18)
    await stage.trigger('keydown', { key: 'Home' })
    expect(w.get('.overview-spark-callout').text()).toContain('11/18')

    // Hovering quick-recharge tier previews projected balance + runway
    const quickTile = w.get('.overview-quick-tile')
    await quickTile.trigger('mouseenter')
    expect(w.get('.overview-card-note').classes()).toContain('is-preview')
    expect(w.get('.overview-card-note').text()).toContain('$130.00')
    expect(w.get('.overview-card-note').text()).toContain('13')
  })

  it('纯订阅用户默认展示订阅额度总控表盘，支持多套餐切换且不重复渲染底部订阅条', async () => {
    const now = Date.now()
    const subs = [
      {
        id: 201,
        user_id: 1001,
        group_id: 1,
        status: 'active',
        starts_at: new Date(now - 8 * 86_400_000).toISOString(),
        expires_at: new Date(now + 22 * 86_400_000).toISOString(),
        daily_usage_usd: 8.2,
        weekly_usage_usd: 61.4,
        monthly_usage_usd: 213.5,
        daily_window_start: new Date(now - 10 * 3_600_000).toISOString(),
        weekly_window_start: new Date(now - 3 * 86_400_000).toISOString(),
        monthly_window_start: new Date(now - 12 * 86_400_000).toISOString(),
        group: { id: 1, name: 'Claude 极速专享月卡', platform: 'anthropic', rate_multiplier: 0.65, daily_limit_usd: 25, weekly_limit_usd: 140, monthly_limit_usd: 500 },
      },
      {
        id: 202,
        user_id: 1001,
        group_id: 2,
        status: 'active',
        starts_at: new Date(now - 25 * 86_400_000).toISOString(),
        expires_at: new Date(now + 5 * 86_400_000).toISOString(),
        daily_usage_usd: 19.2,
        weekly_usage_usd: 106.4,
        monthly_usage_usd: 382,
        daily_window_start: new Date(now - 16 * 3_600_000).toISOString(),
        weekly_window_start: new Date(now - 5 * 86_400_000).toISOString(),
        monthly_window_start: new Date(now - 25 * 86_400_000).toISOString(),
        group: { id: 2, name: 'OpenAI Codex Pro 月卡', platform: 'openai', rate_multiplier: 0.7, daily_limit_usd: 25, weekly_limit_usd: 140, monthly_limit_usd: 500 },
      },
    ] as unknown as UserSubscription[]

    const w = render({
      balance: 0,
      subscriptionEnabled: true,
      activeSubscriptions: subs,
      stats: {
        ...base,
        today_actual_cost: 13.4,
        today_balance_actual_cost: 0,
        today_subscription_actual_cost: 13.4,
        recent_7d_balance_actual_cost: 0,
        recent_7d_subscription_actual_cost: 86,
      },
    })

    // Left chamber defaults to Subscription Hero & suppresses duplicate below-Row-2 strip
    expect(w.findAll('.overview-account-card')).toHaveLength(2)
    expect(w.find('.overview-account-card.is-subscription-hero').exists()).toBe(true)
    expect(w.find('.overview-sub-strip').exists()).toBe(false)
    expect(w.findAll('.overview-sub-plan-tab')).toHaveLength(2)
    expect(w.findAll('.overview-sub-window-cell')).toHaveLength(3)
    expect(w.get('.overview-account-card.is-subscription-hero .overview-amount').text()).toBe('$16.80')
    expect(w.get('.overview-today-footer').text()).toContain('todayPureSubHint')

    // Switching to second plan updates hero remaining ($5.80) and surfaces Renew primary CTA
    await w.findAll('.overview-sub-plan-tab')[1].trigger('click')
    expect(w.get('.overview-account-card.is-subscription-hero .overview-amount').text()).toBe('$5.80')
    expect(w.get('.overview-balance-cta-cluster .overview-primary-button').text()).toContain('renewSubscription')

    // Switching chamber to Cash Balance reveals the upgraded multi-plan Subscription Ledger Ribbon
    await w.findAll('.overview-chamber-tab')[1].trigger('click')
    expect(w.find('.overview-account-card.is-subscription-hero').exists()).toBe(false)
    expect(w.find('.overview-sub-strip.is-ledger').exists()).toBe(true)
    expect(w.findAll('.overview-sub-ledger-row')).toHaveLength(2)
  })

  it('混合用户默认保留现金余额主控并在下方展示多窗口订阅账本，今日页脚拆分套餐与余额消耗', async () => {
    const now = Date.now()
    const subs = [
      {
        id: 301,
        user_id: 1001,
        group_id: 1,
        status: 'active',
        starts_at: new Date(now - 10 * 86_400_000).toISOString(),
        expires_at: new Date(now + 15 * 86_400_000).toISOString(),
        daily_usage_usd: 10,
        weekly_usage_usd: 40,
        monthly_usage_usd: 120,
        daily_window_start: new Date(now - 6 * 3_600_000).toISOString(),
        group: { id: 1, name: 'Claude 月卡', platform: 'anthropic', daily_limit_usd: 25, weekly_limit_usd: 140, monthly_limit_usd: 500 },
      },
    ] as unknown as UserSubscription[]

    const w = render({
      balance: 45,
      subscriptionEnabled: true,
      activeSubscriptions: subs,
      stats: {
        ...base,
        today_actual_cost: 13.4,
        today_balance_actual_cost: 3.2,
        today_subscription_actual_cost: 10.2,
        recent_7d_balance_actual_cost: 21,
        recent_7d_subscription_actual_cost: 65,
      },
    })

    expect(w.find('.overview-account-card.is-subscription-hero').exists()).toBe(false)
    expect(w.find('.overview-sub-strip.is-ledger').exists()).toBe(true)
    expect(w.findAll('.overview-sub-ledger-win')).toHaveLength(3)
    expect(w.get('.overview-today-footer').text()).toContain('todaySplitSubAndCash')

    // One-click switch to Subscription Hero hides the duplicate ledger automatically
    await w.findAll('.overview-chamber-tab')[1].trigger('click')
    expect(w.find('.overview-account-card.is-subscription-hero').exists()).toBe(true)
    expect(w.find('.overview-sub-strip').exists()).toBe(false)
  })
})
