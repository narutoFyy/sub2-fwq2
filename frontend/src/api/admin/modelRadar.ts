import { apiClient } from '../client'
import type { ModelRadarOverview, ModelRadarResult } from '@/api/modelRadar'

export type { ModelRadarOverview, ModelRadarResult }

export interface ModelRadarConfig extends Omit<NonNullable<ModelRadarOverview['groups'][number]>, 'timeline' | 'logic' | 'drawing'> {
  id?: number
}

export async function getOverview(): Promise<ModelRadarOverview> {
  const { data } = await apiClient.get<ModelRadarOverview>('/admin/model-radar/overview')
  return data
}

export async function updateConfigs(configs: ModelRadarConfig[]): Promise<{ configs: ModelRadarConfig[] }> {
  const { data } = await apiClient.put<{ configs: ModelRadarConfig[] }>('/admin/model-radar/configs', { configs })
  return data
}

export async function runNow(groupIds?: number[]): Promise<{ groups_started: number }> {
  const { data } = await apiClient.post<{ groups_started: number }>('/admin/model-radar/run', { group_ids: groupIds ?? [] })
  return data
}

export async function reviewResult(id: number, status: 'passed' | 'failed'): Promise<ModelRadarResult> {
  const { data } = await apiClient.put<ModelRadarResult>(`/admin/model-radar/results/${id}/review`, { status })
  return data
}

export const modelRadarAdminAPI = { getOverview, updateConfigs, runNow, reviewResult }
export default modelRadarAdminAPI
