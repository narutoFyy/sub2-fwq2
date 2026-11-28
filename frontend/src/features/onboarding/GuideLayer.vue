<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { keysAPI } from '@/api/keys'
import { useAppStore, useAuthStore, useSubscriptionStore } from '@/stores'
import { tryUseAffiliateSummaryStore } from '@/stores/affiliateSummary'
import { formatRebateRate, isTrialBonusActive } from '@/features/affiliate/helpers'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { useTutorialWelcomeStore } from '@/features/tutorials/welcomeStore'
import { useUiLabels } from '@/features/tutorials/uiLabels'
import GlowDots from '@/components/visual/GlowDots.vue'
import GuideCardBody from './GuideCardBody.vue'
import GuideConfetti from './GuideConfetti.vue'
import GuideFlyHome from './GuideFlyHome.vue'
import GuideInvite from './GuideInvite.vue'
import GuideSpotlight from './GuideSpotlight.vue'
import { useGuideCopy } from './copy'
import { buildDownloadBlocks } from './downloads'
import { useGuideStore } from './guideStore'
import { detectArch, detectOS, type Env } from './platform'
import { useDownloadResources } from './useDownloadResources'
import { useFirstCallWatcher } from './useFirstCallWatcher'
import { afterKeyStep, FUND_STEPS, nextStep, prevStep, stageOf, type GuideContext, type GuideStepId, type PurchaseMode } from './steps'
import { useGuideTarget, type TargetResolver } from './useGuideTarget'
import { chatPathFor, isChatClient, type GuideClient, type ImageMode } from './config'
import { listImageModels } from '@/features/image/api'
import { buildSkillPrompt, imageBaseUrl, maskKey } from '@/features/image/skillPrompt'
import { CHAT_MODEL_CATALOG } from '@/features/chat/modelCatalog'
import type { GuideBlock, GuideOptionId, GuideView } from './viewModel'
import './guide.css'

const route = useRoute()
const router = useRouter()
const { t, locale } = useI18n()
const auth = useAuthStore()
const app = useAppStore()
const subs = useSubscriptionStore()
const affiliateSummary = tryUseAffiliateSummaryStore()
const welcome = useTutorialWelcomeStore()
const guide = useGuideStore()
const copy = useGuideCopy()
const uiLabels = useUiLabels()

const affiliateEnabled = computed(() =>
  resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.affiliate),
)
const isTrialActive = computed(() => isTrialBonusActive(affiliateSummary?.detail ?? null))
const affiliateRebateRate = computed(() =>
  formatRebateRate(affiliateSummary?.detail?.effective_rebate_rate_percent),
)

/** 只在普通用户的控制台页面里出现：管理员、backend 模式、登录前的页面都不触发。 */
const eligible = computed(
  () =>
    auth.isAuthenticated &&
    !!auth.user &&
    !auth.isAdmin &&
    !app.backendModeEnabled &&
    route.meta.requiresAuth === true &&
    !route.meta.requiresAdmin
)

watch(() => auth.user?.id ?? null, (id) => guide.load(id), { immediate: true })

watch(
  [eligible, () => guide.active, affiliateEnabled],
  ([ok, active, enabled]) => {
    if (ok && active && enabled && !auth.isSimpleMode) {
      void affiliateSummary?.ensureLoaded()
    }
  },
  { immediate: true },
)

// 新注册用户（含 OAuth 首登）：复用教程欢迎弹窗的「待提示」标记启动引导。
watch(
  [eligible, () => welcome.pendingUserId, () => auth.user?.id],
  () => {
    const id = auth.user?.id
    if (!eligible.value || id == null || welcome.pendingUserId !== id) return
    if (guide.status === 'idle') guide.start()
    welcome.dismiss(id)
  },
  { immediate: true }
)

// ---------------------------------------------------------------- 上下文

const customItems = computed(() =>
  (app.cachedPublicSettings?.custom_menu_items ?? [])
    .filter((item) => item.visibility === 'user')
    .sort((a, b) => a.sort_order - b.sort_order)
)
const purchaseMode = computed<PurchaseMode>(() => {
  if (auth.isSimpleMode) return 'none'
  if (app.cachedPublicSettings?.payment_enabled) return 'payment'
  return customItems.value.length > 0 ? 'external' : 'none'
})
const hasFunds = computed(() => (auth.user?.balance ?? 0) > 0 || subs.hasActiveSubscriptions)
const ctx = computed<GuideContext>(() => ({
  hasFunds: hasFunds.value,
  purchaseMode: purchaseMode.value,
  redeemAvailable: !auth.isSimpleMode,
  chatOnly: isChatClient(guide.client),
  imageClient: guide.client === 'image',
  imageMode: guide.imageMode
}))
const isImageWeb = computed(() => guide.client === 'image' && guide.imageMode !== 'agent')

// ---------------------------------------------------------------- 系统与下载资源

const env = ref<Env>({ os: detectOS(), arch: 'unknown' })
void detectArch(env.value.os).then((arch) => (env.value = { os: env.value.os, arch }))
const resources = useDownloadResources()

// 进入下载步骤才去取 manifest 和 Node 版本，其余步骤不发这些请求。
watch(
  () => guide.active && (guide.step === 'download' || guide.step === 'ccs'),
  (need) => {
    if (need) void resources.load()
  },
  { immediate: true }
)

// ---------------------------------------------------------------- 目标

const byAttr = (selector: string): TargetResolver => () => document.querySelector<HTMLElement>(selector)
const sidebarLink = (match: (a: HTMLAnchorElement) => boolean): TargetResolver => () =>
  Array.from(document.querySelectorAll<HTMLAnchorElement>('aside a')).find(match) ?? null

/** 购买入口：站内支付页，或自定义菜单里的第一项（如「购买套餐」外链小铺）。 */
const purchaseResolver = computed<TargetResolver>(() => {
  if (purchaseMode.value === 'payment') return sidebarLink((a) => a.getAttribute('href') === '/purchase')
  const item = customItems.value[0]
  if (!item) return () => null
  const href = item.open_in_new_tab && item.url ? item.url : `/custom/${item.id}`
  return sidebarLink((a) => a.getAttribute('href') === href)
})

/**
 * 密钥名称：没填好之前高亮停在这里。填好后按回车、点卡片里的「下一步」、离开输入框，
 * 或停手 1.2 秒，都算填好，高亮移到分组；再改名字又会回到这里。
 */
const keyNameReady = ref(false)
const keyNameFilled = ref(false)
let keyNameTimer: ReturnType<typeof setTimeout> | undefined
const keyNameInput = () => document.querySelector<HTMLInputElement>('[data-tour="key-form-name"]')
const keyNameResolver: TargetResolver = () => {
  const input = keyNameInput()
  if (!input) return null
  return input.value.trim() === '' || !keyNameReady.value ? input : null
}
function confirmKeyName() {
  clearTimeout(keyNameTimer)
  if (keyNameInput()?.value.trim()) keyNameReady.value = true
}
function isKeyNameEvent(event: Event): event is Event & { target: HTMLInputElement } {
  const target = event.target
  return guide.active && guide.step === 'key' && target instanceof HTMLInputElement && target.dataset.tour === 'key-form-name'
}
function onKeyNameInput(event: Event) {
  if (!isKeyNameEvent(event)) return
  keyNameFilled.value = event.target.value.trim() !== ''
  keyNameReady.value = false
  clearTimeout(keyNameTimer)
  if (keyNameFilled.value) keyNameTimer = setTimeout(confirmKeyName, 1200)
}
function onKeyNameKeydown(event: KeyboardEvent) {
  if (!isKeyNameEvent(event) || event.key !== 'Enter' || event.isComposing) return
  // 引导中回车只是「名字填好了」，不直接提交表单（这时多半还没选分组）
  event.preventDefault()
  confirmKeyName()
}
function onKeyNameBlur(event: Event) {
  if (isKeyNameEvent(event)) confirmKeyName()
}
onMounted(() => {
  document.addEventListener('input', onKeyNameInput, true)
  document.addEventListener('keydown', onKeyNameKeydown, true)
  document.addEventListener('blur', onKeyNameBlur, true)
})
watch(
  () => guide.active && guide.step === 'key',
  () => {
    keyNameReady.value = false
    keyNameFilled.value = false
  }
)

/** 分组：还没选时高亮下拉框，选好后高亮弹窗里的「创建」。 */
const keyGroupResolver: TargetResolver = () => {
  const select = document.querySelector<HTMLElement>('[data-tour="key-form-group"]')
  return select && select.dataset.filled !== '1' ? select : null
}

const resolvers = computed<TargetResolver[]>(() => {
  if (!guide.active) return []
  switch (guide.step) {
    case 'buy':
      return [purchaseResolver.value]
    case 'redeem':
      return [byAttr('[data-tour="redeem-form"]'), sidebarLink((a) => a.getAttribute('href') === '/redeem')]
    case 'key':
      return [
        keyNameResolver,
        keyGroupResolver,
        byAttr('[data-tour="key-form-submit"]'),
        byAttr('[data-tour="keys-create-btn"]'),
        byAttr('[data-tour="sidebar-my-keys"]')
      ]
    case 'import': {
      const mine = guide.keyId != null ? `[data-tour="key-import-ccs"][data-key-id="${guide.keyId}"]` : null
      return [
        mine ? byAttr(mine) : () => null,
        byAttr('[data-tour="key-import-ccs"]'),
        byAttr('[data-tour="sidebar-my-keys"]')
      ]
    }
    default:
      return []
  }
})
const tracking = computed(() => eligible.value && guide.active && resolvers.value.length > 0)
const { rect, anchorRect, inDialog, el, index, missing } = useGuideTarget(resolvers, tracking)

/** 侧栏链接上的文字（「购买套餐」「兑换」…），直接引用界面上的真实名称。 */
const targetText = computed(() => (el.value?.textContent ?? '').replace(/\s+/g, ' ').trim())

// ---------------------------------------------------------------- 已有密钥 / 等第一次调用

/** 重新走引导的用户可能已经有密钥：创建密钥那步给一个「我已有密钥」跳过入口。 */
const hasKey = ref(false)
watch(
  () => guide.active && guide.step === 'key',
  async (on) => {
    if (!on) return
    try {
      hasKey.value = (await keysAPI.list(1, 1)).total > 0
    } catch {
      hasKey.value = false
    }
  },
  { immediate: true }
)

// 完成页等用户在 Codex / Claude Code 里发出第一次调用，查到记录就进入庆祝。
useFirstCallWatcher({
  active: computed(() => eligible.value && guide.active && guide.step === 'finish'),
  keyId: computed(() => guide.keyId),
  onDetected: () => guide.go('success')
})

// ---------------------------------------------------------------- 手动配置用到的值

/** 手动配置的 API 地址：Codex 需要带 /v1，Claude Code 不带（与图文教程一致）。 */
const manualEndpoint = computed(() => {
  const base = (app.cachedPublicSettings?.api_base_url || window.location.origin).replace(/\/+$/, '')
  return guide.client === 'codex' && !base.endsWith('/v1') ? `${base}/v1` : base
})
const manualKey = ref<string | null>(null)
async function loadManualKey() {
  if (manualKey.value) return
  try {
    const row = guide.keyId != null ? await keysAPI.getById(guide.keyId) : (await keysAPI.list(1, 1)).items[0]
    manualKey.value = row?.key ?? null
  } catch {
    manualKey.value = null
  }
  if (manualKey.value && guide.step === 'skill') {
    try {
      skillModel.value = (await listImageModels(manualKey.value))[0] ?? null
    } catch {
      skillModel.value = null
    }
  }
}
/** 智能体那段话里写的模型：这把密钥实际能用的最新生图模型。 */
const skillModel = ref<string | null>(null)
watch(
  () => guide.active && (guide.step === 'manual' || guide.step === 'skill'),
  (on) => {
    if (on) void loadManualKey()
  },
  { immediate: true }
)

/** 复制给智能体的那段话：本站生图地址（带 /v1）+ 引导里建的那把密钥。 */
function skillPromptBlock(): GuideBlock | null {
  const c = copy.value
  if (!manualKey.value) return null
  const lang = locale.value.startsWith('zh') ? 'zh' : 'en'
  const baseUrl = imageBaseUrl(app.cachedPublicSettings?.api_base_url)
  return {
    type: 'prompt',
    heading: c.skill.heading,
    shown: buildSkillPrompt({ lang, baseUrl, apiKey: maskKey(manualKey.value), model: skillModel.value }),
    value: buildSkillPrompt({ lang, baseUrl, apiKey: manualKey.value, model: skillModel.value }),
    note: c.skill.note
  }
}

// ---------------------------------------------------------------- 当前这一屏

const view = computed<GuideView | null>(() => {
  if (!guide.active) return null
  const c = copy.value
  const step = guide.step
  const tool = guide.client ? c.clientName(guide.client) : ''
  const advance = () => guide.go(nextStep(step, ctx.value))
  const go = (path: string, page: string) => ({ label: c.goto(page), run: () => void router.push(path) })

  switch (step) {
    case 'welcome':
      return {
        kind: 'modal',
        key: 'welcome',
        title: isTrialActive.value ? c.welcome.trialTitle : c.welcome.title,
        body: isTrialActive.value ? c.welcome.trialBody : c.welcome.body,
        primary: { label: c.welcome.start, run: advance },
        secondary: { label: c.welcome.skip, run: () => guide.skip() },
        resolvers: []
      }
    case 'client':
      return {
        kind: 'modal',
        key: 'client',
        title: c.client.title,
        body: c.client.body,
        options: [
          { id: 'codex', name: 'Codex', desc: c.client.codex },
          { id: 'claude-code', name: 'Claude Code', desc: c.client.claude },
          { id: 'image', name: c.clientName('image'), desc: c.client.image },
          { id: 'chat', name: c.clientName('chat'), desc: c.client.chat },
          { id: 'fortune', name: c.clientName('fortune'), desc: c.client.fortune }
        ],
        resolvers: []
      }
    case 'imageMode':
      return {
        kind: 'modal',
        key: 'imageMode',
        title: c.imageMode.title,
        body: c.imageMode.body,
        options: [
          { id: 'image-web', name: c.imageMode.web, desc: c.imageMode.webDesc },
          { id: 'image-agent', name: c.imageMode.agent, desc: c.imageMode.agentDesc }
        ],
        resolvers: []
      }
    case 'buy': {
      const label = targetText.value || customItems.value[0]?.label || t('nav.recharge')
      return {
        kind: 'spot',
        key: 'buy',
        title: c.buy.title,
        body: c.buy.body(label),
        hint: c.buy.hint,
        secondary: { label: c.buy.already, run: () => guide.go(ctx.value.redeemAvailable ? 'redeem' : 'key'), quiet: true },
        click: true,
        inSidebar: true,
        resolvers: resolvers.value
      }
    }
    case 'returned':
      return {
        kind: 'modal',
        key: 'returned',
        title: c.returned.title,
        body: c.returned.body,
        primary: { label: c.returned.yes, run: advance },
        secondary: { label: c.returned.no, run: () => guide.go('buy') },
        resolvers: []
      }
    case 'redeem': {
      const onPage = index.value === 0
      return {
        kind: 'spot',
        key: onPage ? 'redeem' : 'redeem-nav',
        title: onPage ? c.redeem.title : c.redeemNav.title,
        body: onPage ? c.redeem.body : c.redeemNav.body(targetText.value || t('nav.redeem')),
        click: !onPage,
        inSidebar: !onPage,
        fallback: missing.value ? go('/redeem', c.pageRedeem) : undefined,
        resolvers: resolvers.value
      }
    }
    case 'key': {
      const phase = (['name', 'form', 'submit', 'create', 'nav'] as const)[index.value] ?? 'nav'
      const inForm = phase === 'name' || phase === 'form' || phase === 'submit'
      const clientName = guide.client ? c.clientName(guide.client) : ''
      const formStep = phase === 'name' ? 0 : phase === 'form' ? 1 : 2
      const titles = { name: c.keyName.title, form: c.keyForm.title, submit: c.keySubmit.title, create: c.keyCreate.title, nav: c.keyNav.title }
      const bodies = {
        name: c.keyName.body(clientName),
        form: guide.client === 'image' ? c.keyForm.bodyImage : isChatClient(guide.client) ? c.keyForm.bodyChat : c.keyForm.body(clientName),
        submit: c.keySubmit.body,
        create: c.keyCreate.body,
        nav: c.keyNav.body(targetText.value || t('nav.apiKeys'))
      }
      return {
        kind: 'spot',
        key: `key-${phase}`,
        title: titles[phase],
        body: bodies[phase],
        // 选好分组后提一句多分组，不强调
        hint: phase === 'submit' ? c.keySubmit.multi : undefined,
        checklist: inForm
          ? c.keySteps.map((label, i) => ({ label, state: i < formStep ? 'done' : i === formStep ? 'active' : 'todo' }))
          : undefined,
        primary: phase === 'name' && keyNameFilled.value ? { label: c.keyName.next, run: confirmKeyName } : undefined,
        click: !inForm || phase === 'submit',
        dim: !inForm,
        inSidebar: phase === 'nav',
        secondary: hasKey.value && !inForm ? { label: c.haveKey, run: () => guide.go(afterKeyStep(ctx.value)), quiet: true } : undefined,
        fallback: missing.value ? go('/keys', c.pageKeys) : undefined,
        resolvers: resolvers.value
      }
    }
    case 'ccs':
      return {
        kind: 'modal',
        key: 'ccs',
        title: c.ccs.title(tool),
        body: c.ccs.body(tool),
        primary: { label: c.ccs.yes, run: advance },
        secondary: { label: c.ccs.no, run: () => guide.go('download') },
        resolvers: []
      }
    case 'download':
      return {
        kind: 'modal',
        key: 'download',
        wide: true,
        collapsible: env.value.os !== 'mobile',
        pillLabel: c.download.pill,
        title: c.download.title,
        body: c.download.body,
        blocks: resources.ready.value
          ? buildDownloadBlocks(
              env.value,
              guide.client === 'codex' ? 'codex' : 'claude-code',
              { manifest: resources.manifest.value, nodeVersion: resources.nodeVersion.value, siteUrl: window.location.origin },
              c
            )
          : undefined,
        hint: resources.ready.value ? undefined : c.download.preparing,
        primary: { label: c.download.cont, run: advance },
        resolvers: []
      }
    case 'import': {
      const onButton = index.value === 0 || index.value === 1
      return {
        kind: 'spot',
        key: onButton ? 'import' : 'import-nav',
        title: onButton ? c.import.title(uiLabels.value.importCcs) : c.importNav.title,
        body: onButton ? (guide.keyId != null ? c.import.body : c.importAny) : c.importNav.body(targetText.value || t('nav.apiKeys')),
        hint: onButton ? c.import.hint : undefined,
        click: true,
        inSidebar: !onButton,
        // 站点关闭了一键导入、或按钮找不到时，不让用户卡住。
        secondary: { label: c.imported.no, run: () => guide.go('manual'), quiet: true },
        fallback: missing.value ? go('/keys', c.pageKeys) : undefined,
        resolvers: resolvers.value
      }
    }
    case 'imported':
      return {
        kind: 'modal',
        key: 'imported',
        title: c.imported.title,
        body: c.imported.body,
        hint: c.imported.hint,
        primary: { label: c.imported.yes, run: advance },
        secondary: { label: c.imported.no, run: () => guide.go('manual') },
        back: { label: c.imported.retry, run: () => guide.go(prevStep('imported')) },
        resolvers: []
      }
    case 'manual': {
      const blocks: GuideBlock[] = [{ type: 'copy', heading: c.manual.endpoint, label: 'URL', value: manualEndpoint.value }]
      if (manualKey.value) blocks.push({ type: 'copy', heading: c.manual.key, label: 'Key', value: manualKey.value, secret: true })
      return {
        kind: 'modal',
        key: 'manual',
        title: c.manual.title,
        body: c.manual.body,
        blocks,
        hint: manualKey.value ? undefined : c.manual.loading,
        primary: { label: c.manual.done, run: advance },
        secondary: {
          label: c.manual.tutorial,
          run: () => void router.push(`/tutorials/${guide.client === 'codex' ? 'codex' : 'claude-code'}`)
        },
        back: { label: c.manual.toDownload, run: () => guide.go('download') },
        resolvers: []
      }
    }
    case 'chat': {
      const target = guide.client === 'fortune' ? 'fortune' : 'chat'
      return {
        kind: 'modal',
        key: 'chat',
        title: c.chatGo.title,
        body: target === 'fortune' ? c.chatGo.bodyFortune : c.chatGo.body,
        blocks: [chatModelsBlock(target)],
        primary: {
          label: c.chatGo.open,
          run: () => {
            advance()
            void router.push(chatPathFor(target))
          }
        },
        resolvers: []
      }
    }
    case 'imageGo':
      return {
        kind: 'modal',
        key: 'imageGo',
        title: c.imageGo.title,
        body: c.imageGo.body,
        primary: {
          label: c.imageGo.open,
          run: () => {
            advance()
            void router.push('/image')
          }
        },
        resolvers: []
      }
    case 'skill': {
      const block = skillPromptBlock()
      return {
        kind: 'modal',
        key: 'skill',
        wide: true,
        title: c.skill.title,
        body: c.skill.body,
        blocks: block ? [block] : undefined,
        hint: block ? undefined : c.skill.loading,
        primary: { label: c.skill.sent, run: advance },
        // 还没装智能体：新标签页打开 Codex 的图文教程（含下载），装好回来接着复制
        secondary: { label: c.skill.noAgent, run: () => void window.open(router.resolve('/tutorials/codex').href, '_blank', 'noopener'), quiet: true },
        resolvers: []
      }
    }
    case 'finish':
      if (guide.client === 'image') {
        const web = isImageWeb.value
        return {
          kind: 'modal' as const,
          key: web ? 'finish-image' : 'finish-skill',
          collapsible: true,
          pillLabel: web ? c.finish.pillImage : c.finish.pill,
          title: web ? c.finish.titleImage : c.finish.titleSkill,
          body: web ? c.finish.bodyImage : c.finish.bodySkill,
          list: web ? c.finish.stepsImage : c.finish.stepsSkill,
          blocks: [{ type: 'status', text: c.finish.waiting }],
          primary: web
            ? {
                label: c.finish.openImage,
                run: () => {
                  chatPillHidden.value = false
                  void router.push('/image')
                }
              }
            : { label: c.finish.done, run: () => guide.finish() },
          secondary: web ? { label: c.finish.done, run: () => guide.finish() } : undefined,
          resolvers: []
        }
      }
      if (isChatClient(guide.client)) {
        const target = guide.client
        return {
          kind: 'modal' as const,
          key: 'finish-chat',
          collapsible: true,
          pillLabel: c.finish.pill,
          title: c.finish.titleChat,
          body: c.finish.bodyChat,
          list: target === 'fortune' ? c.finish.stepsFortune : c.finish.stepsChat,
          blocks: [{ type: 'status', text: c.finish.waiting }],
          primary: {
            label: c.finish.openChat,
            run: () => {
              chatPillHidden.value = false
              // 这一步只需回到聊天页，不再新开对话。
              void router.push('/chat')
            }
          },
          secondary: { label: c.finish.done, run: () => guide.finish() },
          resolvers: []
        }
      }
      return {
        kind: 'modal',
        key: 'finish',
        collapsible: true,
        pillLabel: c.finish.pill,
        title: c.finish.title,
        body: c.finish.body,
        list: guide.client === 'codex' ? c.finish.stepsCodex : c.finish.stepsClaude,
        blocks: [{ type: 'status', text: c.finish.waiting }],
        primary: {
          label: c.finish.tutorial,
          run: () => {
            const target = guide.client === 'codex' ? 'codex' : 'claude-code'
            guide.finish()
            void router.push(`/tutorials/${target}`)
          }
        },
        secondary: { label: c.finish.done, run: () => guide.finish() },
        resolvers: []
      }
    case 'success':
      if (guide.client === 'image') {
        const web = isImageWeb.value
        return {
          kind: 'modal',
          key: 'success',
          title: web ? c.success.titleImage : c.success.titleSkill,
          body: web ? c.success.bodyImage : c.success.bodySkill,
          hint: affiliateEnabled.value && !auth.isSimpleMode ? c.success.affiliateHint(affiliateRebateRate.value) : undefined,
          primary: web
            ? { label: c.success.keepGoing, run: () => guide.finish() }
            : { label: c.success.done, run: () => guide.finish() },
          secondary: {
            label: c.success.usage,
            run: () => {
              guide.finish()
              void router.push('/usage')
            }
          },
          resolvers: []
        }
      }
      return {
        kind: 'modal',
        key: 'success',
        title: c.success.title,
        body: c.success.body,
        hint: affiliateEnabled.value && !auth.isSimpleMode ? c.success.affiliateHint(affiliateRebateRate.value) : undefined,
        primary: {
          label: c.success.usage,
          run: () => {
            guide.finish()
            void router.push('/usage')
          }
        },
        secondary: { label: c.success.done, run: () => guide.finish() },
        resolvers: []
      }
  }
  return null
})

/** 「去聊天」那步讲清楚推荐模型各自强在哪（和聊天页下拉框用同一份名单）。 */
function chatModelsBlock(target: 'chat' | 'fortune'): GuideBlock {
  const c = copy.value
  const zh = locale.value.startsWith('zh')
  return {
    type: 'models',
    heading: target === 'fortune' ? c.chatGo.modelsHeadingFortune : c.chatGo.modelsHeading,
    hint: c.chatGo.modelsHint,
    items: CHAT_MODEL_CATALOG.map((entry, i) => ({
      name: entry.name,
      pitch: zh ? entry.pitch.zh : entry.pitch.en,
      tag: i === 0 ? c.chatGo.defaultTag : undefined
    }))
  }
}

// ---------------------------------------------------------------- 返回上一步 / 页脚

/**
 * 左上角返回箭头回到历史里最近一个还能停下来的步骤：已经有额度时，充值那几步会立刻被跳过，
 * 回到那里没有意义，跳过它们。欢迎页、成功页不显示。
 */
const backIndex = computed(() => {
  if (guide.step === 'welcome' || guide.step === 'success') return -1
  const history = guide.history
  for (let i = history.length - 1; i >= 0; i--) {
    const step = history[i]!
    if (step === guide.step) continue
    if (hasFunds.value && FUND_STEPS.includes(step)) continue
    return i
  }
  return -1
})
function goBack() {
  if (backIndex.value >= 0) guide.backTo(backIndex.value)
}

/** 配置完成后（等第一次调用、接通了；聊天用法从「去聊天」起）不再显示「不想配置？加群」。 */
const showSupport = computed(() => !['chat', 'imageGo', 'finish', 'success'].includes(guide.step))

/** 右下角「图文教程」：按选的工具打开对应教程，新标签页里打开，不打断引导。 */
const tutorialHref = computed(() => {
  const client = guide.client === 'codex' || guide.client === 'claude-code' || guide.client === 'image' ? guide.client : null
  return router.resolve(client ? `/tutorials/${client}` : '/tutorials').href
})

const stage = computed(() => (guide.step === 'welcome' ? -1 : guide.step === 'success' ? 4 : stageOf(guide.step)))

function choose(id: GuideOptionId) {
  if (id === 'image-web' || id === 'image-agent') {
    const mode: ImageMode = id === 'image-web' ? 'web' : 'agent'
    guide.chooseImageMode(mode)
    guide.go(nextStep('imageMode', ctx.value))
    return
  }
  guide.chooseClient(id as GuideClient)
  guide.go(nextStep('client', ctx.value))
}

// ---------------------------------------------------------------- 推进与完成

const completing = ref<'redeem' | 'key' | null>(null)
let completeTimer: ReturnType<typeof setTimeout> | undefined

/** 先给一个对勾动画，再进入下一步。 */
function complete(kind: 'redeem' | 'key', next: GuideStepId) {
  if (completing.value) return
  completing.value = kind
  completeTimer = setTimeout(() => {
    completing.value = null
    guide.go(next)
  }, 1100)
}

// 额度到账（兑换成功、订阅生效、别的标签页里兑换）→ 充值阶段完成。
watch(
  [hasFunds, () => guide.step, () => guide.active, () => guide.redeemedTick],
  () => {
    if (guide.active && FUND_STEPS.includes(guide.step) && hasFunds.value) complete('redeem', 'key')
  },
  { immediate: true }
)
watch(
  () => guide.redeemedTick,
  () => {
    if (guide.active && FUND_STEPS.includes(guide.step)) complete('redeem', 'key')
  }
)
watch(
  () => guide.keyCreatedTick,
  () => {
    if (guide.active && guide.step === 'key') complete('key', afterKeyStep(ctx.value))
  }
)

// 点了购买入口（会在新标签页打开小铺）→ 切到「买好了吗」；点了「导入到 CC Switch」→ 切到「弹窗确认了吗」。
// 用户回来时卡片已经在那里等着。
let clickedEl: HTMLElement | null = null
function onTargetClick() {
  if (!guide.active) return
  if (guide.step === 'buy') guide.go('returned')
  else if (guide.step === 'import' && (index.value === 0 || index.value === 1)) guide.go('imported')
}
watch(
  [el, () => guide.step, index],
  ([target, step, idx]) => {
    clickedEl?.removeEventListener('click', onTargetClick, true)
    clickedEl = null
    if (target && (step === 'buy' || (step === 'import' && (idx === 0 || idx === 1)))) {
      clickedEl = target
      target.addEventListener('click', onTargetClick, true)
    }
  },
  { flush: 'post' }
)

// 手机上目标在侧栏抽屉里：目标迟迟不可见时帮用户展开抽屉。
let drawerTimer: ReturnType<typeof setTimeout> | undefined
watch(
  [() => view.value?.key, () => rect.value === null],
  () => {
    clearTimeout(drawerTimer)
    if (!view.value?.inSidebar || rect.value) return
    drawerTimer = setTimeout(() => {
      if (window.innerWidth < 1024 && !rect.value && !app.mobileOpen) app.setMobileOpen(true)
    }, 500)
  },
  { immediate: true }
)

// ---------------------------------------------------------------- 收起成胶囊

/** 下载步骤里用户要去装软件：把大卡片收成角落的小胶囊，切回本页时自动展开。 */
const minimized = ref(false)
/**
 * 聊天类用法在聊天页等第一次调用时，卡片默认收成胶囊，不挡输入框；
 * 用户点胶囊展开后本步内不再自动收起。
 */
const chatPillHidden = ref(false)
const onChatPage = computed(
  () =>
    guide.step === 'finish' &&
    ((route.path === '/chat' && isChatClient(guide.client)) || (route.path === '/image' && isImageWeb.value))
)
const showPill = computed(
  () => guide.active && !!view.value?.collapsible && (minimized.value || (onChatPage.value && !chatPillHidden.value))
)
watch(() => guide.step, () => {
  minimized.value = false
  chatPillHidden.value = false
})
watch(() => route.path, (path) => {
  if (path === '/chat' || path === '/image') chatPillHidden.value = false
})
function expandPill() {
  minimized.value = false
  chatPillHidden.value = true
}
function onVisible() {
  if (document.visibilityState === 'visible' && minimized.value) minimized.value = false
}
onMounted(() => document.addEventListener('visibilitychange', onVisible))

// ---------------------------------------------------------------- 跳过确认

const confirming = ref(false)
watch(() => view.value?.key, () => (confirming.value = false))
function confirmSkip() {
  confirming.value = false
  guide.skip()
}

// ---------------------------------------------------------------- 卡片位置

const cardEl = ref<HTMLElement | null>(null)
const cardSize = ref({ w: 380, h: 260 })
const vw = ref(window.innerWidth)
const vh = ref(window.innerHeight)
let observer: ResizeObserver | undefined

watch(cardEl, (node) => {
  observer?.disconnect()
  if (!node || typeof ResizeObserver === 'undefined') return
  observer = new ResizeObserver(() => {
    const r = node.getBoundingClientRect()
    cardSize.value = { w: r.width, h: r.height }
  })
  observer.observe(node)
})

function onResize() {
  vw.value = window.innerWidth
  vh.value = window.innerHeight
}
onMounted(() => window.addEventListener('resize', onResize))

const isSheet = computed(() => vw.value < 640)
/** 手机上目标落在下半屏、又滚不上去时，把抽屉挪到顶部，免得盖住目标。 */
const sheetTop = computed(() => isSheet.value && !!rect.value && rect.value.y + rect.value.h / 2 > vh.value * 0.5)

/**
 * 贴着目标（弹窗里的目标则贴着整个弹窗面板）放：优先右侧，其次左侧、下方、上方。
 * 弹窗里两侧都放不下时停靠在视口左下角——那里压着的只是表单的标签，不会挡住目标和「创建」按钮。
 */
const floatStyle = computed(() => {
  if (isSheet.value) return {}
  const { w, h } = cardSize.value
  const m = 16
  const gap = 20
  const clamp = (n: number, lo: number, hi: number) => Math.min(Math.max(n, lo), Math.max(lo, hi))
  const r = anchorRect.value
  if (!r) return { transform: `translate3d(${vw.value - w - 24}px, ${vh.value - h - 24}px, 0)` }
  const spots = [
    { x: r.x + r.w + gap, y: r.y + r.h / 2 - h / 2, ok: r.x + r.w + gap + w <= vw.value - m },
    { x: r.x - gap - w, y: r.y + r.h / 2 - h / 2, ok: r.x - gap - w >= m },
    { x: r.x + r.w / 2 - w / 2, y: r.y + r.h + gap, ok: !inDialog.value && r.y + r.h + gap + h <= vh.value - m },
    { x: r.x + r.w / 2 - w / 2, y: r.y - gap - h, ok: !inDialog.value && r.y - gap - h >= m }
  ]
  const pick = spots.find((s) => s.ok) ?? (inDialog.value ? { x: m, y: vh.value - h - m } : spots[2]!)
  return { transform: `translate3d(${clamp(pick.x, m, vw.value - w - m)}px, ${clamp(pick.y, m, vh.value - h - m)}px, 0)` }
})

// ---------------------------------------------------------------- 焦点与键盘

watch(
  () => [view.value?.key, completing.value, confirming.value],
  async () => {
    if (!view.value || view.value.kind !== 'modal') return
    await nextTick()
    cardEl.value?.querySelector<HTMLElement>('[data-og-autofocus]')?.focus({ preventScroll: true })
  },
  { flush: 'post' }
)

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    confirming.value = true
    return
  }
  if (event.key !== 'Tab' || view.value?.kind !== 'modal' || !cardEl.value) return
  const controls = Array.from(cardEl.value.querySelectorAll<HTMLElement>('button, a[href], input')).filter((n) => !n.hasAttribute('disabled'))
  if (!controls.length) return
  const first = controls[0]!
  const last = controls[controls.length - 1]!
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

// ---------------------------------------------------------------- 结束后飞进头像

/** 引导完成或跳过时，卡片飞进右上角头像，并提示「点头像随时再来一遍」。每个账号只演一次。 */
const flyToken = ref(0)
const flyFrom = ref<DOMRect | null>(null)
const flyGhost = ref<HTMLElement | null>(null)
const flyKey = () => (auth.user?.id == null ? null : `onboarding_flyhome_v1:${auth.user.id}`)
watch(
  () => guide.status,
  (next, prev) => {
    if (prev !== 'active' || (next !== 'done' && next !== 'skipped') || !eligible.value) return
    const key = flyKey()
    try {
      if (!key || localStorage.getItem(key)) return
      localStorage.setItem(key, '1')
    } catch {
      /* 存不了也照样演一次 */
    }
    flyFrom.value = cardEl.value?.getBoundingClientRect() ?? null
    // 带着卡片的样子起飞：复制一份当前卡片，飞行途中淡出成光点
    const ghost = cardEl.value?.cloneNode(true) as HTMLElement | undefined
    ghost?.querySelectorAll('[id]').forEach((n) => n.removeAttribute('id'))
    flyGhost.value = ghost ?? null
    flyToken.value++
  },
  { flush: 'sync' }
)

// ---------------------------------------------------------------- 老用户邀请

const inviteVisible = ref(false)
let inviteTimer: ReturnType<typeof setTimeout> | undefined
let inviteToken = 0

/** 老用户（还没创建过 Key）：在控制台首页给一张不打扰的邀请卡片，而不是直接弹全屏。 */
watch(
  [eligible, () => guide.status, () => route.path, () => auth.user?.id, () => welcome.pendingUserId],
  async ([ok, status, path, id, pending]) => {
    clearTimeout(inviteTimer)
    inviteVisible.value = false
    const token = ++inviteToken
    if (!ok || id == null || status !== 'idle' || path !== '/dashboard' || pending === id) return
    try {
      const res = await keysAPI.list(1, 1)
      if (token !== inviteToken || guide.status !== 'idle') return
      if (res.total > 0) {
        guide.dismiss() // 已经有 Key：不需要邀请，也不用下次再查
        return
      }
      inviteTimer = setTimeout(() => (inviteVisible.value = true), 1200)
    } catch {
      /* 查询失败就不邀请 */
    }
  },
  { immediate: true }
)

function startFromInvite() {
  inviteVisible.value = false
  guide.start()
}
function dismissInvite() {
  inviteVisible.value = false
  guide.dismiss()
}

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  observer?.disconnect()
  clickedEl?.removeEventListener('click', onTargetClick, true)
  document.removeEventListener('visibilitychange', onVisible)
  clearTimeout(completeTimer)
  clearTimeout(drawerTimer)
  clearTimeout(inviteTimer)
  clearTimeout(keyNameTimer)
  document.removeEventListener('input', onKeyNameInput, true)
  document.removeEventListener('keydown', onKeyNameKeydown, true)
  document.removeEventListener('blur', onKeyNameBlur, true)
})
</script>

<template>
  <div class="og-root">
    <template v-if="eligible && guide.active && view">
      <Transition name="og-fade">
        <div v-if="view.kind === 'modal' && !showPill" class="og-backdrop" aria-hidden="true">
          <GlowDots v-if="guide.step === 'welcome'" class="og-glow" :center-x="0.5" :center-y="0.46" :cell="5" :max-cells="14000" :gain="0.5" :interactive="false" />
        </div>
      </Transition>

      <GuideSpotlight v-if="view.kind === 'spot'" :rect="rect" :cursor="view.click" :dim="view.dim !== false" />

      <Transition name="og-card" appear>
        <div v-if="view.kind === 'modal' && !showPill" class="og-center">
          <section ref="cardEl" class="og-card og-card--modal" :class="{ 'og-card--wide': view.wide }" role="dialog" aria-modal="true" aria-labelledby="og-title" @keydown="onKeydown">
            <GuideCardBody
              :view="view"
              :stage="stage"
              :completing="completing"
              :confirming="confirming"
              :show-skip="guide.step !== 'welcome' && guide.step !== 'success'"
              :can-back="backIndex >= 0"
              :show-support="showSupport"
              :tutorial-href="tutorialHref"
              @back="goBack"
              @choose="choose"
              @ask-skip="confirming = true"
              @confirm-skip="confirmSkip"
              @cancel-skip="confirming = false"
              @collapse="minimized = true"
            />
          </section>
        </div>
      </Transition>

      <Transition name="og-card" appear>
        <section
          v-if="view.kind === 'spot'"
          ref="cardEl"
          class="og-card og-card--float"
          :class="{ 'og-sheet': isSheet, 'og-sheet--top': sheetTop }"
          :style="floatStyle"
          role="dialog"
          aria-modal="false"
          aria-labelledby="og-title"
          @keydown.esc="confirming = true"
        >
          <GuideCardBody
            :view="view"
            :stage="stage"
            :completing="completing"
            :confirming="confirming"
            :show-skip="true"
            :can-back="backIndex >= 0"
            :show-support="showSupport"
            :tutorial-href="tutorialHref"
            @back="goBack"
            @choose="choose"
            @ask-skip="confirming = true"
            @confirm-skip="confirmSkip"
            @cancel-skip="confirming = false"
          />
        </section>
      </Transition>
      <GuideConfetti v-if="guide.step === 'success' && !completing" :key="guide.step" />

      <Transition name="og-invite">
        <button v-if="showPill" type="button" class="og-pill" :class="{ 'og-pill--chat': onChatPage }" @click="expandPill">
          <span class="og-pill-dot" aria-hidden="true"></span>{{ view?.pillLabel ?? copy.download.pill }}
        </button>
      </Transition>
    </template>

    <GuideInvite :show="inviteVisible && eligible && !guide.active" @start="startFromInvite" @dismiss="dismissInvite" />
    <GuideFlyHome :token="flyToken" :from="flyFrom" :ghost="flyGhost" />
  </div>
</template>
