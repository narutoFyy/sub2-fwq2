import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ThemeToggleIcon from '../ThemeToggleIcon.vue'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

describe('ThemeToggleIcon', () => {
  afterEach(() => {
    document.documentElement.classList.remove('dark')
    localStorage.removeItem('theme')
  })

  it('switches from light to dark and remembers the choice', async () => {
    const wrapper = mount(ThemeToggleIcon)
    expect(wrapper.classes()).not.toContain('is-dark')
    expect(wrapper.attributes('aria-label')).toBe('home.switchToDark')

    await wrapper.trigger('click')

    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem('theme')).toBe('dark')
    expect(wrapper.classes()).toContain('is-dark')
    expect(wrapper.attributes('aria-label')).toBe('home.switchToLight')
  })

  it('switches from dark back to light', async () => {
    document.documentElement.classList.add('dark')
    const wrapper = mount(ThemeToggleIcon)
    expect(wrapper.classes()).toContain('is-dark')

    await wrapper.trigger('click')

    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(localStorage.getItem('theme')).toBe('light')
    expect(wrapper.classes()).not.toContain('is-dark')
  })

  it('renders a single icon with eight sun rays', () => {
    const wrapper = mount(ThemeToggleIcon)
    expect(wrapper.findAll('svg')).toHaveLength(1)
    expect(wrapper.findAll('.rays line')).toHaveLength(8)
  })
})
