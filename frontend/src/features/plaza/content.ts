import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

export function usePlazaCopy() {
  const { locale } = useI18n()
  return computed(() => (locale.value.startsWith('zh') ? zh : en))
}

const zh = {
  nav: '广场',
  home: '返回首页',
  title: '每个分组，价格一目了然。',
  intro: '实付价 = 官方价 × 分组倍率。统一以美元计价，token 模型按每百万 token 计。',
  entryTitle: '按分组计费，价格透明。',
  entryDesc: '同一个模型，选不同分组，价格不同。',
  viewAll: '查看全部价格',
  moreGroups: (n: number) => `还有 ${n} 个分组`,
  rateLabel: '倍率',
  modelsCount: (n: number) => `${n} 个模型`,
  featured: '代表模型',
  input: '输入',
  output: '输出',
  cacheRead: '缓存读',
  cacheWrite: '缓存写',
  model: '模型',
  unitToken: '美元 / 百万 token',
  unitTokenShort: '/ 1M tokens',
  perImage: '/ 张',
  perRequest: '/ 次',
  perImageLabel: '按张计费',
  perRequestLabel: '按次计费',
  official: '官方',
  officialHint: '划线价为官方原价，仅供对比',
  formula: (rate: string) => `实付 = 官方价 × ${rate}`,
  discount: (d: string) => (d ? `官方价 ${d}` : ''),
  premium: '高于官方价',
  sameAsOfficial: '与官方价相同',
  customRate: (base: string) => `你的专属倍率（分组默认 ×${base}）`,
  exclusive: '专属',
  subscription: '订阅',
  peak: (window: string, m: string) => `高峰时段 ${window} 再乘 ×${m}`,
  timePricing: '部分模型分时段计价，见模型名旁的时段标签',
  longContext: '多档价格按单次请求的上下文总长度计价',
  longContextOff: '本分组不按上下文长度分档，超长请求仍按首档计费',
  reasoning: (effort: string, m: number) => `${effort} 思考 ×${m}`,
  allGroups: '全部分组',
  search: '搜索模型',
  noResult: '没有匹配的模型',
  empty: '暂时没有可展示的分组',
  loadFailed: '价格加载失败，请稍后刷新重试',
  loginRequired: '登录后查看价格',
  login: '登录',
  anonymousHint: '登录后可看到你的专属分组与专属倍率。',
  noteTitle: '计费说明',
  dashboard: '进入控制台',
  sortHint: '分组按倍率从低到高排列',
  weekdays: '工作日',
  group: '分组'
}

type PlazaCopy = typeof zh

const en: PlazaCopy = {
  nav: 'Plaza',
  home: 'Back home',
  title: 'Every group, priced in plain sight.',
  intro: 'You pay list price × group rate. All prices in USD; token models are priced per million tokens.',
  entryTitle: 'Pay by group. See every price.',
  entryDesc: 'Same model, different group, different price.',
  viewAll: 'View all prices',
  moreGroups: (n: number) => `${n} more group${n === 1 ? '' : 's'}`,
  rateLabel: 'Rate',
  modelsCount: (n: number) => `${n} model${n === 1 ? '' : 's'}`,
  featured: 'Featured',
  input: 'Input',
  output: 'Output',
  cacheRead: 'Cache read',
  cacheWrite: 'Cache write',
  model: 'Model',
  unitToken: 'USD / 1M tokens',
  unitTokenShort: '/ 1M tokens',
  perImage: '/ image',
  perRequest: '/ request',
  perImageLabel: 'Per image',
  perRequestLabel: 'Per request',
  official: 'List',
  officialHint: 'Struck-through prices are official list prices, for reference',
  formula: (rate: string) => `You pay list × ${rate}`,
  discount: (d: string) => d,
  premium: 'Above list price',
  sameAsOfficial: 'Same as list price',
  customRate: (base: string) => `Your personal rate (group default ×${base})`,
  exclusive: 'Exclusive',
  subscription: 'Subscription',
  peak: (window: string, m: string) => `Peak hours ${window}: extra ×${m}`,
  timePricing: 'Some models are priced by time of day — see the time tag next to the model',
  longContext: 'Tiered prices depend on the total context of each request',
  longContextOff: 'This group does not tier by context length; long requests stay on the first tier',
  reasoning: (effort: string, m: number) => `${effort} reasoning ×${m}`,
  allGroups: 'All groups',
  search: 'Search models',
  noResult: 'No matching models',
  empty: 'No groups to show yet',
  loadFailed: 'Could not load prices. Please refresh later.',
  loginRequired: 'Sign in to see prices',
  login: 'Sign in',
  anonymousHint: 'Sign in to see your exclusive groups and personal rates.',
  noteTitle: 'Billing notes',
  dashboard: 'Open console',
  sortHint: 'Groups sorted by rate, lowest first',
  weekdays: 'Weekdays',
  group: 'Group'
}

/** 英文下的折扣说明：0.25 → "75% off list"。 */
export function discountEn(rate: number): string {
  if (!(rate > 0) || rate >= 1) return ''
  return `${Math.round((1 - rate) * 100)}% off list`
}
