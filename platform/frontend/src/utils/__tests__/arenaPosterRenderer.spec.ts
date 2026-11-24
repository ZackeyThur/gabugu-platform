import { describe, expect, it, vi } from 'vitest'
import {
  POSTER_HEIGHT,
  POSTER_WIDTH,
  buildPosterFilename,
  renderArenaPosterToCanvas,
  truncateCanvasText,
  type ArenaPosterLabels
} from '../arenaPosterRenderer'
import type { ArenaCategoryBoard } from '@/api/arena'

vi.mock('qrcode', () => ({
  default: {
    toCanvas: vi.fn().mockResolvedValue(undefined)
  }
}))

const sampleBoard: ArenaCategoryBoard = {
  id: 'text',
  subset: 'text_style_control',
  style_controlled: true,
  publish_date: '2026-11-19',
  previous_publish_date: '2026-11-18',
  total_models: 426,
  highlights: {
    leader: 'Gemini 4 Carbon (High)',
    leader_organization: 'google',
    leader_score: 1527.0,
    runner_up: 'Gemini 4 Argon (High)',
    runner_up_gap: 7.4,
    new_entries: 1,
    biggest_riser: {
      model_name: 'DeepSeek V4',
      organization: 'deepseek',
      rank: 3,
      rank_delta: 2,
      score_delta: 4.5
    },
    open_model_count: 1,
    top_open_model: 'DeepSeek V4',
    top_open_organization: 'deepseek',
    top_open_rank: 3,
    top_open_score: 1510.2
  },
  entries: [
    {
      rank: 1,
      rank_delta: 0,
      is_new: false,
      model_name: 'Gemini 4 Carbon (High)',
      organization: 'google',
      license: 'Proprietary',
      is_open: false,
      score: 1527.0,
      score_delta: 1.1,
      ci: 10,
      rating_lower: 1517.0,
      rating_upper: 1537.0,
      votes: 4262
    },
    {
      rank: 2,
      rank_delta: 0,
      is_new: false,
      model_name: 'Gemini 4 Argon (High)',
      organization: 'google',
      license: 'Proprietary',
      is_open: false,
      score: 1519.6,
      score_delta: -0.4,
      ci: 5,
      rating_lower: 1514.6,
      rating_upper: 1524.6,
      votes: 27193
    },
    {
      rank: 3,
      rank_delta: 2,
      is_new: false,
      model_name: 'DeepSeek V4',
      organization: 'deepseek',
      license: 'MIT',
      is_open: true,
      score: 1510.2,
      score_delta: 4.5,
      ci: 7,
      rating_lower: 1503.2,
      rating_upper: 1517.2,
      votes: 12000
    }
  ]
}

const sampleLabels: ArenaPosterLabels = {
  categoryLabels: {
    text: '综合对话',
    code: '编程代码',
    vision: '视觉理解',
    image: '图像生成'
  },
  top10Suffix: 'Top 10',
  overviewTitle: 'Arena 四赛道榜首速览',
  overviewSubtitle: '综合对话 · 编程代码 · 视觉理解 · 图像生成',
  publishDateLabel: '官方发布日',
  baselineDateLabel: '对比基线',
  modelCountText: (n) => `共 ${n} 个上榜模型`,
  openBadge: '开源',
  newBadge: '新上榜',
  footerScanHint: '扫码查看完整榜单',
  footerSource: '数据来源：LMArena 官方数据集',
  lastCheckedLabel: '上次检查'
}

function createMockContext() {
  const grad = { addColorStop: vi.fn() }
  return {
    createLinearGradient: vi.fn(() => grad),
    createRadialGradient: vi.fn(() => grad),
    fillRect: vi.fn(),
    beginPath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    quadraticCurveTo: vi.fn(),
    closePath: vi.fn(),
    arc: vi.fn(),
    fill: vi.fn(),
    stroke: vi.fn(),
    fillText: vi.fn(),
    drawImage: vi.fn(),
    save: vi.fn(),
    restore: vi.fn(),
    translate: vi.fn(),
    scale: vi.fn(),
    measureText: vi.fn((text: string) => ({ width: text.length * 10 }))
  } as unknown as CanvasRenderingContext2D
}

describe('arenaPosterRenderer', () => {
  it('builds deterministic filenames for top10 and overview modes', () => {
    expect(buildPosterFilename('top10', 'code', '2026-11-20')).toBe('gabugu-arena-code-2026-11-20.png')
    expect(buildPosterFilename('overview', 'text', '2026-11-19')).toBe('gabugu-arena-overview-2026-11-19.png')
  })

  it('truncates overly wide text with an ellipsis', () => {
    const ctx = createMockContext()
    expect(truncateCanvasText(ctx, 'Short', 100)).toBe('Short')
    expect(truncateCanvasText(ctx, 'VeryLongModelNameExceedingWidth', 80)).toMatch(/…$/)
  })

  it('renders both top10 and overview layouts onto a 1080x1760 canvas', async () => {
    const ctx = createMockContext()
    const canvas = {
      width: 0,
      height: 0,
      getContext: vi.fn(() => ctx)
    } as unknown as HTMLCanvasElement

    await renderArenaPosterToCanvas(canvas, {
      mode: 'top10',
      theme: 'dark',
      siteName: 'GABUGU',
      shareUrl: 'https://gabugu.com/arena?tab=text',
      displayUrl: 'gabugu.com/arena',
      lastCheckedText: '2026-11-23 08:00',
      activeBoard: sampleBoard,
      allBoards: [sampleBoard],
      labels: sampleLabels
    })

    expect(canvas.width).toBe(POSTER_WIDTH)
    expect(canvas.height).toBe(POSTER_HEIGHT)
    expect((ctx.fillText as unknown as ReturnType<typeof vi.fn>).mock.calls.some(([text]) => String(text).includes('Gemini 4 Carbon'))).toBe(true)

    await renderArenaPosterToCanvas(canvas, {
      mode: 'overview',
      theme: 'light',
      siteName: 'GABUGU',
      shareUrl: 'https://gabugu.com/arena',
      displayUrl: 'gabugu.com/arena',
      lastCheckedText: '2026-11-23 08:00',
      activeBoard: sampleBoard,
      allBoards: [sampleBoard],
      labels: sampleLabels
    })

    expect((ctx.fillText as unknown as ReturnType<typeof vi.fn>).mock.calls.some(([text]) => String(text).includes('Arena 四赛道榜首速览'))).toBe(true)
  })
})
