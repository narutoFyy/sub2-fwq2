import { apiClient } from './client'

export type RadarStatus = 'passed' | 'failed' | 'request_failed' | 'pending_review'

export interface ModelRadarResult {
  id: number
  group_id: number
  model_id: string
  test_type: 'logic' | 'drawing'
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
  next_run_at?: string
  last_run_at?: string
  logic?: ModelRadarResult
  drawing?: ModelRadarResult
  timeline: ModelRadarResult[]
}

export interface ModelRadarOverview {
  groups: ModelRadarGroup[]
  interval_minutes: number
  iq_status: string
  recommend_status: string
}

export async function getOverview(): Promise<ModelRadarOverview> {
  const { data } = await apiClient.get<ModelRadarOverview>('/model-radar/overview')
  return data
}

export const modelRadarAPI = { getOverview }
export default modelRadarAPI
