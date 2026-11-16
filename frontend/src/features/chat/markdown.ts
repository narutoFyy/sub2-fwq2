import DOMPurify from 'dompurify'
import { marked } from 'marked'

marked.setOptions({
  breaks: true,
  gfm: true,
})

function escapeText(value: string): string {
  return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

/**
 * 渲染助手回复。传了 copyLabel 时，每个代码块上面加一条「语言 + 复制」的小栏；
 * 复制按钮只带 data-chat-copy 标记，点击由页面统一处理。
 * 包装放在消毒之后做：这时正文里的尖括号都已转义，正则只会碰到真正的 <pre> 标签。
 */
export function renderChatMarkdown(markdown: string, copyLabel = ''): string {
  const html = marked.parse(markdown || '', { async: false }) as string
  const safe = DOMPurify.sanitize(html, {
    USE_PROFILES: { html: true },
  })
  if (!copyLabel) return safe
  const label = escapeText(copyLabel)
  return safe
    .replace(/<pre><code(?: class="language-([\w+#.-]+)")?>/g, (match, lang?: string) =>
      `<div class="chat-code"><div class="chat-code-bar"><span>${lang ?? ''}</span><button type="button" data-chat-copy data-label="${label}">${label}</button></div>${match}`)
    .replace(/<\/code><\/pre>/g, '</code></pre></div>')
}
