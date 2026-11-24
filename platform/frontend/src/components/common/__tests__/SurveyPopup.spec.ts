import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { reactive } from 'vue'
import SurveyPopup from '../SurveyPopup.vue'
import { useSurveyStore } from '@/stores/surveys'
import { useAnnouncementStore } from '@/stores/announcements'
import { surveysAPI, redeemAPI } from '@/api'
import type { UserSurvey } from '@/types/survey'

const mockRoute = reactive({ path: '/dashboard' })

vi.mock('vue-router', () => ({
  useRoute: () => mockRoute
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key
    })
  }
})

const sampleWizardSurvey: UserSurvey = {
  id: 42,
  title: '体验调研',
  description: '30秒问卷',
  display_mode: 'corner',
  has_reward: true,
  reward_type: 'balance',
  reward_value: 0.5,
  snooze_days: 3,
  skip_if_announcement_unread: true,
  delay_seconds: 0,
  show_micro_results: true,
  created_at: '2026-11-23T00:00:00Z',
  questions: [
    {
      id: 'q1',
      type: 'single',
      title: '主要用途？',
      required: true,
      auto_advance: true,
      options: [
        { id: 'coding', label: 'AI 编程助手' },
        { id: 'app', label: '自建应用' }
      ]
    },
    {
      id: 'q2',
      type: 'rating',
      title: '最近稳定性打几分？',
      required: true,
      scale_min: 1,
      scale_max: 5
    },
    {
      id: 'q3',
      type: 'tags_text',
      title: '反馈补充',
      required: false,
      linked_rating_question_id: 'q2',
      low_score_threshold: 3,
      low_title: '哪里最让您不顺手？',
      high_title: '您最期待哪些新功能？',
      low_chips: ['晚高峰变慢', '偶尔断流'],
      high_chips: ['更快上新模型', '细化账单']
    }
  ]
}

describe('SurveyPopup', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    setActivePinia(createPinia())
    mockRoute.path = '/dashboard'
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
    vi.restoreAllMocks()
    document.body.innerHTML = ''
  })

  function mountPopup() {
    return mount(SurveyPopup, {
      global: {
        stubs: {
          Transition: { template: '<div><slot /></div>' }
        }
      }
    })
  }

  it('defers rendering while an unread announcement popup is open and appears after dismiss', async () => {
    const surveyStore = useSurveyStore()
    const announcementStore = useAnnouncementStore()

    announcementStore.loading = true
    surveyStore.currentSurvey = sampleWizardSurvey

    const wrapper = mountPopup()
    await wrapper.vm.$nextTick()
    expect(document.body.querySelector('[data-testid="survey-popup-root"]')).toBeNull()

    // 公告加载完成且弹出未读公告时继续避让
    announcementStore.loading = false
    announcementStore.currentPopup = {
      id: 99,
      title: '系统公告',
      content: '维护通知',
      notify_mode: 'popup',
      created_at: '2026-11-23T00:00:00Z',
      updated_at: '2026-11-23T00:00:00Z'
    }
    await wrapper.vm.$nextTick()
    expect(document.body.querySelector('[data-testid="survey-popup-root"]')).toBeNull()

    // 关闭公告弹窗后问卷自动浮现
    announcementStore.currentPopup = null
    await wrapper.vm.$nextTick()
    expect(document.body.querySelector('[data-testid="survey-wizard-card"]')).not.toBeNull()
    wrapper.unmount()
  })

  it('auto-advances single choice, switches low-score follow-up chips, and submits reward', async () => {
    const submitSpy = vi.spyOn(surveysAPI, 'submit').mockResolvedValue({
      response_id: 7,
      survey_id: 42,
      reward_status: 'granted',
      reward_type: 'balance',
      reward_value: 0.5,
      reward_redeem_code: 'SRV-42-1-ABCD',
      submitted_at: '2026-11-23T00:01:00Z'
    })

    const surveyStore = useSurveyStore()
    surveyStore.currentSurvey = sampleWizardSurvey

    const wrapper = mountPopup()
    await wrapper.vm.$nextTick()

    // 第1题单选点击 -> 280ms 后自动跳第2题
    const opt1 = document.body.querySelector('[data-testid="survey-opt-q1-coding"]') as HTMLButtonElement
    expect(opt1).not.toBeNull()
    opt1.click()
    vi.advanceTimersByTime(300)
    await wrapper.vm.$nextTick()

    // 第2题打低分 2 分 -> 360ms 后自动跳第3题并呈现低分痛点标签
    const rate2 = document.body.querySelector('[data-testid="survey-rate-q2-2"]') as HTMLButtonElement
    expect(rate2).not.toBeNull()
    rate2.click()
    vi.advanceTimersByTime(400)
    await wrapper.vm.$nextTick()

    expect(document.body.textContent).toContain('哪里最让您不顺手？')
    const chip = document.body.querySelector('[data-testid="survey-chip-晚高峰变慢"]') as HTMLButtonElement
    expect(chip).not.toBeNull()
    chip.click()
    await wrapper.vm.$nextTick()

    // 提交
    const submitBtn = document.body.querySelector('[data-testid="survey-submit-btn"]') as HTMLButtonElement
    submitBtn.click()
    await vi.runAllTimersAsync()
    await wrapper.vm.$nextTick()

    expect(submitSpy).toHaveBeenCalledTimes(1)
    expect(submitSpy).toHaveBeenCalledWith(42, {
      answers: {
        q1: { choice_id: 'coding' },
        q2: { rating: 2 },
        q3: { tags: ['晚高峰变慢'] }
      },
      duration_seconds: expect.any(Number)
    })
    expect(document.body.querySelector('[data-testid="survey-reward-granted"]')).not.toBeNull()
    wrapper.unmount()
  })

  it('supports micro poll one-click voting with live results and inline code redeem', async () => {
    vi.spyOn(surveysAPI, 'submit').mockResolvedValue({
      response_id: 9,
      survey_id: 55,
      reward_status: 'code_issued',
      reward_type: 'redeem_code',
      reward_value: 1,
      reward_redeem_code: 'SRV-55-1-BEEF',
      micro_results: [
        { option_id: 'status', label: '延迟看板', count: 7, percent: 70 },
        { option_id: 'budget', label: '日限额', count: 3, percent: 30 }
      ],
      submitted_at: '2026-11-23T00:02:00Z'
    })
    const redeemSpy = vi.spyOn(redeemAPI, 'redeem').mockResolvedValue({
      message: 'ok',
      type: 'balance',
      value: 1
    })

    const surveyStore = useSurveyStore()
    surveyStore.currentSurvey = {
      ...sampleWizardSurvey,
      id: 55,
      display_mode: 'micro',
      reward_type: 'redeem_code',
      reward_value: 1,
      questions: [
        {
          id: 'poll',
          type: 'single',
          title: '下周最想上什么？',
          required: true,
          auto_advance: true,
          options: [
            { id: 'status', label: '延迟看板' },
            { id: 'budget', label: '日限额' }
          ]
        }
      ]
    }

    const wrapper = mountPopup()
    await wrapper.vm.$nextTick()

    const opt = document.body.querySelector('[data-testid="survey-micro-opt-status"]') as HTMLButtonElement
    opt.click()
    await vi.runAllTimersAsync()
    await wrapper.vm.$nextTick()

    expect(document.body.querySelector('[data-testid="survey-micro-results"]')).not.toBeNull()
    expect(document.body.textContent).toContain('70%')
    expect(document.body.textContent).toContain('SRV-55-1-BEEF')
    const redeemBtn = document.body.querySelector('[data-testid="survey-micro-redeem-btn"]') as HTMLButtonElement
    expect(redeemBtn).not.toBeNull()
    redeemBtn.click()
    await vi.runAllTimersAsync()
    await wrapper.vm.$nextTick()
    expect(redeemSpy).toHaveBeenCalledWith('SRV-55-1-BEEF')
    wrapper.unmount()
  })

  it('allows minimizing corner card to pill and snoozing politely', async () => {
    const stateSpy = vi.spyOn(surveysAPI, 'recordState').mockResolvedValue({
      message: 'ok',
      action: 'snooze',
      snooze_until: '2026-11-26T00:00:00Z'
    })

    const surveyStore = useSurveyStore()
    surveyStore.currentSurvey = sampleWizardSurvey

    const wrapper = mountPopup()
    await wrapper.vm.$nextTick()

    // 最小化到右下角蓝胶囊
    const minBtn = document.body.querySelector('[data-testid="survey-minimize-btn"]') as HTMLButtonElement
    minBtn.click()
    await wrapper.vm.$nextTick()
    expect(document.body.querySelector('[data-testid="survey-minimized-pill"]')).not.toBeNull()

    // 点蓝胶囊恢复
    const pill = document.body.querySelector('[data-testid="survey-minimized-pill"]') as HTMLButtonElement
    pill.click()
    await wrapper.vm.$nextTick()
    expect(document.body.querySelector('[data-testid="survey-wizard-card"]')).not.toBeNull()

    // 点稍后再说
    const snoozeBtn = document.body.querySelector('[data-testid="survey-snooze-btn"]') as HTMLButtonElement
    snoozeBtn.click()
    await vi.runAllTimersAsync()
    await wrapper.vm.$nextTick()
    expect(stateSpy).toHaveBeenCalledWith(42, 'snooze')
    expect(document.body.querySelector('[data-testid="survey-popup-root"]')).toBeNull()
    wrapper.unmount()
  })
})
