/**
 * Admin Surveys API endpoints
 */

import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'
import type {
  CreateSurveyRequest,
  Survey,
  SurveyResponseItem,
  SurveyStats,
  UpdateSurveyRequest
} from '@/types/survey'

export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    status?: string
    search?: string
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  },
  options?: {
    signal?: AbortSignal
  }
): Promise<BasePaginationResponse<Survey>> {
  const { data } = await apiClient.get<BasePaginationResponse<Survey>>('/admin/surveys', {
    params: { page, page_size: pageSize, ...filters },
    signal: options?.signal
  })
  return data
}

export async function getById(id: number): Promise<Survey> {
  const { data } = await apiClient.get<Survey>(`/admin/surveys/${id}`)
  return data
}

export async function create(request: CreateSurveyRequest): Promise<Survey> {
  const { data } = await apiClient.post<Survey>('/admin/surveys', request)
  return data
}

export async function update(id: number, request: UpdateSurveyRequest): Promise<Survey> {
  const { data } = await apiClient.put<Survey>(`/admin/surveys/${id}`, request)
  return data
}

export async function deleteSurvey(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/surveys/${id}`)
  return data
}

export async function getStats(id: number): Promise<SurveyStats> {
  const { data } = await apiClient.get<SurveyStats>(`/admin/surveys/${id}/stats`)
  return data
}

export async function listResponses(
  id: number,
  page: number = 1,
  pageSize: number = 20,
  filters?: { search?: string },
  options?: { signal?: AbortSignal }
): Promise<BasePaginationResponse<SurveyResponseItem>> {
  const { data } = await apiClient.get<BasePaginationResponse<SurveyResponseItem>>(
    `/admin/surveys/${id}/responses`,
    {
      params: { page, page_size: pageSize, ...filters },
      signal: options?.signal
    }
  )
  return data
}

const adminSurveysAPI = {
  list,
  getById,
  create,
  update,
  delete: deleteSurvey,
  getStats,
  listResponses
}

export default adminSurveysAPI
