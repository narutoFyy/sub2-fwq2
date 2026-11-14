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
  /** 欢迎页没有「跳过」页脚（欢迎页上已有显式按钮）。 */
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
const stageLabel = computed(() => (props.stage >= 0 ? `${props.stage + 1} / ${copy.value.stages.length} · ${copy.value.stages[props.stage]}` : ''))
</script>

<template>
  <div class="og-body">
    <div v-if="stage >= 0" class="og-stages" role="progressbar" aria-valuemin="1" :aria-valuemax="copy.stages.length" :aria-valuenow="stage + 1" :aria-label="stageLabel">
      <span v-for="(_, i) in copy.stages" :key="i" class="og-stage" :class="{ 'is-done': i < stage, 'is-active': i === stage }"><i></i></span>
      <span class="og-stage-label">{{ stageLabel }}</span>
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
            <ModelIcon :model="opt.id === 'codex' ? 'gpt-5' : 'claude'" :class="{ 'og-codex-icon': opt.id === 'codex' }" size="30px" />
            <span class="og-option-text"><strong>{{ opt.name }}</strong><small>{{ opt.desc }}</small></span>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><path d="M5 12h14m-6-6 6 6-6 6" /></svg>
          </button>
        </div>

        <GuideBlocks v-if="view.blocks" :blocks="view.blocks" />

        <p v-if="view.hint" class="og-hint">{{ view.hint }}</p>

        <div v-if="view.primary || view.secondary || view.fallback || view.back" class="og-actions">
          <button v-if="view.back" type="button" class="og-btn og-btn--text" @click="view.back.run()">{{ view.back.label }}</button>
          <button v-if="view.collapsible" type="button" class="og-btn og-btn--ghost" @click="emit('collapse')">{{ copy.download.collapse }}</button>
          <button v-if="view.secondary" type="button" class="og-btn og-btn--ghost" @click="view.secondary.run()">{{ view.secondary.label }}</button>
          <button v-if="view.fallback" type="button" class="og-btn og-btn--ghost" @click="view.fallback.run()">{{ view.fallback.label }}</button>
          <button v-if="view.primary" type="button" class="og-btn og-btn--primary" data-og-autofocus @click="view.primary.run()">{{ view.primary.label }}</button>
        </div>
      </div>
    </Transition>

    <footer class="og-foot">
      <button v-if="showSkip && !confirming && !completing" type="button" class="og-link og-link--quiet" @click="emit('askSkip')">{{ copy.skip }}</button>
      <span v-else></span>
      <GuideSupport />
    </footer>
  </div>
</template>
