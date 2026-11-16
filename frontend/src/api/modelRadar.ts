import { apiClient } from './client'

export type RadarStatus = 'passed' | 'failed' | 'request_failed' | 'pending_review'

export interface ModelRadarResult {
  id: number
  group_id: number
  model_id: string
  test_type: 'ciallo' | 'drawing'
  prompt: string
  status: RadarStatus
  response_text?: string
  error_message?: string
  latency_ms: number
  detected_at: string
  review_status?: string
  reviewed_at?: string
}

export interface ModelRadarGroup {
  group_id: number
  group_name: string
  model_id: string
  reasoning_effort: string
  enabled: boolean
  /** 0 表示跟随全局间隔 */
  interval_minutes: number
  /** false 时照常检测，但不出现在用户页 */
  visible: boolean
  verdict?: 'ok' | 'nerfed' | 'watch' | 'error' | 'none'
  next_run_at?: string
  last_run_at?: string
  ciallo?: ModelRadarResult
  drawing?: ModelRadarResult
  timeline: ModelRadarResult[]
}

export interface ModelRadarOverview {
  groups: ModelRadarGroup[]
  interval_minutes: number
  prompt: string
  expected_reply: string
  settings?: ModelRadarSettings
}

export interface ModelRadarSettings {
  interval_minutes: number
  ciallo_attempts: number
  nerf_confirm_rounds: number
  drawing_enabled: boolean
  retention_days: number
}

export async function getOverview(): Promise<ModelRadarOverview> {
  const { data } = await apiClient.get<ModelRadarOverview>('/model-radar/overview')
  return data
}

export const modelRadarAPI = { getOverview }
export default modelRadarAPI
