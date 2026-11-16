import { apiClient } from '../client'
import type { ModelRadarOverview, ModelRadarResult, ModelRadarSettings } from '@/api/modelRadar'

export type { ModelRadarOverview, ModelRadarResult }

export interface ModelRadarConfig extends Omit<NonNullable<ModelRadarOverview['groups'][number]>, 'timeline' | 'ciallo' | 'drawing' | 'verdict'> {
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

export async function updateSettings(settings: ModelRadarSettings): Promise<ModelRadarSettings> {
  const { data } = await apiClient.put<ModelRadarSettings>('/admin/model-radar/settings', settings)
  return data
}

export async function deleteResult(id: number): Promise<void> {
  await apiClient.delete(`/admin/model-radar/results/${id}`)
}

export async function clearGroupResults(groupId: number): Promise<{ deleted: number }> {
  const { data } = await apiClient.delete<{ deleted: number }>(`/admin/model-radar/groups/${groupId}/results`)
  return data
}

export async function clearAllResults(): Promise<{ deleted: number }> {
  const { data } = await apiClient.delete<{ deleted: number }>('/admin/model-radar/results')
  return data
}

export const modelRadarAdminAPI = { getOverview, updateConfigs, runNow, reviewResult, updateSettings, deleteResult, clearGroupResults, clearAllResults }
export default modelRadarAdminAPI
