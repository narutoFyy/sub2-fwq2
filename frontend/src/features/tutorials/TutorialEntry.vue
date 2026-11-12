<script setup lang="ts">
import { useTutorialCopy, tutorialClients, clientName } from './content'
import ModelIcon from '@/components/common/ModelIcon.vue'
const copy = useTutorialCopy()
</script>

<template>
  <section class="tutorial-entry" aria-labelledby="tutorial-entry-title">
    <div class="entry-heading">
      <h2 id="tutorial-entry-title">{{ copy.entryTitle }}</h2>
      <p>{{ copy.entryDesc }}</p>
    </div>
    <div class="entry-links">
      <RouterLink v-for="client in tutorialClients" :key="client" :to="`/tutorials/${client}`" class="entry-link">
        <ModelIcon :model="client === 'codex' ? 'gpt-5' : 'claude'" :class="{ 'tutorial-codex-icon': client === 'codex' }" size="32px" />
        <span><strong>{{ clientName(client) }}</strong><small>{{ copy.read }}</small></span>
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true"><path d="M5 12h14m-6-6 6 6-6 6" /></svg>
      </RouterLink>
    </div>
  </section>
</template>

<style scoped>
.tutorial-entry { max-width: 1280px; margin: 0 auto; padding: 80px 48px; color: #0a0a0b; }
:global(.dark) .tutorial-entry { color: #ededef; }
.entry-heading { display: flex; align-items: baseline; justify-content: space-between; gap: 24px; margin-bottom: 40px; }
h2 { font-size: clamp(26px, 3vw, 38px); line-height: 1.25; font-weight: 500; }
p, small { color: #63636b; }
:global(.dark) .tutorial-entry :is(p, small) { color: #a0a4ad; }
.entry-links { display: grid; grid-template-columns: 1fr 1fr; gap: 48px; }
.entry-link { display: flex; align-items: center; gap: 20px; padding: 28px 0; border-bottom: 1px solid #dedee1; }
:global(.dark) .entry-link { border-color: #2b2b30; }
.entry-link span { flex: 1; }
strong { display: block; font-size: 24px; font-weight: 500; }
small { display: block; margin-top: 8px; font-size: 14px; }
.entry-link svg { transition: transform .2s; }
.entry-link:hover svg { transform: translateX(4px); }
.entry-link:focus-visible { outline: 2px solid currentColor; outline-offset: 6px; border-radius: 4px; }
@media (max-width: 700px) { .tutorial-entry { padding: 56px 24px; } .entry-heading { display: block; margin-bottom: 16px; } p { margin-top: 14px; } .entry-links { grid-template-columns: 1fr; gap: 0; } }
@media (prefers-reduced-motion: reduce) { .entry-link svg { transition: none; } }
</style>
