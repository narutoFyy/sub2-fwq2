import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import LocaleToggle from '../LocaleToggle.vue'

const { locale, setLocale } = vi.hoisted(() => ({
  locale: { value: 'zh' },
  setLocale: vi.fn(),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key, locale: localeRef }) }
})
vi.mock('@/i18n', () => ({
  availableLocales: [
    { code: 'en', name: 'English', flag: '' },
    { code: 'zh', name: '中文', flag: '' },
  ],
  setLocale: (code: string) => {
    setLocale(code)
    localeRef.value = code
    return Promise.resolve()
  },
}))

const localeRef = ref(locale.value)

describe('LocaleToggle', () => {
  it('shows the current language without a flag and switches on click', async () => {
    const wrapper = mount(LocaleToggle)
    expect(wrapper.text()).toBe('中')

    await wrapper.trigger('click')
    await flushPromises()

    expect(setLocale).toHaveBeenCalledWith('en')
    expect(wrapper.text()).toContain('EN')
    expect(wrapper.attributes('aria-label')).toBe('home.switchLanguage')
  })
})
