import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PlazaGroupSection from '../PlazaGroupSection.vue'
import PlazaModelPricingTable from '../PlazaModelPricingTable.vue'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const campaignState = vi.hoisted(() => ({ available: true, discount: 0.2 }))
vi.mock('@/composables/useBillingCampaign', () => ({
  useBillingCampaign: () => ({
    campaign: { value: campaignState.available ? {} : null },
    phase: { __v_isRef: true, get value() { return campaignState.available ? 'championship-day' : 'unavailable' } },
    discountForGroup: (id: number) => [4, 5, 23, 24].includes(id) && campaignState.available ? campaignState.discount : 1
  })
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ cachedPublicSettings: null })
}))

function ladderModel(tiers: number): PlazaModel {
  const intervals = Array.from({ length: tiers }, (_, i) => ({
    min_tokens: i * 272000,
    max_tokens: i === tiers - 1 ? null : (i + 1) * 272000,
    tier_label: '',
    input_price: 5e-6,
    output_price: 3e-5,
    cache_write_price: null,
    cache_read_price: null,
    per_request_price: null
  }))
  return {
    name: 'gpt-5.6-sol',
    platform: 'openai',
    pricing: {
      billing_mode: 'token',
      input_price: 5e-6,
      output_price: 3e-5,
      cache_write_price: null,
      cache_read_price: null,
      image_input_price: null,
      image_output_price: null,
      per_request_price: null,
      intervals: []
    },
    official_pricing: {
      input_price: 5e-6,
      output_price: 3e-5,
      cache_write_price: null,
      cache_read_price: null,
      intervals
    }
  }
}

function group(overrides: Partial<ModelPlazaGroup> = {}): ModelPlazaGroup {
  return {
    id: 1,
    name: 'g',
    description: '',
    platform: 'openai',
    subscription_type: 'standard',
    rate_multiplier: 1,
    peak_rate_enabled: false,
    peak_start: '',
    peak_end: '',
    peak_rate_multiplier: 1,
    is_exclusive: false,
    image_rate_independent: false,
    image_rate_multiplier: 1,
    video_rate_independent: false,
    video_rate_multiplier: 1,
    long_context_pricing_enabled: true,
    models: [ladderModel(2)],
    ...overrides
  }
}

function mountSection(g: ModelPlazaGroup) {
  return mount(PlazaGroupSection, {
    props: { group: g },
    global: {
      stubs: {
        GroupBadge: true,
        Icon: true,
        PlazaModelPricingTable: true
      }
    }
  })
}

const NOTE = 'modelPlaza.detail.longContextDisabledNote'

describe('PlazaGroupSection 长上下文说明', () => {
  it('discloses configured and free additional charges with their units', () => {
    const wrapper = mountSection(group({ additional_charges: { web_search_per_call: 0.5, search_per_1k: 0, audio_realtime_per_min: 0.05, audio_tts_per_million_chars: 15, audio_stt_per_hour: 0.1 } }))
    const details = wrapper.find('[data-additional-charges]')
    expect(details.text()).toContain('$0.5')
    expect(details.text()).toContain('$0 / 1000')
    expect(details.text()).toContain('$0.05 / min')
    expect(details.text()).toContain('$0.1 / h')
    wrapper.unmount()
  })
  it('passes video-specific rates to the pricing table', () => {
    const wrapper = mountSection(group({ video_rate_independent: true, video_rate_multiplier: 0 }))
    expect(wrapper.findComponent(PlazaModelPricingTable).props()).toMatchObject({
      videoRateIndependent: true,
      videoRateMultiplier: 0
    })
    wrapper.unmount()
  })
  it('分组关闭阶梯且组内有官方阶梯模型时显示说明', () => {
    const wrapper = mountSection(group({ long_context_pricing_enabled: false }))
    expect(wrapper.text()).toContain(NOTE)
  })

  it('分组开启阶梯时不显示', () => {
    const wrapper = mountSection(group({ long_context_pricing_enabled: true }))
    expect(wrapper.text()).not.toContain(NOTE)
  })

  it('分组关闭但没有官方阶梯模型时不显示', () => {
    const wrapper = mountSection(
      group({ long_context_pricing_enabled: false, models: [ladderModel(1)] })
    )
    expect(wrapper.text()).not.toContain(NOTE)
  })

  it('旧后端缺少开关字段时不显示', () => {
    const g = group()
    delete (g as Partial<ModelPlazaGroup>).long_context_pricing_enabled
    const wrapper = mountSection(g)
    expect(wrapper.text()).not.toContain(NOTE)
  })
})

describe('PlazaGroupSection 高峰配置传递', () => {
  it('分组启用高峰时把窗口描述与倍率传给价格表', () => {
    const wrapper = mountSection(
      group({
        subscription_type: 'subscription',
        peak_rate_enabled: true,
        peak_start: '14:00',
        peak_end: '18:00',
        peak_rate_multiplier: 1.5
      })
    )
    const table = wrapper.findComponent(PlazaModelPricingTable)
    // appStore mock 无 server_utc_offset,窗口描述不带时区标注
    expect(table.props('peakWindow')).toBe('14:00-18:00 ×1.5')
    expect(table.props('peakRateMultiplier')).toBe(1.5)
  })

  it('分组未启用高峰时窗口描述为空串', () => {
    const wrapper = mountSection(group())
    expect(wrapper.findComponent(PlazaModelPricingTable).props('peakWindow')).toBe('')
  })
})

it('uses timed campaign pricing only for eligible groups and ignores an unverified backend factor', async () => {
  const w = mountSection(group({ id: 4, token_campaign_discount: 0.2 }))
  expect(w.findComponent(PlazaModelPricingTable).props('tokenCampaignDiscount')).toBe(0.2)
  campaignState.discount = 0.9
  await w.setProps({ group: group({ id: 4 }) })
  expect(w.findComponent(PlazaModelPricingTable).props('tokenCampaignDiscount')).toBe(0.9)
  await w.setProps({ group: group({ id: 1, token_campaign_discount: 0.2 }) })
  expect(w.findComponent(PlazaModelPricingTable).props('tokenCampaignDiscount')).toBe(1)
  campaignState.available = false
  await w.setProps({ group: group({ id: 4, token_campaign_discount: 0.2 }) })
  expect(w.findComponent(PlazaModelPricingTable).props('tokenCampaignDiscount')).toBe(1)
  expect(w.text()).toContain('usageRecordDetails.campaignPriceUnavailable')
  campaignState.available = true
  campaignState.discount = 0.2
  w.unmount()
})