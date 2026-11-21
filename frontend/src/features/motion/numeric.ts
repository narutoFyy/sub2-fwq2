/**
 * 把「$0.75」「×0.25」「1,280 次」这类文字拆成 前缀 + 数字 + 后缀，供数字滚动使用。
 * 滚动过程中保持原文的小数位数和千分位写法，滚完后显示原文。
 */
export interface NumericParts {
  prefix: string
  value: number
  decimals: number
  grouped: boolean
  suffix: string
}

const NUMERIC = /^(\D*?)(\d[\d,]*(?:\.\d+)?)(.*)$/s

export function parseNumeric(text: string): NumericParts | null {
  const m = NUMERIC.exec(text)
  if (!m) return null
  const [, prefix, raw, suffix] = m
  const value = Number(raw.replace(/,/g, ''))
  if (!Number.isFinite(value)) return null
  const dot = raw.indexOf('.')
  return { prefix, value, decimals: dot < 0 ? 0 : raw.length - dot - 1, grouped: raw.includes(','), suffix }
}

export function formatNumeric(parts: NumericParts, value: number): string {
  const digits = parts.grouped
    ? value.toLocaleString('en-US', { minimumFractionDigits: parts.decimals, maximumFractionDigits: parts.decimals })
    : value.toFixed(parts.decimals)
  return parts.prefix + digits + parts.suffix
}
