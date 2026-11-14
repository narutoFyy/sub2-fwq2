<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import shots from './shots.json'

interface Mark { n?: number; x: number; y: number; w: number; h: number }
interface ShotMeta { src: string; w: number; h: number; css?: number; marks: Mark[] }

const props = defineProps<{ id: string; legend?: string[]; caption?: string }>()
const { locale } = useI18n()
const open = computed(() => (locale.value.startsWith('zh') ? '点击查看大图' : 'Open full size'))

/** 优先用当前语言的截图，其次是不分语言的（小铺、CC Switch）。没有这张截图就整块不渲染。 */
const meta = computed<ShotMeta | null>(() => {
  const all = shots as unknown as Record<string, Record<string, ShotMeta>>
  const lang = locale.value.startsWith('zh') ? 'zh' : 'en'
  return all[lang]?.[props.id] ?? all.shared?.[props.id] ?? null
})

/** 截图按 1.5 倍像素密度截的；按实际 CSS 宽度显示，窄的图不被拉大。 */
const frameStyle = computed(() => ({
  maxWidth: `${meta.value?.css ?? meta.value?.w ?? 800}px`,
  aspectRatio: `${meta.value?.w} / ${meta.value?.h}`
}))

function markStyle(m: Mark) {
  return { left: `${m.x}%`, top: `${m.y}%`, width: `${m.w}%`, height: `${m.h}%` }
}
</script>

<template>
  <figure v-if="meta" class="shot">
    <div class="shot-frame" :style="frameStyle">
      <a :href="meta.src" target="_blank" rel="noopener" class="shot-open" :title="open">
        <img :src="meta.src" :width="meta.w" :height="meta.h" alt="" loading="lazy" decoding="async" />
      </a>
      <span v-for="(m, i) in meta.marks" :key="i" class="shot-mark" :class="{ 'is-single': !m.n }" :style="markStyle(m)" aria-hidden="true">
        <b v-if="m.n">{{ m.n }}</b>
      </span>
    </div>
    <ol v-if="legend?.length" class="shot-legend">
      <li v-for="(line, i) in legend" :key="i"><b>{{ i + 1 }}</b><span>{{ line }}</span></li>
    </ol>
    <figcaption v-else-if="caption" class="shot-caption">{{ caption }}</figcaption>
  </figure>
</template>
