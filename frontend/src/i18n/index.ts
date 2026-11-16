import { createI18n } from 'vue-i18n'

type LocaleCode = 'en' | 'zh'

type LocaleMessages = Record<string, any>

const LOCALE_KEY = 'sub2api_locale'
const DEFAULT_LOCALE: LocaleCode = 'en'

// 语言包分两份：基础文案（所有页面都要）和后台文案（登录后的页面才要）。
const baseLoaders: Record<LocaleCode, () => Promise<{ default: LocaleMessages }>> = {
  en: () => import('./locales/en/base'),
  zh: () => import('./locales/zh/base')
}

const adminLoaders: Record<LocaleCode, () => Promise<{ default: LocaleMessages }>> = {
  en: () => import('./locales/en/adminPack'),
  zh: () => import('./locales/zh/adminPack')
}

function isLocaleCode(value: string): value is LocaleCode {
  return value === 'en' || value === 'zh'
}

function getDefaultLocale(): LocaleCode {
  const saved = localStorage.getItem(LOCALE_KEY)
  if (saved && isLocaleCode(saved)) {
    return saved
  }

  const browserLang = navigator.language.toLowerCase()
  if (browserLang.startsWith('zh')) {
    return 'zh'
  }

  return DEFAULT_LOCALE
}

export const i18n = createI18n({
  legacy: false,
  locale: getDefaultLocale(),
  fallbackLocale: DEFAULT_LOCALE,
  messages: {},
  // 禁用 HTML 消息警告 - 引导步骤使用富文本内容（driver.js 支持 HTML）
  // 这些内容是内部定义的，不存在 XSS 风险
  warnHtmlMessage: false,
  // 公开页面偶尔也会用到后台文案（例如共用的徽标组件），缺失时自动补加载，加载完成后界面会自动刷新。
  missing: (_locale, key) => {
    if (key.startsWith('admin.') || key.startsWith('adminAccessKeys.')) {
      void ensureAdminMessages()
    }
    return undefined
  }
})

const basePromises = new Map<LocaleCode, Promise<void>>()
const adminPromises = new Map<LocaleCode, Promise<void>>()
// 一旦进入过需要后台文案的页面，之后切换语言时也一并加载后台文案。
let adminWanted = false

function loadBase(locale: LocaleCode): Promise<void> {
  let p = basePromises.get(locale)
  if (!p) {
    p = baseLoaders[locale]().then((module) => {
      i18n.global.setLocaleMessage(locale, module.default)
    })
    p.catch(() => basePromises.delete(locale))
    basePromises.set(locale, p)
  }
  return p
}

function loadAdmin(locale: LocaleCode): Promise<void> {
  let p = adminPromises.get(locale)
  if (!p) {
    // 两份同时下载，但必须在基础文案写入之后再合并，否则会被覆盖。
    const pack = adminLoaders[locale]()
    p = Promise.all([pack, loadBase(locale)]).then(([module]) => {
      i18n.global.mergeLocaleMessage(locale, module.default)
    })
    p.catch(() => adminPromises.delete(locale))
    adminPromises.set(locale, p)
  }
  return p
}

export async function loadLocaleMessages(locale: LocaleCode): Promise<void> {
  await loadBase(locale)
  if (adminWanted) await loadAdmin(locale)
}

/** 加载当前语言的后台文案；登录后的页面在路由解析前调用。 */
export function ensureAdminMessages(): Promise<void> {
  adminWanted = true
  return loadAdmin(getLocale())
}

export async function initI18n(): Promise<void> {
  const current = getLocale()
  await loadLocaleMessages(current)
  document.documentElement.setAttribute('lang', current)
}

export async function setLocale(locale: string): Promise<void> {
  if (!isLocaleCode(locale)) {
    return
  }

  await loadLocaleMessages(locale)
  i18n.global.locale.value = locale
  localStorage.setItem(LOCALE_KEY, locale)
  document.documentElement.setAttribute('lang', locale)

  // 同步更新浏览器页签标题，使其跟随语言切换
  const { resolveRouteDocumentTitle } = await import('@/router/title')
  const { default: router } = await import('@/router')
  const { useAppStore } = await import('@/stores/app')
  const { useAuthStore } = await import('@/stores/auth')
  const { useAdminSettingsStore } = await import('@/stores/adminSettings')
  const route = router.currentRoute.value
  const appStore = useAppStore()
  const authStore = useAuthStore()
  const adminSettingsStore = useAdminSettingsStore()
  const customMenuItems = [
    ...(appStore.cachedPublicSettings?.custom_menu_items ?? []),
    ...(authStore.isAdmin ? adminSettingsStore.customMenuItems : []),
  ]
  document.title = resolveRouteDocumentTitle(route, appStore.siteName, customMenuItems)
}

export function getLocale(): LocaleCode {
  const current = i18n.global.locale.value
  return isLocaleCode(current) ? current : DEFAULT_LOCALE
}

export const availableLocales = [
  { code: 'en', name: 'English', flag: '🇺🇸' },
  { code: 'zh', name: '中文', flag: '🇨🇳' }
] as const

export default i18n
