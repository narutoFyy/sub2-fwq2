<script setup lang="ts">
import '@fontsource-variable/geist'
import '@fontsource-variable/geist-mono'
import { computed, ref, watch, onBeforeUnmount, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import BrandLogo from '@/components/brand/BrandLogo.vue'
import LocaleToggle from '@/components/brand/LocaleToggle.vue'
import ThemeToggleIcon from '@/components/common/ThemeToggleIcon.vue'
import RollLink from '@/components/brand/RollLink.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import GlowDots from '@/components/visual/GlowDots.vue'
import TutorialFigure from './TutorialFigure.vue'
import TutorialCode from './TutorialCode.vue'
import { clientName, tutorialClients, useTutorialCopy, type TutorialClient } from './content'
import './tutorials.css'

const route = useRoute()
const app = useAppStore()
const auth = useAuthStore()
const copy = useTutorialCopy()
const client = computed<TutorialClient>(() => route.params.client === 'codex' ? 'codex' : 'claude-code')
const isIndex = computed(() => !route.params.client)
const name = computed(() => clientName(client.value))
const siteName = computed(() => app.cachedPublicSettings?.site_name || app.siteName || 'Sub2API')
const endpoint = computed(() => (app.cachedPublicSettings?.api_base_url || window.location.origin).replace(/\/+$/, ''))
const manualEndpoint = computed(() => client.value === 'codex' && !endpoint.value.endsWith('/v1') ? `${endpoint.value}/v1` : endpoint.value)
const cli = computed(() => client.value === 'codex' ? 'codex' : 'claude')
const installCommand = computed(() => client.value === 'codex' ? 'npm install -g @openai/codex' : 'npm install -g @anthropic-ai/claude-code')
const installDocs = computed(() => client.value === 'codex' ? 'https://developers.openai.com/codex/cli' : 'https://code.claude.com/docs/en/setup')
const group = computed(() => client.value === 'codex' ? copy.value.codexGroup : copy.value.claudeGroup)
/** 站内支付没开（如只走外部小铺 + 兑换码）时，不展示指向 /purchase、/orders 的入口。 */
const paymentOn = computed(() => app.cachedPublicSettings?.payment_enabled === true)
const activeStep = ref(1)
let observer: IntersectionObserver | undefined
const steps = computed(() => [
  { id: 'credit', title: copy.value.rechargeTitle, body: copy.value.rechargeBody, note: copy.value.rechargeNote },
  { id: 'key', title: copy.value.keyTitle, body: copy.value.keyBody, note: copy.value.keyNote },
  { id: 'install', title: copy.value.installTitle, body: copy.value.installBody },
  { id: 'import', title: copy.value.importTitle, body: copy.value.importBody },
  { id: 'launch', title: copy.value.activateTitle, body: copy.value.activateBody, note: copy.value.activateNote },
  { id: 'verify', title: copy.value.verifyTitle, body: copy.value.verifyBody },
])

watch(() => route.path, async () => {
  observer?.disconnect()
  activeStep.value = 1
  await nextTick()
  if (typeof IntersectionObserver === 'undefined') return
  observer = new IntersectionObserver(entries => {
    for (const entry of entries) {
      if (entry.isIntersecting) activeStep.value = Number((entry.target as HTMLElement).dataset.step)
    }
  }, { rootMargin: '-10% 0px -65% 0px' })
  document.querySelectorAll('[data-guide-step]').forEach(el => observer?.observe(el))
}, { immediate: true })
onBeforeUnmount(() => observer?.disconnect())
</script>

<template>
  <div class="tutorial-surface">
    <header class="guide-header">
      <nav class="guide-nav" :aria-label="copy.nav">
        <BrandLogo to="/home" :name="siteName" />
        <div class="guide-nav-actions"><RouterLink to="/home" class="back-home">{{ copy.home }}</RouterLink><LocaleToggle /><ThemeToggleIcon /></div>
      </nav>
    </header>

    <main v-if="isIndex" class="guide-index">
      <div class="index-glow" aria-hidden="true"><GlowDots :center-x="0.78" :center-y="0.45" :gain="0.48" :max-cells="9000" /></div>
      <h1>{{ copy.title }}</h1>
      <p class="index-intro">{{ copy.intro }}</p>
      <div class="index-guides">
        <RouterLink v-for="item in tutorialClients" :key="item" :to="`/tutorials/${item}`" class="index-guide">
          <ModelIcon :model="item === 'codex' ? 'gpt-5' : 'claude'" :class="{ 'tutorial-codex-icon': item === 'codex' }" size="38px" />
          <h2>{{ clientName(item) }}</h2>
          <p>{{ item === 'codex' ? copy.codexDesc : copy.claudeDesc }}</p>
          <span>{{ copy.read }} <span aria-hidden="true">↗</span></span>
        </RouterLink>
      </div>
      <ol class="index-steps"><li v-for="(label, i) in copy.steps" :key="label"><span>{{ String(i + 1).padStart(2, '0') }}</span>{{ label }}</li></ol>
    </main>

    <main v-else class="guide-main">
      <div class="guide-hero">
        <RouterLink to="/tutorials" class="guide-breadcrumb">{{ copy.nav }}</RouterLink><span class="breadcrumb-slash" aria-hidden="true">/</span><span class="guide-breadcrumb">{{ name }}</span>
        <div class="guide-title-row"><ModelIcon :model="client === 'codex' ? 'gpt-5' : 'claude'" :class="{ 'tutorial-codex-icon': client === 'codex' }" size="42px" /><h1>{{ name }}<span>{{ copy.guideSuffix }}</span></h1></div>
        <p class="guide-intro">{{ copy.guideIntro }}</p>
        <div class="client-switch" :aria-label="copy.switchClient"><RouterLink v-for="item in tutorialClients" :key="item" :to="`/tutorials/${item}`" :class="{ selected: item === client }" :aria-current="item === client ? 'page' : undefined">{{ clientName(item) }}</RouterLink></div>
      </div>

      <div class="guide-layout">
        <aside class="guide-toc">
          <nav :aria-label="copy.contents">
            <p>{{ copy.contents }}</p>
            <ol><li v-for="(step, i) in steps" :key="step.id"><RouterLink :to="{ path: route.path, hash: `#${step.id}` }" :class="{ current: activeStep === i + 1 }" :aria-current="activeStep === i + 1 ? 'step' : undefined"><span>{{ String(i + 1).padStart(2, '0') }}</span>{{ copy.steps[i] }}</RouterLink></li></ol>
            <a href="#troubleshooting" class="toc-help">{{ copy.faqTitle }}</a>
          </nav>
        </aside>

        <div class="guide-article">
          <div class="before-start"><h2>{{ copy.prereq }}</h2><p>{{ copy.prereqText }}</p><RouterLink :to="auth.isAuthenticated ? '/keys' : '/register'">{{ auth.isAuthenticated ? copy.actionKeys : copy.register }}</RouterLink></div>
          <p class="figure-explainer">{{ copy.figureNote }}</p>

          <section v-for="(step, i) in steps" :id="step.id" :key="step.id" :data-step="i + 1" data-guide-step class="guide-step">
            <div class="step-title"><span>{{ String(i + 1).padStart(2, '0') }}</span><h2>{{ step.title }}</h2></div>
            <ol class="step-instructions"><li v-for="line in step.body" :key="line">{{ line }}</li></ol>
            <p v-if="i === 1" class="group-hint">{{ group }}</p>
            <TutorialFigure :step="i + 1" :client="client" :site-name="siteName" :endpoint="endpoint" />
            <template v-if="i === 0"><div class="guide-actions"><RouterLink v-if="paymentOn" to="/purchase">{{ copy.actionRecharge }} <span aria-hidden="true">↗</span></RouterLink><RouterLink to="/redeem">{{ $t('nav.redeem') }}</RouterLink><RouterLink v-if="paymentOn" to="/orders">{{ $t('nav.myOrders') }}</RouterLink></div></template>
            <template v-if="i === 1"><div class="guide-actions"><RouterLink to="/keys">{{ copy.actionKeys }} <span aria-hidden="true">↗</span></RouterLink><RouterLink to="/available-channels">{{ copy.actionModels }}</RouterLink></div></template>
            <template v-if="i === 2">
              <div class="guide-actions"><a href="https://github.com/farion1231/cc-switch/releases/latest" target="_blank" rel="noopener noreferrer">{{ copy.downloads }} ↗</a><a :href="installDocs" target="_blank" rel="noopener noreferrer">{{ copy.officialInstall }} ↗</a><a href="https://nodejs.org/en/download" target="_blank" rel="noopener noreferrer">{{ copy.nodeDownload }} ↗</a></div>
              <TutorialCode :code="installCommand" :label="`${name} / npm`" />
              <TutorialCode :code="`${cli} --version`" :label="copy.versionCheck" />
            </template>
            <template v-if="i === 3">
              <div class="guide-actions"><RouterLink to="/keys">{{ copy.actionKeys }} <span aria-hidden="true">↗</span></RouterLink></div>
              <details class="manual-import"><summary>{{ copy.importHidden }}</summary><dl><dt>{{ copy.endpoint }}</dt><dd><code>{{ manualEndpoint }}</code></dd><dt>{{ copy.apiKey }}</dt><dd>{{ copy.keyPlaceholder }}</dd></dl><p>{{ copy.endpointNote }}</p></details>
            </template>
            <template v-if="i === 4"><TutorialCode :code="cli" :label="name" /></template>
            <template v-if="i === 5"><TutorialCode :code="copy.prompt" :label="name" /><div class="guide-actions"><RouterLink to="/usage">{{ copy.actionUsage }} <span aria-hidden="true">↗</span></RouterLink></div></template>
            <p v-if="step.note" class="guide-note">{{ step.note }}</p>
          </section>

          <div class="connection-success"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12 4 4L19 6" /></svg><div><h2>{{ copy.success }}</h2><p>{{ copy.successDesc }}</p></div></div>
          <section id="troubleshooting" class="guide-faq"><h2>{{ copy.faqTitle }}</h2><details v-for="faq in copy.faqs" :key="faq.q"><summary>{{ faq.q }}</summary><p>{{ faq.a }}</p></details></section>
          <div class="next-guide"><p>{{ copy.nextGuide }}</p><RollLink :to="`/tutorials/${client === 'codex' ? 'claude-code' : 'codex'}`" :label="client === 'codex' ? 'Claude Code' : 'Codex'" arrow /></div>
        </div>
      </div>
    </main>
    <footer class="guide-footer"><BrandLogo to="/home" :name="siteName" :size="22" /><RouterLink to="/tutorials">{{ copy.nav }}</RouterLink><RouterLink to="/home">{{ copy.home }}</RouterLink></footer>
  </div>
</template>
