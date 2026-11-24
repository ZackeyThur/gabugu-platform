/**
 * GABUGU 画室 — 按用户隔离的本机 IndexedDB 历史持久化与切页保活运行时。
 *
 * 解决用户离开 `/studio`（返回控制台、刷新页面或重新打开标签页）后历史作品丢失的问题：
 * 1. 每张生成作品的原图 Blob、缩略图、提示词、模型/比例/清晰度参数、收藏状态与选中图序
 *    按 `userId` 隔离写入 IndexedDB，绝不跨账号串数据，也绝不写入任何 API 密钥或令牌；
 * 2. 离开 `/studio` 时若仍有生图任务在途，后台继续等待完成并直接落盘到该用户的 IndexedDB，
 *    用户切回画室即可无缝看到进度或成品；
 * 3. 超过容量预算或触发浏览器配额上限时，优先保留带星标收藏的作品，自动淘汰最旧的未收藏记录；
 * 4. 在禁止 IndexedDB 的严格隐私模式下安全降级为内存模式，零崩溃。
 */

import type {
  StudioBackground,
  StudioComplexity,
  StudioInputFidelity,
  StudioModeration,
  StudioOrientation,
  StudioOutputFormat,
  StudioTierId,
} from './models'
import type { StudioGeneratedImage } from './api'

export interface StudioVersion {
  id: number
  createdAt: number
  prompt: string
  model: string
  baseRatio: string
  orientation: StudioOrientation
  ratioCode: string
  tier: StudioTierId
  apiSize: string
  dimensions: string
  w: number
  h: number
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
  branchLabel: string
  cost: number
  costKnown: boolean
  images: StudioGeneratedImage[]
  favs: boolean[]
  selectedIdx: number
}

export interface StudioPendingJob {
  prompt: string
  model: string
  ratioCode: string
  tier: StudioTierId
  w: number
  h: number
  count: number
  refCount: number
  branchLabel: string
  cost: number
  costKnown: boolean
}

export interface StudioStoredImage {
  blob: Blob | null
  remoteUrl?: string
  thumbUrl: string
  mimeType: string
  ext: string
  width: number
  height: number
  revisedPrompt?: string
}

export interface StudioStoredVersion {
  storageKey: string
  userId: number
  id: number
  createdAt: number
  prompt: string
  model: string
  baseRatio: string
  orientation: StudioOrientation
  ratioCode: string
  tier: StudioTierId
  apiSize: string
  dimensions: string
  w: number
  h: number
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
  branchLabel: string
  cost: number
  costKnown: boolean
  images: StudioStoredImage[]
  favs: boolean[]
  selectedIdx: number
  byteSize: number
}

interface StudioStoredMeta {
  userId: number
  draftPrompt: string
  updatedAt: number
}

export interface StudioHydratedHistory {
  versions: StudioVersion[]
  nextVersionId: number
  totalCost: number
  draftPrompt: string
  storageAvailable: boolean
}

const DB_NAME = 'gabugu-studio-history-v1'
const DB_VERSION = 1
const STORE_VERSIONS = 'versions'
const STORE_META = 'meta'
const INDEX_BY_USER = 'byUser'

export const MAX_STUDIO_VERSIONS_PER_USER = 60
export const MAX_STUDIO_BYTES_PER_USER = 250 * 1024 * 1024 // 250 MB

const SAFE_RASTER_MIMES = new Set(['image/png', 'image/jpeg', 'image/webp'])
const SAFE_DATA_IMAGE_PREFIX = /^data:image\/(?:png|jpeg|jpg|webp);base64,/i

function isValidUserId(userId: number | null | undefined): userId is number {
  return typeof userId === 'number' && Number.isInteger(userId) && userId > 0
}

function isSafeHttpUrl(raw: string | undefined): raw is string {
  if (!raw) return false
  try {
    const u = new URL(raw)
    return u.protocol === 'https:' || u.protocol === 'http:'
  } catch {
    return false
  }
}

function isSafeThumbUrl(raw: string | undefined): raw is string {
  if (!raw) return false
  if (SAFE_DATA_IMAGE_PREFIX.test(raw)) return true
  return isSafeHttpUrl(raw)
}

/**
 * 无论读写，都强制确保 Blob MIME 属于光栅白名单（阻断 image/svg+xml 进入同源 blob: URL）。
 */
export function sanitizeRasterBlob(blob: Blob, fallbackMime = 'image/png'): Blob {
  const normalized = (blob.type || '').toLowerCase().trim()
  if (SAFE_RASTER_MIMES.has(normalized)) {
    return blob
  }
  const safeFallback = SAFE_RASTER_MIMES.has(fallbackMime.toLowerCase())
    ? fallbackMime.toLowerCase()
    : 'image/png'
  return new Blob([blob], { type: safeFallback })
}

function makeStorageKey(userId: number, versionId: number): string {
  return `${userId}:${versionId}`
}

function estimateRecordBytes(images: StudioStoredImage[]): number {
  let total = 0
  for (const img of images) {
    if (img.blob) total += img.blob.size
    if (img.thumbUrl && img.thumbUrl.startsWith('data:')) {
      total += img.thumbUrl.length
    }
  }
  return total
}

function serializeVersion(userId: number, v: StudioVersion): StudioStoredVersion {
  const images: StudioStoredImage[] = v.images.map((img) => {
    const safeBlob = img.blob ? sanitizeRasterBlob(img.blob, img.mimeType) : null
    const safeRemote = isSafeHttpUrl(img.remoteUrl) ? img.remoteUrl : undefined
    // 不保存临时的 blob: URL（跨刷新即失效），仅保存安全 data: 缩略图或远程链接
    const safeThumb = isSafeThumbUrl(img.thumbUrl) ? img.thumbUrl : ''
    return {
      blob: safeBlob,
      remoteUrl: safeRemote,
      thumbUrl: safeThumb,
      mimeType: safeBlob?.type || (SAFE_RASTER_MIMES.has(img.mimeType.toLowerCase()) ? img.mimeType.toLowerCase() : 'image/png'),
      ext: img.ext || 'png',
      width: img.width || 0,
      height: img.height || 0,
      revisedPrompt: img.revisedPrompt,
    }
  })

  return {
    storageKey: makeStorageKey(userId, v.id),
    userId,
    id: v.id,
    createdAt: v.createdAt || Date.now(),
    prompt: v.prompt,
    model: v.model,
    baseRatio: v.baseRatio,
    orientation: v.orientation,
    ratioCode: v.ratioCode,
    tier: v.tier,
    apiSize: v.apiSize,
    dimensions: v.dimensions,
    w: v.w,
    h: v.h,
    count: v.count,
    quality: v.quality,
    complexity: v.complexity,
    outputFormat: v.outputFormat,
    background: v.background,
    outputCompression: v.outputCompression,
    inputFidelity: v.inputFidelity,
    moderation: v.moderation,
    gptStyle: v.gptStyle,
    refCount: v.refCount,
    branchLabel: v.branchLabel,
    cost: v.cost,
    costKnown: v.costKnown,
    images,
    favs: [...v.favs],
    selectedIdx: v.selectedIdx,
    byteSize: estimateRecordBytes(images),
  }
}

export function hydrateStoredVersion(
  record: StudioStoredVersion,
  mintedObjectUrls: string[]
): StudioVersion | null {
  if (!record || !Array.isArray(record.images)) return null
  const hydratedImages: StudioGeneratedImage[] = []
  for (const img of record.images) {
    if (img.blob instanceof Blob) {
      const safeBlob = sanitizeRasterBlob(img.blob, img.mimeType)
      const objectUrl = URL.createObjectURL(safeBlob)
      mintedObjectUrls.push(objectUrl)
      hydratedImages.push({
        objectUrl,
        blob: safeBlob,
        remoteUrl: isSafeHttpUrl(img.remoteUrl) ? img.remoteUrl : undefined,
        thumbUrl: isSafeThumbUrl(img.thumbUrl) ? img.thumbUrl : objectUrl,
        mimeType: safeBlob.type || 'image/png',
        ext: img.ext || 'png',
        width: img.width || 0,
        height: img.height || 0,
        revisedPrompt: img.revisedPrompt,
      })
      continue
    }
    if (isSafeHttpUrl(img.remoteUrl)) {
      hydratedImages.push({
        objectUrl: img.remoteUrl,
        remoteUrl: img.remoteUrl,
        blob: null,
        thumbUrl: isSafeThumbUrl(img.thumbUrl) ? img.thumbUrl : img.remoteUrl,
        mimeType: SAFE_RASTER_MIMES.has((img.mimeType || '').toLowerCase())
          ? img.mimeType.toLowerCase()
          : 'image/png',
        ext: img.ext || 'png',
        width: img.width || 0,
        height: img.height || 0,
        revisedPrompt: img.revisedPrompt,
      })
    }
  }
  if (!hydratedImages.length) return null

  const favs = hydratedImages.map((_, idx) => Boolean(record.favs?.[idx]))
  const selectedIdx =
    typeof record.selectedIdx === 'number' &&
    record.selectedIdx >= 0 &&
    record.selectedIdx < hydratedImages.length
      ? record.selectedIdx
      : 0

  return {
    id: record.id,
    createdAt: record.createdAt || 0,
    prompt: record.prompt || '',
    model: record.model,
    baseRatio: record.baseRatio,
    orientation: record.orientation,
    ratioCode: record.ratioCode,
    tier: record.tier,
    apiSize: record.apiSize,
    dimensions: record.dimensions,
    w: record.w,
    h: record.h,
    count: hydratedImages.length,
    quality: record.quality ?? null,
    complexity: record.complexity ?? null,
    outputFormat: record.outputFormat ?? null,
    background: record.background ?? null,
    outputCompression: record.outputCompression ?? null,
    inputFidelity: record.inputFidelity ?? null,
    moderation: record.moderation ?? null,
    gptStyle: record.gptStyle || '',
    refCount: record.refCount || 0,
    branchLabel: record.branchLabel || '',
    cost: typeof record.cost === 'number' && Number.isFinite(record.cost) ? record.cost : 0,
    costKnown: Boolean(record.costKnown),
    images: hydratedImages,
    favs,
    selectedIdx,
  }
}

/**
 * 依据容量预算挑出需要淘汰的旧记录：
 * - 永不淘汰 protectedKey（刚刚保存的新版本）
 * - 优先淘汰最旧的「未收藏」版本，保护用户点亮星标的作品
 * - 仅当所有剩余记录都已收藏且仍超预算时，才淘汰最旧的收藏版本
 */
export function selectVersionsToPrune(
  records: ReadonlyArray<Pick<StudioStoredVersion, 'storageKey' | 'createdAt' | 'id' | 'favs' | 'byteSize'>>,
  protectedKey: string,
  maxCount = MAX_STUDIO_VERSIONS_PER_USER,
  maxBytes = MAX_STUDIO_BYTES_PER_USER
): string[] {
  let remainingCount = records.length
  let remainingBytes = records.reduce((sum, r) => sum + Math.max(0, r.byteSize || 0), 0)
  if (remainingCount <= maxCount && remainingBytes <= maxBytes) {
    return []
  }

  const ordered = [...records]
    .filter((r) => r.storageKey !== protectedKey)
    .sort((a, b) => {
      const aFav = Array.isArray(a.favs) && a.favs.some(Boolean) ? 1 : 0
      const bFav = Array.isArray(b.favs) && b.favs.some(Boolean) ? 1 : 0
      if (aFav !== bFav) return aFav - bFav // 未收藏(0)先于已收藏(1)淘汰
      if (a.createdAt !== b.createdAt) return a.createdAt - b.createdAt // 最旧先淘汰
      return a.id - b.id
    })

  const toDelete: string[] = []
  for (const rec of ordered) {
    if (remainingCount <= maxCount && remainingBytes <= maxBytes) break
    toDelete.push(rec.storageKey)
    remainingCount -= 1
    remainingBytes -= Math.max(0, rec.byteSize || 0)
  }
  return toDelete
}

let dbPromise: Promise<IDBDatabase | null> | null = null

export function resetStudioHistoryDbForTests(): void {
  dbPromise = null
}

function openStudioHistoryDb(): Promise<IDBDatabase | null> {
  if (typeof indexedDB === 'undefined') {
    return Promise.resolve(null)
  }
  if (dbPromise) return dbPromise

  dbPromise = new Promise<IDBDatabase | null>((resolve) => {
    try {
      const req = indexedDB.open(DB_NAME, DB_VERSION)
      req.onupgradeneeded = () => {
        const db = req.result
        if (!db.objectStoreNames.contains(STORE_VERSIONS)) {
          const store = db.createObjectStore(STORE_VERSIONS, { keyPath: 'storageKey' })
          store.createIndex(INDEX_BY_USER, 'userId', { unique: false })
        }
        if (!db.objectStoreNames.contains(STORE_META)) {
          db.createObjectStore(STORE_META, { keyPath: 'userId' })
        }
      }
      req.onsuccess = () => {
        const db = req.result
        db.onversionchange = () => {
          db.close()
          dbPromise = null
        }
        resolve(db)
      }
      req.onerror = () => {
        dbPromise = null
        resolve(null)
      }
      req.onblocked = () => {
        dbPromise = null
        resolve(null)
      }
    } catch {
      dbPromise = null
      resolve(null)
    }
  })

  return dbPromise
}

function reqToPromise<T>(req: IDBRequest<T>): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error || new Error('IndexedDB request failed'))
  })
}

function txDone(tx: IDBTransaction): Promise<void> {
  return new Promise<void>((resolve, reject) => {
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error || new Error('IndexedDB transaction failed'))
    tx.onabort = () => reject(tx.error || new Error('IndexedDB transaction aborted'))
  })
}

async function listUserRecords(db: IDBDatabase, userId: number): Promise<StudioStoredVersion[]> {
  const tx = db.transaction(STORE_VERSIONS, 'readonly')
  const index = tx.objectStore(STORE_VERSIONS).index(INDEX_BY_USER)
  const rows = (await reqToPromise(index.getAll(IDBKeyRange.only(userId)))) as StudioStoredVersion[]
  return Array.isArray(rows) ? rows : []
}

export async function loadStudioHistory(userId: number): Promise<StudioHydratedHistory> {
  if (!isValidUserId(userId)) {
    return { versions: [], nextVersionId: 1, totalCost: 0, draftPrompt: '', storageAvailable: false }
  }
  const db = await openStudioHistoryDb()
  if (!db) {
    return { versions: [], nextVersionId: 1, totalCost: 0, draftPrompt: '', storageAvailable: false }
  }

  const mintedUrls: string[] = []
  try {
    const tx = db.transaction([STORE_VERSIONS, STORE_META], 'readonly')
    const versionsIdx = tx.objectStore(STORE_VERSIONS).index(INDEX_BY_USER)
    const metaStore = tx.objectStore(STORE_META)
    const [rawRows, rawMeta] = await Promise.all([
      reqToPromise(versionsIdx.getAll(IDBKeyRange.only(userId))) as Promise<StudioStoredVersion[]>,
      reqToPromise(metaStore.get(userId)) as Promise<StudioStoredMeta | undefined>,
    ])

    const sorted = (Array.isArray(rawRows) ? rawRows : []).sort((a, b) => {
      if ((a.createdAt || 0) !== (b.createdAt || 0)) return (a.createdAt || 0) - (b.createdAt || 0)
      return (a.id || 0) - (b.id || 0)
    })

    const versions: StudioVersion[] = []
    let maxId = 0
    let totalCost = 0
    for (const row of sorted) {
      if (typeof row.id === 'number' && row.id > maxId) maxId = row.id
      const hydrated = hydrateStoredVersion(row, mintedUrls)
      if (!hydrated) continue
      versions.push(hydrated)
      if (hydrated.costKnown && hydrated.cost > 0) {
        totalCost = Math.round((totalCost + hydrated.cost) * 10000) / 10000
      }
    }

    return {
      versions,
      nextVersionId: maxId + 1,
      totalCost,
      draftPrompt: typeof rawMeta?.draftPrompt === 'string' ? rawMeta.draftPrompt : '',
      storageAvailable: true,
    }
  } catch {
    for (const u of mintedUrls) {
      URL.revokeObjectURL(u)
    }
    return { versions: [], nextVersionId: 1, totalCost: 0, draftPrompt: '', storageAvailable: false }
  }
}

function isQuotaExceededError(err: unknown): boolean {
  if (!err || typeof err !== 'object') return false
  const name = (err as { name?: string }).name || ''
  return name === 'QuotaExceededError' || name === 'NS_ERROR_DOM_QUOTA_REACHED'
}

export async function saveStudioVersion(
  userId: number,
  version: StudioVersion
): Promise<{ ok: boolean; prunedCount: number }> {
  if (!isValidUserId(userId)) return { ok: false, prunedCount: 0 }
  const db = await openStudioHistoryDb()
  if (!db) return { ok: false, prunedCount: 0 }

  const record = serializeVersion(userId, version)

  const writeAndPrune = async (maxCount: number, maxBytes: number): Promise<number> => {
    const existing = await listUserRecords(db, userId)
    const combined = [
      ...existing.filter((r) => r.storageKey !== record.storageKey),
      record,
    ]
    const toDelete = selectVersionsToPrune(combined, record.storageKey, maxCount, maxBytes)
    const tx = db.transaction(STORE_VERSIONS, 'readwrite')
    const store = tx.objectStore(STORE_VERSIONS)
    for (const key of toDelete) {
      store.delete(key)
    }
    store.put(record)
    await txDone(tx)
    return toDelete.length
  }

  try {
    const prunedCount = await writeAndPrune(MAX_STUDIO_VERSIONS_PER_USER, MAX_STUDIO_BYTES_PER_USER)
    return { ok: true, prunedCount }
  } catch (err) {
    if (isQuotaExceededError(err)) {
      try {
        // 触发浏览器磁盘配额上限时，收紧保留条目并重写入当前新版本
        const existing = await listUserRecords(db, userId)
        const targetCount = Math.max(1, Math.floor(existing.length / 2))
        const prunedCount = await writeAndPrune(targetCount, Math.floor(MAX_STUDIO_BYTES_PER_USER / 2))
        return { ok: true, prunedCount }
      } catch {
        return { ok: false, prunedCount: 0 }
      }
    }
    return { ok: false, prunedCount: 0 }
  }
}

export async function clearStudioHistory(userId: number): Promise<boolean> {
  if (!isValidUserId(userId)) return false
  const db = await openStudioHistoryDb()
  if (!db) return false

  try {
    const existing = await listUserRecords(db, userId)
    const tx = db.transaction([STORE_VERSIONS, STORE_META], 'readwrite')
    const versionStore = tx.objectStore(STORE_VERSIONS)
    for (const rec of existing) {
      versionStore.delete(rec.storageKey)
    }
    tx.objectStore(STORE_META).delete(userId)
    await txDone(tx)
    return true
  } catch {
    return false
  }
}

export async function saveStudioDraft(userId: number, draftPrompt: string): Promise<boolean> {
  if (!isValidUserId(userId)) return false
  const db = await openStudioHistoryDb()
  if (!db) return false

  try {
    const tx = db.transaction(STORE_META, 'readwrite')
    const store = tx.objectStore(STORE_META)
    const trimmed = draftPrompt.slice(0, 8000)
    if (!trimmed.trim()) {
      store.delete(userId)
    } else {
      const record: StudioStoredMeta = {
        userId,
        draftPrompt: trimmed,
        updatedAt: Date.now(),
      }
      store.put(record)
    }
    await txDone(tx)
    return true
  } catch {
    return false
  }
}

export function revokeVersionObjectUrls(v: StudioVersion): void {
  for (const img of v.images) {
    if (img.objectUrl && img.objectUrl.startsWith('blob:')) {
      URL.revokeObjectURL(img.objectUrl)
    }
  }
}

// ==========================================================================
// 切页在途生图保活（仅存内存，按 userId 隔离，不存任何 API 密钥）
// ==========================================================================

export interface StudioJobListener {
  onSuccess: (version: StudioVersion) => void
  onError: (err: unknown) => void
}

interface ActiveStudioJob {
  userId: number
  pendingJob: StudioPendingJob
  listener: StudioJobListener | null
}

const activeJobsByUser = new Map<number, ActiveStudioJob>()

export function getActiveStudioJob(userId: number): StudioPendingJob | null {
  if (!isValidUserId(userId)) return null
  return activeJobsByUser.get(userId)?.pendingJob ?? null
}

export function attachStudioJobListener(userId: number, listener: StudioJobListener | null): void {
  if (!isValidUserId(userId)) return
  const job = activeJobsByUser.get(userId)
  if (job) {
    job.listener = listener
  }
}

/**
 * 在模块生命周期内托管一次生图任务：
 * 即使中途离开 `/studio`，任务完成后也会立即把成品写入 IndexedDB，
 * 若页面仍在则直接回调渲染，若已离开则安全释放临时 objectUrl 等用户返回时从 IndexedDB 恢复。
 */
export function runManagedStudioJob(
  userId: number,
  pendingJob: StudioPendingJob,
  listener: StudioJobListener,
  execute: () => Promise<StudioVersion>
): Promise<void> {
  const entry: ActiveStudioJob = { userId, pendingJob, listener }
  activeJobsByUser.set(userId, entry)

  return (async () => {
    try {
      const version = await execute()
      // 先完成 IndexedDB 落盘，再清理 activeJobsByUser，消除切页重挂载竞态窗口
      await saveStudioVersion(userId, version)
      // 成功后清理草稿
      await saveStudioDraft(userId, '')
      if (activeJobsByUser.get(userId) === entry) {
        activeJobsByUser.delete(userId)
      }
      if (entry.listener) {
        entry.listener.onSuccess(version)
      } else {
        // 用户已离开画室：原图 Blob 已安全存入 IndexedDB，立即释放本次临时 blob: URL 防泄漏
        revokeVersionObjectUrls(version)
      }
    } catch (err) {
      if (activeJobsByUser.get(userId) === entry) {
        activeJobsByUser.delete(userId)
      }
      if (entry.listener) {
        entry.listener.onError(err)
      }
    }
  })()
}
