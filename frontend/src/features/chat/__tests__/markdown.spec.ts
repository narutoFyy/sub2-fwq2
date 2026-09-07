import { describe, expect, it } from 'vitest'
import { renderChatMarkdown } from '../markdown'

describe('renderChatMarkdown', () => {
  it('renders markdown and strips script and javascript urls', () => {
    const html = renderChatMarkdown('Hello **world**\n\n<script>alert(1)</script>\n\n[x](javascript:alert(1))')
    expect(html).toContain('<strong>world</strong>')
    expect(html).not.toContain('<script')
    expect(html).not.toContain('javascript:')
  })

  it('keeps fenced code as a copyable block', () => {
    const html = renderChatMarkdown('```ts\nconst x = 1\n```')
    expect(html).toContain('<pre>')
    expect(html).toContain('<code')
    expect(html).toContain('const x = 1')
  })

  it('strips inline event handlers from raw HTML', () => {
    const html = renderChatMarkdown('<img src=x onerror="alert(1)"><a href="https://evil.test" target="_blank">x</a>')
    expect(html).not.toContain('onerror')
    expect(html).not.toContain('alert(1)')
  })
})
