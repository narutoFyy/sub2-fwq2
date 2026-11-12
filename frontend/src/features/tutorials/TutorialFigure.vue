<script setup lang="ts">
import { computed } from 'vue'
import { clientName, useTutorialCopy, type TutorialClient } from './content'
const props = defineProps<{ step: number; client: TutorialClient; siteName: string; endpoint: string }>()
const copy = useTutorialCopy()
const name = computed(() => clientName(props.client))
</script>

<template>
  <figure class="guide-figure" :aria-label="`${copy.figure} ${step}: ${copy.steps[step - 1]}`">
    <div class="window-bar" aria-hidden="true"><span class="window-dots"><i></i><i></i><i></i></span><span>{{ step === 3 || step === 5 ? 'CC Switch' : siteName }}</span><span>{{ copy.figure }}</span></div>
    <div class="figure-body" aria-hidden="true">
      <template v-if="step === 1">
        <div class="mini-heading">{{ copy.ui.recharge }}<span class="mini-label">{{ siteName }}</span></div>
        <p class="mini-label">{{ copy.ui.amount }}</p>
        <div class="amounts"><span>10</span><span class="selected">20</span><span>50</span><span>100</span></div>
        <div class="figure-line"><span>{{ copy.ui.pay }}</span><b class="callout">1</b></div>
        <div class="done-line"><svg viewBox="0 0 24 24"><path d="m5 12 4 4L19 6" /></svg>{{ copy.ui.paid }}<b class="callout">2</b></div>
      </template>
      <template v-else-if="step === 2">
        <div class="mini-heading">{{ copy.ui.create }}<b class="callout">1</b></div>
        <div class="form-label">{{ copy.ui.keyName }}</div><div class="fake-input">{{ name }} · {{ siteName }}</div>
        <div class="form-label">{{ copy.ui.provider }}</div>
        <div class="provider-tabs"><span :class="{ selected: client === 'claude-code' }">Claude</span><span :class="{ selected: client === 'codex' }">OpenAI</span><b class="callout">2</b></div>
        <div class="fake-input group-input">{{ copy.ui.group }}<span>⌄</span><b class="callout">3</b></div>
        <div class="figure-line">{{ copy.ui.create }}</div>
      </template>
      <template v-else-if="step === 3">
        <div class="install-flow"><div class="app-mark">CC<span>Switch</span></div><span class="flow-plus">+</span><div class="cli-mark"><span>&gt;_</span>{{ name }}</div></div>
        <div class="terminal-mini"><span>$ {{ client === 'codex' ? 'codex' : 'claude' }} --version</span><span class="terminal-comment">{{ copy.ui.install }}</span></div>
      </template>
      <template v-else-if="step === 4">
        <div class="mini-heading">{{ copy.ui.keys }}</div>
        <div class="key-row"><span>{{ name }}<small>sk-••••••••••••</small></span><span class="import-button">{{ copy.ui.import }}</span><b class="callout">1</b></div>
        <div class="flow-down">↓</div>
        <div class="import-dialog"><div>{{ copy.ui.allow }}<b class="callout">2</b></div><p>{{ name }} / {{ siteName }}</p><div class="endpoint-preview">{{ endpoint }}</div><span class="confirm-button">{{ copy.ui.confirm }} <b class="callout">3</b></span></div>
      </template>
      <template v-else-if="step === 5">
        <div class="mini-heading">{{ name }}<span class="mini-label">CC Switch</span></div>
        <div class="active-provider"><div class="provider-monogram">{{ siteName.slice(0, 1) }}</div><span>{{ siteName }}<small>{{ copy.ui.enabled }}</small></span><span class="enable-button">{{ copy.ui.enable }}</span><b class="callout">1</b></div>
        <div class="flow-down">↓</div>
        <div class="terminal-mini"><span>$ {{ client === 'codex' ? 'codex' : 'claude' }}</span><span class="terminal-comment">{{ name }}</span></div>
      </template>
      <template v-else>
        <div class="terminal-mini verify-terminal"><span>&gt; {{ copy.prompt }}</span><span class="terminal-answer">{{ copy.ui.reply }}</span></div>
        <div class="flow-down">↓</div>
        <div class="mini-heading">{{ copy.ui.record }}</div>
        <div class="record-row"><span>{{ name }}</span><code>200</code><span class="success-pill">{{ copy.ui.success }}</span><b class="callout">✓</b></div>
      </template>
    </div>
    <figcaption>{{ String(step).padStart(2, '0') }} / {{ copy.steps[step - 1] }} <span>{{ copy.figure }}</span></figcaption>
  </figure>
</template>

<style scoped>
.guide-figure { margin: 28px 0; overflow: hidden; border-radius: 16px; background: var(--guide-panel); border: 1px solid var(--guide-line); font-size: 13px; }
.window-bar { padding: 13px 18px; display: flex; gap: 20px; align-items: center; color: var(--guide-muted); border-bottom: 1px solid var(--guide-line); font-size: 11px; }
.window-bar > span:last-child { margin-left: auto; }
.window-dots { display: flex; gap: 5px; }
.window-dots i { width: 6px; height: 6px; border-radius: 50%; background: var(--guide-line); }
.figure-body { max-width: 540px; padding: 32px 28px; margin: auto; }
.mini-heading { display: flex; align-items: center; justify-content: space-between; gap: 10px; font-size: 17px; font-weight: 500; margin-bottom: 24px; }
.mini-label, .form-label { font-size: 11px; font-weight: 400; color: var(--guide-muted); }
.form-label { margin: 16px 0 7px; }
.amounts { display: flex; gap: 10px; margin: 10px 0 24px; }
.amounts > span { flex: 1; text-align: center; border-radius: 6px; padding: 12px 6px; background: var(--guide-bg); font-family: 'Geist Mono Variable', monospace; }
.selected { box-shadow: inset 0 0 0 1px var(--guide-accent); color: var(--guide-ink); }
.figure-line { background: var(--guide-ink); color: var(--guide-bg); padding: 12px; border-radius: 7px; display: flex; justify-content: center; align-items: center; gap: 14px; margin-top: 20px; }
.callout { display: inline-grid; place-items: center; flex-shrink: 0; width: 21px; height: 21px; border-radius: 50%; background: var(--guide-accent); color: #082b30; font: 600 11px 'Geist Variable', sans-serif; }
.done-line { display: flex; align-items: center; gap: 8px; margin-top: 20px; }
.done-line svg { width: 16px; height: 16px; stroke: var(--guide-accent); fill: none; stroke-width: 2; }
.done-line b { margin-left: auto; }
.fake-input { border: 1px solid var(--guide-line); padding: 12px; border-radius: 6px; }
.group-input { display: flex; gap: 16px; margin-top: 16px; }
.group-input > span { margin-left: auto; }
.provider-tabs { display: flex; align-items: center; gap: 12px; }
.provider-tabs > span { flex: 1; padding: 12px; text-align: center; border-radius: 6px; background: var(--guide-bg); }
.install-flow { display: flex; justify-content: center; align-items: center; gap: 32px; padding: 24px 0 40px; }
.app-mark { width: 84px; height: 84px; background: var(--guide-ink); color: var(--guide-bg); border-radius: 20px; display: flex; flex-direction: column; align-items: center; justify-content: center; font-size: 28px; font-weight: 500; line-height: 1.1; }
.app-mark span { font-size: 12px; margin-top: 6px; }
.flow-plus, .flow-down { color: var(--guide-muted); font-size: 22px; text-align: center; }
.flow-down { margin: 18px 0; }
.cli-mark { display: flex; flex-direction: column; align-items: center; gap: 10px; }
.cli-mark > span { font-family: monospace; font-size: 38px; }
.terminal-mini { background: #111115; color: #eeeef1; border-radius: 8px; padding: 18px; display: grid; gap: 12px; font-family: 'Geist Mono Variable', monospace; overflow-wrap: anywhere; }
.terminal-comment { color: #a2a5b0; font-size: 11px; }
.key-row { display: flex; align-items: center; gap: 12px; border-block: 1px solid var(--guide-line); padding: 15px 0; }
small { display: block; color: var(--guide-muted); margin-top: 6px; font-size: 11px; }
.import-button { margin-left: auto; font-size: 11px; color: var(--guide-accent-ink); }
.import-dialog { border: 1px solid var(--guide-line); border-radius: 10px; padding: 20px; }
.import-dialog > div:first-child { display: flex; justify-content: space-between; gap: 12px; }
.import-dialog p { margin-top: 15px; font-weight: 500; }
.endpoint-preview { color: var(--guide-muted); overflow-wrap: anywhere; font-size: 11px; margin: 6px 0 20px; }
.confirm-button, .enable-button { display: inline-flex; align-items: center; gap: 10px; border-radius: 6px; padding: 8px 12px; background: var(--guide-ink); color: var(--guide-bg); font-size: 11px; }
.active-provider { display: flex; align-items: center; gap: 12px; padding: 18px 14px; border: 1px solid var(--guide-accent); border-radius: 10px; }
.active-provider .enable-button { margin-left: auto; }
.provider-monogram { width: 34px; height: 34px; display: grid; place-items: center; border-radius: 8px; background: var(--guide-bg); }
.terminal-answer { color: #86e3c5; padding-top: 8px; }
.record-row { display: flex; align-items: center; gap: 16px; padding: 15px 0; border-top: 1px solid var(--guide-line); }
.record-row > span:first-child { flex: 1; }
.success-pill { color: var(--guide-accent-ink); font-size: 11px; }
figcaption { border-top: 1px solid var(--guide-line); padding: 12px 18px; color: var(--guide-muted); display: flex; justify-content: space-between; font-size: 11px; }
@media (max-width: 480px) { .figure-body { padding: 24px 16px; } .active-provider { gap: 8px; padding: 14px 10px; } .provider-monogram { display: none; } .install-flow { gap: 24px; } }
</style>
