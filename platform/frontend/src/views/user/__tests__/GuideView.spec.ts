import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick, reactive } from 'vue'
import type { PublicSettings } from '@/types'
import zh from '@/i18n/locales/zh/guide'
import en from '@/i18n/locales/en/guide'
import GuideView from '../GuideView.vue'

const state = vi.hoisted(() => ({ app: {} as { cachedPublicSettings: Partial<PublicSettings> | null }, auth: {} as { isSimpleMode: boolean }, copy: vi.fn(), locale: 'zh' }))
vi.mock('@/stores', () => ({ useAppStore: () => state.app, useAuthStore: () => state.auth }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard: state.copy }) }))
// The production Vite build enables i18n JIT. Vitest uses the runtime-only alias;
// use real copy through a lookup here, with message compilation tested separately.
vi.mock('vue-i18n', async importOriginal => ({ ...await importOriginal<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => {
  const messages = state.locale === 'zh' ? zh : en
  const value = key.split('.').reduce<unknown>((node, segment) => (node as Record<string, unknown>)?.[segment], messages)
  return typeof value === 'string' ? value : key
} }) }))

function render(locale = 'zh') {
  state.locale = locale
  return mount(GuideView, {
    global: {
      stubs: {
        AppLayout: { props: ['autoStartTour'], template: '<div><slot /></div>' },
        GuideMedia: true,
        RouterLink: { props: ['to'], template: '<a :href="typeof to === \'string\' ? to : to.path + (to.hash || \'\')"><slot /></a>' }
      }
    }
  })
}

beforeEach(() => {
  state.app = reactive({ cachedPublicSettings: {} })
  state.auth = reactive({ isSimpleMode: false })
  state.copy.mockReset()
})

describe('beginner guide', () => {
  it('defaults to desktop Codex and preserves each tool’s actual workflow across systems', async () => {
    const wrapper = render()
    expect(wrapper.get('[data-testid="client-codex"]').attributes('aria-pressed')).toBe('true')
    for (const os of ['windows', 'macos']) {
      await wrapper.get(`[data-testid="os-${os}"]`).trigger('click')
      for (const client of ['claude', 'codex']) {
        await wrapper.get(`[data-testid="client-${client}"]`).trigger('click')
        if (client === 'codex') {
          expect(wrapper.find('[data-testid="launch-command"]').exists()).toBe(false)
          expect(wrapper.find('[data-testid="copy-command"]').exists()).toBe(false)
          expect(wrapper.get('[data-testid="codex-desktop-download"]').attributes('href')).toBe('https://chatgpt.com/download/')
          expect(wrapper.get('[data-testid="step-launch"]').text()).toContain('新建聊天')
          expect(wrapper.findAll('.guide-step')).toHaveLength(6)
          expect(wrapper.get('.guide-path code').text()).toBe(os === 'windows' ? '%USERPROFILE%\\.codex' : '~/.codex')
          continue
        }
        const command = wrapper.get('[data-testid="launch-command"]').text()
        expect(command.endsWith(client)).toBe(true)
        expect(command.includes('Set-Location')).toBe(os === 'windows')
        expect(command.includes('mkdir -p')).toBe(os === 'macos')
        await wrapper.get('[data-testid="copy-command"]').trigger('click')
        expect(state.copy).toHaveBeenLastCalledWith(command)
      }
    }
    wrapper.unmount()
  })
  it('changes the actual configuration step to manual setup when CCS import is hidden', () => {
    state.app.cachedPublicSettings = { hide_ccs_import_button: true }
    const wrapper = render()
    const connect = wrapper.get('[data-testid="step-connect"]')
    expect(connect.find('[data-testid="manual-route"]').exists()).toBe(true)
    expect(connect.find('[data-testid="import-route"]').exists()).toBe(false)
    expect(connect.text()).toContain('使用密钥')
    expect(connect.text()).not.toContain('导入到 CCS')
    expect(wrapper.get('[data-testid="step-installSwitch"]').text()).toContain('可以跳过安装 CC Switch')
    wrapper.unmount()
  })
  it('omits unavailable links and usage instructions in simple mode', () => {
    state.auth.isSimpleMode = true
    state.app.cachedPublicSettings = { channel_monitor_enabled: false }
    const wrapper = render()
    for (const path of ['/usage', '/purchase', '/subscriptions', '/available-channels', '/monitor']) {
      expect(wrapper.find(`a[href="${path}"]`).exists()).toBe(false)
    }
    expect(wrapper.find('a[href="/keys"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="step-verify"]').text()).toContain(zh.guide.steps.verify.simpleResult)
    expect(wrapper.get('[data-testid="step-verify"] .guide-step-content > p').text()).toBe(zh.guide.steps.verify.simpleBody)
    wrapper.unmount()
  })
  it('updates the tutorial when public settings arrive after the page renders', async () => {
    state.app.cachedPublicSettings = null
    const wrapper = render()
    expect(wrapper.find('[data-testid="step-connect"]').exists()).toBe(true)
    state.app.cachedPublicSettings = { hide_ccs_import_button: true, available_channels_enabled: false, channel_monitor_enabled: false, payment_enabled: false, subscription_enabled: false }
    await nextTick()
    expect(wrapper.find('[data-testid="manual-route"]').exists()).toBe(true)
    expect(wrapper.find('guide-media-stub[src="/guide/keys-actions-focused.webp"]').exists()).toBe(false)
    for (const path of ['/available-channels', '/monitor', '/purchase', '/subscriptions']) expect(wrapper.find(`a[href="${path}"]`).exists()).toBe(false)
    wrapper.unmount()
  })
  it('renders all required chapters and a working English prompt', async () => {
    const wrapper = render('en')
    for (const chapter of ['basics', 'setup', 'resources', 'faq']) expect(wrapper.get(`#${chapter}`).exists()).toBe(true)
    expect(wrapper.findAll('.guide-step')).toHaveLength(6)
    expect(wrapper.findAll('.guide-faq details')).toHaveLength(8)
    await wrapper.get('[data-testid="copy-prompt"]').trigger('click')
    expect(state.copy).toHaveBeenCalledWith(en.guide.steps.verify.prompt)
    expect(wrapper.text()).not.toContain('guide.steps.')
    wrapper.unmount()
  })
  it('pairs the Codex route with an OpenAI-specific image and desktop-only instructions', () => {
    const wrapper = render()
    const connect = wrapper.get('[data-testid="step-connect"]')
    expect(connect.text()).toContain('OpenAI')
    expect(connect.find('guide-media-stub[src="/guide/keys-create-codex-detail.webp"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('Codex CLI')
    expect(wrapper.get('[data-testid="step-launch"]').text()).not.toContain('PowerShell')
    wrapper.unmount()
  })
  it('surfaces Model Plaza when enabled and renders protocol quick-reference endpoints', () => {
    state.app.cachedPublicSettings = { model_plaza_enabled: true, available_channels_enabled: false, api_base_url: 'https://gabugu.com' }
    const wrapper = render()
    expect(wrapper.find('a[href="/model-plaza?embedded=1"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/available-channels"]').exists()).toBe(false)
    const refEl = wrapper.get('[data-testid="guide-protocol-reference"]')
    const endpoints = refEl.findAll('code').map((el) => el.text())
    expect(endpoints).toEqual([
      'https://gabugu.com/v1',
      'https://gabugu.com',
      'https://gabugu.com',
    ])
    expect(refEl.text()).toContain('api_version=v1beta')
    wrapper.unmount()
  })
})
