import type { ModelRadarGroup, ModelRadarResult } from '@/api/modelRadar'

export type RadarVerdict = 'ok' | 'nerfed' | 'error' | 'none'

/** 把后端状态折成页面上的四种结论。pending_review 是旧数据里的画图状态，按通过算。 */
export function verdictOf(result?: Pick<ModelRadarResult, 'status'> | null): RadarVerdict {
  if (!result) return 'none'
  if (result.status === 'passed' || result.status === 'pending_review') return 'ok'
  if (result.status === 'failed') return 'nerfed'
  return 'error'
}

export const verdictLabelKey: Record<RadarVerdict, string> = {
  ok: 'modelRadar.verdictOk',
  nerfed: 'modelRadar.verdictNerfed',
  error: 'modelRadar.verdictError',
  none: 'modelRadar.verdictNone'
}

export const verdictReasonKey: Record<RadarVerdict, string> = {
  ok: 'modelRadar.reasonOk',
  nerfed: 'modelRadar.reasonNerfed',
  error: 'modelRadar.reasonError',
  none: 'modelRadar.reasonNone'
}

export function summarize(groups: ModelRadarGroup[]) {
  const counts = { ok: 0, nerfed: 0, unknown: 0 }
  for (const group of groups) {
    const verdict = verdictOf(group.ciallo)
    if (verdict === 'ok') counts.ok++
    else if (verdict === 'nerfed') counts.nerfed++
    else counts.unknown++
  }
  return counts
}
