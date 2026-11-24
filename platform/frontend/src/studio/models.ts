/**
 * GABUGU 画室 — 线上生图模型契约、厂商标识、比例与精确分辨率换算。
 *
 * 仅列出线上「图像生成」分组实际在线且可调度的模型，参数能力严格对齐后端
 * `service/openai_images.go`、`service/openai_images_gemini.go` 与 `service/image_billing_size.go`。
 */

export type StudioVendorId = 'openai' | 'gemini' | 'grok'
export type StudioOrientation = 'landscape' | 'portrait'
export type StudioEffectiveDir = 'none' | 'landscape' | 'portrait'
export type StudioTierId = '1K' | '2K' | '4K'
export type StudioOutputFormat = 'png' | 'jpeg' | 'webp'
export type StudioBackground = 'auto' | 'opaque' | 'transparent'
export type StudioComplexity = 'low' | 'medium' | 'high'
export type StudioInputFidelity = 'low' | 'high'
export type StudioModeration = 'auto' | 'low'

export interface VendorMeta {
  id: StudioVendorId
  name: string
  path: string
}

export interface BaseRatioDef {
  id: string
  name: string
  pName?: string
  w: number
  h: number
  directional: boolean
}

export interface ModelRatioEntry {
  base: string
  dirs: StudioEffectiveDir[]
}

export interface StudioModelSpec {
  id: string
  name: string
  shortName: string
  protocol: StudioVendorId
  featured: boolean
  summary: string
  ratios: ModelRatioEntry[]
  tiers: StudioTierId[]
  defaultTier: StudioTierId
  qualityLevels: string[]
  defaultQuality: string | null
  supportsComplexity?: boolean
  defaultComplexity?: StudioComplexity
  outputFormats?: StudioOutputFormat[]
  defaultFormat?: StudioOutputFormat
  supportsBackground?: boolean
  supportsCompression?: boolean
  supportsInputFidelity?: boolean
  supportsModeration?: boolean
  supportsGptStylePassthrough?: boolean
  supportsEdit: boolean
  maxRefImages: number
  maxN: number
}

export interface ResolvedDimensions {
  w: number
  h: number
  pixels: string
  apiSize: string
  isAuto: boolean
}

export const VENDOR_META: Record<StudioVendorId, VendorMeta> = {
  openai: {
    id: 'openai',
    name: 'OpenAI',
    path: 'M21.55 10.004a5.416 5.416 0 00-.478-4.501c-1.217-2.09-3.662-3.166-6.05-2.66A5.59 5.59 0 0010.831 1C8.39.995 6.224 2.546 5.473 4.838A5.553 5.553 0 001.76 7.496a5.487 5.487 0 00.691 6.5 5.416 5.416 0 00.477 4.502c1.217 2.09 3.662 3.165 6.05 2.66A5.586 5.586 0 0013.168 23c2.443.006 4.61-1.546 5.361-3.84a5.553 5.553 0 003.715-2.66 5.488 5.488 0 00-.693-6.497v.001zm-8.381 11.558a4.199 4.199 0 01-2.675-.954c.034-.018.093-.05.132-.074l4.44-2.53a.71.71 0 00.364-.623v-6.176l1.877 1.069c.02.01.033.029.036.05v5.115c-.003 2.274-1.87 4.118-4.174 4.123zM4.192 17.78a4.059 4.059 0 01-.498-2.763c.032.02.09.055.131.078l4.44 2.53c.225.13.504.13.73 0l5.42-3.088v2.138a.068.068 0 01-.027.057L9.9 19.288c-1.999 1.136-4.552.46-5.707-1.51h-.001zM3.023 8.216A4.15 4.15 0 015.198 6.41l-.002.151v5.06a.711.711 0 00.364.624l5.42 3.087-1.876 1.07a.067.067 0 01-.063.005l-4.489-2.559c-1.995-1.14-2.679-3.658-1.53-5.63h.001zm15.417 3.54l-5.42-3.088L14.896 7.6a.067.067 0 01.063-.006l4.489 2.557c1.998 1.14 2.683 3.662 1.529 5.633a4.163 4.163 0 01-2.174 1.807V12.38a.71.71 0 00-.363-.623zm1.867-2.773a6.04 6.04 0 00-.132-.078l-4.44-2.53a.731.731 0 00-.729 0l-5.42 3.088V7.325a.068.068 0 01.027-.057L14.1 4.713c2-1.137 4.555-.46 5.707 1.513.487.833.664 1.809.499 2.757h.001zm-11.741 3.81l-1.877-1.068a.065.065 0 01-.036-.051V6.559c.001-2.277 1.873-4.122 4.181-4.12.976 0 1.92.338 2.671.954-.034.018-.092.05-.131.073l-4.44 2.53a.71.71 0 00-.365.623l-.003 6.173v.002zm1.02-2.168L12 9.25l2.414 1.375v2.75L12 14.75l-2.415-1.375v-2.75z',
  },
  gemini: {
    id: 'gemini',
    name: 'Google',
    path: 'M20.616 10.835a14.147 14.147 0 01-4.45-3.001 14.111 14.111 0 01-3.678-6.452.503.503 0 00-.975 0 14.134 14.134 0 01-3.679 6.452 14.155 14.155 0 01-4.45 3.001c-.65.28-1.318.505-2.002.678a.502.502 0 000 .975c.684.172 1.35.397 2.002.677a14.147 14.147 0 014.45 3.001 14.112 14.112 0 013.679 6.453.502.502 0 00.975 0c.172-.685.397-1.351.677-2.003a14.145 14.145 0 013.001-4.45 14.113 14.113 0 016.453-3.678.503.503 0 000-.975 13.245 13.245 0 01-2.003-.678z',
  },
  grok: {
    id: 'grok',
    name: 'xAI',
    path: 'M9.27 15.29l7.978-5.897c.391-.29.95-.177 1.137.272.98 2.369.542 5.215-1.41 7.169-1.951 1.954-4.667 2.382-7.149 1.406l-2.711 1.257c3.889 2.661 8.611 2.003 11.562-.953 2.341-2.344 3.066-5.539 2.388-8.42l.006.007c-.983-4.232.242-5.924 2.75-9.383.06-.082.12-.164.179-.248l-3.301 3.305v-.01L9.267 15.292M7.623 16.723c-2.792-2.67-2.31-6.801.071-9.184 1.761-1.763 4.647-2.483 7.166-1.425l2.705-1.25a7.808 7.808 0 00-1.829-1A8.975 8.975 0 005.984 5.83c-2.533 2.536-3.33 6.436-1.962 9.764 1.022 2.487-.653 4.246-2.34 6.022-.599.63-1.199 1.259-1.682 1.925l7.62-6.815',
  },
}

export const VENDOR_ORDER: StudioVendorId[] = ['openai', 'gemini', 'grok']

export const BASE_RATIOS: Record<string, BaseRatioDef> = {
  auto: { id: 'auto', name: '自动', w: 1, h: 1, directional: false },
  '1:1': { id: '1:1', name: '正方', w: 1, h: 1, directional: false },
  '4:3': { id: '4:3', name: '经典', pName: '画册', w: 4, h: 3, directional: true },
  '3:2': { id: '3:2', name: '相机', pName: '人像', w: 3, h: 2, directional: true },
  '5:4': { id: '5:4', name: '画幅', pName: '海报', w: 5, h: 4, directional: true },
  '16:9': { id: '16:9', name: '宽屏', pName: '竖屏', w: 16, h: 9, directional: true },
  '2:1': { id: '2:1', name: '宽画', pName: '长竖', w: 2, h: 1, directional: true },
  '21:9': { id: '21:9', name: '电影', pName: '电影', w: 21, h: 9, directional: true },
  '19.5:9': { id: '19.5:9', name: '横全面屏', pName: '竖全面屏', w: 19.5, h: 9, directional: true },
  '20:9': { id: '20:9', name: '长横屏', pName: '长竖屏', w: 20, h: 9, directional: true },
}

export const TIER_META: Record<StudioTierId, { id: StudioTierId; label: string; note: string }> = {
  '1K': { id: '1K', label: '1K', note: '标准清晰 · 省费' },
  '2K': { id: '2K', label: '2K', note: '高清细节 · 推荐' },
  '4K': { id: '4K', label: '4K', note: '超清成品 · 精细' },
}

export const QUALITY_META: Record<string, { label: string; desc: string }> = {
  auto: { label: '自动', desc: '由模型自动权衡细节与速度' },
  low: { label: '低', desc: '最快最省，适合快速试构图' },
  medium: { label: '中', desc: '速度与细节平衡（默认）' },
  high: { label: '高', desc: '纹理与文字更精细' },
  xhigh: { label: '超高', desc: '极高细节密度，耗时较长' },
  max: { label: '最高', desc: '满格渲染上限，精细文字/微距' },
}

export const COMPLEXITY_META: Record<StudioComplexity, { label: string; desc: string }> = {
  low: { label: '低', desc: '简单单主体或局部换底，更快' },
  medium: { label: '中', desc: '常规构图与多元素平衡' },
  high: { label: '高', desc: '复杂排版、多主体空间关系与密集文字' },
}

export const FORMAT_META: Record<StudioOutputFormat, { label: string; desc: string; ext: string; mime: string }> = {
  png: { label: 'PNG', desc: '无损，支持透明背景', ext: 'png', mime: 'image/png' },
  jpeg: { label: 'JPEG', desc: '体积小，可调压缩率', ext: 'jpg', mime: 'image/jpeg' },
  webp: { label: 'WebP', desc: '高压缩，支持透明与压缩率', ext: 'webp', mime: 'image/webp' },
}

const GPT_RATIOS: ModelRatioEntry[] = [
  { base: 'auto', dirs: ['none'] },
  { base: '1:1', dirs: ['none'] },
  { base: '4:3', dirs: ['landscape', 'portrait'] },
  { base: '3:2', dirs: ['landscape', 'portrait'] },
  { base: '5:4', dirs: ['landscape', 'portrait'] },
  { base: '16:9', dirs: ['landscape', 'portrait'] },
  { base: '2:1', dirs: ['landscape', 'portrait'] },
  { base: '21:9', dirs: ['landscape'] },
]

// 对齐后端 `openai_images_gemini.go` 的 nearestGeminiImageAspectRatio 白名单
const GEMINI_RATIOS: ModelRatioEntry[] = [
  { base: '1:1', dirs: ['none'] },
  { base: '4:3', dirs: ['landscape', 'portrait'] },
  { base: '3:2', dirs: ['landscape', 'portrait'] },
  { base: '5:4', dirs: ['landscape', 'portrait'] },
  { base: '16:9', dirs: ['landscape', 'portrait'] },
  { base: '21:9', dirs: ['landscape'] },
]

const GROK_RATIOS: ModelRatioEntry[] = [
  { base: 'auto', dirs: ['none'] },
  { base: '1:1', dirs: ['none'] },
  { base: '4:3', dirs: ['landscape', 'portrait'] },
  { base: '3:2', dirs: ['landscape', 'portrait'] },
  { base: '16:9', dirs: ['landscape', 'portrait'] },
  { base: '2:1', dirs: ['landscape', 'portrait'] },
  { base: '19.5:9', dirs: ['landscape', 'portrait'] },
  { base: '20:9', dirs: ['landscape', 'portrait'] },
  { base: '21:9', dirs: ['landscape'] },
]

export const DEFAULT_STUDIO_MODEL_ID = 'gpt-image-2.6-sunburst'

export const STUDIO_MODELS: Record<string, StudioModelSpec> = {
  'gpt-image-2.6-sunburst': {
    id: 'gpt-image-2.6-sunburst',
    name: 'GPT Image 2.6 Sunburst',
    shortName: 'Image 2.6 Sunburst',
    protocol: 'openai',
    featured: true,
    summary: '6 档画质 · 构图规划 · 1K–4K · 透明背景 · 8 图参考',
    ratios: GPT_RATIOS,
    tiers: ['1K', '2K', '4K'],
    defaultTier: '2K',
    qualityLevels: ['auto', 'low', 'medium', 'high', 'xhigh', 'max'],
    defaultQuality: 'medium',
    supportsComplexity: true,
    defaultComplexity: 'high',
    outputFormats: ['png', 'jpeg', 'webp'],
    defaultFormat: 'png',
    supportsBackground: true,
    supportsCompression: true,
    supportsInputFidelity: true,
    supportsModeration: true,
    supportsGptStylePassthrough: true,
    supportsEdit: true,
    maxRefImages: 8,
    maxN: 4,
  },
  'gpt-image-2.6-flare': {
    id: 'gpt-image-2.6-flare',
    name: 'GPT Image 2.6 Flare',
    shortName: 'Image 2.6 Flare',
    protocol: 'openai',
    featured: true,
    summary: '极速 2.6 · 6 档画质 · 构图规划 · 1K–4K · 透明背景',
    ratios: GPT_RATIOS,
    tiers: ['1K', '2K', '4K'],
    defaultTier: '2K',
    qualityLevels: ['auto', 'low', 'medium', 'high', 'xhigh', 'max'],
    defaultQuality: 'medium',
    supportsComplexity: true,
    defaultComplexity: 'high',
    outputFormats: ['png', 'jpeg', 'webp'],
    defaultFormat: 'png',
    supportsBackground: true,
    supportsCompression: true,
    supportsInputFidelity: true,
    supportsModeration: true,
    supportsGptStylePassthrough: true,
    supportsEdit: true,
    maxRefImages: 8,
    maxN: 4,
  },
  'gemini-3-pro-image': {
    id: 'gemini-3-pro-image',
    name: 'Gemini 3 Pro Image',
    shortName: 'Gemini 3 Pro',
    protocol: 'gemini',
    featured: true,
    summary: '10 比例（含 21:9 横）· 1K–4K · 14 图参考 · 单次 1 张',
    ratios: GEMINI_RATIOS,
    tiers: ['1K', '2K', '4K'],
    defaultTier: '2K',
    qualityLevels: [],
    defaultQuality: null,
    supportsEdit: true,
    maxRefImages: 14,
    maxN: 1,
  },
  'gemini-3.1-flash-image': {
    id: 'gemini-3.1-flash-image',
    name: 'Gemini 3.1 Flash Image',
    shortName: 'Gemini 3.1 Flash',
    protocol: 'gemini',
    featured: true,
    summary: '极速响应 · 10 比例 · 1K–4K · 14 图参考 · 单次 1 张',
    ratios: GEMINI_RATIOS,
    tiers: ['1K', '2K', '4K'],
    defaultTier: '2K',
    qualityLevels: [],
    defaultQuality: null,
    supportsEdit: true,
    maxRefImages: 14,
    maxN: 1,
  },
  'grok-imagine-image-2.0': {
    id: 'grok-imagine-image-2.0',
    name: 'Grok Imagine 2.0',
    shortName: 'Grok Imagine 2.0',
    protocol: 'grok',
    featured: true,
    summary: '16 比例（含全面屏）· 1K / 2K · 2 档画质 · 5 图参考',
    ratios: GROK_RATIOS,
    tiers: ['1K', '2K'],
    defaultTier: '2K',
    qualityLevels: ['low', 'medium'],
    defaultQuality: 'medium',
    supportsEdit: true,
    maxRefImages: 5,
    maxN: 4,
  },
  'gpt-image-2.5-sunburst': {
    id: 'gpt-image-2.5-sunburst',
    name: 'GPT Image 2.5 Sunburst',
    shortName: 'Image 2.5 Sunburst',
    protocol: 'openai',
    featured: false,
    summary: '5 档画质（low–max）· 1K–4K · 透明背景 · 8 图参考',
    ratios: GPT_RATIOS,
    tiers: ['1K', '2K', '4K'],
    defaultTier: '2K',
    qualityLevels: ['low', 'medium', 'high', 'xhigh', 'max'],
    defaultQuality: 'medium',
    outputFormats: ['png', 'jpeg', 'webp'],
    defaultFormat: 'png',
    supportsBackground: true,
    supportsCompression: true,
    supportsInputFidelity: true,
    supportsModeration: true,
    supportsGptStylePassthrough: true,
    supportsEdit: true,
    maxRefImages: 8,
    maxN: 4,
  },
  'gpt-image-2.5-flare': {
    id: 'gpt-image-2.5-flare',
    name: 'GPT Image 2.5 Flare',
    shortName: 'Image 2.5 Flare',
    protocol: 'openai',
    featured: false,
    summary: '极速 2.5 · 5 档画质（low–max）· 1K–4K · 透明背景',
    ratios: GPT_RATIOS,
    tiers: ['1K', '2K', '4K'],
    defaultTier: '2K',
    qualityLevels: ['low', 'medium', 'high', 'xhigh', 'max'],
    defaultQuality: 'medium',
    outputFormats: ['png', 'jpeg', 'webp'],
    defaultFormat: 'png',
    supportsBackground: true,
    supportsCompression: true,
    supportsInputFidelity: true,
    supportsModeration: true,
    supportsGptStylePassthrough: true,
    supportsEdit: true,
    maxRefImages: 8,
    maxN: 4,
  },
  'gpt-image-2': {
    id: 'gpt-image-2',
    name: 'GPT Image 2',
    shortName: 'GPT Image 2',
    protocol: 'openai',
    featured: false,
    summary: '通用稳健 · 4 档画质（auto/低/中/高）· 1K–4K · 透明背景',
    ratios: GPT_RATIOS,
    tiers: ['1K', '2K', '4K'],
    defaultTier: '2K',
    qualityLevels: ['auto', 'low', 'medium', 'high'],
    defaultQuality: 'medium',
    outputFormats: ['png', 'jpeg', 'webp'],
    defaultFormat: 'png',
    supportsBackground: true,
    supportsCompression: true,
    supportsInputFidelity: true,
    supportsModeration: true,
    supportsGptStylePassthrough: true,
    supportsEdit: true,
    maxRefImages: 8,
    maxN: 4,
  },
  'gemini-3.1-flash-lite-image': {
    id: 'gemini-3.1-flash-lite-image',
    name: 'Gemini 3.1 Flash Lite Image',
    shortName: 'Gemini 3.1 Lite',
    protocol: 'gemini',
    featured: false,
    summary: '轻量极速 · 10 比例 · 1K–4K · 14 图参考 · 单次 1 张',
    ratios: GEMINI_RATIOS,
    tiers: ['1K', '2K', '4K'],
    defaultTier: '2K',
    qualityLevels: [],
    defaultQuality: null,
    supportsEdit: true,
    maxRefImages: 14,
    maxN: 1,
  },
  'grok-imagine-image-quality': {
    id: 'grok-imagine-image-quality',
    name: 'Grok Imagine Quality',
    shortName: 'Grok Quality',
    protocol: 'grok',
    featured: false,
    summary: '高精细模式 · 16 比例 · 1K / 2K · 5 图参考',
    ratios: GROK_RATIOS,
    tiers: ['1K', '2K'],
    defaultTier: '2K',
    qualityLevels: ['low', 'medium'],
    defaultQuality: 'medium',
    supportsEdit: true,
    maxRefImages: 5,
    maxN: 4,
  },
  'grok-imagine-image': {
    id: 'grok-imagine-image',
    name: 'Grok Imagine',
    shortName: 'Grok Imagine',
    protocol: 'grok',
    featured: false,
    summary: '标准快出 · 16 比例 · 1K / 2K · 5 图参考',
    ratios: GROK_RATIOS,
    tiers: ['1K', '2K'],
    defaultTier: '2K',
    qualityLevels: ['low', 'medium'],
    defaultQuality: 'medium',
    supportsEdit: true,
    maxRefImages: 5,
    maxN: 4,
  },
}

export const STUDIO_MODEL_LIST: StudioModelSpec[] = Object.values(STUDIO_MODELS)

export function getStudioModel(id: string): StudioModelSpec {
  return STUDIO_MODELS[id] || STUDIO_MODELS[DEFAULT_STUDIO_MODEL_ID]
}

export function ratioLabelForDir(def: BaseRatioDef | undefined, dir: StudioEffectiveDir): string {
  if (!def) return ''
  return dir === 'portrait' && def.pName ? def.pName : def.name
}

export function getRatioEntry(model: StudioModelSpec, baseId: string): ModelRatioEntry | null {
  return model.ratios.find((r) => r.base === baseId) || null
}

export function effectiveDir(
  model: StudioModelSpec,
  baseId: string,
  orientation: StudioOrientation
): StudioEffectiveDir {
  const def = BASE_RATIOS[baseId]
  if (!def || !def.directional) return 'none'
  const entry = getRatioEntry(model, baseId)
  if (entry && !entry.dirs.includes(orientation)) return entry.dirs[0]
  return orientation
}

export function resolveRatioCode(baseId: string, dir: StudioEffectiveDir): string {
  if (baseId === 'auto') return 'auto'
  if (baseId === '1:1') return '1:1'
  const parts = baseId.split(':')
  return dir === 'portrait' ? `${parts[1]}:${parts[0]}` : baseId
}

export function numericAspect(baseId: string, dir: StudioEffectiveDir): number {
  const def = BASE_RATIOS[baseId] || BASE_RATIOS['1:1']
  if (!def.directional) return 1
  return dir === 'portrait' ? def.h / def.w : def.w / def.h
}

function round16(n: number): number {
  return Math.max(64, Math.round(n / 16) * 16)
}

function floor16(n: number): number {
  return Math.max(64, Math.floor(n / 16) * 16)
}

/**
 * 与后端 `service/image_billing_size.go` `ClassifyImageBillingTier` 保持 1:1 对应。
 */
export function classifyImageBillingTier(size: string): StudioTierId {
  const normalized = size.trim().toLowerCase()
  if (normalized === '1k') return '1K'
  if (normalized === '2k' || normalized === '' || normalized === 'auto') return '2K'
  if (normalized === '4k') return '4K'
  if (normalized === '2048x2048' || normalized === '2048x1152') return '2K'
  if (normalized === '3840x2160' || normalized === '2160x3840') return '4K'
  const parts = normalized.split('x')
  if (parts.length !== 2) return '2K'
  const w = Number.parseInt(parts[0], 10)
  const h = Number.parseInt(parts[1], 10)
  if (!Number.isFinite(w) || !Number.isFinite(h) || w <= 0 || h <= 0) return '2K'
  const maxEdge = Math.max(w, h)
  if (maxEdge <= 1024) return '1K'
  if (maxEdge <= 2048) return '2K'
  return '4K'
}

/**
 * 16 像素对齐的精确尺寸换算，严格保证：
 * - 1K 长边 <= 1024（落入后端 1K 计费档）
 * - 2K 长边在 1025..2048（落入后端 2K 计费档）
 * - 4K 长边 > 2048 且单边 <= 3840、面积 <= 3840×2160（落入后端 4K 计费档）
 */
export function resolveDimensions(
  _model: StudioModelSpec,
  tierId: StudioTierId,
  baseId: string,
  dir: StudioEffectiveDir
): ResolvedDimensions {
  if (baseId === 'auto') {
    return { w: 1024, h: 1024, pixels: 'auto', apiSize: 'auto', isAuto: true }
  }

  const ratio = numericAspect(baseId, dir)
  let w: number
  let h: number

  if (tierId === '1K') {
    const longEdge = 1024
    if (ratio >= 1) {
      w = longEdge
      h = round16(longEdge / ratio)
    } else {
      h = longEdge
      w = round16(longEdge * ratio)
    }
  } else if (tierId === '2K') {
    const longEdge = 2048
    if (ratio >= 1) {
      w = longEdge
      h = round16(longEdge / ratio)
    } else {
      h = longEdge
      w = round16(longEdge * ratio)
    }
  } else {
    const maxPixels = 3840 * 2160
    const rawW = Math.sqrt(maxPixels * ratio)
    const rawH = rawW / ratio
    const scale = Math.min(1, 3840 / Math.max(rawW, rawH))
    w = floor16(rawW * scale)
    h = floor16(rawH * scale)
    if (Math.max(w, h) <= 2048) {
      if (ratio >= 1) {
        w = 2176
        h = floor16(w / ratio)
      } else {
        h = 2176
        w = floor16(h * ratio)
      }
    }
  }

  return {
    w,
    h,
    pixels: `${w}×${h}`,
    apiSize: `${w}x${h}`,
    isAuto: false,
  }
}

export interface StudioSnippetVersion {
  prompt: string
  model: string
  ratioCode: string
  tier: StudioTierId
  apiSize: string
  count: number
  quality: string | null
  complexity: StudioComplexity | null
  outputFormat: StudioOutputFormat | null
  background: StudioBackground | null
  outputCompression: number | null
  inputFidelity: StudioInputFidelity | null
  moderation: StudioModeration | null
  gptStyle: string
  refCount: number
}

/**
 * 构建可复制的真实 GABUGU API 调用代码（永远只输出 YOUR_GABUGU_API_KEY 占位符）。
 */
export function buildApiSnippet(
  v: StudioSnippetVersion,
  lang: 'curl' | 'python',
  origin = 'https://gabugu.com'
): string {
  const baseOrigin = origin.replace(/\/$/, '')
  const m = getStudioModel(v.model)
  const isEdit = v.refCount > 0
  const endpoint = `${baseOrigin}/v1/images/${isEdit ? 'edits' : 'generations'}`

  const fields: Record<string, string | number> = {
    model: v.model,
    prompt: v.prompt,
    size: v.apiSize,
  }

  if (m.protocol === 'gemini') {
    fields.n = 1
    fields.response_format = 'b64_json'
  } else {
    if (v.count > 1) fields.n = v.count
    if (v.quality) fields.quality = v.quality
    if (m.supportsComplexity && v.complexity) fields.complexity = v.complexity
    if (m.outputFormats && v.outputFormat && v.outputFormat !== 'png') {
      fields.output_format = v.outputFormat
    }
    if (m.supportsBackground && v.background && v.background !== 'auto') {
      fields.background = v.background
      if (v.background === 'transparent' && !fields.output_format) {
        fields.output_format = 'png'
      }
    }
    if (
      m.supportsCompression &&
      v.outputCompression != null &&
      (v.outputFormat === 'jpeg' || v.outputFormat === 'webp')
    ) {
      fields.output_compression = v.outputCompression
    }
    if (m.supportsInputFidelity && isEdit && v.inputFidelity && v.inputFidelity !== 'low') {
      fields.input_fidelity = v.inputFidelity
    }
    if (m.supportsModeration && v.moderation && v.moderation !== 'auto') {
      fields.moderation = v.moderation
    }
    if (m.supportsGptStylePassthrough && v.gptStyle.trim()) {
      fields.style = v.gptStyle.trim()
    }
  }

  if (isEdit) {
    // 改图统一走 multipart/form-data（兼容 OpenAI / Gemini / Grok 全系）
    if (lang === 'python') {
      const fileEntries = Array.from({ length: v.refCount }, (_, i) => {
        const key = v.refCount === 1 ? 'image' : 'image[]'
        const name = v.refCount === 1 ? 'reference.png' : `reference-${i + 1}.png`
        return `    ("${key}", ("${name}", open("${name}", "rb"), "image/png")),`
      })
      const dataPairs = Object.entries(fields).map(
        ([k, val]) => `    ${JSON.stringify(k)}: ${JSON.stringify(String(val))},`
      )
      return [
        'import requests',
        '',
        `url = ${JSON.stringify(endpoint)}`,
        'headers = {"Authorization": "Bearer YOUR_GABUGU_API_KEY"}',
        'data = {',
        ...dataPairs,
        '}',
        'files = [',
        ...fileEntries,
        ']',
        '',
        'resp = requests.post(url, headers=headers, data=data, files=files, timeout=300)',
        'resp.raise_for_status()',
        'print(resp.json())',
      ].join('\n')
    }

    const formFlags: string[] = []
    for (const [k, val] of Object.entries(fields)) {
      const escaped = String(val).replace(/'/g, "'\\''")
      formFlags.push(`  -F '${k}=${escaped}'`)
    }
    for (let i = 0; i < v.refCount; i++) {
      const key = v.refCount === 1 ? 'image' : 'image[]'
      const name = v.refCount === 1 ? 'reference.png' : `reference-${i + 1}.png`
      formFlags.push(`  -F "${key}=@${name}"`)
    }
    return [
      `curl ${endpoint} \\`,
      '  -H "Authorization: Bearer YOUR_GABUGU_API_KEY" \\',
      formFlags.join(' \\\n'),
    ].join('\n')
  }

  if (lang === 'python') {
    const jsonPayload = JSON.stringify(fields, null, 2)
    return [
      'import requests',
      '',
      `url = ${JSON.stringify(endpoint)}`,
      'headers = {',
      '    "Authorization": "Bearer YOUR_GABUGU_API_KEY",',
      '    "Content-Type": "application/json",',
      '}',
      `payload = ${jsonPayload}`,
      '',
      'resp = requests.post(url, headers=headers, json=payload, timeout=300)',
      'resp.raise_for_status()',
      'print(resp.json())',
    ].join('\n')
  }

  return [
    `curl ${endpoint} \\`,
    '  -H "Authorization: Bearer YOUR_GABUGU_API_KEY" \\',
    '  -H "Content-Type: application/json" \\',
    `  -d '${JSON.stringify(fields, null, 2).replace(/'/g, "'\\''")}'`,
  ].join('\n')
}
