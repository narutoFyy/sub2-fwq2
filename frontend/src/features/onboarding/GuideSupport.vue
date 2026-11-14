<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { SUPPORT_QQ_GROUP } from './config'
import { copyText } from './copyText'
import { useGuideCopy } from './copy'

const copy = useGuideCopy()
const open = ref(false)
const copied = ref(false)
const root = ref<HTMLElement>()
let timer: ReturnType<typeof setTimeout> | undefined

async function copyGroup() {
  if (!(await copyText(SUPPORT_QQ_GROUP))) return
  copied.value = true
  clearTimeout(timer)
  timer = setTimeout(() => (copied.value = false), 1600)
}

async function toggle() {
  open.value = !open.value
  if (open.value) {
    await nextTick()
    root.value?.querySelector<HTMLElement>('.og-support-copy')?.focus()
  }
}

function onOutside(event: MouseEvent) {
  if (open.value && root.value && !root.value.contains(event.target as Node)) open.value = false
}
onMounted(() => document.addEventListener('mousedown', onOutside))
onBeforeUnmount(() => {
  clearTimeout(timer)
  document.removeEventListener('mousedown', onOutside)
})
</script>

<template>
  <div ref="root" class="og-support" @keydown.esc.stop="open = false">
    <button type="button" class="og-link" :aria-expanded="open" @click="toggle">{{ copy.support }}</button>
    <Transition name="og-pop">
      <div v-if="open" class="og-support-pop" role="group" :aria-label="copy.supportTitle">
        <p>{{ copy.supportTitle }}</p>
        <div class="og-support-row">
          <span class="og-support-label">{{ copy.supportGroup }}</span>
          <strong class="og-support-number">{{ SUPPORT_QQ_GROUP }}</strong>
          <button type="button" class="og-support-copy" @click="copyGroup">{{ copied ? copy.copied : copy.copy }}</button>
        </div>
      </div>
    </Transition>
  </div>
</template>
