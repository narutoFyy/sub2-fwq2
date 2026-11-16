import type { ModelRadarGroup, ModelRadarResult } from '@/api/modelRadar'

export type RadarVerdict = 'ok' | 'nerfed' | 'watch' | 'error' | 'none'

/** 把后端状态折成页面上的四种结论。pending_review 是旧数据里的画图状态，按通过算。 */
export function verdictOf(result?: Pick<ModelRadarResult, 'status'> | null): RadarVerdict {
  if (!result) return 'none'
  if (result.status === 'passed' || result.status === 'pending_review') return 'ok'
  if (result.status === 'failed') return 'nerfed'
  return 'error'
}

/** 分组结论以后端为准（它知道「连续几轮才算降智」）；旧后端没给时退回看最近一次。 */
export function groupVerdict(group: Pick<ModelRadarGroup, 'verdict' | 'ciallo'>): RadarVerdict {
  return group.verdict ?? verdictOf(group.ciallo)
}

export const verdictLabelKey: Record<RadarVerdict, string> = {
  ok: 'modelRadar.verdictOk',
  nerfed: 'modelRadar.verdictNerfed',
  watch: 'modelRadar.verdictWatch',
  error: 'modelRadar.verdictError',
  none: 'modelRadar.verdictNone'
}

export const verdictReasonKey: Record<RadarVerdict, string> = {
  ok: 'modelRadar.reasonOk',
  nerfed: 'modelRadar.reasonNerfed',
  watch: 'modelRadar.reasonWatch',
  error: 'modelRadar.reasonError',
  none: 'modelRadar.reasonNone'
}

export function summarize(groups: ModelRadarGroup[]) {
  const counts = { ok: 0, nerfed: 0, unknown: 0 }
  for (const group of groups) {
    const verdict = groupVerdict(group)
    if (verdict === 'ok') counts.ok++
    else if (verdict === 'nerfed') counts.nerfed++
    else counts.unknown++
  }
  return counts
}

const INTERVAL_PRESETS = [5, 10, 15, 30, 60, 120, 180, 360, 720, 1440]

/** 间隔下拉的选项；库里存的值不在预设里时也带上，避免一打开就被改掉。 */
export function intervalOptions(current?: number): number[] {
  return [...new Set(current && current > 0 ? [...INTERVAL_PRESETS, current] : INTERVAL_PRESETS)].sort((a, b) => a - b)
}

export function intervalLabel(minutes: number, t: (key: string, named: Record<string, unknown>) => string): string {
  return minutes % 60 === 0 ? t('modelRadar.hours', { n: minutes / 60 }) : t('modelRadar.minutes', { n: minutes })
}
