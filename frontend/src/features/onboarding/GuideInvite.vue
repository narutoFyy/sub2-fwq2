<script setup lang="ts">
import { useGuideCopy } from './copy'

defineProps<{ show: boolean }>()
const emit = defineEmits<{ start: []; dismiss: [] }>()
const copy = useGuideCopy()
</script>

<template>
  <Transition name="og-invite">
    <aside v-if="show" class="og-invite" role="region" :aria-label="copy.invite.title">
      <span class="og-invite-glow" aria-hidden="true"></span>
      <div class="og-invite-text">
        <strong>{{ copy.invite.title }}</strong>
        <span>{{ copy.invite.body }}</span>
      </div>
      <div class="og-invite-actions">
        <button type="button" class="og-btn og-btn--text" @click="emit('dismiss')">{{ copy.invite.no }}</button>
        <button type="button" class="og-btn og-btn--primary" @click="emit('start')">{{ copy.invite.start }}</button>
      </div>
    </aside>
  </Transition>
</template>
