<template>
  <article class="card flex min-w-0 flex-col p-5 sm:p-6" :data-verdict="verdict">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0">
        <h3 class="truncate text-base font-semibold text-gray-900 dark:text-white">{{ group.group_name }}</h3>
        <p class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400">{{ group.model_id }} · {{ group.reasoning_effort }}</p>
        <slot name="badges" />
      </div>
      <span class="inline-flex shrink-0 items-center gap-2 rounded-full px-3 py-1.5 text-sm font-semibold" :class="tone.pill">
        <span class="h-2 w-2 rounded-full" :class="tone.dot"></span>{{ t(verdictLabelKey[verdict]) }}
      </span>
    </div>

    <slot name="config" />

    <!-- 最近一次问答 -->
    <div class="mt-5 space-y-2">
      <div class="flex justify-end">
        <p class="rounded-2xl rounded-br-md bg-gray-950 px-3.5 py-1.5 font-mono text-sm text-gray-50 dark:bg-gray-200 dark:text-gray-950">{{ prompt }}</p>
      </div>
      <div class="flex justify-start">
        <p class="max-w-full whitespace-pre-wrap break-words rounded-2xl rounded-bl-md px-3.5 py-2 text-base leading-7" :class="tone.bubble">{{ replyText }}</p>
      </div>
      <p class="text-xs" :class="tone.text">
        {{ t(verdictReasonKey[verdict], { n: confirmRounds }) }}
        <span v-if="latest" class="text-gray-500 dark:text-gray-400">· {{ t('modelRadar.seconds', { s: (latest.latency_ms / 1000).toFixed(1) }) }} · {{ formatRelativeTime(latest.detected_at) }}</span>
      </p>
    </div>

    <!-- 历史 -->
    <div class="mt-5">
      <div class="mb-2 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
        <span>{{ t('modelRadar.history', { n: slots.length }) }}</span>
        <span v-if="judged">{{ t('modelRadar.passRate', { passed, total: judged }) }}</span>
      </div>
      <div class="flex h-6 gap-[3px]" role="img" :aria-label="judged ? t('modelRadar.passRate', { passed, total: judged }) : t('modelRadar.noData')">
        <span v-for="slot in slots" :key="slot.key" class="min-w-0 flex-1 rounded-[3px]" :class="slot.tone" :title="slot.title"></span>
      </div>
    </div>

    <!-- 画图样张默认展开 -->
    <section v-if="group.drawing?.response_text" class="mt-5 border-t border-gray-900/[0.06] pt-4 dark:border-white/[0.09]">
      <h4 class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('modelRadar.drawing') }}</h4>
      <p class="mb-3 mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('modelRadar.drawingHint') }}</p>
      <RadarDrawingPreview :response-text="group.drawing.response_text" :title="t('modelRadar.drawing')" />
    </section>

    <slot name="actions" />
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelRadarGroup } from '@/api/modelRadar'
import { formatRelativeTime } from '@/utils/format'
import RadarDrawingPreview from './RadarDrawingPreview.vue'
import { groupVerdict, verdictLabelKey, verdictOf, verdictReasonKey, type RadarVerdict } from './radar'

const SLOT_COUNT = 24

const props = withDefaults(defineProps<{ group: ModelRadarGroup; prompt: string; confirmRounds?: number }>(), { confirmRounds: 2 })
const { t } = useI18n()

const latest = computed(() => props.group.ciallo)
const verdict = computed(() => groupVerdict(props.group))

const tones: Record<RadarVerdict, { pill: string; dot: string; bubble: string; text: string; bar: string }> = {
  ok: {
    pill: 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
    dot: 'bg-emerald-500',
    bubble: 'bg-emerald-500/10 text-gray-900 dark:text-gray-50',
    text: 'text-emerald-700 dark:text-emerald-400',
    bar: 'bg-emerald-500'
  },
  nerfed: {
    pill: 'bg-red-500/10 text-red-700 dark:text-red-300',
    dot: 'bg-red-500',
    bubble: 'bg-red-500/10 text-gray-900 dark:text-gray-50',
    text: 'text-red-700 dark:text-red-400',
    bar: 'bg-red-500'
  },
  watch: {
    pill: 'bg-amber-500/10 text-amber-700 dark:text-amber-300',
    dot: 'bg-amber-500',
    bubble: 'bg-amber-500/10 text-gray-900 dark:text-gray-50',
    text: 'text-amber-700 dark:text-amber-400',
    bar: 'bg-amber-400'
  },
  error: {
    pill: 'bg-amber-500/10 text-amber-700 dark:text-amber-300',
    dot: 'bg-amber-500',
    bubble: 'bg-gray-900/[0.04] text-gray-500 dark:bg-white/[0.06] dark:text-gray-400',
    text: 'text-amber-700 dark:text-amber-400',
    bar: 'bg-amber-400'
  },
  none: {
    pill: 'bg-gray-900/[0.05] text-gray-600 dark:bg-white/[0.07] dark:text-gray-300',
    dot: 'bg-gray-400',
    bubble: 'bg-gray-900/[0.04] text-gray-500 dark:bg-white/[0.06] dark:text-gray-400',
    text: 'text-gray-500 dark:text-gray-400',
    bar: 'bg-gray-900/[0.06] dark:bg-white/[0.08]'
  }
}
const tone = computed(() => tones[verdict.value])

const replyText = computed(() => {
  if (!latest.value) return '…'
  return latest.value.response_text?.trim() || latest.value.error_message || t('modelRadar.emptyReply')
})

// 后端按新到旧给，页面上旧的在左、最新的在右，不足的用空格子补在左边。
const slots = computed(() => {
  const items = props.group.timeline.slice(0, SLOT_COUNT).reverse()
  const padding = Array.from({ length: SLOT_COUNT - items.length }, (_, index) => ({
    key: `empty-${index}`,
    tone: tones.none.bar,
    title: t('modelRadar.noData')
  }))
  return padding.concat(items.map(item => {
    const itemVerdict = verdictOf(item)
    const head = t('modelRadar.slotTitle', { verdict: t(verdictLabelKey[itemVerdict]), time: new Date(item.detected_at).toLocaleString() })
    return { key: String(item.id), tone: tones[itemVerdict].bar, title: item.response_text ? `${head}\n${item.response_text}` : head }
  }))
})
const judged = computed(() => props.group.timeline.filter(item => verdictOf(item) === 'ok' || verdictOf(item) === 'nerfed').length)
const passed = computed(() => props.group.timeline.filter(item => verdictOf(item) === 'ok').length)
</script>
