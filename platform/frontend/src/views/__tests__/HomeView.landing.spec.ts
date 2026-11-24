import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { brotliDecompressSync } from 'node:zlib'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick } from 'vue'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'

import HomeView from '../HomeView.vue'

const { appStore, authStore, i18n } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {} as Record<string, unknown>,
    siteName: 'Gabugu',
    siteLogo: '',
    docUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
  },
  authStore: {
    isAuthenticated: false,
    isAdmin: false,
    checkAuth: vi.fn(),
  },
  i18n: { locale: { value: 'zh' } },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key, locale: i18n.locale }),
  }
})

/** Stands in for the component that downloads Three.js and draws the scene. */
const EarthStageStub = defineComponent({
  name: 'EarthStage',
  props: ['surface', 'journey', 'opening', 'after'],
  emits: ['ready', 'failed'],
  methods: { wake() {} },
  template: '<div class="earth-stage-stub" />',
})

function mountLanding(locale = 'zh', settings: Record<string, unknown> = {}) {
  i18n.locale.value = locale
  appStore.cachedPublicSettings = {
    site_name: 'Gabugu',
    model_plaza_enabled: true,
    model_plaza_require_auth: false,
    ...settings,
  }
  return mount(HomeView, {
    attachTo: document.body,
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
        LocaleSwitcher: { template: '<div />' },
        EarthStage: EarthStageStub,
      },
    },
  })
}

/** Every link of the page, in the order the page holds them. */
const destinations = (wrapper: ReturnType<typeof mountLanding>) =>
  wrapper.findAllComponents(RouterLinkStub).map((link) => link.props('to'))

/** A browser that can draw the scene: WebGL 2 with float render targets and large textures. */
function fakeWebGL() {
  const context = {
    MAX_TEXTURE_SIZE: 0x0d33,
    getExtension: () => ({ loseContext: vi.fn() }),
    getParameter: () => 8192,
  }
  vi.stubGlobal('WebGL2RenderingContext', class {})
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(context as unknown as RenderingContext)
}

/** Code points a WOFF2 font can draw, read from the format 4 subtables of its cmap table. */
function woff2CodePoints(file: string) {
  const font = readFileSync(file)
  const tableCount = font.readUInt16BE(12)
  const compressedSize = font.readUInt32BE(20)
  let offset = 48
  const readBase128 = () => {
    let value = 0
    for (;;) {
      const byte = font[offset++]
      value = value * 128 + (byte & 0x7f)
      if (!(byte & 0x80)) return value
    }
  }
  let position = 0
  let cmapStart = -1
  for (let index = 0; index < tableCount; index++) {
    const flags = font[offset++]
    const tag = flags & 0x3f
    if (tag === 0x3f) offset += 4
    const length = readBase128()
    // glyf (10) and loca (11) are stored transformed at version 0; every other table at any other version.
    const transformed = tag === 10 || tag === 11 ? flags >> 6 === 0 : flags >> 6 !== 0
    const stored = transformed ? readBase128() : length
    if (tag === 0) cmapStart = position
    position += stored
  }
  const tables = brotliDecompressSync(font.subarray(offset, offset + compressedSize))
  const codePoints = new Set<number>()
  for (let record = 0; record < tables.readUInt16BE(cmapStart + 2); record++) {
    const subtable = cmapStart + tables.readUInt32BE(cmapStart + 8 + record * 8)
    if (tables.readUInt16BE(subtable) !== 4) continue
    const segments = tables.readUInt16BE(subtable + 6) / 2
    const ends = subtable + 14
    const starts = ends + segments * 2 + 2
    const deltas = starts + segments * 2
    const rangeOffsets = deltas + segments * 2
    for (let segment = 0; segment < segments; segment++) {
      const start = tables.readUInt16BE(starts + segment * 2)
      const end = tables.readUInt16BE(ends + segment * 2)
      const delta = tables.readUInt16BE(deltas + segment * 2)
      const rangeOffset = tables.readUInt16BE(rangeOffsets + segment * 2)
      for (let code = start; code <= end && code !== 0xffff; code++) {
        const listed = rangeOffset
          ? tables.readUInt16BE(rangeOffsets + segment * 2 + rangeOffset + (code - start) * 2)
          : code
        if (listed && (listed + delta) % 0x10000) codePoints.add(code)
      }
    }
  }
  if (!codePoints.size) throw new Error(`No format 4 cmap subtable in ${file}`)
  return codePoints
}

describe('HomeView default Landing', () => {
  beforeEach(() => {
    authStore.isAuthenticated = false
    authStore.isAdmin = false
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false } as MediaQueryList)
  })

  afterEach(() => {
    window.history.replaceState(null, '', '/')
    window.sessionStorage.clear()
    document.body.innerHTML = ''
    vi.unstubAllGlobals()
  })

  it('holds every Chinese character of the page in the Landing font subset', () => {
    const styles = readFileSync(resolve(process.cwd(), 'src/components/landing/landing.css'), 'utf8')
    const subset = styles.match(/\/brand\/noto-sans-sc-landing[\w.-]*\.woff2/)?.[0]
    expect(subset).toBeDefined()
    const held = woff2CodePoints(resolve(process.cwd(), `public${subset}`))
    expect(held.has('龘'.codePointAt(0)!)).toBe(false)

    const wrapper = mountLanding('zh')
    const text = wrapper.text()
    // A character outside the subset still renders, from the workspace font, at the cost of a far larger download.
    const missing = [...new Set(text)].filter((character) => {
      const code = character.codePointAt(0)!
      return code >= 0x2e80 && !held.has(code)
    })

    expect(text).toContain('让想法')
    expect(missing).toEqual([])
    wrapper.unmount()
  })

  it('is one screen for each part of the story, in order, with a way to any of them', () => {
    const wrapper = mountLanding()
    const names = wrapper.findAll('.landing-screen').map((screen) => screen.attributes('data-screen'))

    expect(names).toEqual(['hero', 'models', 'connect', 'tools', 'pricing', 'control', 'faq', 'start'])
    // The screens a link can lead to carry an id and take the focus.
    expect(wrapper.findAll('.landing-screen[id]').map((screen) => screen.attributes('id')))
      .toEqual(['models', 'connect', 'tools', 'pricing', 'control', 'faq'])
    // Each screen tells the picture where its words are (see earth/story.ts).
    expect(wrapper.findAll('.landing-screen').map((screen) => screen.attributes('data-earth')))
      .toEqual(['below', 'beside', 'beside', 'beside', 'below', 'beside', 'beside', 'below'])

    const dots = wrapper.findAll('.screen-dots button')
    expect(dots).toHaveLength(8)
    // The page opens on its first screen, and every dot is named.
    expect(dots.map((dot) => dot.attributes('aria-current'))).toEqual(['true', ...Array(7).fill(undefined)])
    expect(dots.map((dot) => dot.text())).toEqual(['首屏', '模型', '接入', '工具', '计费', '掌控', '常见问题', '开始使用'])
    expect(wrapper.get('.screen-count').text()).toBe('01 / 08')
    wrapper.unmount()
  })

  it('takes a header link to its screen itself instead of handing it to the router', () => {
    const wrapper = mountLanding()
    const link = wrapper.get('.landing-nav-links a[href="#pricing"]').element
    const click = new MouseEvent('click', { bubbles: true, cancelable: true })
    link.dispatchEvent(click)

    // Left alone, the click would change the route hash and the router would scroll back to the top.
    expect(click.defaultPrevented).toBe(true)
    expect(window.location.hash).toBe('#pricing')
    expect(document.activeElement).toBe(wrapper.get('#pricing').element)
    wrapper.unmount()
  })

  it('takes the reader along when a dot is chosen, and says where they are', () => {
    // Scrolling alone moves nothing the keyboard or a screen reader follows.
    const wrapper = mountLanding()
    const dot = wrapper.findAll('.screen-dots button')[6]
    dot.element.click()

    expect(document.activeElement).toBe(wrapper.get('#faq').element)
    expect(document.activeElement?.getAttribute('aria-labelledby')).toBe('landing-faq-title')
    expect(window.location.hash).toBe('#faq')
    wrapper.unmount()
  })

  it('offers the integration screen as the documentation where no documentation site is set', () => {
    const plain = mountLanding()
    expect(plain.get('[data-testid="home-docs-link"]').attributes('href')).toBe('#connect')
    plain.unmount()

    const documented = mountLanding('zh', { doc_url: 'https://docs.example.com/guide' })
    const link = documented.get('[data-testid="home-docs-link"]')
    expect(link.attributes('href')).toBe('https://docs.example.com/guide')
    expect(link.attributes('target')).toBe('_blank')
    documented.unmount()
  })

  it('lists the model families the site serves, each with the protocol its groups are called by', () => {
    const wrapper = mountLanding()
    const rows = wrapper.findAll('#models .provider-example')

    expect(rows.map((row) => row.get('.provider-model').text())).toEqual(['GPT', 'Claude', 'Gemini', 'Grok', 'DeepSeek'])
    // DeepSeek is named by its model alone.
    expect(rows.flatMap((row) => row.find('.provider-company').exists() ? [row.get('.provider-company').text()] : []))
      .toEqual(['OpenAI', 'Anthropic', 'Google', 'xAI'])
    // Grok and DeepSeek are served by the OpenAI-compatible endpoints, so their rows open that example.
    expect(rows.map((row) => row.get('.provider-protocol').text()))
      .toEqual(['OpenAI 协议', 'Anthropic 协议', 'Gemini 协议', 'OpenAI 协议', 'OpenAI 协议'])
    wrapper.unmount()
  })

  it('opens the request example of the model row that was chosen', async () => {
    const wrapper = mountLanding()
    await wrapper.findAll('#models .provider-example')[1].trigger('click')

    expect(wrapper.get('#protocol-tab-anthropic').attributes('aria-selected')).toBe('true')
    expect(wrapper.get('[role="tabpanel"]').text()).toContain('from anthropic import Anthropic')
    expect(document.activeElement).toBe(wrapper.get('#protocol-tab-anthropic').element)
    wrapper.unmount()
  })

  it('points each official Python SDK at this site, in the shape that SDK sends requests to', async () => {
    const wrapper = mountLanding()
    const origin = window.location.origin
    const pane = (id: string) => wrapper.get(`#protocol-panel-${id} code`).element.textContent ?? ''

    // OpenAI's SDK appends /v1/chat/completions to its base URL...
    expect(pane('openai')).toContain(`base_url="${origin}/v1"`)
    // ...Anthropic's tells the version itself and takes the address without it...
    expect(pane('anthropic')).toContain(`base_url="${origin}"`)
    // ...and Google's needs to be told which version the gateway serves.
    expect(pane('gemini')).toContain(`"base_url": "${origin}"`)
    expect(pane('gemini')).toContain('"api_version": "v1beta"')

    // Every example asks for the key in the same place.
    for (const id of ['openai', 'anthropic', 'gemini']) expect(pane(id)).toContain('YOUR_API_KEY')
    wrapper.unmount()
  })

  it('copies the example as the plain command, and says so when it cannot', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const wrapper = mountLanding()
    await wrapper.get('.code-copy').trigger('click')

    expect(writeText).toHaveBeenCalledWith(wrapper.get('.console-pane.active code').element.textContent)
    expect(writeText.mock.calls[0][0]).toMatch(/^from openai import OpenAI/)
    expect(wrapper.get('.code-copy').text()).toContain('已复制')
    wrapper.unmount()

    // A browser that denies the clipboard gets a message instead of silence.
    vi.stubGlobal('navigator', { clipboard: { writeText: vi.fn().mockRejectedValue(new Error('denied')) } })
    const blocked = mountLanding()
    await blocked.get('.code-copy').trigger('click')
    await flushPromises()
    expect(blocked.get('.copy-error').text()).toBe('暂时无法复制，请选中代码手动复制。')
    blocked.unmount()
  })

  it('keeps all three examples in the panel and offers only the chosen one', async () => {
    const wrapper = mountLanding()
    // Laid out together, the panel is as tall as the longest example and does not jump when the protocol changes.
    expect(wrapper.findAll('.console-pane')).toHaveLength(3)
    expect(wrapper.findAll('.console-pane[role="tabpanel"]')).toHaveLength(1)
    expect(wrapper.get('.console-pane[role="tabpanel"]').attributes('id')).toBe('protocol-panel-openai')

    await wrapper.get('#protocol-tab-gemini').trigger('keydown', { key: 'ArrowLeft' })
    expect(wrapper.get('#protocol-tab-anthropic').attributes('aria-selected')).toBe('true')
    await wrapper.get('#protocol-tab-anthropic').trigger('keydown', { key: 'End' })
    expect(wrapper.get('#protocol-tab-gemini').attributes('aria-selected')).toBe('true')
    expect(wrapper.get('[role="tabpanel"]').text()).toContain('google')
    wrapper.unmount()
  })

  it('explains how a call is priced, term by term', () => {
    const wrapper = mountLanding()
    expect(wrapper.get('.formula-cost').text()).toBe('费用')
    expect(wrapper.findAll('.formula-part .formula-term').map((term) => term.text())).toEqual(['基础单价', '分组倍率', '用量'])
    expect(wrapper.findAll('.formula-sign').map((sign) => sign.text())).toEqual(['=', '×', '×'])
    // Each term of the line has its explanation under it.
    expect(wrapper.findAll('.formula-note dt').map((term) => term.text())).toEqual(['基础单价', '分组倍率', '用量'])
    expect(wrapper.findAll('.formula-note dd')).toHaveLength(3)
    wrapper.unmount()
  })

  it('names the ways to pay only as far as the site has them switched on', () => {
    const none = mountLanding('zh', { payment_enabled: false })
    expect(none.find('.pay-list').exists()).toBe(false)
    // Without any way to pay, the question that compares the two is not asked either.
    expect(none.findAll('.faq-questions button')).toHaveLength(3)
    none.unmount()

    const both = mountLanding('zh', { payment_enabled: true })
    expect(both.findAll('#pricing .pay-list li').map((item) => item.text()))
      .toEqual(['余额：充值后按量扣', '订阅：买套餐，有效期内按套餐额度用'])
    expect(both.findAll('.faq-questions button')).toHaveLength(4)
    both.unmount()

    const subscriptionOnly = mountLanding('zh', { payment_enabled: true, payment_balance_disabled: true })
    expect(subscriptionOnly.findAll('#pricing .pay-list li')).toHaveLength(1)
    expect(subscriptionOnly.get('#pricing .pay-list').text()).toContain('订阅')
    expect(subscriptionOnly.findAll('.faq-questions button')).toHaveLength(3)
    subscriptionOnly.unmount()

    const balanceOnly = mountLanding('zh', { payment_enabled: true, subscription_enabled: false })
    expect(balanceOnly.get('#pricing .pay-list').text()).toContain('余额')
    expect(balanceOnly.findAll('.faq-questions button')).toHaveLength(3)
    balanceOnly.unmount()
  })

  it('answers one question at a time and moves between them with the keyboard', async () => {
    const wrapper = mountLanding()
    const tabs = () => wrapper.findAll('.faq-questions button')
    expect(tabs().map((tab) => tab.attributes('aria-selected'))).toEqual(['true', 'false', 'false'])

    await tabs()[0].trigger('keydown', { key: 'ArrowDown' })
    expect(tabs()[1].attributes('aria-selected')).toBe('true')
    expect(wrapper.get('.faq-answers [role="tabpanel"]').text()).toContain('一个 Key 属于一个分组')
    expect(document.activeElement).toBe(tabs()[1].element)

    await tabs()[1].trigger('keydown', { key: 'Home' })
    expect(tabs()[0].attributes('aria-selected')).toBe('true')
    // Every answer is in place, so the screen keeps its height.
    expect(wrapper.findAll('.faq-answer')).toHaveLength(3)
    expect(wrapper.findAll('.faq-answers [role="tabpanel"]')).toHaveLength(1)
    wrapper.unmount()
  })

  it('shows how to hold a key and check its usage, with a sample that says so', () => {
    const wrapper = mountLanding()
    expect(wrapper.get('#control .sample-note').text()).toContain('示例数据')
    expect(wrapper.findAll('.sample-limits li')).toHaveLength(3)
    expect(wrapper.findAll('.sample-record dt').map((term) => term.text())).toEqual(['模型', 'Token', '费用', '耗时'])
    // The usage page takes a key, so it works without signing in; the note says so.
    expect(destinations(wrapper)).toContain('/key-usage')
    expect(wrapper.get('#control .screen-note').text()).toContain('不登录也能用 Key 查额度和用量汇总')
    wrapper.unmount()
  })

  it('sends a visitor to the model catalog, or to where they can reach it', () => {
    const open = mountLanding('zh')
    expect(destinations(open)).toContain('/model-plaza')
    open.unmount()

    // A site that takes new accounts sends a visitor to sign up...
    const membersOnly = mountLanding('zh', { model_plaza_require_auth: true, registration_enabled: true })
    expect(destinations(membersOnly)).not.toContain('/model-plaza')
    expect(destinations(membersOnly)).toContain('/register')
    membersOnly.unmount()

    // ...and one that does not, to sign in and be brought back.
    const closed = mountLanding('zh', { model_plaza_require_auth: true, registration_enabled: false })
    expect(destinations(closed)).toContainEqual({ path: '/login', query: { redirect: '/model-plaza' } })
    closed.unmount()

    // A visitor who is signed in goes straight there.
    authStore.isAuthenticated = true
    const member = mountLanding('zh', { model_plaza_require_auth: true })
    expect(destinations(member)).toContain('/model-plaza')
    member.unmount()

    // Without the catalog there is nothing to offer; the page leads on to the site itself.
    const off = mountLanding('zh', { model_plaza_enabled: false })
    expect(destinations(off)).not.toContain('/model-plaza')
    expect(destinations(off)).not.toContain('/register')
    off.unmount()
  })

  it('names the tools the site offers configurations for, and the guide for a first key', () => {
    const wrapper = mountLanding()
    expect(wrapper.findAll('.tool-names > span').map((name) => name.text()))
      .toEqual(['Claude Code', 'Codex', 'Gemini CLI', 'Grok CLI', 'OpenCode'])
    expect(wrapper.get('#tools .screen-note').text()).toContain('Codex 桌面端和 Claude Code')
    wrapper.unmount()

    // A site that has taken the CC Switch import out of its key dialog does not offer it here either.
    const plain = mountLanding('zh', { hide_ccs_import_button: true })
    expect(plain.get('#tools .tool-steps').text()).not.toContain('CC Switch')
    wrapper.unmount()
    plain.unmount()
  })

  it('links the terms the site has published, and only those', () => {
    const wrapper = mountLanding('zh', {
      login_agreement_documents: [
        { id: 'terms', title: '服务条款', content_md: '第一条' },
        { id: 'draft', title: '未写完的条款', content_md: '   ' }
      ]
    })
    // A document with no text yet gets no link: there would be nothing to read.
    expect(destinations(wrapper)).toEqual(expect.arrayContaining(['/legal/terms']))
    expect(destinations(wrapper).filter((to) => typeof to === 'string' && to.startsWith('/legal/')))
      .toEqual(['/legal/terms'])
    wrapper.unmount()
  })

  it('names the site, its contact and its independence in the keep', () => {
    const wrapper = mountLanding('zh', { site_name: 'Bubu', contact_info: ' hi@example.com ' })
    const keep = wrapper.get('.footer-disclaimer').text()

    expect(keep).toContain('Bubu 是独立的第三方服务')
    expect(keep).toContain('与 OpenAI、Anthropic、Google、xAI、DeepSeek 没有隶属或授权关系')
    expect(keep).toContain('联系方式：hi@example.com')
    // The Chinese page addresses the visitor as the site names itself, not as the default.
    expect(wrapper.get('#connect .section-description').text()).toContain('Bubu')
    wrapper.unmount()
  })

  it('shows the still picture where the live Earth cannot be drawn', async () => {
    // jsdom has no WebGL, which is the case this fallback exists for.
    const wrapper = mountLanding()
    await flushPromises()

    expect(wrapper.findComponent(EarthStageStub).exists()).toBe(false)
    expect(wrapper.get('.gabugu-landing').classes()).not.toContain('staged')
    // The same sunrise as a picture, at the back of the page.
    expect(wrapper.find('.landing-backdrop').exists()).toBe(true)
    expect(wrapper.find('.footer-bottom .footer-credits a').attributes('href'))
      .toBe('https://visibleearth.nasa.gov/collection/1484/blue-marble')
    wrapper.unmount()
  })

  it('puts the live Earth behind the page where it can be drawn', () => {
    fakeWebGL()
    const wrapper = mountLanding()

    expect(wrapper.findComponent(EarthStageStub).exists()).toBe(true)
    // The layout for the live picture, and no still picture under it.
    expect(wrapper.get('.gabugu-landing').classes()).toContain('staged')
    expect(wrapper.find('.landing-backdrop').exists()).toBe(false)
    wrapper.unmount()
  })

  it('keeps to the still picture for a visitor who asked for less motion or less data', () => {
    fakeWebGL()
    vi.spyOn(window, 'matchMedia').mockImplementation((query) => ({ matches: query.includes('prefers-reduced-motion') } as MediaQueryList))
    const calm = mountLanding()
    expect(calm.findComponent(EarthStageStub).exists()).toBe(false)
    expect(calm.find('.landing-backdrop').exists()).toBe(true)
    // Nothing waits in an opening pose either.
    expect(calm.get('.gabugu-landing').classes()).not.toContain('intro-pending')
    calm.unmount()

    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false } as MediaQueryList)
    vi.stubGlobal('navigator', { hardwareConcurrency: 16, connection: { saveData: true } })
    const saving = mountLanding()
    expect(saving.findComponent(EarthStageStub).exists()).toBe(false)
    expect(saving.find('.landing-backdrop').exists()).toBe(true)
    saving.unmount()
  })

  it('gives a scene that failed one lighter second chance, then shows the still picture', async () => {
    fakeWebGL()
    vi.stubGlobal('navigator', { hardwareConcurrency: 16 })
    const wrapper = mountLanding()
    const stage = () => wrapper.findComponent(EarthStageStub)
    expect(stage().props('surface').quality).toBe('high')

    stage().vm.$emit('failed')
    await nextTick()
    expect(stage().props('surface').quality).toBe('low')

    stage().vm.$emit('failed')
    await nextTick()
    expect(stage().exists()).toBe(false)
    expect(wrapper.get('.gabugu-landing').classes()).not.toContain('staged')
    expect(wrapper.find('.landing-backdrop').exists()).toBe(true)
    wrapper.unmount()
  })

  it('reads the headline as one sentence although it is drawn in pieces', () => {
    const chinese = mountLanding('zh')
    expect(chinese.get('.landing-hero h1').attributes('aria-label')).toBe('让想法走得更远')
    // One piece for each character; the lines themselves are hidden from assistive technology.
    expect(chinese.findAll('.hero-piece').map((piece) => piece.text())).toEqual([...'让想法走得更远'])
    expect(chinese.findAll('.hero-line').every((line) => line.attributes('aria-hidden') === 'true')).toBe(true)
    chinese.unmount()

    const english = mountLanding('en')
    expect(english.get('.landing-hero h1').attributes('aria-label')).toBe('Ideas move. Possibilities follow.')
    expect(english.findAll('.hero-piece').map((piece) => piece.text())).toEqual(['Ideas', 'move.', 'Possibilities', 'follow.'])
    english.unmount()
  })

  it('addresses an English visitor in English throughout', () => {
    const wrapper = mountLanding('en')
    expect(wrapper.get('.gabugu-landing').attributes('lang')).toBe('en')
    expect(wrapper.get('#models h2').text()).toBe('Many models. One entry.')
    expect(wrapper.get('.screen-count').text()).toBe('01 / 08')
    expect(wrapper.get('.footer-disclaimer').text()).toContain('independent service')
    expect(wrapper.findAll('.formula-part .formula-term').map((term) => term.text()))
      .toEqual(['Base price', 'Group multiplier', 'Usage'])
    wrapper.unmount()
  })

  it('credits the sources of the Earth imagery in the footer', () => {
    const wrapper = mountLanding()
    const links = wrapper.findAll('.footer-credits a')

    expect(links.map((link) => link.text())).toEqual(['地球影像：NASA Blue Marble 与 Black Marble', 'Solar System Scope', 'CC BY 4.0'])
    expect(links.map((link) => new URL(link.attributes('href')!).hostname)).toEqual(['visibleearth.nasa.gov', 'www.solarsystemscope.com', 'creativecommons.org'])
    // CC BY 4.0 asks for the author, the source, the licence, and a note that the work was changed.
    expect(wrapper.get('.footer-credits > span').text()).toBe('云层：Solar System Scope，CC BY 4.0，有改动')
    wrapper.unmount()
  })

  it('holds the first screen in its opening pose only on a visit that starts at the top', async () => {
    const first = mountLanding()
    expect(first.get('.gabugu-landing').classes()).toContain('intro-pending')
    // The pose is released when the animation library has arrived, and at the latest a moment after.
    await vi.waitFor(() => expect(first.get('.gabugu-landing').classes()).not.toContain('intro-pending'), { timeout: 4000 })
    first.unmount()

    // A visitor who follows a link to a section must find the page in place, not mid-animation.
    window.sessionStorage.clear()
    window.history.replaceState(null, '', '/#control')
    const linked = mountLanding()
    expect(linked.get('.gabugu-landing').classes()).not.toContain('intro-pending')
    linked.unmount()
  })
})
