import type { CreateSurveyRequest, SurveyDeliveryPolicy } from '@/types/survey'

export function defaultSurveyDeliveryPolicy(): SurveyDeliveryPolicy {
  return {
    snooze_days: 3,
    min_account_age_days: 0,
    require_recent_usage_days: 0,
    skip_if_announcement_unread: true,
    delay_seconds: 4,
    show_micro_results: true
  }
}

export function buildDailyExperienceTemplate(): CreateSurveyRequest {
  return {
    title: '帮我们把 GABUGU 做得更顺手',
    description: '只需轻点 4 下，约 30 秒，您的反馈将直接决定下一批优化优先级',
    status: 'draft',
    display_mode: 'corner',
    reward_type: 'none',
    reward_value: 0,
    reward_budget_limit: null,
    reward_code_valid_days: 30,
    targeting: { any_of: [] },
    delivery_policy: {
      snooze_days: 3,
      min_account_age_days: 3,
      require_recent_usage_days: 14,
      skip_if_announcement_unread: true,
      delay_seconds: 4,
      show_micro_results: true
    },
    questions: [
      {
        id: 'q1',
        type: 'single',
        title: '您目前主要用 GABUGU 来做什么？',
        hint: '点选一项，自动进入下一题',
        required: true,
        auto_advance: true,
        options: [
          { id: 'coding', label: 'AI 编程助手', description: 'Claude Code / Codex / Cursor / OpenCode 等开发工具' },
          { id: 'app', label: '自建应用与 API 集成', description: '接入自有网站、自动化脚本或 Agent 工作流' },
          { id: 'chat', label: '日常对话与效率办公', description: '写作、翻译、资料总结或个人知识库问答' },
          { id: 'eval', label: '多模型对比与测试', description: '同时试跑 GPT / Claude / Gemini / Grok 效果' }
        ]
      },
      {
        id: 'q2',
        type: 'multi',
        title: '您平时最常调用哪些模型家族？',
        hint: '可多选，帮助我们优先扩充高需求线路容量',
        required: false,
        options: [
          { id: 'claude', label: 'Claude 系列', tag: '编程主力' },
          { id: 'gpt', label: 'GPT / Codex', tag: '综合高频' },
          { id: 'gemini', label: 'Gemini 系列', tag: '长上下文' },
          { id: 'grok', label: 'Grok 系列', tag: '实时信息' },
          { id: 'deepseek', label: 'DeepSeek', tag: '高性价比' },
          { id: 'other', label: '其他开源/多模态', tag: '图像/语音' }
        ]
      },
      {
        id: 'q3',
        type: 'rating',
        title: '最近两周的响应速度和稳定性您打几分？',
        hint: '轻点分值，低分将自动展开痛点标签供一键勾选',
        required: true,
        scale_min: 1,
        scale_max: 5
      },
      {
        id: 'q4',
        type: 'tags_text',
        title: '您最希望接下来优先优化哪方面？',
        hint: '点选标签即可直接提交，文字补充完全选填',
        required: false,
        linked_rating_question_id: 'q3',
        low_score_threshold: 3,
        low_title: '抱歉带来不便，最影响您体验的是哪方面？',
        high_title: '感谢认可！您最希望接下来优先增强哪项？',
        low_chips: ['晚高峰首字变慢', '偶发报错或断流', '某款新模型未上线', '错误提示不够直观', '价格倍率偏高', '接入文档不够清楚'],
        high_chips: ['更快跟进新模型', '更细的按模型账单', '增加更多充值方式', '团队/多Key额度管理', '线路状态实时看板', '保持当前稳定就很好'],
        placeholder: '还有想对站长说的话？（完全选填，不写也可直接提交）'
      }
    ]
  }
}

export function buildRewardedDeepDiveTemplate(): CreateSurveyRequest {
  const base = buildDailyExperienceTemplate()
  return {
    ...base,
    title: '有奖体验回访 · 填完即送 $0.50 体验金',
    description: '约 30 秒，前 200 名有效填写将自动赠送 $0.50 到账户余额',
    display_mode: 'modal',
    reward_type: 'balance',
    reward_value: 0.5,
    reward_budget_limit: 200
  }
}

export function buildMicroPollTemplate(): CreateSurveyRequest {
  return {
    title: '5 秒快问 · 下周优先上线哪项能力？',
    description: '点一下投票，立即可看全站实时投票占比',
    status: 'draft',
    display_mode: 'micro',
    reward_type: 'none',
    reward_value: 0,
    reward_budget_limit: null,
    reward_code_valid_days: 30,
    targeting: { any_of: [] },
    delivery_policy: {
      snooze_days: 3,
      min_account_age_days: 0,
      require_recent_usage_days: 0,
      skip_if_announcement_unread: true,
      delay_seconds: 3,
      show_micro_results: true
    },
    questions: [
      {
        id: 'poll',
        type: 'single',
        title: '下一批您最希望优先上线哪项功能？',
        required: true,
        auto_advance: true,
        options: [
          { id: 'budget', label: '每个 API Key 独立日限额提醒' },
          { id: 'status', label: '各模型线路延迟与可用率看板' },
          { id: 'batch', label: '批量低价异步任务接口（Batch）' },
          { id: 'team', label: '小团队子账号与分账统计' }
        ]
      }
    ]
  }
}
