import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import ArenaLeaderboardContent from '@/components/arena/ArenaLeaderboardContent.vue'
import zhArena from '@/i18n/locales/zh/arena'
import type { ArenaLeaderboardsResponse } from '@/api/arena'

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    cachedPublicSettings: { site_name: 'GABUGU', site_logo: '' },
    siteName: 'GABUGU'
  })
}))

vi.mock('@/utils/arenaPosterRenderer', () => ({
  POSTER_WIDTH: 1080,
  POSTER_HEIGHT: 1760,
  buildPosterFilename: () => 'gabugu-arena-text-2026-11-19.png',
  renderArenaPosterToCanvas: vi.fn().mockResolvedValue(undefined)
}))

const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  messages: {
    zh: { ...zhArena }
  }
})

const fixtureResponse: ArenaLeaderboardsResponse = {
  source: {
    name: 'LMArena (Chatbot Arena)',
    site_url: 'https://arena.ai/leaderboard',
    dataset_url: 'https://huggingface.co/datasets/lmarena-ai/leaderboard-dataset'
  },
  last_checked_at: '2026-11-23T08:00:00+08:00',
  last_synced_at: '2026-11-23T08:00:02+08:00',
  sync_status: 'ok',
  categories: [
    {
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
        runner_up: 'DeepSeek V4',
        runner_up_gap: 16.8,
        new_entries: 1,
        biggest_riser: {
          model_name: 'DeepSeek V4',
          organization: 'deepseek',
          rank: 2,
          rank_delta: 3,
          score_delta: 4.2
        },
        open_model_count: 1,
        top_open_model: 'DeepSeek V4',
        top_open_organization: 'deepseek',
        top_open_rank: 2,
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
          rank_delta: 3,
          is_new: false,
          model_name: 'DeepSeek V4',
          organization: 'deepseek',
          license: 'MIT',
          is_open: true,
          score: 1510.2,
          score_delta: 4.2,
          ci: 7,
          rating_lower: 1503.2,
          rating_upper: 1517.2,
          votes: 12000
        },
        {
          rank: 3,
          rank_delta: -1,
          is_new: false,
          model_name: 'Claude Fable 5.2 (xHigh)',
          organization: 'anthropic',
          license: 'Proprietary',
          is_open: false,
          score: 1504.5,
          score_delta: -0.5,
          ci: 8,
          rating_lower: 1496.5,
          rating_upper: 1512.5,
          votes: 9100
        },
        {
          rank: 4,
          rank_delta: 1,
          is_new: false,
          model_name: 'GPT-6.2 Astra (Max)',
          organization: 'openai',
          license: 'Proprietary',
          is_open: false,
          score: 1499.4,
          score_delta: 0.8,
          ci: 9,
          rating_lower: 1490.4,
          rating_upper: 1508.4,
          votes: 15600
        }
      ]
    },
    {
      id: 'code',
      subset: 'webdev',
      style_controlled: false,
      publish_date: '2026-11-20',
      previous_publish_date: '2026-11-19',
      total_models: 161,
      highlights: {
        leader: 'Claude Fable 5.2 (xHigh)',
        leader_organization: 'anthropic',
        leader_score: 1823.6,
        runner_up_gap: 4.6,
        new_entries: 0,
        open_model_count: 0
      },
      entries: [
        {
          rank: 1,
          rank_delta: 0,
          is_new: false,
          model_name: 'Claude Fable 5.2 (xHigh)',
          organization: 'anthropic',
          license: 'Proprietary',
          is_open: false,
          score: 1823.6,
          score_delta: 0.2,
          ci: 16,
          rating_lower: 1807.8,
          rating_upper: 1839.5,
          votes: 2042
        },
        {
          rank: 2,
          rank_delta: 1,
          is_new: false,
          model_name: 'Gemini 4 Carbon (High)',
          organization: 'google',
          license: 'Proprietary',
          is_open: false,
          score: 1819.0,
          score_delta: 0.5,
          ci: 12,
          rating_lower: 1807.0,
          rating_upper: 1831.0,
          votes: 3100
        }
      ]
    }
  ]
}

describe('ArenaLeaderboardContent', () => {
  it('switches tracks, filters open-source models, searches, and opens share poster modal', async () => {
    const wrapper = mount(ArenaLeaderboardContent, {
      props: {
        response: fixtureResponse,
        loading: false
      },
      global: {
        plugins: [i18n],
        stubs: {
          Teleport: true
        }
      }
    })

    // Initially displays Text track leader & open model in podium + #4 in list
    expect(wrapper.text()).toContain('Gemini 4 Carbon (High)')
    expect(wrapper.text()).toContain('DeepSeek V4')
    expect(wrapper.text()).toContain('GPT-6.2 Astra (Max)')

    // Filter to open-source only
    await wrapper.get('[data-testid="arena-license-open"]').trigger('click')
    expect(wrapper.text()).toContain('DeepSeek V4')
    expect(wrapper.get('[data-testid="arena-list"]').text()).not.toContain('Gemini 4 Carbon (High)')

    // Reset to all & switch to Code track
    await wrapper.get('[data-testid="arena-license-all"]').trigger('click')
    await wrapper.get('[data-testid="arena-tab-code"]').trigger('click')
    expect(wrapper.emitted('update:category')?.[0]).toEqual(['code'])
    expect(wrapper.text()).toContain('Claude Fable 5.2 (xHigh)')

    // Open share poster modal
    await wrapper.get('[data-testid="open-poster-modal-btn"]').trigger('click')
    expect(wrapper.find('[data-testid="arena-poster-canvas"]').exists()).toBe(true)
  })

  it('opens inline model inspector, jumps across tracks, filters by org chip, and sorts by votes', async () => {
    const wrapper = mount(ArenaLeaderboardContent, {
      props: {
        response: fixtureResponse,
        loading: false
      },
      global: {
        plugins: [i18n],
        stubs: {
          Teleport: true
        }
      }
    })

    // Click #1 podium card to open spotlight inspector with 95% CI and cross-track link
    const podiumCards = wrapper.findAll('[data-testid="arena-podium-card"]')
    expect(podiumCards.length).toBe(3)
    await podiumCards[0].trigger('click')

    const inspector = wrapper.get('[data-testid="arena-inspector"]')
    expect(inspector.text()).toContain('1527.0')
    expect(inspector.text()).toContain('1517.0 – 1537.0')
    expect(inspector.find('[data-testid="arena-cross-track-code"]').exists()).toBe(true)

    // Click cross-track chip to jump to Code track
    await inspector.get('[data-testid="arena-cross-track-code"]').trigger('click')
    expect(wrapper.emitted('update:category')?.[0]).toEqual(['code'])

    // Switch back to text, filter by OpenAI org chip
    await wrapper.get('[data-testid="arena-tab-text"]').trigger('click')
    await wrapper.get('[data-testid="arena-org-chip-openai"]').trigger('click')
    expect(wrapper.findAll('[data-testid="arena-row"]').length).toBe(1)
    expect(wrapper.get('[data-testid="arena-list"]').text()).toContain('GPT-6.2 Astra (Max)')

    // Reset org filter & sort by votes descending -> GPT-6.2 Astra (15,600 votes) should come first
    await wrapper.get('[data-testid="arena-org-chip-all"]').trigger('click')
    await wrapper.get('[data-testid="arena-sort-votes"]').trigger('click')
    const rows = wrapper.findAll('[data-testid="arena-row"]')
    expect(rows.length).toBe(4)
    expect(rows[0].text()).toContain('GPT-6.2 Astra (Max)')
  })

  it('renders initializing state when backend returns 503', () => {
    const wrapper = mount(ArenaLeaderboardContent, {
      props: {
        response: null,
        loading: false,
        initializing: true
      },
      global: {
        plugins: [i18n]
      }
    })

    expect(wrapper.text()).toContain('首次榜单同步进行中')
  })
})
