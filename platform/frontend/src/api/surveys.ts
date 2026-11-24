/**
 * User Surveys API endpoints
 */

import { apiClient } from './client'
import type { SubmitSurveyResult, SurveyAnswers, UserSurvey } from '@/types/survey'

export async function listActive(): Promise<UserSurvey[]> {
  const { data } = await apiClient.get<UserSurvey[]>('/surveys/active')
  return data
}

export async function submit(
  id: number,
  payload: { answers: SurveyAnswers; duration_seconds: number }
): Promise<SubmitSurveyResult> {
  const { data } = await apiClient.post<SubmitSurveyResult>(`/surveys/${id}/submit`, payload)
  return data
}

export async function recordState(
  id: number,
  action: 'snooze' | 'dismiss'
): Promise<{ message: string; action: string; snooze_until?: string | null }> {
  const { data } = await apiClient.post<{
    message: string
    action: string
    snooze_until?: string | null
  }>(`/surveys/${id}/state`, { action })
  return data
}

const surveysAPI = {
  listActive,
  submit,
  recordState
}

export default surveysAPI
