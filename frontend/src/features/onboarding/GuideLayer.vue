<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { keysAPI } from '@/api/keys'
import { useAppStore, useAuthStore, useSubscriptionStore } from '@/stores'
import { useTutorialWelcomeStore } from '@/features/tutorials/welcomeStore'
import GlowDots from '@/components/visual/GlowDots.vue'
import GuideCardBody from './GuideCardBody.vue'
import GuideInvite from './GuideInvite.vue'
import GuideSpotlight from './GuideSpotlight.vue'
import { useGuideCopy } from './copy'
import { useGuideStore } from './guideStore'
import { FUND_STEPS, nextStep, prevStep, stageOf, type GuideContext, type PurchaseMode } from './steps'
import { useGuideTarget, type TargetResolver } from './useGuideTarget'
import type { GuideClient } from './config'
import type { GuideView } from './viewModel'
import './guide.css'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const auth = useAuthStore()
const app = useAppStore()
const subs = useSubscriptionStore()
const welcome = useTutorialWelcomeStore()
const guide = useGuideStore()
const copy = useGuideCopy()

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
  redeemAvailable: !auth.isSimpleMode
}))

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

const resolvers = computed<TargetResolver[]>(() => {
  if (!guide.active) return []
  switch (guide.step) {
    case 'buy':
      return [purchaseResolver.value]
    case 'redeem':
      return [byAttr('[data-tour="redeem-form"]'), sidebarLink((a) => a.getAttribute('href') === '/redeem')]
    case 'key':
      return [
        byAttr('[data-tour="key-form-group"]'),
        byAttr('[data-tour="keys-create-btn"]'),
        byAttr('[data-tour="sidebar-my-keys"]')
      ]
    default:
      return []
  }
})
const tracking = computed(() => eligible.value && guide.active && resolvers.value.length > 0)
const { rect, anchorRect, inDialog, el, index, missing } = useGuideTarget(resolvers, tracking)

/** 侧栏链接上的文字（「购买套餐」「兑换」…），直接引用界面上的真实名称。 */
const targetText = computed(() => (el.value?.textContent ?? '').replace(/\s+/g, ' ').trim())

// ---------------------------------------------------------------- 当前这一屏

const view = computed<GuideView | null>(() => {
  if (!guide.active) return null
  const c = copy.value
  const step = guide.step
  const advance = () => guide.go(nextStep(step, ctx.value))
  const go = (path: string, page: string) => ({ label: c.goto(page), run: () => void router.push(path) })

  switch (step) {
    case 'welcome':
      return {
        kind: 'modal',
        key: 'welcome',
        title: c.welcome.title,
        body: c.welcome.body,
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
          { id: 'claude-code', name: 'Claude Code', desc: c.client.claude }
        ],
        back: { label: c.back, run: () => guide.go(prevStep('client')) },
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
        secondary: { label: c.buy.already, run: () => guide.go(ctx.value.redeemAvailable ? 'redeem' : 'key') },
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
      const phase = index.value === 0 ? 'form' : index.value === 1 ? 'create' : 'nav'
      const clientName = guide.client ? c.clientName(guide.client) : ''
      return {
        kind: 'spot',
        key: `key-${phase}`,
        title: phase === 'form' ? c.keyForm.title : phase === 'create' ? c.keyCreate.title : c.keyNav.title,
        body:
          phase === 'form'
            ? c.keyForm.body(clientName)
            : phase === 'create'
              ? c.keyCreate.body
              : c.keyNav.body(targetText.value || t('nav.apiKeys')),
        click: phase !== 'form',
        dim: phase !== 'form',
        inSidebar: phase === 'nav',
        fallback: missing.value ? go('/keys', c.pageKeys) : undefined,
        resolvers: resolvers.value
      }
    }
    case 'finish':
      return {
        kind: 'modal',
        key: 'finish',
        title: c.finish.title,
        body: c.finish.body,
        list: c.finish.steps,
        primary: {
          label: c.finish.tutorial,
          run: () => {
            const target = guide.client ?? 'claude-code'
            guide.finish()
            void router.push(`/tutorials/${target}`)
          }
        },
        secondary: { label: c.finish.done, run: () => guide.finish() },
        resolvers: []
      }
  }
  return null
})

const stage = computed(() => (guide.step === 'welcome' ? -1 : stageOf(guide.step)))

function choose(client: GuideClient) {
  guide.chooseClient(client)
  guide.go(nextStep('client', ctx.value))
}

// ---------------------------------------------------------------- 推进与完成

const completing = ref<'redeem' | 'key' | null>(null)
let completeTimer: ReturnType<typeof setTimeout> | undefined

/** 先给一个对勾动画，再进入下一步。 */
function complete(kind: 'redeem' | 'key', next: 'key' | 'finish') {
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
    if (guide.active && guide.step === 'key') complete('key', 'finish')
  }
)

// 点了购买入口（会在新标签页打开小铺）→ 切到「买好了吗」，用户回来就能看到。
let clickedEl: HTMLElement | null = null
function onPurchaseClick() {
  if (guide.active && guide.step === 'buy') guide.go('returned')
}
watch(
  [el, () => guide.step],
  ([target, step]) => {
    clickedEl?.removeEventListener('click', onPurchaseClick, true)
    clickedEl = null
    if (step === 'buy' && target) {
      clickedEl = target
      target.addEventListener('click', onPurchaseClick, true)
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
    cardEl.value?.querySelector<HTMLElement>('[data-og-autofocus]')?.focus()
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
  clickedEl?.removeEventListener('click', onPurchaseClick, true)
  clearTimeout(completeTimer)
  clearTimeout(drawerTimer)
  clearTimeout(inviteTimer)
})
</script>

<template>
  <div class="og-root">
    <template v-if="eligible && guide.active && view">
      <Transition name="og-fade">
        <div v-if="view.kind === 'modal'" class="og-backdrop" aria-hidden="true">
          <GlowDots v-if="guide.step === 'welcome'" class="og-glow" :center-x="0.5" :center-y="0.46" :cell="5" :max-cells="14000" :gain="0.5" :interactive="false" />
        </div>
      </Transition>

      <GuideSpotlight v-if="view.kind === 'spot'" :rect="rect" :cursor="view.click" :dim="view.dim !== false" />

      <Transition name="og-card" appear>
        <div v-if="view.kind === 'modal'" class="og-center">
          <section ref="cardEl" class="og-card og-card--modal" role="dialog" aria-modal="true" aria-labelledby="og-title" @keydown="onKeydown">
            <GuideCardBody
              :view="view"
              :stage="stage"
              :completing="completing"
              :confirming="confirming"
              :show-skip="guide.step !== 'welcome'"
              @choose="choose"
              @ask-skip="confirming = true"
              @confirm-skip="confirmSkip"
              @cancel-skip="confirming = false"
            />
          </section>
        </div>
      </Transition>

      <Transition name="og-card" appear>
        <section
          v-if="view.kind === 'spot'"
          ref="cardEl"
          class="og-card og-card--float"
          :class="{ 'og-sheet': isSheet }"
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
            @choose="choose"
            @ask-skip="confirming = true"
            @confirm-skip="confirmSkip"
            @cancel-skip="confirming = false"
          />
        </section>
      </Transition>
    </template>

    <GuideInvite :show="inviteVisible && eligible && !guide.active" @start="startFromInvite" @dismiss="dismissInvite" />
  </div>
</template>
