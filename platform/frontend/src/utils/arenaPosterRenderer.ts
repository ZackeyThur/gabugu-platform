import QRCode from 'qrcode'
import type { ArenaCategoryBoard, ArenaCategoryId, ArenaModelEntry } from '@/api/arena'
import { wordmarkPaths } from '@/components/brand/wordmarkPaths'
import { formatArenaScore, formatScoreDelta, resolveArenaOrgMeta } from '@/utils/arenaOrg'

export const POSTER_WIDTH = 1080
export const POSTER_HEIGHT = 1760

export type ArenaPosterMode = 'top10' | 'overview'
export type ArenaPosterTheme = 'dark' | 'light'

export interface ArenaPosterLabels {
  categoryLabels: Record<ArenaCategoryId, string>
  top10Suffix: string
  overviewTitle: string
  overviewSubtitle: string
  publishDateLabel: string
  baselineDateLabel: string
  modelCountText: (count: number) => string
  openBadge: string
  newBadge: string
  footerScanHint: string
  footerSource: string
  lastCheckedLabel: string
}

export interface RenderArenaPosterOptions {
  mode: ArenaPosterMode
  theme: ArenaPosterTheme
  siteName: string
  siteLogoUrl?: string
  shareUrl: string
  displayUrl: string
  lastCheckedText: string
  activeBoard: ArenaCategoryBoard
  allBoards: ArenaCategoryBoard[]
  labels: ArenaPosterLabels
}

interface Palette {
  bg: string
  glowBlue: string
  glowGold: string
  gridLine: string
  championWash: string
  cardBg: string
  barTrack: string
  barFill: string
  text: string
  text2: string
  muted: string
  faint: string
  line: string
  up: string
  down: string
  newTag: string
  openText: string
  openBg: string
  gold: string
  silver: string
  bronze: string
  wordmark: string
  qrBorder: string
}

const DARK: Palette = {
  bg: '#07080d',
  glowBlue: 'rgba(91, 140, 255, 0.14)',
  glowGold: 'rgba(230, 198, 138, 0.08)',
  gridLine: 'rgba(255, 255, 255, 0.025)',
  championWash: 'rgba(230, 198, 138, 0.065)',
  cardBg: 'rgba(255, 255, 255, 0.026)',
  barTrack: 'rgba(255, 255, 255, 0.08)',
  barFill: 'rgba(184, 198, 224, 0.45)',
  text: '#f3f5f9',
  text2: '#c7ccd8',
  muted: '#8b92a1',
  faint: '#5d6473',
  line: 'rgba(255, 255, 255, 0.09)',
  up: '#3dd68c',
  down: '#ff6b70',
  newTag: '#78a9ff',
  openText: '#3dd68c',
  openBg: 'rgba(61, 214, 140, 0.13)',
  gold: '#e6c68a',
  silver: '#cfd5e0',
  bronze: '#cf9f78',
  wordmark: '#ffffff',
  qrBorder: 'rgba(255, 255, 255, 0.14)'
}

const LIGHT: Palette = {
  bg: '#fafbfd',
  glowBlue: 'rgba(37, 99, 235, 0.075)',
  glowGold: 'rgba(166, 114, 34, 0.055)',
  gridLine: 'rgba(15, 23, 42, 0.026)',
  championWash: 'rgba(166, 114, 34, 0.055)',
  cardBg: 'rgba(255, 255, 255, 0.85)',
  barTrack: 'rgba(15, 23, 42, 0.075)',
  barFill: 'rgba(71, 85, 105, 0.36)',
  text: '#0b1220',
  text2: '#334155',
  muted: '#5f6c80',
  faint: '#8e9aaf',
  line: 'rgba(15, 23, 42, 0.09)',
  up: '#059669',
  down: '#e11d48',
  newTag: '#2563eb',
  openText: '#059669',
  openBg: 'rgba(5, 150, 105, 0.09)',
  gold: '#a67222',
  silver: '#5b6678',
  bronze: '#995c32',
  wordmark: '#101723',
  qrBorder: 'rgba(15, 23, 42, 0.1)'
}

const FONT = "Inter, 'Gabugu Arena SC', 'Gabugu Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif"
const PAD = 72
const CONTENT_W = POSTER_WIDTH - PAD * 2

export function truncateCanvasText(ctx: CanvasRenderingContext2D, text: string, maxWidth: number): string {
  if (!text) return ''
  if (ctx.measureText(text).width <= maxWidth) return text
  const ellipsis = '…'
  if (ctx.measureText(ellipsis).width >= maxWidth) return ellipsis
  let low = 0
  let high = text.length
  while (low < high) {
    const mid = Math.ceil((low + high) / 2)
    if (ctx.measureText(text.slice(0, mid)).width + ctx.measureText(ellipsis).width <= maxWidth) low = mid
    else high = mid - 1
  }
  return text.slice(0, Math.max(1, low)) + ellipsis
}

export function buildPosterFilename(mode: ArenaPosterMode, categoryId: ArenaCategoryId, publishDate?: string): string {
  const dateSlug = (publishDate || new Date().toISOString().slice(0, 10)).replace(/[^0-9-]/g, '')
  return `gabugu-arena-${mode === 'overview' ? 'overview' : categoryId}-${dateSlug}.png`
}

function roundRect(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number): void {
  const radius = Math.min(r, w / 2, h / 2)
  ctx.beginPath()
  ctx.moveTo(x + radius, y)
  ctx.lineTo(x + w - radius, y)
  ctx.quadraticCurveTo(x + w, y, x + w, y + radius)
  ctx.lineTo(x + w, y + h - radius)
  ctx.quadraticCurveTo(x + w, y + h, x + w - radius, y + h)
  ctx.lineTo(x + radius, y + h)
  ctx.quadraticCurveTo(x, y + h, x, y + h - radius)
  ctx.lineTo(x, y + radius)
  ctx.quadraticCurveTo(x, y, x + radius, y)
  ctx.closePath()
}

function hairline(ctx: CanvasRenderingContext2D, y: number, color: string, x1 = PAD, x2 = POSTER_WIDTH - PAD): void {
  ctx.save()
  ctx.beginPath()
  ctx.moveTo(x1, y + 0.5)
  ctx.lineTo(x2, y + 0.5)
  ctx.strokeStyle = color
  ctx.lineWidth = 1
  ctx.stroke()
  ctx.restore()
}

function loadImageSafe(src: string, timeoutMs = 2500): Promise<HTMLImageElement | null> {
  if (!src || typeof Image === 'undefined') return Promise.resolve(null)
  return new Promise((resolve) => {
    const img = new Image()
    img.crossOrigin = 'anonymous'
    let settled = false
    const finish = (value: HTMLImageElement | null) => {
      if (settled) return
      settled = true
      window.clearTimeout(timer)
      resolve(value)
    }
    const timer = window.setTimeout(() => finish(null), timeoutMs)
    img.onload = () => finish(img)
    img.onerror = () => finish(null)
    img.src = src
  })
}

function drawWordmark(ctx: CanvasRenderingContext2D, siteName: string, x: number, y: number, height: number, fill: string): void {
  ctx.save()
  ctx.fillStyle = fill
  if (/^gabugu(?: ai)?$/i.test(siteName.trim()) && typeof Path2D !== 'undefined') {
    const scale = height / 154
    ctx.translate(x, y)
    ctx.scale(scale, scale)
    const glyphs: Array<[string, number]> = [
      [wordmarkPaths.G, 0],
      [wordmarkPaths.A, 162],
      [wordmarkPaths.B, 340],
      [wordmarkPaths.U, 490],
      [wordmarkPaths.G, 630],
      [wordmarkPaths.U, 796]
    ]
    for (const [d, tx] of glyphs) {
      ctx.save()
      ctx.translate(tx, 0)
      ctx.fill(new Path2D(d), 'evenodd')
      ctx.restore()
    }
  } else {
    ctx.font = `700 ${Math.round(height)}px ${FONT}`
    ctx.textBaseline = 'top'
    ctx.fillText(siteName || 'GABUGU', x, y)
  }
  ctx.restore()
}

function rankColor(rank: number, p: Palette): string {
  if (rank === 1) return p.gold
  if (rank === 2) return p.silver
  if (rank === 3) return p.bronze
  return p.faint
}

function rankDeltaLabel(entry: ArenaModelEntry, p: Palette, newLabel: string): { text: string; color: string } | null {
  if (entry.is_new) return { text: newLabel, color: p.newTag }
  if (entry.rank_delta != null && entry.rank_delta > 0) return { text: `▲${entry.rank_delta}`, color: p.up }
  if (entry.rank_delta != null && entry.rank_delta < 0) return { text: `▼${Math.abs(entry.rank_delta)}`, color: p.down }
  return null
}

function drawOpenTag(ctx: CanvasRenderingContext2D, label: string, x: number, centerY: number, p: Palette): void {
  ctx.save()
  ctx.font = `500 15px ${FONT}`
  const w = ctx.measureText(label).width + 16
  roundRect(ctx, x, centerY - 12, w, 24, 5)
  ctx.fillStyle = p.openBg
  ctx.fill()
  ctx.fillStyle = p.openText
  ctx.textBaseline = 'middle'
  ctx.fillText(label, x + 8, centerY + 1)
  ctx.restore()
}

function drawOrgDot(ctx: CanvasRenderingContext2D, x: number, y: number, color: string): void {
  ctx.save()
  ctx.beginPath()
  ctx.arc(x, y, 4.5, 0, Math.PI * 2)
  ctx.fillStyle = color
  ctx.fill()
  ctx.restore()
}

function drawTop10(ctx: CanvasRenderingContext2D, board: ArenaCategoryBoard, p: Palette, labels: ArenaPosterLabels): void {
  const entries = board.entries.slice(0, 10)
  const top = 356
  const rowH = 107

  const maxScore = entries[0]?.score ?? 1
  const minScore = entries[entries.length - 1]?.score ?? 0
  const span = Math.max(10, maxScore - minScore)

  hairline(ctx, top, p.line)

  entries.forEach((entry, i) => {
    const y = top + i * rowH
    const mid = y + rowH / 2
    const orgMeta = resolveArenaOrgMeta(entry.organization, entry.model_name)

    // Subtle focal spotlight for #01 (bleeds subtly outside content column so numbers stay ruler-aligned)
    if (i === 0) {
      ctx.save()
      ctx.fillStyle = p.championWash
      ctx.fillRect(PAD - 18, y + 1, CONTENT_W + 36, rowH - 1)
      ctx.fillStyle = p.gold
      ctx.fillRect(PAD - 18, y + 16, 3, rowH - 32)
      ctx.restore()
    }

    // Rank numeral, with this release's movement under it.
    ctx.save()
    ctx.textBaseline = 'alphabetic'
    ctx.fillStyle = rankColor(entry.rank, p)
    ctx.font = `600 30px ${FONT}`
    ctx.fillText(String(entry.rank).padStart(2, '0'), PAD, mid + 4)
    const delta = rankDeltaLabel(entry, p, labels.newBadge)
    if (delta) {
      ctx.font = `600 15px ${FONT}`
      ctx.fillStyle = delta.color
      ctx.fillText(delta.text, PAD, mid + 30)
    }
    ctx.restore()

    // Model and organization with subtle brand color dot.
    const nameX = PAD + 96
    ctx.save()
    ctx.textBaseline = 'alphabetic'
    ctx.fillStyle = p.text
    ctx.font = `600 ${i === 0 ? 31 : 29}px ${FONT}`
    ctx.fillText(truncateCanvasText(ctx, entry.model_name, 510), nameX, mid - 2)
    drawOrgDot(ctx, nameX + 5, mid + 23, orgMeta.color)
    ctx.fillStyle = p.muted
    ctx.font = `400 19px ${FONT}`
    ctx.fillText(orgMeta.displayName, nameX + 18, mid + 30)
    const orgW = ctx.measureText(orgMeta.displayName).width
    ctx.restore()
    if (entry.is_open) drawOpenTag(ctx, labels.openBadge, nameX + 18 + orgW + 12, mid + 23, p)

    // Mini relative Elo bar (subtle right-aligned meter)
    const barW = 108
    const barX = POSTER_WIDTH - PAD - 246
    const barY = mid - 4
    const ratio = 0.24 + Math.max(0, Math.min(1, (entry.score - minScore) / span)) * 0.76
    ctx.save()
    roundRect(ctx, barX, barY, barW, 5, 2.5)
    ctx.fillStyle = p.barTrack
    ctx.fill()
    roundRect(ctx, barX, barY, Math.max(8, barW * ratio), 5, 2.5)
    ctx.fillStyle = i === 0 ? p.gold : p.barFill
    ctx.fill()
    ctx.restore()

    // Score, with error margin and change (ruler-aligned on right edge across all 10 rows).
    const right = POSTER_WIDTH - PAD
    ctx.save()
    ctx.textAlign = 'right'
    ctx.textBaseline = 'alphabetic'
    ctx.fillStyle = p.text
    ctx.font = `600 ${i === 0 ? 40 : 37}px ${FONT}`
    ctx.fillText(formatArenaScore(entry.score), right, mid + 4)
    ctx.font = `500 16px ${FONT}`
    let cursor = right
    const sd = formatScoreDelta(entry.score_delta)
    if (sd) {
      ctx.fillStyle = (entry.score_delta ?? 0) > 0 ? p.up : p.down
      ctx.fillText(sd, cursor, mid + 30)
      cursor -= ctx.measureText(sd).width + 12
    }
    if (entry.ci > 0) {
      ctx.fillStyle = p.faint
      ctx.fillText(`±${entry.ci}`, cursor, mid + 30)
    }
    ctx.restore()

    hairline(ctx, y + rowH, p.line)
  })
}

function drawOverview(ctx: CanvasRenderingContext2D, boards: ArenaCategoryBoard[], p: Palette, labels: ArenaPosterLabels): void {
  const top = 356
  const gap = 18
  const cardH = 254

  boards.slice(0, 4).forEach((board, i) => {
    const y = top + i * (cardH + gap)

    ctx.save()
    roundRect(ctx, PAD, y, CONTENT_W, cardH, 16)
    ctx.fillStyle = p.cardBg
    ctx.fill()
    ctx.strokeStyle = p.line
    ctx.lineWidth = 1
    ctx.stroke()
    ctx.restore()

    ctx.save()
    ctx.textBaseline = 'alphabetic'
    ctx.fillStyle = p.text
    ctx.font = `600 25px ${FONT}`
    ctx.fillText(labels.categoryLabels[board.id] || board.id, PAD + 28, y + 48)
    ctx.textAlign = 'right'
    ctx.fillStyle = p.faint
    ctx.font = `400 17px ${FONT}`
    ctx.fillText(`${board.publish_date}  ·  ${labels.modelCountText(board.total_models)}`, POSTER_WIDTH - PAD - 28, y + 48)
    ctx.restore()

    hairline(ctx, y + 68, p.line, PAD + 28, POSTER_WIDTH - PAD - 28)

    board.entries.slice(0, 3).forEach((entry, r) => {
      const rowY = y + 114 + r * 54
      const orgMeta = resolveArenaOrgMeta(entry.organization, entry.model_name)

      ctx.save()
      ctx.textBaseline = 'alphabetic'
      ctx.fillStyle = rankColor(entry.rank, p)
      ctx.font = `600 21px ${FONT}`
      ctx.fillText(String(entry.rank).padStart(2, '0'), PAD + 28, rowY)

      ctx.fillStyle = r === 0 ? p.text : p.text2
      ctx.font = `${r === 0 ? 600 : 500} 23px ${FONT}`
      ctx.fillText(truncateCanvasText(ctx, entry.model_name, 480), PAD + 84, rowY)

      ctx.textAlign = 'right'
      ctx.fillStyle = r === 0 ? p.text : p.text2
      ctx.font = `600 24px ${FONT}`
      const scoreText = formatArenaScore(entry.score)
      ctx.fillText(scoreText, POSTER_WIDTH - PAD - 28, rowY)
      const scoreW = ctx.measureText(scoreText).width
      ctx.fillStyle = p.muted
      ctx.font = `400 17px ${FONT}`
      const orgRightX = POSTER_WIDTH - PAD - 28 - scoreW - 26
      ctx.fillText(orgMeta.displayName, orgRightX, rowY)
      const orgW = ctx.measureText(orgMeta.displayName).width
      drawOrgDot(ctx, orgRightX - orgW - 12, rowY - 6, orgMeta.color)
      ctx.restore()
    })
  })
}

/** Draws the tall share image onto the canvas. */
export async function renderArenaPosterToCanvas(canvas: HTMLCanvasElement, options: RenderArenaPosterOptions): Promise<void> {
  if (typeof document !== 'undefined' && document.fonts?.ready) {
    try {
      await document.fonts.ready
    } catch {
      // System fonts are an acceptable fallback.
    }
  }

  canvas.width = POSTER_WIDTH
  canvas.height = POSTER_HEIGHT
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('Canvas 2D context unavailable')
  const p = options.theme === 'dark' ? DARK : LIGHT
  const { labels } = options

  // Background with architectural zenith dual glow + whisper-fine upper grid.
  ctx.fillStyle = p.bg
  ctx.fillRect(0, 0, POSTER_WIDTH, POSTER_HEIGHT)

  const glowBlue = ctx.createRadialGradient(POSTER_WIDTH * 0.18, 0, 20, POSTER_WIDTH * 0.18, 0, 720)
  glowBlue.addColorStop(0, p.glowBlue)
  glowBlue.addColorStop(1, 'rgba(0, 0, 0, 0)')
  ctx.fillStyle = glowBlue
  ctx.fillRect(0, 0, POSTER_WIDTH, 860)

  const glowGold = ctx.createRadialGradient(POSTER_WIDTH * 0.88, 20, 10, POSTER_WIDTH * 0.88, 20, 620)
  glowGold.addColorStop(0, p.glowGold)
  glowGold.addColorStop(1, 'rgba(0, 0, 0, 0)')
  ctx.fillStyle = glowGold
  ctx.fillRect(0, 0, POSTER_WIDTH, 720)

  // Subtle architectural header grid fading out smoothly
  ctx.save()
  ctx.strokeStyle = p.gridLine
  ctx.lineWidth = 1
  for (let gx = 72; gx < POSTER_WIDTH; gx += 72) {
    ctx.beginPath()
    ctx.moveTo(gx + 0.5, 0)
    ctx.lineTo(gx + 0.5, 340)
    ctx.stroke()
  }
  for (let gy = 72; gy <= 320; gy += 72) {
    ctx.beginPath()
    ctx.moveTo(0, gy + 0.5)
    ctx.lineTo(POSTER_WIDTH, gy + 0.5)
    ctx.stroke()
  }
  ctx.restore()

  // Top hairline specular bar
  ctx.save()
  const topSpec = ctx.createLinearGradient(0, 0, POSTER_WIDTH, 0)
  topSpec.addColorStop(0, 'rgba(91, 140, 255, 0)')
  topSpec.addColorStop(0.35, options.theme === 'dark' ? 'rgba(143, 180, 255, 0.55)' : 'rgba(37, 99, 235, 0.35)')
  topSpec.addColorStop(0.7, options.theme === 'dark' ? 'rgba(230, 198, 138, 0.45)' : 'rgba(166, 114, 34, 0.3)')
  topSpec.addColorStop(1, 'rgba(230, 198, 138, 0)')
  ctx.fillStyle = topSpec
  ctx.fillRect(0, 0, POSTER_WIDTH, 3)
  ctx.restore()

  // Brand row.
  const logo = await loadImageSafe(options.siteLogoUrl || (options.theme === 'dark' ? '/brand/gabugu-logo-dark.png' : '/brand/gabugu-logo-light.png'))
  let brandX = PAD
  if (logo) {
    ctx.drawImage(logo, brandX, 72, 34, 38)
    brandX += 48
  }
  drawWordmark(ctx, options.siteName || 'GABUGU', brandX, 79, 24, p.wordmark)
  ctx.save()
  ctx.textAlign = 'right'
  ctx.textBaseline = 'middle'
  ctx.fillStyle = p.muted
  ctx.font = `500 15px ${FONT}`
  ctx.fillText('LMARENA LEADERBOARD', POSTER_WIDTH - PAD, 92)
  ctx.restore()

  // Title and meta line.
  ctx.save()
  ctx.textBaseline = 'alphabetic'
  ctx.fillStyle = p.text
  if (options.mode === 'top10') {
    const name = labels.categoryLabels[options.activeBoard.id] || options.activeBoard.id
    ctx.font = `600 64px ${FONT}`
    ctx.fillText(`${name} ${labels.top10Suffix}`, PAD, 232)
    const meta: string[] = []
    if (options.activeBoard.publish_date) meta.push(`${labels.publishDateLabel} ${options.activeBoard.publish_date}`)
    if (options.activeBoard.previous_publish_date) meta.push(`${labels.baselineDateLabel} ${options.activeBoard.previous_publish_date}`)
    meta.push(labels.modelCountText(options.activeBoard.total_models))
    ctx.fillStyle = p.muted
    ctx.font = `400 21px ${FONT}`
    ctx.fillText(meta.join('  ·  '), PAD, 286)
  } else {
    ctx.font = `600 64px ${FONT}`
    ctx.fillText(labels.overviewTitle, PAD, 232)
    ctx.fillStyle = p.muted
    ctx.font = `400 21px ${FONT}`
    ctx.fillText(labels.overviewSubtitle, PAD, 286)
  }
  ctx.restore()

  if (options.mode === 'top10') drawTop10(ctx, options.activeBoard, p, labels)
  else drawOverview(ctx, options.allBoards, p, labels)

  // Footer: address, source and a crisp QR card back to the page.
  const footerTop = 1496
  ctx.save()
  ctx.textBaseline = 'alphabetic'
  ctx.fillStyle = p.text
  ctx.font = `600 28px ${FONT}`
  ctx.fillText(truncateCanvasText(ctx, options.displayUrl, CONTENT_W - 220), PAD, footerTop + 54)
  ctx.fillStyle = p.text2
  ctx.font = `400 20px ${FONT}`
  ctx.fillText(truncateCanvasText(ctx, labels.footerScanHint, CONTENT_W - 220), PAD, footerTop + 94)
  ctx.fillStyle = p.faint
  ctx.font = `400 16px ${FONT}`
  ctx.fillText(truncateCanvasText(ctx, labels.footerSource, CONTENT_W - 220), PAD, footerTop + 150)
  if (options.lastCheckedText) {
    ctx.fillText(`${labels.lastCheckedLabel}  ${options.lastCheckedText}`, PAD, footerTop + 178)
  }
  ctx.restore()

  const qrSize = 168
  const qrX = POSTER_WIDTH - PAD - qrSize
  const qrY = footerTop + 16
  ctx.save()
  roundRect(ctx, qrX, qrY, qrSize, qrSize, 16)
  ctx.fillStyle = '#ffffff'
  ctx.fill()
  ctx.strokeStyle = p.qrBorder
  ctx.lineWidth = 1.5
  ctx.stroke()
  ctx.restore()
  try {
    const qrCanvas = document.createElement('canvas')
    await QRCode.toCanvas(qrCanvas, options.shareUrl, { width: 148, margin: 1, color: { dark: '#0b0c10', light: '#ffffff' } })
    ctx.drawImage(qrCanvas, qrX + 10, qrY + 10, 148, 148)
  } catch {
    // The rest of the image is still useful without the code.
  }
}
