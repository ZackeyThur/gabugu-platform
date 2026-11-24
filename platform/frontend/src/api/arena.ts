/**
 * Arena Leaderboard API（公开端点，免登录可访问）
 * 由后端定时同步 LMArena 官方 Hugging Face 数据集并提供本地缓存快照与历史涨跌。
 */

import { apiClient } from './client'

export type ArenaCategoryId = 'text' | 'code' | 'vision' | 'image'

export interface ArenaSourceInfo {
  name: string
  site_url: string
  dataset_url: string
}

export interface ArenaBiggestRiser {
  model_name: string
  organization: string
  rank: number
  rank_delta: number
  score_delta?: number | null
}

export interface ArenaCategoryHighlights {
  leader: string
  leader_organization: string
  leader_score: number
  runner_up?: string
  runner_up_gap: number
  new_entries: number
  biggest_riser?: ArenaBiggestRiser | null
  open_model_count: number
  top_open_model?: string
  top_open_organization?: string
  top_open_rank?: number
  top_open_score?: number
}

export interface ArenaModelEntry {
  rank: number
  rank_delta?: number | null
  is_new: boolean
  model_name: string
  organization: string
  license: string
  is_open: boolean
  score: number
  score_delta?: number | null
  ci: number
  rating_lower: number
  rating_upper: number
  votes: number
}

export interface ArenaCategoryBoard {
  id: ArenaCategoryId
  subset: string
  style_controlled: boolean
  publish_date: string
  previous_publish_date?: string
  total_models: number
  highlights: ArenaCategoryHighlights
  entries: ArenaModelEntry[]
}

export interface ArenaLeaderboardsResponse {
  source: ArenaSourceInfo
  last_checked_at: string
  last_synced_at: string
  sync_status: 'ok' | 'partial' | 'error' | string
  categories: ArenaCategoryBoard[]
}

export async function getArenaLeaderboards(options?: {
  category?: ArenaCategoryId
  signal?: AbortSignal
}): Promise<ArenaLeaderboardsResponse> {
  const { data } = await apiClient.get<ArenaLeaderboardsResponse>('/arena/leaderboards', {
    params: options?.category ? { category: options.category } : undefined,
    signal: options?.signal
  })
  return data
}

export const arenaAPI = { getArenaLeaderboards }

export default arenaAPI
