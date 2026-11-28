<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import GlowDots from '@/components/visual/GlowDots.vue'
import { useAuthStore } from '@/stores/auth'
import { useGuideStore } from '@/features/onboarding/guideStore'
import { GatewayError, generateImages, ImageTimeoutError, isAbortError, listImageModels, toLocalSrc, type ImageQuality, type ImageRatio } from './api'
import { useImageCopy } from './copy'
import { clearRecords, deleteRecord, loadRecords, saveRecord, type ImageRecord } from './history'
import { MY_IMAGE_DEFAULT_MODEL } from './skillPrompt'
import SkillPromptDialog from './SkillPromptDialog.vue'
import { useImageKeys } from './useImageKeys'
import './image.css'

const auth = useAuthStore()
const guide = useGuideStore()
const copy = useImageCopy()
const userId = computed(() => auth.user?.id ?? null)

// ---------------------------------------------------------------- 密钥与模型

const { keys, loaded: keysLoaded, load: loadKeys, hasKeys } = useImageKeys()
const keyId = ref<number | null>(null)
const keyStorage = () => (userId.value == null ? null : `image_page_key:${userId.value}`)
const selectedKey = computed(() => keys.value.find((k) => k.id === keyId.value) ?? null)

const models = ref<string[]>([])
const model = ref(MY_IMAGE_DEFAULT_MODEL)
const modelsChecked = ref(false)

watch(keys, (list) => {
  if (list.some((k) => k.id === keyId.value)) return
  let saved: number | null = null
  try {
    const key = keyStorage()
    saved = key ? Number(localStorage.getItem(key)) || null : null
  } catch {
    saved = null
  }
  // 新手引导里刚建的那把优先（引导卡片承诺了「自动用上刚建的密钥」），其次上次用的，最后是列表第一把
  const guided = guide.active && guide.client === 'image' ? list.find((k) => k.id === guide.keyId) : undefined
  keyId.value = (guided ?? list.find((k) => k.id === saved) ?? list[0])?.id ?? null
})

let modelsToken = 0
watch(selectedKey, async (key) => {
  try {
    const storageKey = keyStorage()
    if (storageKey && key) localStorage.setItem(storageKey, String(key.id))
  } catch {
    /* ignore */
  }
  models.value = []
  modelsChecked.value = false
  if (!key) return
  const token = ++modelsToken
  try {
    const found = await listImageModels(key.key)
    if (token !== modelsToken) return
    models.value = found
    model.value = found[0] ?? MY_IMAGE_DEFAULT_MODEL
  } catch {
    if (token !== modelsToken) return
    models.value = []
    model.value = MY_IMAGE_DEFAULT_MODEL
  } finally {
    if (token === modelsToken) modelsChecked.value = true
  }
})
const modelOptions = computed(() => (models.value.length ? models.value : [MY_IMAGE_DEFAULT_MODEL]))
const noImageModel = computed(() => modelsChecked.value && models.value.length === 0)

// ---------------------------------------------------------------- 表单

const prompt = ref('')
const ratio = ref<ImageRatio>('square')
/**
 * 每次只生 1 张：张数越多越慢，线上请求超过约 100 秒会被 Cloudflare 断开，
 * 扣了费却拿不到图（异步生图没开）。所以页面上不提供张数选择。
 */
const IMAGES_PER_RUN = 1
const quality = ref<ImageQuality>('auto')
const promptEl = ref<HTMLTextAreaElement | null>(null)
const stageEl = ref<HTMLElement | null>(null)
const ratios: ImageRatio[] = ['square', 'landscape', 'portrait']

function useExample(text: string) {
  prompt.value = text
  void nextTick(() => promptEl.value?.focus())
}

const reference = ref<Blob | null>(null)
const referenceUrl = ref<string | null>(null)
const referenceError = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const dragging = ref(false)

function setReference(blob: Blob | null) {
  if (referenceUrl.value) URL.revokeObjectURL(referenceUrl.value)
  referenceError.value = ''
  reference.value = blob
  referenceUrl.value = blob ? URL.createObjectURL(blob) : null
}
function pickFile(file: File | undefined | null) {
  if (!file || !file.type.startsWith('image/')) return
  if (file.size > 20 * 1024 * 1024) {
    referenceError.value = copy.value.referenceTooBig
    return
  }
  setReference(file)
}
function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  pickFile(input.files?.[0])
  input.value = ''
}
function onDrop(event: DragEvent) {
  dragging.value = false
  pickFile(event.dataTransfer?.files?.[0])
}

// ---------------------------------------------------------------- 生成

type Status = 'idle' | 'busy' | 'done' | 'error'
const status = ref<Status>('idle')
const errorText = ref('')
const errorCode = ref<string | null>(null)
const current = ref<ImageRecord | null>(null)
/** 生成中的那一次用的比例：画布形状跟着它，而不是跟着表单里正在改的值。 */
const busyRatio = ref<ImageRatio>('square')
const elapsed = ref(0)
/** 刚生成的那一组才播「显影」动画；从历史里打开的直接显示。 */
const freshId = ref<string | null>(null)
let tick: ReturnType<typeof setInterval> | undefined
let controller: AbortController | null = null

const canGenerate = computed(() => !!selectedKey.value && prompt.value.trim().length > 0 && status.value !== 'busy')
const stageRatio = computed<ImageRatio>(() => (status.value === 'busy' ? busyRatio.value : current.value?.ratio ?? ratio.value))

function errorMessage(error: unknown): string {
  const c = copy.value.errors
  if (error instanceof ImageTimeoutError) return c.timeout
  if (error instanceof GatewayError) {
    if (error.code === 'invalid_key') return c.invalid_key
    // 线上 403：Image generation is not enabled for this group —— 说清楚是分组权限，不是余额
    if (error.code === 'billing' && /image generation|not enabled|permission/i.test(error.message)) return c.noPermission
    if (error.code === 'billing') return c.billing
    if (error.code === 'rate_limit') return c.rate_limit
    return c.upstream(error.message)
  }
  if (error instanceof TypeError) return c.network
  return c.upstream(error instanceof Error ? error.message : String(error))
}

async function generate() {
  if (!canGenerate.value || !selectedKey.value || userId.value == null) return
  const text = prompt.value.trim()
  const usedRatio = ratio.value
  // 窄屏时画布在表单下面：点了生成要把画布带到眼前，不然用户看不到在生成
  if (window.innerWidth < 1280) void nextTick(() => stageEl.value?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
  controller = new AbortController()
  status.value = 'busy'
  busyRatio.value = usedRatio
  errorText.value = ''
  elapsed.value = 0
  clearInterval(tick)
  tick = setInterval(() => elapsed.value++, 1000)
  try {
    const images = await generateImages({
      apiKey: selectedKey.value.key,
      model: model.value,
      prompt: text,
      ratio: usedRatio,
      count: IMAGES_PER_RUN,
      quality: quality.value,
      reference: reference.value,
      signal: controller.signal
    })
    if (!images.length) throw new Error(copy.value.errors.empty)
    const record: ImageRecord = {
      id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      userId: userId.value,
      prompt: text,
      ratio: usedRatio,
      model: model.value,
      createdAt: Date.now(),
      images: images.map((img) => img.src)
    }
    current.value = record
    freshId.value = record.id
    history.value = [record, ...history.value]
    status.value = 'done'
    void persist(record)
  } catch (error) {
    if (isAbortError(error)) {
      status.value = current.value ? 'done' : 'idle'
    } else {
      errorCode.value = error instanceof GatewayError ? error.code : null
      errorText.value = error instanceof Error && error.message === copy.value.errors.empty ? error.message : errorMessage(error)
      status.value = 'error'
    }
  } finally {
    clearInterval(tick)
    controller = null
  }
}

/** 图床链接会过期：先转成本地数据再存历史。正在看的这张不换地址，免得图片闪一下。 */
async function persist(record: ImageRecord) {
  const images = await Promise.all(record.images.map(toLocalSrc))
  const saved = { ...record, images }
  history.value = history.value.map((r) => (r.id === record.id ? saved : r))
  await saveRecord(saved)
}

function stop() {
  controller?.abort()
}

function onPromptKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' && (event.metaKey || event.ctrlKey) && !event.isComposing) {
    event.preventDefault()
    void generate()
  }
}

const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform)

// ---------------------------------------------------------------- 结果操作

/** 跨域链接直接加 download 属性不会下载、只会新开标签页，所以先取回本地再存。 */
async function download(src: string, index: number) {
  const name = `image-${current.value?.createdAt ?? Date.now()}-${index + 1}.png`
  // 生成后已经在后台把图存到本地了，有本地版本就直接用，不用再等一次下载
  const local = history.value.find((r) => r.id === current.value?.id)?.images[index]
  let href = local?.startsWith('data:') ? local : src
  let revoke = false
  if (/^https?:/i.test(href)) {
    try {
      const response = await fetch(href)
      if (!response.ok) throw new Error(String(response.status))
      href = URL.createObjectURL(await response.blob())
      revoke = true
    } catch {
      window.open(src, '_blank', 'noopener')
      return
    }
  }
  const a = document.createElement('a')
  a.href = href
  a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()
  if (revoke) setTimeout(() => URL.revokeObjectURL(href), 1000)
}

async function useAsReference(src: string) {
  try {
    const blob = await (await fetch(src)).blob()
    setReference(blob)
    // 改图默认沿用原图的比例，描述清空让用户只写「怎么改」
    if (current.value) ratio.value = current.value.ratio
    prompt.value = ''
    void nextTick(() => promptEl.value?.focus())
  } catch {
    /* 跨域图片取不到就算了 */
  }
}

function again() {
  if (!current.value) return
  prompt.value = current.value.prompt
  ratio.value = current.value.ratio
  void generate()
}

const lightbox = ref<string | null>(null)
function onWindowKey(event: KeyboardEvent) {
  if (event.key === 'Escape') lightbox.value = null
}

// ---------------------------------------------------------------- 历史

const history = ref<ImageRecord[]>([])
watch(
  userId,
  async (id) => {
    history.value = id == null ? [] : await loadRecords(id)
  },
  { immediate: true }
)
function openRecord(record: ImageRecord) {
  // 输入框空着就把那次的描述和比例带回来，方便微调后再生成；正在写的草稿不覆盖
  if (!prompt.value.trim()) {
    prompt.value = record.prompt
    ratio.value = record.ratio
  }
  current.value = record
  status.value = 'done'
  errorText.value = ''
}
function removeRecord(record: ImageRecord) {
  history.value = history.value.filter((r) => r.id !== record.id)
  if (current.value?.id === record.id) {
    current.value = null
    status.value = 'idle'
  }
  void deleteRecord(record.id)
}
function clearHistory() {
  if (userId.value == null || !window.confirm(copy.value.historyClearConfirm)) return
  history.value = []
  current.value = null
  status.value = 'idle'
  void clearRecords(userId.value)
}

// ---------------------------------------------------------------- 智能体

const skillOpen = ref(false)

onMounted(() => {
  void loadKeys()
  window.addEventListener('keydown', onWindowKey)
})
onBeforeUnmount(() => {
  controller?.abort()
  clearInterval(tick)
  if (referenceUrl.value) URL.revokeObjectURL(referenceUrl.value)
  window.removeEventListener('keydown', onWindowKey)
})
</script>

<template>
  <AppLayout>
    <div class="img-page">
      <div class="img-grid">
        <!-- 左：怎么画 -->
        <form class="img-panel" @submit.prevent="generate">
          <div class="img-field">
            <label for="img-prompt" class="img-label">{{ copy.promptLabel }}</label>
            <div class="img-prompt-box">
              <textarea
                id="img-prompt"
                ref="promptEl"
                v-model="prompt"
                class="img-prompt"
                data-tour="image-prompt"
                rows="5"
                :placeholder="reference ? copy.promptPlaceholderEdit : copy.promptPlaceholder"
                @keydown="onPromptKeydown"
              />
              <span class="img-kbd" aria-hidden="true"><kbd>{{ isMac ? '⌘' : 'Ctrl' }}</kbd><kbd>↵</kbd></span>
            </div>
            <div class="img-examples">
              <span>{{ copy.examplesLabel }}</span>
              <div>
                <button v-for="ex in copy.examples" :key="ex.label" type="button" class="img-chip" @click="useExample(ex.prompt)">{{ ex.label }}</button>
              </div>
            </div>
          </div>

          <div class="img-field">
            <span class="img-label">{{ copy.ratioLabel }}</span>
            <div class="img-ratios" role="radiogroup" :aria-label="copy.ratioLabel">
              <button
                v-for="r in ratios"
                :key="r"
                type="button"
                role="radio"
                :aria-label="`${copy.ratios[r]} ${copy.ratioHint[r]}`"
                class="img-ratio"
                :class="{ 'is-on': ratio === r }"
                :aria-checked="ratio === r"
                @click="ratio = r"
              >
                <i :class="`img-shape img-shape--${r}`" aria-hidden="true" />
                <span>{{ copy.ratios[r] }}</span>
                <small>{{ copy.ratioHint[r] }}</small>
              </button>
            </div>
          </div>

          <div class="img-field">
            <span class="img-label">{{ copy.referenceLabel }}<em>{{ copy.referenceOptional }}</em></span>
            <div v-if="referenceUrl" class="img-ref">
              <img :src="referenceUrl" alt="" />
              <button type="button" class="img-ref-remove" :aria-label="copy.referenceRemove" :title="copy.referenceRemove" @click="setReference(null)">
                <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18" /></svg>
              </button>
            </div>
            <button
              v-else
              type="button"
              class="img-drop"
              :class="{ 'is-drag': dragging }"
              @click="fileInput?.click()"
              @dragover.prevent="dragging = true"
              @dragleave="dragging = false"
              @drop.prevent="onDrop"
            >
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 16l4.5-4.5a2 2 0 0 1 2.8 0L16 16m-2-2 1.5-1.5a2 2 0 0 1 2.8 0L20 14M14 8h.01M6 20h12a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2z" /></svg>
              {{ copy.referenceAdd }}
            </button>
            <input ref="fileInput" type="file" accept="image/*" class="sr-only" @change="onFileChange" />
            <p v-if="referenceError" class="img-error-text">{{ referenceError }}</p>
          </div>

          <details class="img-more">
            <summary>{{ copy.more }}</summary>
            <div class="img-more-body">
              <label class="img-select">
                <span>{{ copy.keyLabel }}</span>
                <select v-model="keyId" :disabled="!keys.length">
                  <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }}</option>
                </select>
              </label>
              <label class="img-select">
                <span>{{ copy.modelLabel }}</span>
                <select v-model="model">
                  <option v-for="m in modelOptions" :key="m" :value="m">{{ m }}</option>
                </select>
              </label>
              <label class="img-select">
                <span>{{ copy.qualityLabel }}</span>
                <select v-model="quality">
                  <option v-for="q in (['auto', 'low', 'high'] as const)" :key="q" :value="q">{{ copy.qualities[q] }}</option>
                </select>
              </label>
            </div>
          </details>

          <p v-if="noImageModel" class="img-warn">{{ copy.noModel }}</p>

          <router-link v-if="keysLoaded && !hasKeys" to="/keys" class="img-go">{{ copy.noKeyAction }}</router-link>
          <button v-else-if="status === 'busy'" type="button" class="img-go is-busy" :title="copy.stopHint" @click="stop">
            <span class="img-go-spin" aria-hidden="true" />{{ copy.stop }}
          </button>
          <button v-else type="submit" class="img-go" :disabled="!canGenerate">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3.5l1.9 5.1 5.1 1.9-5.1 1.9L12 17.5l-1.9-5.1-5.1-1.9 5.1-1.9z" /><path d="M18.5 15.5l.7 1.8 1.8.7-1.8.7-.7 1.8-.7-1.8-1.8-.7 1.8-.7z" /></svg>
            {{ copy.generate }}
          </button>

          <button v-if="hasKeys" type="button" class="img-agent" @click="skillOpen = true">
            <span>{{ copy.agentTitle }}</span>
            <span class="img-agent-action">{{ copy.agentAction }}<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 12h14m-6-6 6 6-6 6" /></svg></span>
          </button>
        </form>

        <!-- 右：画布 -->
        <section ref="stageEl" class="img-stage" :aria-busy="status === 'busy'">
          <div v-if="keysLoaded && !hasKeys" class="img-nokey">
            <div class="img-stage-glow"><GlowDots :center-x="0.5" :center-y="0.5" :gain="0.45" :max-cells="9000" :interactive="false" /></div>
            <div class="img-nokey-body">
              <h2>{{ copy.noKeyTitle }}</h2>
              <p>{{ copy.noKeyBody }}</p>
              <router-link to="/keys" class="img-go img-go--inline">{{ copy.noKeyAction }}</router-link>
            </div>
          </div>

          <template v-else>
            <div class="img-canvas" :class="[`is-${status}`, `ratio-${stageRatio}`]">
              <div class="img-stage-glow" :class="{ 'is-live': status === 'busy' }">
                <GlowDots :center-x="0.5" :center-y="0.5" :gain="0.7" :max-cells="12000" />
              </div>

              <Transition name="img-swap" mode="out-in">
                <div v-if="status === 'busy'" key="busy" class="img-frame-wrap">
                  <div class="img-frame is-busy" :class="`ratio-${busyRatio}`">
                    <span class="img-frame-ring" aria-hidden="true" />
                    <div class="img-busy" role="status">
                      <span class="img-busy-dot" aria-hidden="true" />
                      <p>{{ copy.generating }}</p>
                      <small class="tabular-nums">{{ copy.elapsed(elapsed) }}</small>
                    </div>
                  </div>
                </div>

                <div v-else-if="status === 'error'" key="error" class="img-empty" role="alert">
                  <span class="img-error-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M12 8v5M12 16.5h.01" /><circle cx="12" cy="12" r="8.5" /></svg></span>
                  <p class="img-error-title">{{ errorText }}</p>
                  <div class="img-error-actions">
                    <button type="button" class="img-ghost img-ghost--strong" :disabled="!canGenerate" @click="generate">{{ copy.retry }}</button>
                    <router-link v-if="errorCode === 'billing' || errorCode === 'invalid_key'" to="/keys" class="img-ghost inline-flex items-center">{{ copy.toKeys }}</router-link>
                  </div>
                </div>

                <div v-else-if="current" :key="current.id" class="img-results" :class="`n-${current.images.length}`">
                  <figure v-for="(src, i) in current.images" :key="i" class="img-result" :class="[`ratio-${current.ratio}`, { 'is-fresh': current.id === freshId }]" :style="{ '--i': i }">
                    <img :src="src" :alt="current.prompt" @click="lightbox = src" />
                    <figcaption class="img-result-actions">
                      <button type="button" :title="copy.download" :aria-label="copy.download" @click="download(src, i)">
                        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 4v11m0 0 4.5-4.5M12 15l-4.5-4.5M5 19h14" /></svg>
                      </button>
                      <button type="button" :title="copy.enlarge" :aria-label="copy.enlarge" @click="lightbox = src">
                        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M14 4h6v6M10 20H4v-6M20 4l-7 7M4 20l7-7" /></svg>
                      </button>
                      <button type="button" :title="copy.useAsReference" :aria-label="copy.useAsReference" @click="useAsReference(src)">
                        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 20h4L19 9a2.8 2.8 0 0 0-4-4L4 16z" /><path d="M13.5 6.5l4 4" /></svg>
                      </button>
                    </figcaption>
                  </figure>
                </div>

                <div v-else key="empty" class="img-empty">
                  <div class="img-frame is-idle" :class="`ratio-${ratio}`">
                    <span class="img-frame-corners" aria-hidden="true"><i /><i /><i /><i /></span>
                    <div>
                      <p>{{ copy.stageEmpty }}</p>
                      <small>{{ copy.stageEmptyHint }}</small>
                    </div>
                  </div>
                </div>
              </Transition>
            </div>

            <div v-if="current && status === 'done'" class="img-meta">
              <p class="img-meta-prompt">{{ current.prompt }}</p>
              <span class="img-meta-model">{{ current.model }}</span>
              <button type="button" class="img-ghost" @click="again">{{ copy.again }}</button>
            </div>
          </template>
        </section>
      </div>

      <!-- 最近生成 -->
      <section v-if="history.length" class="img-history">
        <header>
          <h2>{{ copy.historyLabel }}</h2>
          <span>{{ copy.historyNote }}</span>
          <button type="button" class="img-link" @click="clearHistory">{{ copy.historyClear }}</button>
        </header>
        <ol>
          <li v-for="record in history" :key="record.id" :class="{ 'is-current': current?.id === record.id }">
            <button type="button" class="img-thumb" :title="record.prompt" @click="openRecord(record)">
              <img :src="record.images[0]" :alt="record.prompt" loading="lazy" />
              <span v-if="record.images.length > 1" class="img-thumb-count tabular-nums">{{ record.images.length }}</span>
            </button>
            <button type="button" class="img-thumb-del" :aria-label="copy.historyDelete" :title="copy.historyDelete" @click="removeRecord(record)">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18" /></svg>
            </button>
          </li>
        </ol>
      </section>
    </div>

    <Teleport to="body">
      <Transition name="img-fade">
        <div v-if="lightbox" class="img-lightbox" role="dialog" aria-modal="true" @click="lightbox = null">
          <img :src="lightbox" alt="" @click.stop />
          <button type="button" class="img-lightbox-close" :aria-label="copy.close" @click="lightbox = null">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18" /></svg>
          </button>
        </div>
      </Transition>
    </Teleport>

    <SkillPromptDialog :show="skillOpen" :preferred-key-id="keyId" @close="skillOpen = false" />
  </AppLayout>
</template>
