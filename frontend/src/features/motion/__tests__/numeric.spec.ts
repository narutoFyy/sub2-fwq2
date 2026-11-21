import { describe, expect, it } from 'vitest'
import { formatNumeric, parseNumeric } from '../numeric'

describe('parseNumeric', () => {
  it('拆出价格的前缀、数值和小数位', () => {
    expect(parseNumeric('$0.75')).toEqual({ prefix: '$', value: 0.75, decimals: 2, grouped: false, suffix: '' })
    expect(parseNumeric('$0.0035')).toMatchObject({ value: 0.0035, decimals: 4 })
  })

  it('支持倍率符号、千分位和后缀', () => {
    expect(parseNumeric('×0.25')).toMatchObject({ prefix: '×', value: 0.25, decimals: 2 })
    expect(parseNumeric('1,280 次')).toEqual({ prefix: '', value: 1280, decimals: 0, grouped: true, suffix: ' 次' })
    expect(parseNumeric('×1')).toMatchObject({ prefix: '×', value: 1, decimals: 0 })
  })

  it('没有数字时返回 null', () => {
    expect(parseNumeric('—')).toBeNull()
    expect(parseNumeric('')).toBeNull()
  })
})

describe('formatNumeric', () => {
  it('滚动中的值保持原文的小数位和写法', () => {
    const price = parseNumeric('$12.50')!
    expect(formatNumeric(price, 0)).toBe('$0.00')
    expect(formatNumeric(price, 6.256)).toBe('$6.26')
    expect(formatNumeric(price, 12.5)).toBe('$12.50')
    const count = parseNumeric('1,280 次')!
    expect(formatNumeric(count, 1279.6)).toBe('1,280 次')
    expect(formatNumeric(count, 640)).toBe('640 次')
  })
})
