import type { AnnouncementTargeting } from './index'

export type SurveyStatus = 'draft' | 'active' | 'archived'
export type SurveyDisplayMode = 'corner' | 'modal' | 'micro'
export type SurveyQuestionType = 'single' | 'multi' | 'rating' | 'tags_text'
export type SurveyRewardType = 'none' | 'balance' | 'redeem_code'
export type SurveyRewardStatus =
  | 'none'
  | 'pending'
  | 'granted'
  | 'code_issued'
  | 'budget_exhausted'
  | 'failed'

export interface SurveyQuestionOption {
  id: string
  label: string
  description?: string
  tag?: string
  icon?: string
}

export interface SurveyQuestion {
  id: string
  type: SurveyQuestionType
  title: string
  hint?: string
  required: boolean
  auto_advance?: boolean
  options?: SurveyQuestionOption[]
  scale_min?: number
  scale_max?: number
  linked_rating_question_id?: string
  low_score_threshold?: number
  low_title?: string
  high_title?: string
  low_chips?: string[]
  high_chips?: string[]
  placeholder?: string
}

export interface SurveyDeliveryPolicy {
  snooze_days: number
  min_account_age_days: number
  require_recent_usage_days: number
  skip_if_announcement_unread: boolean
  delay_seconds: number
  show_micro_results: boolean
}

export interface Survey {
  id: number
  title: string
  description: string
  status: SurveyStatus
  display_mode: SurveyDisplayMode
  questions: SurveyQuestion[]
  targeting: AnnouncementTargeting
  delivery_policy: SurveyDeliveryPolicy
  reward_type: SurveyRewardType
  reward_value: number
  reward_budget_limit?: number | null
  reward_issued_count: number
  reward_code_valid_days: number
  response_count: number
  starts_at?: string | null
  ends_at?: string | null
  created_by?: number | null
  updated_by?: number | null
  created_at: string
  updated_at: string
}

export interface UserSurvey {
  id: number
  title: string
  description: string
  display_mode: SurveyDisplayMode
  questions: SurveyQuestion[]
  has_reward: boolean
  reward_type: SurveyRewardType
  reward_value: number
  snooze_days: number
  skip_if_announcement_unread: boolean
  delay_seconds: number
  show_micro_results: boolean
  created_at: string
}

export interface SurveyAnswerValue {
  choice_id?: string
  choice_ids?: string[]
  rating?: number
  tags?: string[]
  text?: string
}

export type SurveyAnswers = Record<string, SurveyAnswerValue>

export interface MicroPollOptionStat {
  option_id: string
  label: string
  count: number
  percent: number
}

export interface SubmitSurveyResult {
  response_id: number
  survey_id: number
  reward_status: SurveyRewardStatus
  reward_type: SurveyRewardType
  reward_value: number
  reward_redeem_code?: string
  micro_results?: MicroPollOptionStat[]
  submitted_at: string
}

export interface SurveyResponseItem {
  id: number
  survey_id: number
  user_id: number
  user_email: string
  username: string
  answers: SurveyAnswers
  duration_seconds: number
  reward_status: SurveyRewardStatus
  reward_type: SurveyRewardType
  reward_value: number
  reward_redeem_code?: string
  submitted_at: string
}

export interface SurveyOptionStat {
  option_id: string
  label: string
  count: number
  percent: number
}

export interface SurveyRatingBucket {
  score: number
  count: number
  percent: number
}

export interface SurveyTagStat {
  tag: string
  count: number
}

export interface SurveyTextFeedback {
  user_email: string
  text: string
  submitted_at: string
}

export interface SurveyQuestionStats {
  question_id: string
  type: SurveyQuestionType
  title: string
  answer_count: number
  options?: SurveyOptionStat[]
  average_rating?: number
  rating_buckets?: SurveyRatingBucket[]
  top_tags?: SurveyTagStat[]
  recent_texts?: SurveyTextFeedback[]
}

export interface SurveyStats {
  survey_id: number
  total_responses: number
  snoozed_count: number
  dismissed_count: number
  average_duration_seconds: number
  reward_issued_count: number
  reward_total_value: number
  questions: SurveyQuestionStats[]
}

export interface CreateSurveyRequest {
  title: string
  description?: string
  status?: SurveyStatus
  display_mode?: SurveyDisplayMode
  questions: SurveyQuestion[]
  targeting?: AnnouncementTargeting
  delivery_policy?: SurveyDeliveryPolicy
  reward_type?: SurveyRewardType
  reward_value?: number
  reward_budget_limit?: number | null
  reward_code_valid_days?: number
  starts_at?: number
  ends_at?: number
}

export interface UpdateSurveyRequest {
  title?: string
  description?: string
  status?: SurveyStatus
  display_mode?: SurveyDisplayMode
  questions?: SurveyQuestion[]
  targeting?: AnnouncementTargeting
  delivery_policy?: SurveyDeliveryPolicy
  reward_type?: SurveyRewardType
  reward_value?: number
  reward_budget_limit?: number
  reward_code_valid_days?: number
  starts_at?: number
  ends_at?: number
}
