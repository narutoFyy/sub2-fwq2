<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores'
import { copyText } from '@/features/onboarding/copyText'
import { useImageCopy } from './copy'
import { listImageModels } from './api'
import { buildSkillPrompt, imageBaseUrl, maskKey, MY_IMAGE_REPO } from './skillPrompt'
import { useImageKeys } from './useImageKeys'

const props = defineProps<{ show: boolean; preferredKeyId?: number | null }>()
const emit = defineEmits<{ close: [] }>()

const { locale } = useI18n()
const app = useAppStore()
const copy = useImageCopy()
const { keys, loading, load } = useImageKeys()
const selectedId = ref<number | null>(null)
const copied = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined

watch(
  () => props.show,
  async (open) => {
    if (!open) return
    copied.value = false
    await load()
    const preferred = keys.value.find((k) => k.id === props.preferredKeyId)
    selectedId.value = (preferred ?? keys.value[0])?.id ?? null
  },
  { immediate: true }
)

const lang = computed<'zh' | 'en'>(() => (locale.value.startsWith('zh') ? 'zh' : 'en'))
const baseUrl = computed(() => imageBaseUrl(app.cachedPublicSettings?.api_base_url))
const selected = computed(() => keys.value.find((k) => k.id === selectedId.value) ?? null)
// 写进那段话的模型：用这把密钥实际能用的最新生图模型，查不到就用默认值
const model = ref<string | null>(null)
let modelToken = 0
watch(selected, async (key) => {
  const token = ++modelToken
  model.value = null
  if (!key) return
  try {
    const found = await listImageModels(key.key)
    if (token === modelToken) model.value = found[0] ?? null
  } catch {
    /* 用默认模型 */
  }
})
const fullPrompt = computed(() => (selected.value ? buildSkillPrompt({ lang: lang.value, baseUrl: baseUrl.value, apiKey: selected.value.key, model: model.value }) : ''))
const shownPrompt = computed(() => (selected.value ? buildSkillPrompt({ lang: lang.value, baseUrl: baseUrl.value, apiKey: maskKey(selected.value.key), model: model.value }) : ''))

async function onCopy() {
  if (!fullPrompt.value || !(await copyText(fullPrompt.value))) return
  copied.value = true
  clearTimeout(timer)
  timer = setTimeout(() => (copied.value = false), 2400)
}
onBeforeUnmount(() => clearTimeout(timer))
</script>

<template>
  <BaseDialog :show="show" :title="copy.skill.title" width="normal" @close="emit('close')">
    <div class="skill-dialog">
      <p class="skill-body">{{ copy.skill.body }}</p>

      <p v-if="loading && !keys.length" class="skill-muted">{{ copy.skill.loading }}</p>
      <div v-else-if="!keys.length" class="skill-empty">
        <p>{{ copy.noKeyTitle }}</p>
        <router-link to="/keys" class="btn btn-secondary btn-sm" @click="emit('close')">{{ copy.noKeyAction }}</router-link>
      </div>
      <template v-else>
        <label v-if="keys.length > 1" class="skill-key">
          <span>{{ copy.skill.keyLabel }}</span>
          <select v-model="selectedId" class="input">
            <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }}</option>
          </select>
        </label>
        <pre class="skill-prompt">{{ shownPrompt }}</pre>
        <p class="skill-muted">{{ copy.skill.note }}</p>
      </template>
    </div>
    <template #footer>
      <div class="flex w-full items-center justify-between gap-3">
        <a :href="MY_IMAGE_REPO" target="_blank" rel="noopener noreferrer" class="text-sm text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-dark-100">{{ copy.skill.repo }}</a>
        <button type="button" class="btn btn-primary" :disabled="!fullPrompt" @click="onCopy">{{ copied ? copy.skill.copied : copy.skill.copy }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<style scoped>
.skill-dialog { display: grid; gap: 16px; }
.skill-body { font-size: 14px; line-height: 1.7; color: rgb(63 63 70); }
.dark .skill-body { color: rgb(212 212 216); }
.skill-muted { font-size: 12px; color: rgb(113 113 122); }
.skill-key { display: grid; gap: 8px; font-size: 13px; color: rgb(82 82 91); }
.dark .skill-key { color: rgb(161 161 170); }
.skill-prompt {
  max-height: 280px;
  overflow: auto;
  padding: 16px;
  border-radius: 12px;
  background: #111113;
  color: #e4e4e7;
  font: 12.5px/1.75 'Geist Mono Variable', ui-monospace, monospace;
  white-space: pre-wrap;
  word-break: break-word;
}
.skill-empty { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 16px; border-radius: 12px; background: rgb(244 244 245); font-size: 14px; }
.dark .skill-empty { background: rgb(24 24 27); }
</style>
