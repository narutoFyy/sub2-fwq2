import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../../')

describe('chat integration surface', () => {
  it('registers an authenticated /chat route and sidebar entry', () => {
    const router = readFileSync(resolve(root, 'router/index.ts'), 'utf8')
    const sidebar = readFileSync(resolve(root, 'components/layout/AppSidebar.vue'), 'utf8')
    expect(router).toContain("path: '/chat'")
    expect(router).toContain("titleKey: 'chat.title'")
    expect(router).toContain('requiresAuth: true')
    expect(sidebar).toContain("{ path: '/chat', label: t('nav.chat'), icon: ChatIcon }")
    expect(sidebar).toMatch(/\{ path: '\/chat', label: t\('nav\.chat'\), icon: ChatIcon \}/)
  })

  it('does not wrap chat in the dashboard AppLayout and uses the chat typefaces', () => {
    const view = readFileSync(resolve(root, 'features/chat/ChatView.vue'), 'utf8')
    const css = readFileSync(resolve(root, 'features/chat/chat.css'), 'utf8')
    expect(view).toContain('chat-shell')
    expect(view).not.toContain('AppLayout')
    expect(css).toContain('Instrument Sans')
    expect(css).toContain('Noto Serif SC')
    expect(css).toContain('Source Serif 4')
  })
})
