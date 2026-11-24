import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import { createI18n } from 'vue-i18n'
import HomeView from '../HomeView.vue'

const mockAppStore = reactive({
  siteName: 'Gabugu AI',
  siteLogo: '/logo.svg',
  docUrl: '',
  publicSettingsLoaded: true,
  cachedPublicSettings: {} as Record<string, unknown>,
  fetchPublicSettings: vi.fn(),
})

const mockAuthStore = reactive({
  isAuthenticated: false,
  isAdmin: false,
  checkAuth: vi.fn(),
})

vi.mock('@/stores', () => ({
  useAppStore: () => mockAppStore,
  useAuthStore: () => mockAuthStore,
}))

vi.mock('@/api/billingCampaign', () => ({
  getBillingCampaign: vi.fn().mockResolvedValue({
    campaign: null,
    server_now: '2026-11-23T02:00:00+08:00',
    status: 'disabled',
  }),
}))

const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  messages: { zh: {}, en: {} },
})

describe('HomeView custom home content security', () => {
  beforeEach(() => {
    mockAppStore.cachedPublicSettings = {}
    mockAuthStore.checkAuth.mockReset()
  })

  it('sanitizes custom home HTML with DOMPurify and strips XSS payloads', async () => {
    mockAppStore.cachedPublicSettings = {
      home_content: '<h1>Welcome</h1><script>window.__xss=1</script><img src=x onerror="alert(1)"><iframe src="https://evil.example"></iframe>',
    }

    const wrapper = mount(HomeView, {
      global: {
        plugins: [i18n],
        stubs: {
          RouterLink: true,
          LocaleSwitcher: true,
          BillingCampaignBanner: true,
          WeekendRateBanner: true,
          EarthStage: true,
          HeroPoster: true,
          CompactHome: true,
        },
      },
    })

    await flushPromises()
    expect(wrapper.find('h1').text()).toBe('Welcome')
    expect(wrapper.find('script').exists()).toBe(false)
    expect(wrapper.find('iframe').exists()).toBe(false)
    const img = wrapper.find('img')
    if (img.exists()) {
      expect(img.attributes('onerror')).toBeUndefined()
    }
    wrapper.unmount()
  })

  it('applies sandbox and no-referrer to custom home URL iframes', async () => {
    mockAppStore.cachedPublicSettings = {
      home_content: 'https://landing.example.com/promo',
    }

    const wrapper = mount(HomeView, {
      global: {
        plugins: [i18n],
        stubs: {
          RouterLink: true,
          LocaleSwitcher: true,
          BillingCampaignBanner: true,
          WeekendRateBanner: true,
          EarthStage: true,
          HeroPoster: true,
          CompactHome: true,
        },
      },
    })

    await flushPromises()
    const iframe = wrapper.get('iframe')
    expect(iframe.attributes('src')).toBe('https://landing.example.com/promo')
    expect(iframe.attributes('sandbox')).toContain('allow-scripts')
    expect(iframe.attributes('sandbox')).not.toContain('allow-same-origin')
    expect(iframe.attributes('referrerpolicy')).toBe('no-referrer')
    wrapper.unmount()
  })
})
