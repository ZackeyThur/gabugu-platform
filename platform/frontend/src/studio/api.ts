/**
 * GABUGU 画室 — 线上生图 / 改图网关调用、密钥自动管理与参考图预处理。
 *
 * 安全原则：
 * - 自动绑定或创建的生图密钥仅驻留内存，绝不写入 localStorage/sessionStorage/URL/控制台日志。
 * - 改图统一采用 multipart/form-data (`POST /v1/images/edits`)，同时兼容 OpenAI / Gemini 原生 / Grok 上游。
 * - 文生图采用 JSON (`POST /v1/images/generations`)，对 Gemini 模型严格剥离不支持参数避免 400。
 */

import { keysAPI } from '@/api/keys'
import { userAPI } from '@/api/user'
import { userGroupsAPI } from '@/api/groups'
import { buildGatewayUrl } from '@/api/url'
import type { ApiKey, Group, User } from '@/types'
import {
  FORMAT_META,
  getStudioModel,
  type StudioBackground,
  type StudioComplexity,
  type StudioInputFidelity,
  type StudioModeration,
  type StudioOutputFormat,
} from './models'
import { selectStudioImageGroup } from './pricing'

export const STUDIO_AUTO_KEY_NAME = 'GABUGU 画室'
export const MAX_REF_UPLOAD_BYTES = 20 * 1024 * 1024 // 20 MB per file

export interface StudioContextData {
  user: User
  imageGroup: Group | null
  userGroupRates: Record<number, number>
  existingKey: ApiKey | null
}

export interface StudioRefPayload {
  blob: Blob
  filename: string
  mimeType: string
}

export interface StudioGenerateRequest {
  apiKey: string
  model: string
  prompt: string
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
  refs: StudioRefPayload[]
  signal?: AbortSignal
}

export interface StudioGeneratedImage {
  objectUrl: string
  remoteUrl?: string
  blob: Blob | null
  thumbUrl: string
  mimeType: string
  ext: string
  width: number
  height: number
  revisedPrompt?: string
}

export class StudioApiError extends Error {
  status: number
  code?: string
  isBalanceError: boolean
  isAuthKeyError: boolean

  constructor(
    message: string,
    opts: { status?: number; code?: string; isBalanceError?: boolean; isAuthKeyError?: boolean } = {}
  ) {
    super(message)
    this.name = 'StudioApiError'
    this.status = opts.status ?? 0
    this.code = opts.code
    this.isBalanceError = !!opts.isBalanceError
    this.isAuthKeyError = !!opts.isAuthKeyError
  }
}

export function isEligibleStudioKey(key: ApiKey, groupId: number): boolean {
  if (key.status !== 'active') return false
  if (key.group_id !== groupId) return false
  if (!key.key || !key.key.trim()) return false
  if (key.expires_at) {
    const exp = Date.parse(key.expires_at)
    if (Number.isFinite(exp) && exp <= Date.now()) return false
  }
  if (typeof key.quota === 'number' && key.quota > 0 && key.quota_used >= key.quota) {
    return false
  }
  if (Array.isArray(key.ip_whitelist) && key.ip_whitelist.length > 0) {
    return false
  }
  if (Array.isArray(key.ip_blacklist) && key.ip_blacklist.length > 0) {
    return false
  }
  if (
    typeof key.rate_limit_5h === 'number' &&
    key.rate_limit_5h > 0 &&
    typeof key.usage_5h === 'number' &&
    key.usage_5h >= key.rate_limit_5h
  ) {
    return false
  }
  if (
    typeof key.rate_limit_1d === 'number' &&
    key.rate_limit_1d > 0 &&
    typeof key.usage_1d === 'number' &&
    key.usage_1d >= key.rate_limit_1d
  ) {
    return false
  }
  if (
    typeof key.rate_limit_7d === 'number' &&
    key.rate_limit_7d > 0 &&
    typeof key.usage_7d === 'number' &&
    key.usage_7d >= key.rate_limit_7d
  ) {
    return false
  }
  return true
}

/**
 * 仅复用同分组下名为「GABUGU 画室」的专属密钥，绝不静默借用用户的其他生产密钥。
 */
export function pickStudioApiKey(keys: ApiKey[], groupId: number): ApiKey | null {
  const candidates = keys.filter((k) => isEligibleStudioKey(k, groupId))
  if (!candidates.length) return null
  return candidates.find((k) => k.name.trim() === STUDIO_AUTO_KEY_NAME) ?? null
}

async function listActiveKeysForGroup(groupId: number): Promise<ApiKey[]> {
  const collected: ApiKey[] = []
  let page = 1
  const pageSize = 100
  while (page <= 5) {
    const res = await keysAPI.list(page, pageSize, {
      status: 'active',
      group_id: groupId,
      sort_by: 'created_at',
      sort_order: 'desc',
    })
    const items = res.items || []
    collected.push(...items)
    if (page >= (res.pages || 1) || items.length < pageSize) break
    page += 1
  }
  return collected
}

export async function loadStudioContext(): Promise<StudioContextData> {
  const [user, groups, rates] = await Promise.all([
    userAPI.getProfile(),
    userGroupsAPI.getAvailable(),
    userGroupsAPI.getUserGroupRates().catch(() => ({} as Record<number, number>)),
  ])
  const imageGroup = selectStudioImageGroup(groups || [])
  let existingKey: ApiKey | null = null
  if (imageGroup) {
    try {
      const keys = await listActiveKeysForGroup(imageGroup.id)
      existingKey = pickStudioApiKey(keys, imageGroup.id)
    } catch {
      existingKey = null
    }
  }
  return { user, imageGroup, userGroupRates: rates, existingKey }
}

const pendingKeyCreation = new Map<number, Promise<ApiKey>>()

/**
 * 懒加载解析或自动创建一个绑定到图像分组的专属画室 API 密钥。
 * 并发点击共享同一个 Promise，避免重复创建密钥。
 */
export async function resolveOrCreateStudioApiKey(
  groupId: number,
  cachedKey: ApiKey | null
): Promise<ApiKey> {
  if (cachedKey && isEligibleStudioKey(cachedKey, groupId)) {
    return cachedKey
  }
  const existing = pendingKeyCreation.get(groupId)
  if (existing) return existing

  const task = (async () => {
    const keys = await listActiveKeysForGroup(groupId)
    const found = pickStudioApiKey(keys, groupId)
    if (found) return found
    const created = await keysAPI.create(STUDIO_AUTO_KEY_NAME, groupId)
    if (!created || !created.key) {
      throw new StudioApiError('自动创建画室生图凭证失败，请到「API 密钥」页面手动创建后再试。')
    }
    return created
  })().finally(() => {
    pendingKeyCreation.delete(groupId)
  })

  pendingKeyCreation.set(groupId, task)
  return task
}

export function buildGenerationRequestBody(req: StudioGenerateRequest): Record<string, unknown> {
  const m = getStudioModel(req.model)
  const payload: Record<string, unknown> = {
    model: req.model,
    prompt: req.prompt,
    size: req.apiSize,
  }

  if (m.protocol === 'gemini') {
    payload.n = 1
    payload.response_format = 'b64_json'
    return payload
  }

  payload.n = Math.max(1, Math.min(req.count, m.maxN))
  if (req.quality && m.qualityLevels.includes(req.quality)) {
    payload.quality = req.quality
  }
  if (m.supportsComplexity && req.complexity) {
    payload.complexity = req.complexity
  }
  if (m.outputFormats && req.outputFormat && m.outputFormats.includes(req.outputFormat)) {
    if (req.outputFormat !== 'png' || (m.supportsBackground && req.background === 'transparent')) {
      payload.output_format = req.outputFormat
    }
  }
  if (m.supportsBackground && req.background && req.background !== 'auto') {
    payload.background = req.background
    if (req.background === 'transparent' && !payload.output_format) {
      payload.output_format = 'png'
    }
  }
  if (
    m.supportsCompression &&
    req.outputCompression != null &&
    (req.outputFormat === 'jpeg' || req.outputFormat === 'webp')
  ) {
    payload.output_compression = req.outputCompression
  }
  if (m.supportsModeration && req.moderation && req.moderation !== 'auto') {
    payload.moderation = req.moderation
  }
  if (m.supportsGptStylePassthrough && req.gptStyle.trim()) {
    payload.style = req.gptStyle.trim()
  }
  return payload
}

export function buildEditFormData(req: StudioGenerateRequest): FormData {
  const m = getStudioModel(req.model)
  const form = new FormData()
  form.append('model', req.model)
  form.append('prompt', req.prompt)
  form.append('size', req.apiSize)

  if (m.protocol === 'gemini') {
    form.append('n', '1')
    form.append('response_format', 'b64_json')
  } else {
    form.append('n', String(Math.max(1, Math.min(req.count, m.maxN))))
    if (req.quality && m.qualityLevels.includes(req.quality)) {
      form.append('quality', req.quality)
    }
    if (m.supportsComplexity && req.complexity) {
      form.append('complexity', req.complexity)
    }
    if (m.outputFormats && req.outputFormat && m.outputFormats.includes(req.outputFormat)) {
      if (req.outputFormat !== 'png' || (m.supportsBackground && req.background === 'transparent')) {
        form.append('output_format', req.outputFormat)
      }
    }
    if (m.supportsBackground && req.background && req.background !== 'auto') {
      form.append('background', req.background)
      if (req.background === 'transparent' && !form.has('output_format')) {
        form.append('output_format', 'png')
      }
    }
    if (
      m.supportsCompression &&
      req.outputCompression != null &&
      (req.outputFormat === 'jpeg' || req.outputFormat === 'webp')
    ) {
      form.append('output_compression', String(req.outputCompression))
    }
    if (m.supportsInputFidelity && req.inputFidelity && req.inputFidelity !== 'low') {
      form.append('input_fidelity', req.inputFidelity)
    }
    if (m.supportsModeration && req.moderation && req.moderation !== 'auto') {
      form.append('moderation', req.moderation)
    }
    if (m.supportsGptStylePassthrough && req.gptStyle.trim()) {
      form.append('style', req.gptStyle.trim())
    }
  }

  const fieldName = req.refs.length === 1 ? 'image' : 'image[]'
  req.refs.forEach((ref, idx) => {
    const safeName = ref.filename || `reference-${idx + 1}.png`
    form.append(fieldName, ref.blob, safeName)
  })
  return form
}

function isSafeHttpUrl(raw: string): boolean {
  try {
    const u = new URL(raw)
    return u.protocol === 'https:' || u.protocol === 'http:'
  } catch {
    return false
  }
}

function normalizeMimeAndExt(
  formatHint: string | undefined,
  fallbackFormat: StudioOutputFormat | null
): { mimeType: string; ext: string } {
  const hint = (formatHint || fallbackFormat || 'png').toLowerCase().trim()
  if (hint === 'jpg' || hint === 'jpeg' || hint === 'image/jpeg') {
    return { mimeType: FORMAT_META.jpeg.mime, ext: FORMAT_META.jpeg.ext }
  }
  if (hint === 'webp' || hint === 'image/webp') {
    return { mimeType: FORMAT_META.webp.mime, ext: FORMAT_META.webp.ext }
  }
  return { mimeType: FORMAT_META.png.mime, ext: FORMAT_META.png.ext }
}

const ALLOWED_RASTER_MIMES = new Set(['image/png', 'image/jpeg', 'image/webp'])

export function base64ToBlob(rawBase64: string, mimeType: string): Blob {
  let clean = rawBase64.trim()
  let resolvedMime = ALLOWED_RASTER_MIMES.has(mimeType.toLowerCase())
    ? mimeType.toLowerCase()
    : FORMAT_META.png.mime
  const dataPrefix = /^data:(image\/[a-z0-9.+-]+);base64,/i.exec(clean)
  if (dataPrefix) {
    const candidate = dataPrefix[1].toLowerCase()
    if (ALLOWED_RASTER_MIMES.has(candidate)) {
      resolvedMime = candidate
    }
    // 明确拒绝 image/svg+xml 等脚本可执行格式进入同源 blob: URL
    clean = clean.slice(dataPrefix[0].length)
  }
  clean = clean.replace(/\s+/g, '')
  let binary: string
  try {
    binary = atob(clean)
  } catch {
    throw new StudioApiError('服务端返回了损坏的图像数据，请重试。')
  }
  const len = binary.length
  const bytes = new Uint8Array(len)
  for (let i = 0; i < len; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return new Blob([bytes], { type: resolvedMime })
}

function loadImageDimensionsAndThumb(
  srcUrl: string,
  maxThumbEdge = 160
): Promise<{ width: number; height: number; thumbUrl: string }> {
  return new Promise((resolve) => {
    if (typeof Image === 'undefined' || typeof document === 'undefined') {
      resolve({ width: 0, height: 0, thumbUrl: srcUrl })
      return
    }
    const isRemote = /^https?:\/\//i.test(srcUrl)
    let settled = false
    const finish = (val: { width: number; height: number; thumbUrl: string }) => {
      if (settled) return
      settled = true
      if (timer != null) clearTimeout(timer)
      resolve(val)
    }
    const timer = setTimeout(() => finish({ width: 0, height: 0, thumbUrl: srcUrl }), 5000)

    const attemptLoad = (useCors: boolean) => {
      const img = new Image()
      if (isRemote && useCors) {
        img.crossOrigin = 'anonymous'
      }
      img.onload = () => {
        const w = img.naturalWidth || img.width || 0
        const h = img.naturalHeight || img.height || 0
        let thumbUrl = srcUrl
        if (!isRemote || useCors) {
          try {
            const scale = Math.min(1, maxThumbEdge / Math.max(1, w, h))
            const canvas = document.createElement('canvas')
            canvas.width = Math.max(1, Math.round(w * scale))
            canvas.height = Math.max(1, Math.round(h * scale))
            const ctx = canvas.getContext('2d')
            if (ctx) {
              ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
              thumbUrl = canvas.toDataURL('image/jpeg', 0.72)
            }
          } catch {
            thumbUrl = srcUrl
          }
        }
        finish({ width: w, height: h, thumbUrl })
      }
      img.onerror = () => {
        if (isRemote && useCors) {
          attemptLoad(false)
        } else {
          finish({ width: 0, height: 0, thumbUrl: srcUrl })
        }
      }
      img.src = srcUrl
    }
    attemptLoad(isRemote)
  })
}

function inferStatusFromErrorCode(code: string, message: string): number {
  const lower = `${code} ${message}`.toLowerCase()
  if (lower.includes('insufficient_balance') || lower.includes('余额不足')) return 402
  if (lower.includes('api_key_quota') || lower.includes('rate_limit') || lower.includes('429')) return 429
  if (lower.includes('invalid_api_key') || lower.includes('unauthorized')) return 401
  if (lower.includes('access_denied') || lower.includes('forbidden')) return 403
  return 502
}

export function extractErrorMessage(
  status: number,
  bodyText: string
): { message: string; code?: string; isBalanceError: boolean; isAuthKeyError: boolean } {
  let rawMessage = ''
  let code = ''
  try {
    const parsed = JSON.parse(bodyText) as {
      error?: { message?: string; code?: string; type?: string }
      message?: string
      code?: string
    }
    rawMessage = parsed.error?.message || parsed.message || ''
    code = String(parsed.error?.code || parsed.error?.type || parsed.code || '')
  } catch {
    rawMessage = bodyText.trim().slice(0, 240)
  }

  const lower = `${code} ${rawMessage}`.toLowerCase()
  const isUserAccountInactive =
    lower.includes('user_inactive') ||
    lower.includes('user disabled') ||
    lower.includes('account disabled')
  const isApiKeyQuotaOrAccessError =
    lower.includes('api_key_quota') ||
    lower.includes('api key quota') ||
    lower.includes('key_quota') ||
    lower.includes('key quota') ||
    (status === 403 && (lower.includes('access_denied') || lower.includes('ip') || lower.includes('api key')))
  const isAuthKeyError =
    !isUserAccountInactive &&
    (isApiKeyQuotaOrAccessError ||
      lower.includes('invalid_api_key') ||
      lower.includes('api_key_expired') ||
      lower.includes('api_key_disabled') ||
      (status === 401 &&
        (lower.includes('api key') ||
          lower.includes('invalid') ||
          lower.includes('expired') ||
          lower.includes('disabled') ||
          lower.includes('unauthorized') ||
          !rawMessage)))
  const isBalanceError =
    !isAuthKeyError &&
    (status === 402 ||
      lower.includes('insufficient_balance') ||
      lower.includes('insufficient balance') ||
      lower.includes('insufficient_quota') ||
      lower.includes('balance') ||
      lower.includes('余额不足'))

  if (isAuthKeyError) {
    return {
      message: '画室生图凭证已失效或达到限额，系统正在刷新凭证，请稍后重试。',
      code,
      isBalanceError: false,
      isAuthKeyError: true,
    }
  }
  if (isBalanceError) {
    return {
      message: rawMessage || '账户余额不足，请充值后重试（已保留您的草稿与参考图）。',
      code,
      isBalanceError: true,
      isAuthKeyError: false,
    }
  }
  if (status === 429) {
    return {
      message: rawMessage || '当前请求过于频繁，请稍等片刻后重试。',
      code,
      isBalanceError: false,
      isAuthKeyError: false,
    }
  }
  if (status === 413) {
    return {
      message: '参考图体积过大，请适当压缩或减少参考图数量后重试。',
      code,
      isBalanceError: false,
      isAuthKeyError: false,
    }
  }
  if (status >= 500) {
    return {
      message: rawMessage || `上游生图服务暂时繁忙（HTTP ${status}），未扣费，请稍后重试。`,
      code,
      isBalanceError: false,
      isAuthKeyError: false,
    }
  }
  return {
    message: rawMessage || `生成失败（HTTP ${status}），请检查参数后重试。`,
    code,
    isBalanceError: false,
    isAuthKeyError: false,
  }
}

export async function generateStudioImages(
  req: StudioGenerateRequest
): Promise<StudioGeneratedImage[]> {
  const isEdit = req.refs.length > 0
  const endpoint = buildGatewayUrl(isEdit ? '/v1/images/edits' : '/v1/images/generations')

  const headers: Record<string, string> = {
    Authorization: `Bearer ${req.apiKey}`,
  }
  let body: BodyInit
  if (isEdit) {
    body = buildEditFormData(req)
  } else {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(buildGenerationRequestBody(req))
  }

  let response: Response
  try {
    response = await fetch(endpoint, {
      method: 'POST',
      headers,
      body,
      signal: req.signal,
    })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') {
      throw new StudioApiError('请求已取消。')
    }
    throw new StudioApiError('网络连接异常或请求超时，请稍后重试（您的草稿与参考图已保留）。')
  }

  const responseText = await response.text()
  if (!response.ok) {
    const info = extractErrorMessage(response.status, responseText)
    throw new StudioApiError(info.message, {
      status: response.status,
      code: info.code,
      isBalanceError: info.isBalanceError,
      isAuthKeyError: info.isAuthKeyError,
    })
  }

  let parsed: {
    error?: { message?: string; code?: string; type?: string }
    output_format?: string
    data?: Array<{
      b64_json?: string
      url?: string
      output_format?: string
      revised_prompt?: string
    }>
  }
  try {
    parsed = JSON.parse(responseText)
  } catch {
    throw new StudioApiError('服务端返回了无法解析的响应格式，请稍后重试。')
  }

  // 后端长请求保活（openai_images_json_keepalive.go）在 15s 心跳后会先提交 HTTP 200，
  // 后续上游失败会在 200 JSON 体内返回 {"error": {...}}，此处必须显式识别并透传真实错误。
  if (parsed.error) {
    const inferredStatus = inferStatusFromErrorCode(
      String(parsed.error.code || parsed.error.type || ''),
      String(parsed.error.message || '')
    )
    const info = extractErrorMessage(inferredStatus, responseText)
    throw new StudioApiError(info.message, {
      status: inferredStatus,
      code: info.code,
      isBalanceError: info.isBalanceError,
      isAuthKeyError: info.isAuthKeyError,
    })
  }

  const items = Array.isArray(parsed.data) ? parsed.data : []
  if (!items.length) {
    throw new StudioApiError('模型未返回任何图片，请调整描述词或切换模型后重试。')
  }

  const results: StudioGeneratedImage[] = []
  const createdObjectUrls: string[] = []
  try {
    for (const item of items) {
      const fmt = normalizeMimeAndExt(item.output_format || parsed.output_format, req.outputFormat)
      const inlineData =
        item.b64_json && item.b64_json.trim()
          ? item.b64_json.trim()
          : item.url && /^data:image\/[a-z0-9.+-]+;base64,/i.test(item.url.trim())
            ? item.url.trim()
            : ''
      if (inlineData) {
        const blob = base64ToBlob(inlineData, fmt.mimeType)
        const objectUrl = URL.createObjectURL(blob)
        createdObjectUrls.push(objectUrl)
        const meta = await loadImageDimensionsAndThumb(objectUrl, 160)
        results.push({
          objectUrl,
          blob,
          thumbUrl: meta.thumbUrl,
          mimeType: blob.type || fmt.mimeType,
          ext: fmt.ext,
          width: meta.width,
          height: meta.height,
          revisedPrompt: item.revised_prompt,
        })
        continue
      }
      if (item.url && isSafeHttpUrl(item.url.trim())) {
        const remoteUrl = item.url.trim()
        const meta = await loadImageDimensionsAndThumb(remoteUrl, 160)
        results.push({
          objectUrl: remoteUrl,
          remoteUrl,
          blob: null,
          thumbUrl: meta.thumbUrl,
          mimeType: fmt.mimeType,
          ext: fmt.ext,
          width: meta.width,
          height: meta.height,
          revisedPrompt: item.revised_prompt,
        })
      }
    }
  } catch (err) {
    for (const u of createdObjectUrls) {
      URL.revokeObjectURL(u)
    }
    throw err
  }

  if (!results.length) {
    throw new StudioApiError('未能提取有效的生成图像数据，请重试。')
  }
  return results
}

const SUPPORTED_UPLOAD_MIMES = new Set(['image/png', 'image/jpeg', 'image/webp'])

/**
 * 预处理用户上传/粘贴/拖入的本地参考图：
 * - 校验图片类型与体积上限（20MB）
 * - PNG / JPEG / WebP 直接保留原文件 Blob（高保真送入 /v1/images/edits），其余图片格式转 PNG
 * - 生成轻量 160px 缩略图用于底部参考槽显示
 */
export async function prepareUploadReference(
  file: File
): Promise<{ blob: Blob; filename: string; mimeType: string; thumbUrl: string }> {
  if (!file.type || !file.type.startsWith('image/')) {
    throw new Error('请提供图片文件（PNG / JPEG / WebP）')
  }
  if (file.size > MAX_REF_UPLOAD_BYTES) {
    throw new Error(`单张参考图不能超过 ${Math.round(MAX_REF_UPLOAD_BYTES / (1024 * 1024))}MB`)
  }

  const tempUrl = URL.createObjectURL(file)
  try {
    const img = await new Promise<HTMLImageElement>((resolve, reject) => {
      const el = new Image()
      el.onload = () => resolve(el)
      el.onerror = () => reject(new Error('无法读取该图片文件'))
      el.src = tempUrl
    })

    const w = img.naturalWidth || img.width
    const h = img.naturalHeight || img.height
    const scale = Math.min(1, 160 / Math.max(1, w, h))
    const thumbCanvas = document.createElement('canvas')
    thumbCanvas.width = Math.max(1, Math.round(w * scale))
    thumbCanvas.height = Math.max(1, Math.round(h * scale))
    const tCtx = thumbCanvas.getContext('2d')
    if (tCtx) {
      tCtx.drawImage(img, 0, 0, thumbCanvas.width, thumbCanvas.height)
    }
    const thumbUrl = thumbCanvas.toDataURL('image/jpeg', 0.78)

    if (SUPPORTED_UPLOAD_MIMES.has(file.type.toLowerCase())) {
      return {
        blob: file,
        filename: file.name || 'reference.png',
        mimeType: file.type.toLowerCase(),
        thumbUrl,
      }
    }

    // 非常规图片格式（如 BMP/GIF/AVIF）转为标准 PNG，兼容 Gemini/OpenAI 上传校验
    const outCanvas = document.createElement('canvas')
    outCanvas.width = w
    outCanvas.height = h
    const oCtx = outCanvas.getContext('2d')
    if (oCtx) {
      oCtx.drawImage(img, 0, 0, w, h)
    }
    const pngBlob = await new Promise<Blob | null>((resolve) =>
      outCanvas.toBlob((b) => resolve(b), 'image/png')
    )
    if (!pngBlob) {
      throw new Error('图片格式转换失败')
    }
    const baseName = (file.name || 'reference').replace(/\.[^.]+$/, '')
    return {
      blob: pngBlob,
      filename: `${baseName}.png`,
      mimeType: 'image/png',
      thumbUrl,
    }
  } finally {
    URL.revokeObjectURL(tempUrl)
  }
}

/**
 * 将画室历史结果转为可直接送入 `/v1/images/edits` 的高保真 Blob。
 */
export async function resolveImageToBlob(image: StudioGeneratedImage): Promise<Blob> {
  if (image.blob) return image.blob
  const src = image.remoteUrl || image.objectUrl
  let resp: Response
  try {
    resp = await fetch(src)
  } catch {
    throw new StudioApiError('无法读取该历史图片（受跨域限制或链接已失效），请先下载原图后拖入上传。')
  }
  if (!resp.ok) {
    throw new StudioApiError('无法读取该历史图片内容，请先下载后手动上传作为参考图。')
  }
  return resp.blob()
}
