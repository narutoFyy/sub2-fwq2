<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import GuideBlocks from '@/features/onboarding/GuideBlocks.vue'
import { useGuideCopy } from '@/features/onboarding/copy'
import { buildDownloadBlocks } from '@/features/onboarding/downloads'
import { detectArch, detectOS, type Env } from '@/features/onboarding/platform'
import { useDownloadResources } from '@/features/onboarding/useDownloadResources'
import '@/features/onboarding/guide.css'
import type { TutorialClient } from './tutorialSteps'

/** 和站内引导同一套下载内容（按系统自动识别；CC Switch 优先自托管，Codex 走国内镜像，Claude Code 走 npmmirror）。 */
const props = defineProps<{ client: TutorialClient }>()
const copy = useGuideCopy()
const env = ref<Env>({ os: detectOS(), arch: 'unknown' })
const resources = useDownloadResources()

onMounted(async () => {
  void resources.load()
  env.value = { os: env.value.os, arch: await detectArch(env.value.os) }
})

const blocks = computed(() =>
  resources.ready.value
    ? buildDownloadBlocks(env.value, props.client, { manifest: resources.manifest.value, nodeVersion: resources.nodeVersion.value, siteUrl: window.location.origin }, copy.value)
    : []
)
</script>

<template>
  <div class="og-root og-root--inline tutorial-downloads">
    <GuideBlocks v-if="blocks.length" :blocks="blocks" />
    <p v-else class="og-block-hint">{{ copy.download.preparing }}</p>
  </div>
</template>
