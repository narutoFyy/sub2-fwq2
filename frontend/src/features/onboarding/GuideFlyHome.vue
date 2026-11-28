<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useGuideCopy } from './copy'

/**
 * 引导结束（完成或跳过）时：卡片缩成一颗光点飞进右上角头像，头像亮一下，
 * 再在头像下面轻轻说一句「忘了怎么操作？点头像，随时再来一遍」。
 * token 每变一次播一次；from 是卡片最后的位置（拿不到就从屏幕中间起飞）。
 */
const props = defineProps<{ token: number; from: DOMRect | null; ghost?: HTMLElement | null }>()
const emit = defineEmits<{ done: [] }>()

const copy = useGuideCopy()
const flyEl = ref<HTMLElement | null>(null)
const flying = ref(false)
const flyStyle = ref<Record<string, string>>({})
const ping = ref<Record<string, string> | null>(null)
const tip = ref<Record<string, string> | null>(null)
let tipTimer: ReturnType<typeof setTimeout> | undefined
let pingTimer: ReturnType<typeof setTimeout> | undefined
let animation: Animation | null = null

const SELECTOR = '[data-tour="header-avatar"]'

function findAvatar(): HTMLElement | null {
  const el = document.querySelector<HTMLElement>(SELECTOR)
  if (!el) return null
  const r = el.getBoundingClientRect()
  return r.width > 0 && r.height > 0 ? el : null
}

/** 换页后头像可能还没挂上：最多等 1 秒。 */
async function waitForAvatar(): Promise<HTMLElement | null> {
  for (let i = 0; i < 20; i++) {
    const el = findAvatar()
    if (el) return el
    await new Promise((r) => setTimeout(r, 50))
  }
  return null
}

function showTip(avatar: DOMRect) {
  const right = Math.max(16, window.innerWidth - avatar.right)
  tip.value = {
    top: `${avatar.bottom + 12}px`,
    right: `${right}px`,
    '--tip-arrow': `${Math.max(12, avatar.width / 2 - 5)}px`
  }
  clearTimeout(tipTimer)
  tipTimer = setTimeout(closeTip, 12000)
  // 用户点了头像（打开菜单）就不用再提示了
  avatarListener = () => closeTip()
  findAvatar()?.addEventListener('click', avatarListener, { once: true })
}

function showPing(avatar: DOMRect) {
  ping.value = { left: `${avatar.left}px`, top: `${avatar.top}px`, width: `${avatar.width}px`, height: `${avatar.height}px` }
  clearTimeout(pingTimer)
  pingTimer = setTimeout(() => (ping.value = null), 950)
}

let avatarListener: (() => void) | null = null
function closeTip() {
  clearTimeout(tipTimer)
  if (avatarListener) findAvatar()?.removeEventListener('click', avatarListener)
  avatarListener = null
  if (!tip.value) return
  tip.value = null
  emit('done')
}

async function play() {
  const avatarEl = await waitForAvatar()
  if (!avatarEl) return emit('done')
  const target = avatarEl.getBoundingClientRect()
  const reduced = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
  if (reduced || typeof Element.prototype.animate !== 'function') {
    showTip(target)
    return
  }

  const vw = window.innerWidth
  const vh = window.innerHeight
  const from = props.from && props.from.width > 0 ? props.from : new DOMRect(vw / 2 - 190, vh / 2 - 130, 380, 260)
  const w = from.width
  const h = from.height
  const dot = 14
  const endX = target.left + target.width / 2 - dot / 2
  const endY = target.top + target.height / 2 - dot / 2
  // 弧线：先收成一颗小光点，往右上方抛过去
  const midX = from.left + (endX - from.left) * 0.55
  const midY = Math.min(from.top, endY) + (endY - from.top) * 0.2 - 60

  flyStyle.value = { width: `${w}px`, height: `${h}px`, transformOrigin: '0 0', borderRadius: '20px' }
  flying.value = true
  await nextTick()
  const el = flyEl.value
  if (!el) return emit('done')
  let ghostAnim: Animation | null = null
  if (props.ghost) {
    const ghost = props.ghost
    Object.assign(ghost.style, { position: 'absolute', inset: '0', width: '100%', height: '100%', margin: '0', transform: 'none', pointerEvents: 'none', boxShadow: 'none' })
    el.appendChild(ghost)
    ghostAnim = ghost.animate([{ opacity: 1 }, { opacity: 0, offset: 0.3 }, { opacity: 0 }], { duration: 1000, fill: 'forwards' })
  }

  const sx = dot / w
  const sy = dot / h
  animation = el.animate(
    [
      { transform: `translate(${from.left}px, ${from.top}px) scale(1, 1)`, borderRadius: '20px', opacity: 1, offset: 0 },
      { transform: `translate(${from.left + w * 0.35}px, ${from.top + h * 0.35}px) scale(0.3, 0.3)`, borderRadius: '40px', opacity: 1, offset: 0.28 },
      { transform: `translate(${midX}px, ${midY}px) scale(${sx * 1.6}, ${sy * 1.6})`, borderRadius: '50%', opacity: 1, offset: 0.62 },
      { transform: `translate(${endX}px, ${endY}px) scale(${sx}, ${sy})`, borderRadius: '50%', opacity: 0.2, offset: 1 }
    ],
    { duration: 1000, easing: 'cubic-bezier(0.65, 0, 0.25, 1)', fill: 'forwards' }
  )
  try {
    await animation.finished
  } catch {
    /* 被打断 */
  }
  ghostAnim?.cancel()
  flying.value = false
  animation = null
  const fresh = findAvatar()?.getBoundingClientRect() ?? target
  showPing(fresh)
  showTip(fresh)
}

watch(
  () => props.token,
  (next, prev) => {
    if (next > 0 && next !== prev) void play()
  }
)

onBeforeUnmount(() => {
  animation?.cancel()
  clearTimeout(tipTimer)
  clearTimeout(pingTimer)
})
</script>

<template>
  <Teleport to="body">
    <div class="og-root og-fly-layer">
      <div v-if="flying" ref="flyEl" class="og-fly" :style="flyStyle" aria-hidden="true"></div>
      <span v-if="ping" class="og-avatar-ping" :style="ping" aria-hidden="true"></span>
      <Transition name="og-tip">
        <div v-if="tip" class="og-home-tip" :style="tip" role="status">
          <span>{{ copy.flyHome.text }}</span>
          <button type="button" @click="closeTip">{{ copy.flyHome.ok }}</button>
        </div>
      </Transition>
    </div>
  </Teleport>
</template>
