<script setup lang="ts">
import { computed } from 'vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import GuideBlocks from './GuideBlocks.vue'
import GuideSupport from './GuideSupport.vue'
import { useGuideCopy } from './copy'
import type { GuideClient } from './config'
import type { GuideView } from './viewModel'

const props = defineProps<{
  view: GuideView
  /** 当前阶段（0–3）；欢迎页为 -1，只展示路线预览。 */
  stage: number
  completing: string | null
  confirming: boolean
  /** 右上角的「跳过引导」；欢迎页（已有显式按钮）和成功页不显示。 */
  showSkip: boolean
}>()

const emit = defineEmits<{
  choose: [client: GuideClient]
  askSkip: []
  confirmSkip: []
  cancelSkip: []
  collapse: []
}>()

const copy = useGuideCopy()
const doneText = computed(() => (props.completing === 'redeem' ? copy.value.done.redeem : copy.value.done.key))
const isWelcome = computed(() => props.view.key === 'welcome')
/** 「我已有密钥，跳过这步」这类次要出口放在底部一行（和客服入口同一行，右对齐）。 */
const quietAction = computed(() => (props.view.secondary?.quiet ? props.view.secondary : null))
const hasActions = computed(
  () => !!(props.view.primary || (props.view.secondary && !props.view.secondary.quiet) || props.view.fallback || props.view.back)
)
const stageLabel = computed(() => {
  if (props.stage < 0) return ''
  if (props.stage >= copy.value.stages.length) return copy.value.stageDone
  return `${props.stage + 1} / ${copy.value.stages.length} · ${copy.value.stages[props.stage]}`
})
</script>

<template>
  <div class="og-body">
    <div v-if="stage >= 0" class="og-top">
      <div class="og-stages" role="progressbar" aria-valuemin="1" :aria-valuemax="copy.stages.length" :aria-valuenow="stage + 1" :aria-label="stageLabel">
        <span v-for="(_, i) in copy.stages" :key="i" class="og-stage" :class="{ 'is-done': i < stage, 'is-active': i === stage }"><i></i></span>
        <span class="og-stage-label">{{ stageLabel }}</span>
      </div>
      <button v-if="showSkip && !confirming && !completing" type="button" class="og-link og-link--skip" @click="emit('askSkip')">{{ copy.skip }}</button>
    </div>

    <Transition name="og-swap" mode="out-in">
      <div v-if="completing" key="completing" class="og-done" role="status">
        <svg class="og-check" width="56" height="56" viewBox="0 0 56 56" fill="none" aria-hidden="true">
          <circle class="og-check-ring" cx="28" cy="28" r="24" />
          <path class="og-check-mark" d="M17 29.5l7.5 7.5L39 21" />
        </svg>
        <p>{{ doneText }}</p>
      </div>

      <div v-else-if="confirming" key="confirm" class="og-content">
        <h2 id="og-title" class="og-title">{{ copy.skipConfirmTitle }}</h2>
        <p class="og-text">{{ copy.skipConfirmBody }}</p>
        <div class="og-actions">
          <button type="button" class="og-btn og-btn--ghost" @click="emit('confirmSkip')">{{ copy.skipConfirm }}</button>
          <button type="button" class="og-btn og-btn--primary" data-og-autofocus @click="emit('cancelSkip')">{{ copy.skipCancel }}</button>
        </div>
      </div>

      <div v-else :key="view.key" class="og-content">
        <h2 id="og-title" class="og-title">{{ view.title }}</h2>
        <p v-if="view.body" class="og-text">{{ view.body }}</p>

        <ol v-if="isWelcome" class="og-path">
          <li v-for="(label, i) in copy.stages" :key="label" :style="{ '--i': i }"><span>{{ i + 1 }}</span>{{ label }}</li>
        </ol>

        <ol v-if="view.list" class="og-list">
          <li v-for="(item, i) in view.list" :key="item" :style="{ '--i': i }"><span>{{ i + 1 }}</span>{{ item }}</li>
        </ol>

        <div v-if="view.options" class="og-options">
          <button v-for="(opt, i) in view.options" :key="opt.id" type="button" class="og-option" :style="{ '--i': i }" :data-og-autofocus="i === 0 ? '' : undefined" @click="emit('choose', opt.id)">
            <span v-if="opt.id === 'chat'" class="og-option-glyph" aria-hidden="true">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 5.5A2.5 2.5 0 0 1 6.5 3h11A2.5 2.5 0 0 1 20 5.5v8a2.5 2.5 0 0 1-2.5 2.5H10l-4.5 4v-4h0A1.5 1.5 0 0 1 4 14.5z" /><path d="M8.5 8.5h7M8.5 11.5h4" /></svg>
            </span>
            <span v-else-if="opt.id === 'fortune'" class="og-option-glyph og-option-glyph--fortune" aria-hidden="true">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M20 14.5A8.5 8.5 0 1 1 9.5 4a6.8 6.8 0 0 0 10.5 10.5z" /><path d="M16.5 3.5l.6 1.4 1.4.6-1.4.6-.6 1.4-.6-1.4-1.4-.6 1.4-.6z" /></svg>
            </span>
            <ModelIcon v-else :model="opt.id === 'codex' ? 'gpt-5' : 'claude'" :class="{ 'og-codex-icon': opt.id === 'codex' }" size="30px" />
            <span class="og-option-text"><strong>{{ opt.name }}</strong><small>{{ opt.desc }}</small></span>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><path d="M5 12h14m-6-6 6 6-6 6" /></svg>
          </button>
        </div>

        <GuideBlocks v-if="view.blocks" :blocks="view.blocks" />

        <p v-if="view.hint" class="og-hint">{{ view.hint }}</p>

        <div v-if="hasActions" class="og-actions">
          <button v-if="view.back" type="button" class="og-btn og-btn--text" @click="view.back.run()">{{ view.back.label }}</button>
          <button v-if="view.collapsible" type="button" class="og-btn og-btn--ghost" @click="emit('collapse')">{{ copy.download.collapse }}</button>
          <button v-if="view.secondary && !view.secondary.quiet" type="button" class="og-btn og-btn--ghost" @click="view.secondary.run()">{{ view.secondary.label }}</button>
          <button v-if="view.fallback" type="button" class="og-btn og-btn--ghost" @click="view.fallback.run()">{{ view.fallback.label }}</button>
          <button v-if="view.primary" type="button" class="og-btn og-btn--primary" data-og-autofocus @click="view.primary.run()">{{ view.primary.label }}</button>
        </div>
      </div>
    </Transition>

    <footer class="og-foot">
      <GuideSupport />
      <button v-if="quietAction && !confirming && !completing" type="button" class="og-link" @click="quietAction.run()">{{ quietAction.label }}</button>
    </footer>
  </div>
</template>
