import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { ApiKey, Group } from '@/types'
import {
  STUDIO_MODEL_LIST,
  STUDIO_MODELS,
  buildApiSnippet,
  classifyImageBillingTier,
  resolveDimensions,
} from '../models'
import {
  estimateModelMinPrice,
  estimateStudioCost,
  estimateStudioCostRange,
  formatStudioUsd,
  formatWalletUsd,
  selectStudioImageGroup,
} from '../pricing'
import {
  STUDIO_AUTO_KEY_NAME,
  base64ToBlob,
  buildEditFormData,
  buildGenerationRequestBody,
  extractErrorMessage,
  generateStudioImages,
  isEligibleStudioKey,
  pickStudioApiKey,
  type StudioGenerateRequest,
} from '../api'
import {
  SITE_THEME_STORAGE_KEY,
  applyStudioTheme,
  readInheritedStudioTheme,
} from '../theme'
import {
  attachStudioJobListener,
  getActiveStudioJob,
  hydrateStoredVersion,
  loadStudioHistory,
  resetStudioHistoryDbForTests,
  runManagedStudioJob,
  sanitizeRasterBlob,
  selectVersionsToPrune,
  type StudioStoredVersion,
  type StudioVersion,
} from '../historyStore'

function makeGroup(overrides: Partial<Group> = {}): Group {
  return {
    id: 7,
    name: '图像生成',
    description: null,
    platform: 'openai',
    rate_multiplier: 1,
    is_exclusive: false,
    status: 'active',
    subscription_type: 'standard',
    daily_limit_usd: null,
    weekly_limit_usd: null,
    monthly_limit_usd: null,
    long_context_pricing_enabled: false,
    allow_image_generation: true,
    allow_batch_image_generation: false,
    image_count_billing: true,
    image_rate_independent: true,
    image_rate_multiplier: 1,
    batch_image_discount_multiplier: 0.5,
    batch_image_hold_multiplier: 1,
    image_price_1k: 0.189,
    image_price_2k: 0.268,
    image_price_4k: 0.371,
    video_rate_independent: false,
    video_rate_multiplier: 1,
    video_price_480p: null,
    video_price_720p: null,
    video_price_1080p: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function makeApiKey(overrides: Partial<ApiKey> = {}): ApiKey {
  return {
    id: 101,
    user_id: 1,
    key: 'sk-test-studio-placeholder',
    name: STUDIO_AUTO_KEY_NAME,
    group_id: 7,
    status: 'active',
    ip_whitelist: [],
    ip_blacklist: [],
    quota: 0,
    quota_used: 0,
    expires_at: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('studio models & billing tier alignment', () => {
  it('ensures every non-auto dimension falls into the exact backend ClassifyImageBillingTier', () => {
    for (const model of STUDIO_MODEL_LIST) {
      for (const tier of model.tiers) {
        for (const ratioEntry of model.ratios) {
          if (ratioEntry.base === 'auto') continue
          for (const dir of ratioEntry.dirs) {
            const dims = resolveDimensions(model, tier, ratioEntry.base, dir)
            expect(dims.w % 16).toBe(0)
            expect(dims.h % 16).toBe(0)
            expect(classifyImageBillingTier(dims.apiSize)).toBe(tier)
          }
        }
      }
    }
  })

  it('enforces Gemini n=1 and no unsupported quality levels', () => {
    for (const model of STUDIO_MODEL_LIST.filter((m) => m.protocol === 'gemini')) {
      expect(model.maxN).toBe(1)
      expect(model.qualityLevels).toEqual([])
      expect(model.defaultQuality).toBeNull()
    }
  })

  it('never leaks real API keys in generated cURL or Python snippets', () => {
    const snippetCurl = buildApiSnippet(
      {
        prompt: 'warm beige ceramic vase',
        model: 'gpt-image-2.6-sunburst',
        ratioCode: '16:9',
        tier: '2K',
        apiSize: '2048x1152',
        count: 2,
        quality: 'high',
        complexity: 'high',
        outputFormat: 'png',
        background: 'transparent',
        outputCompression: null,
        inputFidelity: null,
        moderation: 'auto',
        gptStyle: 'vivid',
        refCount: 0,
      },
      'curl',
      'https://gabugu.com'
    )
    expect(snippetCurl).toContain('https://gabugu.com/v1/images/generations')
    expect(snippetCurl).toContain('YOUR_GABUGU_API_KEY')
    expect(snippetCurl).toContain('"complexity": "high"')
    expect(snippetCurl).toContain('"style": "vivid"')

    const snippetEditPy = buildApiSnippet(
      {
        prompt: 'change sky to sunset',
        model: 'gemini-3-pro-image',
        ratioCode: '4:3',
        tier: '4K',
        apiSize: '3328x2496',
        count: 1,
        quality: null,
        complexity: null,
        outputFormat: null,
        background: null,
        outputCompression: null,
        inputFidelity: null,
        moderation: null,
        gptStyle: '',
        refCount: 2,
      },
      'python',
      'https://gabugu.com'
    )
    expect(snippetEditPy).toContain('https://gabugu.com/v1/images/edits')
    expect(snippetEditPy).toContain('YOUR_GABUGU_API_KEY')
    expect(snippetEditPy).toContain('"image[]"')
    expect(snippetEditPy).not.toContain('quality')
  })
})

describe('studio pricing & group resolution', () => {
  it('selects dedicated active openai image group (Group 7) over generic GPT group (Group 4)', () => {
    const gpt = makeGroup({ id: 4, name: 'GPT', image_price_1k: 0.213, image_price_2k: 0.257 })
    const img = makeGroup({ id: 7, name: '图像生成', image_price_1k: 0.189, image_price_2k: 0.268 })
    const gemini = makeGroup({ id: 24, name: 'Gemini', platform: 'gemini', allow_image_generation: false })
    const inactive = makeGroup({ id: 1, name: '图像停用', status: 'inactive' })

    const selected = selectStudioImageGroup([gpt, gemini, inactive, img])
    expect(selected?.id).toBe(7)
  })

  it('matches live Group 7 USD prices and independent multiplier rules', () => {
    const group7 = makeGroup()
    // Independent multiplier = 1 ignores userGroupRates override
    const est1k = estimateStudioCost(group7, { 7: 0.5 }, '1024x1024', 1)
    expect(est1k).toMatchObject({ known: true, tier: '1K', unitPrice: 0.189, totalPrice: 0.189 })

    const est2k = estimateStudioCost(group7, { 7: 0.5 }, '2048x1152', 2)
    expect(est2k).toMatchObject({ known: true, tier: '2K', unitPrice: 0.268, totalPrice: 0.536 })

    const est4k = estimateStudioCost(group7, {}, '3840x2160', 1)
    expect(est4k).toMatchObject({ known: true, tier: '4K', unitPrice: 0.371, totalPrice: 0.371 })

    // When image_rate_independent is false, custom user group rate applies
    const dependent = makeGroup({ image_rate_independent: false, rate_multiplier: 1.0 })
    const estCustom = estimateStudioCost(dependent, { 7: 0.5 }, '2048x2048', 1)
    expect(estCustom.totalPrice).toBeCloseTo(0.134, 6)

    // Non count-billing group reports known: false honestly
    const tokenOnly = makeGroup({ image_count_billing: false })
    expect(estimateStudioCost(tokenOnly, {}, '2048x2048', 1).known).toBe(false)

    // Min model price picks 1K
    expect(estimateModelMinPrice(group7, {}, STUDIO_MODELS['gpt-image-2.6-sunburst'].tiers)).toBe(0.189)
    expect(formatStudioUsd(0.268)).toBe('$0.268')
    expect(formatStudioUsd(1.5)).toBe('$1.50')
    expect(formatWalletUsd(12.4)).toBe('$ 12.40')
  })
})

describe('studio api request builders & key eligibility', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('strips Gemini-forbidden options in JSON generations and multipart edits', () => {
    const req: StudioGenerateRequest = {
      apiKey: 'sk-dummy',
      model: 'gemini-3-pro-image',
      prompt: 'draw a mountain',
      apiSize: '2048x1152',
      count: 4,
      quality: 'high',
      complexity: 'high',
      outputFormat: 'webp',
      background: 'transparent',
      outputCompression: 80,
      inputFidelity: 'high',
      moderation: 'low',
      gptStyle: 'vivid',
      refs: [],
    }
    const jsonBody = buildGenerationRequestBody(req)
    expect(jsonBody).toEqual({
      model: 'gemini-3-pro-image',
      prompt: 'draw a mountain',
      size: '2048x1152',
      n: 1,
      response_format: 'b64_json',
    })

    const blob = new Blob(['fake'], { type: 'image/png' })
    const editForm = buildEditFormData({
      ...req,
      refs: [
        { blob, filename: 'ref1.png', mimeType: 'image/png' },
        { blob, filename: 'ref2.png', mimeType: 'image/png' },
      ],
    })
    expect(editForm.get('n')).toBe('1')
    expect(editForm.get('response_format')).toBe('b64_json')
    expect(editForm.get('quality')).toBeNull()
    expect(editForm.get('style')).toBeNull()
    expect(editForm.get('background')).toBeNull()
    expect(editForm.getAll('image[]')).toHaveLength(2)
  })

  it('uses single image field for 1 ref and preserves GPT 2.6 native fields', () => {
    const blob = new Blob(['fake'], { type: 'image/png' })
    const form = buildEditFormData({
      apiKey: 'sk-dummy',
      model: 'gpt-image-2.6-sunburst',
      prompt: 'refine lighting',
      apiSize: '2048x2048',
      count: 2,
      quality: 'xhigh',
      complexity: 'high',
      outputFormat: 'webp',
      background: 'transparent',
      outputCompression: 90,
      inputFidelity: 'high',
      moderation: 'low',
      gptStyle: 'natural',
      refs: [{ blob, filename: 'a.png', mimeType: 'image/png' }],
    })
    expect(form.getAll('image')).toHaveLength(1)
    expect(form.get('complexity')).toBe('high')
    expect(form.get('quality')).toBe('xhigh')
    expect(form.get('output_format')).toBe('webp')
    expect(form.get('background')).toBe('transparent')
    expect(form.get('output_compression')).toBe('90')
    expect(form.get('input_fidelity')).toBe('high')
    expect(form.get('moderation')).toBe('low')
    expect(form.get('style')).toBe('natural')
  })

  it('picks only dedicated studio key and skips expired, IP-restricted, blacklisted, or exhausted keys', () => {
    const expired = makeApiKey({ id: 1, expires_at: '2020-01-01T00:00:00Z' })
    const ipLocked = makeApiKey({ id: 2, ip_whitelist: ['1.2.3.4'] })
    const ipBlocked = makeApiKey({ id: 6, ip_blacklist: ['5.6.7.8'] })
    const exhausted = makeApiKey({ id: 3, quota: 10, quota_used: 10 })
    const rateLimited = makeApiKey({ id: 7, rate_limit_5h: 10, usage_5h: 10 })
    const generic = makeApiKey({ id: 4, name: 'My General Key' })
    const dedicated = makeApiKey({ id: 5, name: STUDIO_AUTO_KEY_NAME })

    expect(isEligibleStudioKey(expired, 7)).toBe(false)
    expect(isEligibleStudioKey(ipLocked, 7)).toBe(false)
    expect(isEligibleStudioKey(ipBlocked, 7)).toBe(false)
    expect(isEligibleStudioKey(exhausted, 7)).toBe(false)
    expect(isEligibleStudioKey(rateLimited, 7)).toBe(false)
    expect(pickStudioApiKey([expired, ipLocked, ipBlocked, exhausted, rateLimited, generic, dedicated], 7)?.id).toBe(5)
    // Never silently borrow an unrelated user production key
    expect(pickStudioApiKey([expired, ipLocked, exhausted, generic], 7)).toBeNull()
  })

  it('blocks image/svg+xml in base64ToBlob (same-origin XSS prevention)', () => {
    const svgData = 'data:image/svg+xml;base64,' + btoa('<svg onload="alert(1)"></svg>')
    const blob = base64ToBlob(svgData, 'image/png')
    expect(blob.type).toBe('image/png')
    expect(blob.type).not.toContain('svg')
  })

  it('catches HTTP 200 keepalive late JSON errors and decodes data: URI item.url', async () => {
    const keepaliveErrFetch = vi.fn().mockResolvedValue(
      new Response('   \n   \n' + JSON.stringify({ error: { code: 'upstream_timeout', message: '上游超时未扣费' } }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', keepaliveErrFetch)
    await expect(
      generateStudioImages({
        apiKey: 'sk-dummy',
        model: 'gemini-3-pro-image',
        prompt: 'test',
        apiSize: '2048x2048',
        count: 1,
        quality: null,
        complexity: null,
        outputFormat: null,
        background: null,
        outputCompression: null,
        inputFidelity: null,
        moderation: null,
        gptStyle: '',
        refs: [],
      })
    ).rejects.toThrow('上游超时未扣费')

    const dataUriFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ data: [{ url: 'data:image/png;base64,' + btoa('PNG') }] }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', dataUriFetch)
    const origCreateObjectURL = URL.createObjectURL
    const origImage = globalThis.Image
    URL.createObjectURL = vi.fn(() => 'blob:mock-1')
    vi.stubGlobal('Image', undefined)
    try {
      const imgs = await generateStudioImages({
        apiKey: 'sk-dummy',
        model: 'gpt-image-2.6-sunburst',
        prompt: 'test',
        apiSize: '1024x1024',
        count: 1,
        quality: 'medium',
        complexity: 'high',
        outputFormat: 'png',
        background: 'auto',
        outputCompression: null,
        inputFidelity: null,
        moderation: 'auto',
        gptStyle: '',
        refs: [],
      })
      expect(imgs).toHaveLength(1)
      expect(imgs[0].blob?.type).toBe('image/png')
    } finally {
      URL.createObjectURL = origCreateObjectURL
      vi.stubGlobal('Image', origImage)
    }
  })

  it('classifies balance, api_key_quota, 403 access_denied, and user_inactive cleanly', () => {
    const bal = extractErrorMessage(402, JSON.stringify({ error: { message: 'insufficient balance' } }))
    expect(bal.isBalanceError).toBe(true)
    expect(bal.isAuthKeyError).toBe(false)

    const keyQuota = extractErrorMessage(429, JSON.stringify({ error: { code: 'API_KEY_QUOTA_EXHAUSTED', message: 'API key quota exhausted' } }))
    expect(keyQuota.isAuthKeyError).toBe(true)
    expect(keyQuota.isBalanceError).toBe(false)

    const accessDenied = extractErrorMessage(403, JSON.stringify({ error: { code: 'ACCESS_DENIED', message: 'IP blocked' } }))
    expect(accessDenied.isAuthKeyError).toBe(true)

    const userInactive = extractErrorMessage(401, JSON.stringify({ error: { code: 'USER_INACTIVE', message: 'User account is disabled' } }))
    expect(userInactive.isAuthKeyError).toBe(false)
    expect(userInactive.message).toContain('User account is disabled')

    const auth = extractErrorMessage(401, JSON.stringify({ error: { code: 'invalid_api_key', message: 'Invalid key' } }))
    expect(auth.isAuthKeyError).toBe(true)
    expect(auth.isBalanceError).toBe(false)
  })

  it('enforces 4K 4:3 pixel budget, preserves true/false in Python prompts, and computes auto range', () => {
    const dims4k43 = resolveDimensions(STUDIO_MODELS['gpt-image-2.6-sunburst'], '4K', '4:3', 'landscape')
    expect(dims4k43.w * dims4k43.h).toBeLessThanOrEqual(3840 * 2160)
    expect(classifyImageBillingTier(dims4k43.apiSize)).toBe('4K')

    const py = buildApiSnippet(
      {
        prompt: 'a true masterpiece with false shadows',
        model: 'gpt-image-2.6-sunburst',
        ratioCode: '1:1',
        tier: '2K',
        apiSize: '2048x2048',
        count: 1,
        quality: 'medium',
        complexity: 'high',
        outputFormat: 'jpeg',
        background: 'auto',
        outputCompression: 85,
        inputFidelity: null,
        moderation: 'auto',
        gptStyle: '',
        refCount: 0,
      },
      'python'
    )
    expect(py).toContain('a true masterpiece with false shadows')
    expect(py).toContain('"output_compression": 85')

    const range = estimateStudioCostRange(makeGroup(), {}, STUDIO_MODELS['gpt-image-2.6-sunburst'].tiers, 2)
    expect(range).toEqual({
      known: true,
      minUnitPrice: 0.189,
      maxUnitPrice: 0.371,
      minTotalPrice: 0.378,
      maxTotalPrice: 0.742,
    })
  })
})

describe('studio site-theme inheritance', () => {
  const mockMatchMedia = (prefersDark: boolean) => {
    vi.stubGlobal(
      'matchMedia',
      vi.fn().mockImplementation((query: string) => ({
        matches: query.includes('dark') ? prefersDark : !prefersDark,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }))
    )
  }

  beforeEach(() => {
    document.documentElement.classList.remove('dark')
    localStorage.clear()
    mockMatchMedia(false)
  })

  it('inherits dark theme when user entered from dark mode (html.dark) even under OS light', () => {
    document.documentElement.classList.add('dark')
    localStorage.setItem(SITE_THEME_STORAGE_KEY, 'dark')
    expect(readInheritedStudioTheme()).toBe('dark')
  })

  it('inherits light theme when user entered from light mode even under OS dark and legacy studio dark key', () => {
    mockMatchMedia(true)
    localStorage.setItem(SITE_THEME_STORAGE_KEY, 'light')
    localStorage.setItem('gabugu-studio-theme-v2', 'dark')
    expect(readInheritedStudioTheme()).toBe('light')
  })

  it('falls back to OS prefers-color-scheme when no explicit site theme is stored', () => {
    mockMatchMedia(true)
    expect(readInheritedStudioTheme()).toBe('dark')
    mockMatchMedia(false)
    expect(readInheritedStudioTheme()).toBe('light')
  })

  it('syncs manual studio theme toggles back to html.dark and site localStorage[theme] and cleans legacy key', () => {
    localStorage.setItem('gabugu-studio-theme-v2', 'dark')
    applyStudioTheme('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem(SITE_THEME_STORAGE_KEY)).toBe('dark')
    expect(localStorage.getItem('gabugu-studio-theme-v2')).toBeNull()

    applyStudioTheme('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(localStorage.getItem(SITE_THEME_STORAGE_KEY)).toBe('light')
  })
})

describe('studio history persistence & managed background jobs', () => {
  it('blocks image/svg+xml blobs during sanitization and hydration', () => {
    const evilBlob = new Blob(['<svg onload="alert(1)"/>'], { type: 'image/svg+xml' })
    const safe = sanitizeRasterBlob(evilBlob)
    expect(safe.type).toBe('image/png')

    const origCreate = URL.createObjectURL
    URL.createObjectURL = vi.fn(() => 'blob:hydrated-safe-1')
    try {
      const minted: string[] = []
      const stored: StudioStoredVersion = {
        storageKey: '42:1',
        userId: 42,
        id: 1,
        createdAt: 1000,
        prompt: 'warm ceramic cup',
        model: 'gpt-image-2.6-sunburst',
        baseRatio: '1:1',
        orientation: 'landscape',
        ratioCode: '1:1',
        tier: '2K',
        apiSize: '2048x2048',
        dimensions: '2048×2048',
        w: 2048,
        h: 2048,
        count: 1,
        quality: 'medium',
        complexity: 'high',
        outputFormat: 'png',
        background: 'auto',
        outputCompression: null,
        inputFidelity: null,
        moderation: 'auto',
        gptStyle: '',
        refCount: 0,
        branchLabel: '',
        cost: 0.268,
        costKnown: true,
        images: [
          {
            blob: evilBlob,
            thumbUrl: 'javascript:alert(1)',
            mimeType: 'image/svg+xml',
            ext: 'png',
            width: 2048,
            height: 2048,
          },
        ],
        favs: [true],
        selectedIdx: 0,
        byteSize: evilBlob.size,
      }
      const hydrated = hydrateStoredVersion(stored, minted)
      expect(hydrated).not.toBeNull()
      expect(hydrated?.images[0].mimeType).toBe('image/png')
      expect(hydrated?.images[0].blob?.type).toBe('image/png')
      expect(hydrated?.images[0].thumbUrl).toBe('blob:hydrated-safe-1')
      expect(hydrated?.favs).toEqual([true])
      expect(minted).toEqual(['blob:hydrated-safe-1'])
    } finally {
      URL.createObjectURL = origCreate
    }
  })

  it('prunes oldest non-favorited versions first while protecting starred works and the newly saved key', () => {
    const records = [
      { storageKey: '7:1', id: 1, createdAt: 100, favs: [true], byteSize: 50 },
      { storageKey: '7:2', id: 2, createdAt: 200, favs: [false], byteSize: 50 },
      { storageKey: '7:3', id: 3, createdAt: 300, favs: [false], byteSize: 50 },
      { storageKey: '7:4', id: 4, createdAt: 400, favs: [false], byteSize: 50 },
    ]
    // Cap at 2 items: should delete 7:2 and 7:3 (oldest unstarred), keeping starred 7:1 and newly saved 7:4
    const pruned = selectVersionsToPrune(records, '7:4', 2, 10_000)
    expect(pruned).toEqual(['7:2', '7:3'])

    // Tight byte cap (100 bytes): if all remaining except protected are starred, eventually prunes oldest starred
    const tightPruned = selectVersionsToPrune(records, '7:4', 10, 60)
    expect(tightPruned).toEqual(['7:2', '7:3', '7:1'])
  })

  it('degrades safely when IndexedDB is unavailable', async () => {
    resetStudioHistoryDbForTests()
    const origIdb = globalThis.indexedDB
    vi.stubGlobal('indexedDB', undefined)
    try {
      const res = await loadStudioHistory(42)
      expect(res.storageAvailable).toBe(false)
      expect(res.versions).toEqual([])
      expect(res.nextVersionId).toBe(1)
    } finally {
      vi.stubGlobal('indexedDB', origIdb)
      resetStudioHistoryDbForTests()
    }
  })

  it('completes managed background job and revokes temporary blob URL if view unmounts mid-flight', async () => {
    const origRevoke = URL.revokeObjectURL
    const revoked: string[] = []
    URL.revokeObjectURL = vi.fn((u: string) => {
      revoked.push(u)
    })
    const origIdb = globalThis.indexedDB
    vi.stubGlobal('indexedDB', undefined)
    resetStudioHistoryDbForTests()
    try {
      let resolveJob!: (v: StudioVersion) => void
      const deferred = new Promise<StudioVersion>((resolve) => {
        resolveJob = resolve
      })
      const onSuccess = vi.fn()
      const onError = vi.fn()

      const pending = {
        prompt: 'sunlit forest',
        model: 'gpt-image-2.6-sunburst',
        ratioCode: '1:1',
        tier: '2K' as const,
        w: 2048,
        h: 2048,
        count: 1,
        refCount: 0,
        branchLabel: '',
        cost: 0.268,
        costKnown: true,
      }

      const promise = runManagedStudioJob(99, pending, { onSuccess, onError }, () => deferred)
      expect(getActiveStudioJob(99)).toEqual(pending)
      expect(getActiveStudioJob(100)).toBeNull()

      // Simulate user leaving /studio mid-generation
      attachStudioJobListener(99, null)

      resolveJob({
        id: 5,
        createdAt: Date.now(),
        prompt: 'sunlit forest',
        model: 'gpt-image-2.6-sunburst',
        baseRatio: '1:1',
        orientation: 'landscape',
        ratioCode: '1:1',
        tier: '2K',
        apiSize: '2048x2048',
        dimensions: '2048×2048',
        w: 2048,
        h: 2048,
        count: 1,
        quality: 'medium',
        complexity: 'high',
        outputFormat: 'png',
        background: 'auto',
        outputCompression: null,
        inputFidelity: null,
        moderation: 'auto',
        gptStyle: '',
        refCount: 0,
        branchLabel: '',
        cost: 0.268,
        costKnown: true,
        images: [
          {
            objectUrl: 'blob:temp-bg-5',
            blob: new Blob(['png'], { type: 'image/png' }),
            thumbUrl: 'blob:temp-bg-5',
            mimeType: 'image/png',
            ext: 'png',
            width: 2048,
            height: 2048,
          },
        ],
        favs: [false],
        selectedIdx: 0,
      })

      await promise
      expect(onSuccess).not.toHaveBeenCalled()
      expect(onError).not.toHaveBeenCalled()
      expect(getActiveStudioJob(99)).toBeNull()
      expect(revoked).toEqual(['blob:temp-bg-5'])
    } finally {
      URL.revokeObjectURL = origRevoke
      vi.stubGlobal('indexedDB', origIdb)
      resetStudioHistoryDbForTests()
    }
  })
})
