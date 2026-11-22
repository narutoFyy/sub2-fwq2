import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { routerKey } from 'vue-router'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import AnnouncementPopup from '../AnnouncementPopup.vue'
import { useAnnouncementStore } from '@/stores/announcements'

const announcementMarkdownStyles = readFileSync(
  resolve(process.cwd(), 'src/styles/announcement-markdown.css'),
  'utf8',
)

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const announcement = {
  id: 1,
  title: 'Preview announcement',
  content: '## Preview heading\n\n<div>HTML content</div><script>window.__xss = true</script>',
  status: 'draft' as const,
  notify_mode: 'popup' as const,
  targeting: { any_of: [] },
  created_at: '2026-07-24T07:30:00Z',
  updated_at: '2026-07-24T07:30:00Z',
}

describe('AnnouncementPopup', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    document.body.innerHTML = ''
    document.body.style.overflow = ''
  })

  it('renders mixed Markdown and HTML inside the shared styled container', async () => {
    const store = useAnnouncementStore()
    store.currentPopup = {
      id: 1,
      title: 'Mixed content announcement',
      content: [
        '## Markdown heading',
        '',
        '<div><h3>HTML heading</h3><ul><li>HTML list item</li></ul></div>',
        '',
        '<table><thead><tr><th>Status</th></tr></thead><tbody><tr><td>OK</td></tr></tbody></table>',
        '<script>window.__announcementXss = true</script>',
      ].join('\n'),
      notify_mode: 'popup',
      created_at: '2026-07-24T07:30:00Z',
      updated_at: '2026-07-24T07:30:00Z',
    }

    const wrapper = mount(AnnouncementPopup)
    await wrapper.vm.$nextTick()

    const content = document.body.querySelector('.markdown-body')
    expect(content?.querySelector('h2')?.textContent).toBe('Markdown heading')
    expect(content?.querySelector('h3')?.textContent).toBe('HTML heading')
    expect(content?.querySelector('li')?.textContent).toBe('HTML list item')
    expect(content?.querySelector('table td')?.textContent).toBe('OK')
    expect(content?.querySelector('script')).toBeNull()

    wrapper.unmount()
  })

  it.each(['h2', 'h3', 'ul', 'li', 'blockquote', 'table', 'th', 'td', 'code'])(
    'loads a shared style rule for mixed-content <%s> elements',
    (element) => {
      expect(announcementMarkdownStyles).toContain(`.markdown-body ${element}`)
    },
  )

  it('previews an admin announcement without marking it as read', async () => {
    const store = useAnnouncementStore()
    const dismissPopup = vi.spyOn(store, 'dismissPopup')
    const wrapper = mount(AnnouncementPopup, {
      props: {
        announcement,
        preview: true,
      },
    })

    expect(document.body.textContent).toContain('Preview announcement')
    expect(document.body.querySelector('.markdown-body h2')?.textContent).toBe('Preview heading')
    expect(document.body.querySelector('.markdown-body script')).toBeNull()
    expect(document.body.textContent).toContain('common.close')

    const dismissButton = document.body.querySelector<HTMLButtonElement>(
      '[data-testid="announcement-popup-dismiss"]',
    )
    dismissButton?.click()
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('close')).toHaveLength(1)
    expect(dismissPopup).not.toHaveBeenCalled()

    await wrapper.setProps({ announcement: null })
    expect(document.body.style.overflow).toBe('')
    wrapper.unmount()
  })

  it('keeps the existing user popup dismissal behavior', async () => {
    const store = useAnnouncementStore()
    store.currentPopup = announcement
    const dismissPopup = vi.spyOn(store, 'dismissPopup').mockResolvedValue()
    const wrapper = mount(AnnouncementPopup)

    const dismissButton = document.body.querySelector<HTMLButtonElement>(
      '[data-testid="announcement-popup-dismiss"]',
    )
    dismissButton?.click()
    await wrapper.vm.$nextTick()

    expect(dismissPopup).toHaveBeenCalledTimes(1)
    expect(wrapper.emitted('close')).toBeUndefined()
    wrapper.unmount()
  })

  it('navigates internally and dismisses when clicking an internal link in normal mode', async () => {
    const store = useAnnouncementStore()
    store.currentPopup = {
      ...announcement,
      content: '邀请好友赚返利：[立即前往](/affiliate)',
    }
    const dismissPopup = vi.spyOn(store, 'dismissPopup').mockResolvedValue()
    const push = vi.fn().mockResolvedValue(undefined)

    const wrapper = mount(AnnouncementPopup, {
      global: {
        provide: {
          [routerKey as symbol]: { push },
        },
      },
    })
    await wrapper.vm.$nextTick()

    const link = document.body.querySelector<HTMLAnchorElement>('.markdown-body a')
    expect(link?.getAttribute('href')).toBe('/affiliate')
    const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true })
    link?.dispatchEvent(clickEvent)
    await wrapper.vm.$nextTick()

    expect(clickEvent.defaultPrevented).toBe(true)
    expect(push).toHaveBeenCalledWith('/affiliate')
    expect(dismissPopup).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('navigates without marking read when clicking an internal link in preview mode', async () => {
    const store = useAnnouncementStore()
    const dismissPopup = vi.spyOn(store, 'dismissPopup').mockResolvedValue()
    const push = vi.fn().mockResolvedValue(undefined)

    const wrapper = mount(AnnouncementPopup, {
      props: {
        announcement: {
          ...announcement,
          content: '[去返利页](/affiliate?from=popup)',
        },
        preview: true,
      },
      global: {
        provide: {
          [routerKey as symbol]: { push },
        },
      },
    })
    await wrapper.vm.$nextTick()

    const link = document.body.querySelector<HTMLAnchorElement>('.markdown-body a')
    const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true })
    link?.dispatchEvent(clickEvent)
    await wrapper.vm.$nextTick()

    expect(clickEvent.defaultPrevented).toBe(true)
    expect(push).toHaveBeenCalledWith('/affiliate?from=popup')
    expect(dismissPopup).not.toHaveBeenCalled()
    expect(wrapper.emitted('close')).toHaveLength(1)
    wrapper.unmount()
  })

  it('does not intercept external links inside announcement markdown', async () => {
    const store = useAnnouncementStore()
    store.currentPopup = {
      ...announcement,
      content: '[外链](https://example.com/docs)',
    }
    const dismissPopup = vi.spyOn(store, 'dismissPopup').mockResolvedValue()
    const push = vi.fn().mockResolvedValue(undefined)

    const wrapper = mount(AnnouncementPopup, {
      global: {
        provide: {
          [routerKey as symbol]: { push },
        },
      },
    })
    await wrapper.vm.$nextTick()

    const link = document.body.querySelector<HTMLAnchorElement>('.markdown-body a')
    const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true })
    link?.dispatchEvent(clickEvent)
    await wrapper.vm.$nextTick()

    expect(clickEvent.defaultPrevented).toBe(false)
    expect(push).not.toHaveBeenCalled()
    expect(dismissPopup).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('preserves native browser behavior on Ctrl/Cmd modifier clicks on internal links', async () => {
    const store = useAnnouncementStore()
    store.currentPopup = {
      ...announcement,
      content: '[去返利](/affiliate)',
    }
    const dismissPopup = vi.spyOn(store, 'dismissPopup').mockResolvedValue()
    const push = vi.fn().mockResolvedValue(undefined)

    const wrapper = mount(AnnouncementPopup, {
      global: {
        provide: {
          [routerKey as symbol]: { push },
        },
      },
    })
    await wrapper.vm.$nextTick()

    const link = document.body.querySelector<HTMLAnchorElement>('.markdown-body a')
    const ctrlClick = new MouseEvent('click', { bubbles: true, cancelable: true, ctrlKey: true })
    link?.dispatchEvent(ctrlClick)
    const metaClick = new MouseEvent('click', { bubbles: true, cancelable: true, metaKey: true })
    link?.dispatchEvent(metaClick)
    await wrapper.vm.$nextTick()

    expect(ctrlClick.defaultPrevented).toBe(false)
    expect(metaClick.defaultPrevented).toBe(false)
    expect(push).not.toHaveBeenCalled()
    expect(dismissPopup).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('does not swallow internal link clicks when router is unavailable', async () => {
    const store = useAnnouncementStore()
    store.currentPopup = {
      ...announcement,
      content: '[去返利](/affiliate)',
    }
    const dismissPopup = vi.spyOn(store, 'dismissPopup').mockResolvedValue()

    const wrapper = mount(AnnouncementPopup)
    await wrapper.vm.$nextTick()

    const link = document.body.querySelector<HTMLAnchorElement>('.markdown-body a')
    const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true })
    link?.dispatchEvent(clickEvent)
    await wrapper.vm.$nextTick()

    expect(clickEvent.defaultPrevented).toBe(false)
    expect(dismissPopup).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
