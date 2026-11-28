<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import SkillPromptDialog from '@/features/image/SkillPromptDialog.vue'
import { MY_IMAGE_REPO } from '@/features/image/skillPrompt'
import TutorialCode from './TutorialCode.vue'
import { useTutorialCopy } from './content'

const { locale } = useI18n()
const auth = useAuthStore()
const base = useTutorialCopy()
const dialog = ref(false)

const zh = {
  name: '生图',
  suffix: '使用指南',
  intro: '两种用法：在网页上直接生图，或者把 my-image 装进 Codex / Claude Code。大约 3 分钟。',
  prereq: '开始前',
  prereqText: '你需要一把能生图的密钥。创建密钥时，选一个支持生图的分组（名字里一般带「生图」或 image）就行。',
  toKeys: '前往 API 密钥',
  register: '注册账号',
  webTitle: '在网页上生图',
  webLead: '什么都不用装，写一句话、点一下就出图。',
  web: [
    { text: '打开左侧菜单里的「生图」。', link: { to: '/image', label: '打开生图页' } },
    { text: '在「描述画面」里写一句你想要的画面。不知道写什么，就点下面的例子。', hint: '写清楚主体、场景、光线和风格，效果会更稳，比如「雨后的上海街头，霓虹倒映在积水里，电影感」。' },
    { text: '选好比例（方形 / 横版 / 竖版），点「生成」，每次出一张。', hint: '也可以按 ⌘ + 回车（Windows 是 Ctrl + 回车）。' },
    { text: '几十秒后图片出现。鼠标移到图上，可以下载、放大，或者「在这张基础上改」。' },
    { text: '想改一张已有的图：在「参考图」里上传它，再写清楚要怎么改、哪些地方保持不变。' }
  ],
  webNote: '生成记录只保存在你这台设备的浏览器里，换电脑或清理浏览器数据后就看不到了，喜欢的图记得下载。',
  agentTitle: '在 Codex / Claude Code 里生图',
  agentLead: '复制一段话发给你的智能体，它会自己装好 my-image，以后让它画图就行。',
  agent: [
    { text: '先装好 Codex 或 Claude Code。还没装的话，按对应的接入教程装好。', links: [{ to: '/tutorials/codex', label: 'Codex 接入教程' }, { to: '/tutorials/claude-code', label: 'Claude Code 接入教程' }] },
    { text: '复制那段话。生图页左下角、密钥页的提示里都能找到，也可以直接点下面的按钮。', button: true },
    { text: '把这段话粘贴到 Codex 或 Claude Code 的对话框里发送。它会自己克隆 my-image、写好配置，再画一张橘猫测试图。' },
    { text: '之后直接跟它说要画什么、或者要怎么改图，它就会用 my-image 来做。图片会保存在项目的 outputs/my-image 文件夹里。', code: '帮我画一张 16:9 的网站首屏图：清晨海边的咖啡馆，暖色调，留出左侧放标题的位置' }
  ],
  copyButton: '复制这段话',
  example: '比如这样说',
  agentNote: '那段话里带着你的密钥，只发给自己电脑上的智能体，不要发到群里或公开的地方。',
  repo: 'my-image 源码',
  faqTitle: '卡住了？',
  faqs: [
    { q: '提示 403，或「没有生图权限」', a: '这把密钥的分组不支持生图。在密钥页换一个支持生图的分组，或新建一把。' },
    { q: '智能体说找不到 my-image', a: '新开一个对话，或者把 Codex / Claude Code 关掉重开，让它重新发现 skill。还不行，就再发一次那段话。' },
    { q: '提示 429，或「上游繁忙」', a: '现在用的人多，等一会儿再试，或者换一个分组。' }
  ]
}

const en: typeof zh = {
  name: 'Images',
  suffix: 'guide',
  intro: 'Two ways to use it: make images right in the browser, or install my-image into Codex / Claude Code. About 3 minutes.',
  prereq: 'Before you start',
  prereqText: 'You need a key that can make images. When you create one, pick a group that supports images (usually named with “image”).',
  toKeys: 'Open API keys',
  register: 'Create an account',
  webTitle: 'Make images in the browser',
  webLead: 'Nothing to install — write a sentence, click, get a picture.',
  web: [
    { text: 'Open “Images” in the left menu.', link: { to: '/image', label: 'Open the image page' } },
    { text: 'Describe what you want under “Describe the picture”. Not sure? Tap one of the examples.', hint: 'Mention the subject, the setting, the light and the style for steadier results, e.g. “A Shanghai street after rain, neon reflected in puddles, cinematic”.' },
    { text: 'Pick a shape (square / landscape / portrait), then click Generate. You get one picture each time.', hint: 'You can also press ⌘ + Enter (Ctrl + Enter on Windows).' },
    { text: 'The picture shows up in under a minute. Hover it to download, enlarge, or edit from it.' },
    { text: 'To change an existing image, upload it as the reference and describe what should change and what should stay.' }
  ],
  webNote: 'History is saved only in this browser. It is gone on another computer or after clearing site data, so download the ones you like.',
  agentTitle: 'Make images in Codex / Claude Code',
  agentLead: 'Copy one message to your agent and it installs my-image itself. After that, just ask it to draw.',
  agent: [
    { text: 'Install Codex or Claude Code first. Not yet? Follow its setup guide.', links: [{ to: '/tutorials/codex', label: 'Codex setup guide' }, { to: '/tutorials/claude-code', label: 'Claude Code setup guide' }] },
    { text: 'Copy the message. It is at the bottom left of the image page and in the hint on the keys page, or use the button below.', button: true },
    { text: 'Paste it into Codex or Claude Code and send. It clones my-image, writes the config and draws a cat as a test.' },
    { text: 'From then on, tell it what to draw or how to edit an image and it uses my-image. Pictures are saved in your project’s outputs/my-image folder.', code: 'Make a 16:9 hero image for a website: a seaside café at dawn, warm tones, leave room on the left for a headline' }
  ],
  copyButton: 'Copy the message',
  example: 'For example',
  agentNote: 'The message contains your key. Only send it to an agent on your own computer — never in a group chat or anywhere public.',
  repo: 'my-image source',
  faqTitle: 'Stuck?',
  faqs: [
    { q: '403, or “image generation not allowed”', a: 'The key’s group cannot make images. Switch it to a group that supports images on the keys page, or create a new key.' },
    { q: 'The agent cannot find my-image', a: 'Start a new conversation, or quit and reopen Codex / Claude Code so it picks up the skill. If that fails, send the message again.' },
    { q: '429, or “upstream busy”', a: 'It is busy right now. Wait a bit and try again, or use another group.' }
  ]
}

const copy = computed(() => (locale.value.startsWith('zh') ? zh : en))
const sections = computed(() => [
  { id: 'web', title: copy.value.webTitle },
  { id: 'agent', title: copy.value.agentTitle }
])
</script>

<template>
  <div class="guide-hero">
    <RouterLink to="/tutorials" class="guide-breadcrumb">{{ base.nav }}</RouterLink><span class="breadcrumb-slash" aria-hidden="true">/</span><span class="guide-breadcrumb">{{ copy.name }}</span>
    <div class="guide-title-row">
      <span class="tutorial-image-icon" aria-hidden="true">
        <svg viewBox="0 0 24 24"><rect x="3.5" y="4.5" width="17" height="15" rx="2.5" /><path d="M3.5 16l4.6-4.6a1.6 1.6 0 0 1 2.3 0l4.9 4.9m-1.6-1.6 1.2-1.2a1.6 1.6 0 0 1 2.3 0l2.3 2.3" /><path d="M15.5 7.5l.5 1.2 1.2.5-1.2.5-.5 1.2-.5-1.2-1.2-.5 1.2-.5z" /></svg>
      </span>
      <h1>{{ copy.name }}<span>{{ copy.suffix }}</span></h1>
    </div>
    <p class="guide-intro">{{ copy.intro }}</p>
  </div>

  <div class="guide-layout">
    <aside class="guide-toc">
      <nav :aria-label="base.contents">
        <p>{{ base.contents }}</p>
        <ol>
          <li v-for="(s, i) in sections" :key="s.id">
            <a :href="`#${s.id}`"><span>{{ String(i + 1).padStart(2, '0') }}</span>{{ s.title }}</a>
          </li>
        </ol>
        <a href="#troubleshooting" class="toc-help">{{ copy.faqTitle }}</a>
      </nav>
    </aside>

    <div class="guide-article">
      <div class="before-start">
        <h2>{{ copy.prereq }}</h2>
        <p>{{ copy.prereqText }}</p>
        <div class="before-links">
          <RouterLink :to="auth.isAuthenticated ? '/keys' : '/register'">{{ auth.isAuthenticated ? copy.toKeys : copy.register }}</RouterLink>
        </div>
      </div>

      <section id="web" class="guide-step">
        <div class="step-title"><span>01</span><h2>{{ copy.webTitle }}</h2></div>
        <p class="step-lead">{{ copy.webLead }}</p>
        <ol class="action-list">
          <li v-for="(a, j) in copy.web" :key="j" class="action">
            <p class="action-text">{{ a.text }}</p>
            <p v-if="a.hint" class="action-hint">{{ a.hint }}</p>
            <RouterLink v-if="a.link" :to="a.link.to" class="action-link">{{ a.link.label }} <span aria-hidden="true">↗</span></RouterLink>
          </li>
        </ol>
        <p class="guide-note">{{ copy.webNote }}</p>
      </section>

      <section id="agent" class="guide-step">
        <div class="step-title"><span>02</span><h2>{{ copy.agentTitle }}</h2></div>
        <p class="step-lead">{{ copy.agentLead }}</p>
        <ol class="action-list">
          <li v-for="(a, j) in copy.agent" :key="j" class="action">
            <p class="action-text">{{ a.text }}</p>
            <div v-if="a.links" class="tutorial-image-links">
              <RouterLink v-for="l in a.links" :key="l.to" :to="l.to" class="action-link">{{ l.label }} <span aria-hidden="true">↗</span></RouterLink>
            </div>
            <template v-if="a.button">
              <button v-if="auth.isAuthenticated" type="button" class="tutorial-image-btn" @click="dialog = true">{{ copy.copyButton }}</button>
              <RouterLink v-else to="/register" class="action-link">{{ copy.register }} <span aria-hidden="true">↗</span></RouterLink>
            </template>
            <TutorialCode v-if="a.code" :code="a.code" :label="copy.example" />
          </li>
        </ol>
        <p class="guide-note">{{ copy.agentNote }} <a :href="MY_IMAGE_REPO" target="_blank" rel="noopener noreferrer" class="underline underline-offset-4">{{ copy.repo }}</a></p>
      </section>

      <section id="troubleshooting" class="guide-faq">
        <h2>{{ copy.faqTitle }}</h2>
        <details v-for="faq in copy.faqs" :key="faq.q"><summary>{{ faq.q }}</summary><p>{{ faq.a }}</p></details>
      </section>
    </div>
  </div>

  <SkillPromptDialog v-if="auth.isAuthenticated" :show="dialog" @close="dialog = false" />
</template>

<style scoped>
.tutorial-image-icon { display: grid; place-items: center; width: 42px; height: 42px; border-radius: 12px; background: var(--guide-panel); color: var(--guide-accent-ink); }
.tutorial-image-icon svg { width: 24px; height: 24px; fill: none; stroke: currentColor; stroke-width: 1.5; stroke-linecap: round; stroke-linejoin: round; }
.tutorial-image-links { display: flex; flex-wrap: wrap; gap: 20px; }
.tutorial-image-btn { margin-top: 16px; height: 40px; padding: 0 20px; border-radius: 10px; background: #0a0a0b; color: #fafafa; font-size: 14px; font-weight: 500; transition: opacity 0.2s; }
.tutorial-image-btn:hover { opacity: 0.88; }
.dark .tutorial-image-btn { background: #ededef; color: #0a0a0b; }
</style>
