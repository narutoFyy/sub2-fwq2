<script setup lang="ts">
import '@fontsource-variable/geist'
import '@fontsource-variable/geist-mono'
import { computed, ref, watch, onBeforeUnmount, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import BrandLogo from '@/components/brand/BrandLogo.vue'
import LocaleToggle from '@/components/brand/LocaleToggle.vue'
import ThemeToggleIcon from '@/components/common/ThemeToggleIcon.vue'
import RollLink from '@/components/brand/RollLink.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import GlowDots from '@/components/visual/GlowDots.vue'
import { SUPPORT_QQ_GROUP } from '@/features/onboarding/config'
import { useGuideStore } from '@/features/onboarding/guideStore'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import TutorialCode from './TutorialCode.vue'
import TutorialDownloads from './TutorialDownloads.vue'
import TutorialShot from './TutorialShot.vue'
import TutorialImage from './TutorialImage.vue'
import { clientName, tutorialClients, useTutorialCopy, type TutorialClient } from './content'
import { buildSteps } from './tutorialSteps'
import { useUiLabels } from './uiLabels'
import './tutorials.css'

const route = useRoute()
const router = useRouter()
const { locale } = useI18n()
const app = useAppStore()
const auth = useAuthStore()
const guide = useGuideStore()
const copy = useTutorialCopy()
const ui = useUiLabels()
const client = computed<TutorialClient>(() => route.params.client === 'codex' ? 'codex' : 'claude-code')
const isIndex = computed(() => !route.params.client)
const isImage = computed(() => route.params.client === 'image')
const imageCardCopy = computed(() => locale.value.startsWith('zh') ? { name: '生图', desc: '网页里写一句话就出图，也能装进 Codex / Claude Code。' } : { name: 'Images', desc: 'Describe a picture in the browser, or add image making to Codex / Claude Code.' })
const name = computed(() => clientName(client.value))
const siteName = computed(() => app.cachedPublicSettings?.site_name || app.siteName || 'Sub2API')
const endpoint = computed(() => (app.cachedPublicSettings?.api_base_url || window.location.origin).replace(/\/+$/, ''))
/** 手动添加供应商时填的地址：Codex 需要带 /v1，Claude Code 不带。 */
const manualEndpoint = computed(() => client.value === 'codex' && !endpoint.value.endsWith('/v1') ? `${endpoint.value}/v1` : endpoint.value)
const plazaOn = computed(() => resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.modelPlaza))
const canStartGuide = computed(() => auth.isAuthenticated && !auth.isAdmin)
const steps = computed(() => buildSteps({
  lang: locale.value.startsWith('zh') ? 'zh' : 'en',
  client: client.value,
  ui: ui.value,
  siteName: siteName.value,
  endpoint: manualEndpoint.value,
  qqGroup: SUPPORT_QQ_GROUP,
  plaza: plazaOn.value
}))
const activeStep = ref(1)
let observer: IntersectionObserver | undefined

function startInAppGuide() {
  guide.start()
  void router.push('/dashboard')
}

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
        <div class="guide-nav-actions"><RouterLink :to="auth.isAuthenticated ? '/dashboard' : '/home'" class="back-home">{{ auth.isAuthenticated ? copy.console : copy.home }}</RouterLink><LocaleToggle /><ThemeToggleIcon /></div>
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
        <RouterLink to="/tutorials/image" class="index-guide">
          <div class="tutorial-image-glyph" aria-hidden="true"><svg viewBox="0 0 24 24"><rect x="3.5" y="4.5" width="17" height="15" rx="2.5" /><path d="M3.5 16l4.6-4.6a1.6 1.6 0 0 1 2.3 0l4.9 4.9m-1.6-1.6 1.2-1.2a1.6 1.6 0 0 1 2.3 0l2.3 2.3" /><path d="M15.5 7.5l.5 1.2 1.2.5-1.2.5-.5 1.2-.5-1.2-1.2-.5 1.2-.5z" /></svg></div>
          <h2>{{ imageCardCopy.name }}</h2>
          <p>{{ imageCardCopy.desc }}</p>
          <span>{{ copy.read }} <span aria-hidden="true">↗</span></span>
        </RouterLink>
      </div>
      <ol class="index-steps"><li v-for="(label, i) in copy.steps" :key="label"><span>{{ String(i + 1).padStart(2, '0') }}</span>{{ label }}</li></ol>
    </main>

    <main v-else-if="isImage" class="guide-main">
      <TutorialImage />
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
          <div class="before-start">
            <h2>{{ copy.prereq }}</h2>
            <p>{{ copy.prereqText }}</p>
            <div class="before-links">
              <RouterLink :to="auth.isAuthenticated ? '/keys' : '/register'">{{ auth.isAuthenticated ? copy.actionKeys : copy.register }}</RouterLink>
              <button v-if="canStartGuide" type="button" class="guide-cta" @click="startInAppGuide">{{ copy.guideCta }}<span>{{ copy.guideCtaAction }} →</span></button>
            </div>
          </div>
          <p class="figure-explainer">{{ copy.figureNote }}</p>

          <section v-for="(step, i) in steps" :id="step.id" :key="step.id" :data-step="i + 1" data-guide-step class="guide-step">
            <div class="step-title"><span>{{ String(i + 1).padStart(2, '0') }}</span><h2>{{ step.title }}</h2></div>
            <p class="step-lead">{{ step.lead }}</p>
            <ol class="action-list">
              <li v-for="(action, j) in step.actions" :key="j" class="action">
                <p class="action-text">{{ action.text }}</p>
                <p v-if="action.hint" class="action-hint">{{ action.hint }}</p>
                <TutorialDownloads v-if="action.downloads" :client="client" />
                <details v-if="action.terminal" class="terminal-help">
                  <summary>{{ copy.terminalSummary }}</summary>
                  <ul><li v-for="line in copy.terminalLines" :key="line">{{ line }}</li></ul>
                </details>
                <TutorialCode v-if="action.code" :code="action.code.text" :label="action.code.label" />
                <TutorialShot v-if="action.shot" :id="action.shot.id" :legend="action.shot.legend" :caption="action.shot.caption" />
                <RouterLink v-if="action.link" :to="action.link.to" class="action-link">{{ action.link.label }} <span aria-hidden="true">↗</span></RouterLink>
              </li>
            </ol>
            <details v-if="step.detail" class="manual-import">
              <summary>{{ step.detail.summary }}</summary>
              <p v-for="line in step.detail.lines" :key="line">{{ line }}</p>
              <dl v-if="step.detail.fields"><template v-for="field in step.detail.fields" :key="field.label"><dt>{{ field.label }}</dt><dd>{{ field.value }}</dd></template></dl>
            </details>
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
