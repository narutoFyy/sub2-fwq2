<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { copyText } from './copyText'
import { useGuideCopy } from './copy'
import type { GuideBlock } from './viewModel'

defineProps<{ blocks: GuideBlock[] }>()

const copy = useGuideCopy()
const copiedKey = ref<string | null>(null)
let timer: ReturnType<typeof setTimeout> | undefined

async function copyValue(id: string, value: string) {
  if (!(await copyText(value))) return
  copiedKey.value = id
  clearTimeout(timer)
  timer = setTimeout(() => (copiedKey.value = null), 1600)
}

/** 密钥只显示前后几位，复制的是完整值。 */
function mask(value: string): string {
  return value.length <= 12 ? '••••••••' : `${value.slice(0, 5)}••••••••${value.slice(-4)}`
}

/** 带标题的块依次编号（①②③），纯提示块不占编号。 */
function numberOf(blocks: GuideBlock[], index: number): number {
  return blocks.slice(0, index + 1).filter((b) => b.type === 'download' || b.type === 'command' || b.type === 'prompt' || (b.type === 'copy' && b.heading)).length
}

onBeforeUnmount(() => clearTimeout(timer))
</script>

<template>
  <ol class="og-blocks">
    <li v-for="(block, i) in blocks" :key="i" class="og-block" :style="{ '--i': i }">
      <template v-if="block.type === 'note'">
        <p class="og-block-note">{{ block.text }}</p>
      </template>

      <template v-else-if="block.type === 'models'">
        <p class="og-models-head">{{ block.heading }}</p>
        <ul class="og-models">
          <li v-for="item in block.items" :key="item.name">
            <strong>{{ item.name }}<em v-if="item.tag">{{ item.tag }}</em></strong>
            <span>{{ item.pitch }}</span>
          </li>
        </ul>
        <p v-if="block.hint" class="og-block-hint">{{ block.hint }}</p>
      </template>

      <template v-else-if="block.type === 'status'">
        <p class="og-status" role="status"><span class="og-pill-dot" aria-hidden="true"></span>{{ block.text }}</p>
      </template>

      <template v-else-if="block.type === 'download'">
        <h3 class="og-block-head"><span class="og-num">{{ numberOf(blocks, i) }}</span>{{ block.heading }}</h3>
        <a v-if="block.primary" class="og-dl" :href="block.primary.href" target="_blank" rel="noopener noreferrer" download>
          <span class="og-dl-label">{{ copy.download.get }} · {{ block.primary.label }}</span>
          <span v-if="block.primary.meta" class="og-dl-meta">{{ block.primary.meta }}</span>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><path d="M12 4v11m0 0 4.5-4.5M12 15l-4.5-4.5M5 19h14" /></svg>
        </a>
        <p v-if="block.note" class="og-block-hint">{{ block.note }}</p>
        <div v-if="block.alternates?.length" class="og-alt">
          <span>{{ copy.download.other }}</span>
          <a v-for="alt in block.alternates" :key="alt.href" :href="alt.href" target="_blank" rel="noopener noreferrer" download>{{ alt.label }}<small v-if="alt.meta"> · {{ alt.meta }}</small></a>
        </div>
        <div v-if="block.backup || block.verify" class="og-alt og-alt--quiet">
          <a v-if="block.backup" :href="block.backup.href" target="_blank" rel="noopener noreferrer">{{ copy.download.backupLabel }}：{{ block.backup.label }}</a>
          <a v-if="block.verify" :href="block.verify.href" target="_blank" rel="noopener noreferrer">{{ block.verify.label }}</a>
        </div>
      </template>

      <template v-else-if="block.type === 'prompt'">
        <h3 class="og-block-head"><span class="og-num">{{ numberOf(blocks, i) }}</span>{{ block.heading }}</h3>
        <div class="og-prompt">
          <pre>{{ block.shown }}</pre>
          <button type="button" class="og-prompt-copy" :class="{ 'is-done': copiedKey === `p${i}` }" data-og-autofocus @click="copyValue(`p${i}`, block.value)">{{ copiedKey === `p${i}` ? copy.download.copied : copy.download.copy }}</button>
        </div>
        <p v-if="block.note" class="og-block-hint">{{ block.note }}</p>
      </template>

      <template v-else-if="block.type === 'command'">
        <h3 class="og-block-head"><span class="og-num">{{ numberOf(blocks, i) }}</span>{{ block.heading }}</h3>
        <div class="og-code">
          <code>{{ block.command }}</code>
          <button type="button" @click="copyValue(`c${i}`, block.command)">{{ copiedKey === `c${i}` ? copy.download.copied : copy.download.copy }}</button>
        </div>
        <p v-if="block.hint" class="og-block-hint">{{ block.hint }}</p>
      </template>

      <template v-else>
        <h3 v-if="block.heading" class="og-block-head"><span class="og-num">{{ numberOf(blocks, i) }}</span>{{ block.heading }}</h3>
        <div class="og-code og-code--field">
          <span class="og-field-label">{{ block.label }}</span>
          <code>{{ block.secret ? mask(block.value) : block.value }}</code>
          <button type="button" @click="copyValue(`f${i}`, block.value)">{{ copiedKey === `f${i}` ? copy.download.copied : copy.download.copy }}</button>
        </div>
      </template>
    </li>
  </ol>
</template>
