import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GuideClient } from './config'

export function useGuideCopy() {
  const { locale } = useI18n()
  return computed(() => (locale.value.startsWith('zh') ? zh : en))
}

const zh = {
  stages: ['选工具', '兑换额度', '创建密钥', '接入工具'],
  clientName: (c: GuideClient) => (c === 'codex' ? 'Codex' : 'Claude Code'),

  welcome: {
    title: '几分钟，把 AI 编程工具接上来',
    body: '跟着引导走：兑换额度、创建密钥、接入工具。每一步都有指引，随时可以跳过。',
    start: '开始',
    skip: '跳过引导'
  },
  client: {
    title: '你想用哪个工具？',
    body: '选好后，创建密钥时会自动帮你选对类型。',
    codex: 'OpenAI 的编程助手，有桌面版，装好就能用。',
    claude: 'Anthropic 的命令行编程助手。'
  },
  buy: {
    title: '先买一个兑换码',
    body: (label: string) => `点击左侧的「${label}」，在新打开的页面里选购额度，付款后会拿到一个兑换码。`,
    hint: '会在新标签页打开，买好后回到这里继续。',
    already: '我已经有兑换码'
  },
  returned: {
    title: '买好兑换码了吗？',
    body: '先复制好兑换码，下一步到「兑换」页使用它。',
    yes: '买好了，去兑换',
    no: '还没买好，再去看看'
  },
  redeemNav: {
    title: '去兑换',
    body: (label: string) => `点击左侧的「${label}」。`
  },
  redeem: {
    title: '粘贴兑换码并兑换',
    body: '把兑换码粘贴到输入框，点「兑换」，额度会立刻到账。'
  },
  keyNav: {
    title: '创建你的密钥',
    body: (label: string) => `点击左侧的「${label}」。`
  },
  keyCreate: {
    title: '点「创建密钥」',
    body: '密钥就是你的工具连接本站的凭证。'
  },
  keyForm: {
    title: '确认类型并选择分组',
    body: (client: string) => `已按你选的 ${client} 选好类型。选一个分组（倍率越低越便宜），然后点「创建」。`
  },
  finish: {
    title: '密钥创建好了',
    body: '最后一步：安装 CC Switch，一键导入这个密钥。',
    steps: ['下载并安装 CC Switch', '回到「API 密钥」页，点「导入到 CC Switch」', '启用后，在终端或桌面端开始使用'],
    tutorial: '查看图文教程',
    done: '完成'
  },
  done: { redeem: '兑换成功', key: '密钥已创建' },

  back: '上一步',
  skip: '跳过引导',
  skipConfirmTitle: '确定跳过引导吗？',
  skipConfirmBody: '跳过后不会再自动提示，首页上的图文教程随时可以看。',
  skipConfirm: '确认跳过',
  skipCancel: '继续引导',
  goto: (page: string) => `前往${page}`,
  pageRedeem: '兑换页',
  pageKeys: 'API 密钥页',
  support: '不想配置？加群找管理员',
  supportTitle: '加 QQ 群，找管理员帮你配置',
  supportGroup: 'QQ 群号',
  copy: '复制群号',
  copied: '已复制',

  invite: {
    title: '还没创建密钥？',
    body: '几步就能把 Codex / Claude Code 接上来。',
    start: '开始',
    no: '不用了'
  }
}

type GuideCopy = typeof zh

const en: GuideCopy = {
  stages: ['Pick a tool', 'Add credit', 'Create key', 'Connect'],
  clientName: (c: GuideClient) => (c === 'codex' ? 'Codex' : 'Claude Code'),

  welcome: {
    title: 'Connect your AI coding tool in minutes',
    body: 'Follow along: add credit, create a key, connect your tool. Every step is guided, and you can skip any time.',
    start: 'Start',
    skip: 'Skip guide'
  },
  client: {
    title: 'Which tool do you use?',
    body: "We'll pick the right key type for you later.",
    codex: "OpenAI's coding assistant, with a desktop app.",
    claude: "Anthropic's command-line coding assistant."
  },
  buy: {
    title: 'First, get a redeem code',
    body: (label: string) => `Click “${label}” on the left, choose an amount on the page that opens, and you'll get a redeem code after paying.`,
    hint: 'It opens in a new tab. Come back here when you have the code.',
    already: 'I already have a code'
  },
  returned: {
    title: 'Got your redeem code?',
    body: 'Copy it first — you will use it on the Redeem page next.',
    yes: 'Got it, go redeem',
    no: 'Not yet, take another look'
  },
  redeemNav: {
    title: 'Go redeem it',
    body: (label: string) => `Click “${label}” on the left.`
  },
  redeem: {
    title: 'Paste the code and redeem',
    body: 'Paste your code into the box and click Redeem. Credit arrives instantly.'
  },
  keyNav: {
    title: 'Create your key',
    body: (label: string) => `Click “${label}” on the left.`
  },
  keyCreate: {
    title: 'Click “Create key”',
    body: 'A key is the credential your tool uses to connect to this site.'
  },
  keyForm: {
    title: 'Confirm type and pick a group',
    body: (client: string) => `We preselected the type for ${client}. Pick a group (lower rate = cheaper), then click Create.`
  },
  finish: {
    title: 'Your key is ready',
    body: 'Last step: install CC Switch and import this key with one click.',
    steps: ['Download and install CC Switch', 'Back on the API keys page, click “Import to CC Switch”', 'Enable it, then start using it in your terminal or desktop app'],
    tutorial: 'Open the illustrated guide',
    done: 'Done'
  },
  done: { redeem: 'Redeemed', key: 'Key created' },

  back: 'Back',
  skip: 'Skip guide',
  skipConfirmTitle: 'Skip the guide?',
  skipConfirmBody: "You won't be prompted again. The illustrated guides on the home page are always there.",
  skipConfirm: 'Skip',
  skipCancel: 'Keep going',
  goto: (page: string) => `Go to ${page}`,
  pageRedeem: 'Redeem page',
  pageKeys: 'API keys page',
  support: 'Need help? Ask in our group',
  supportTitle: 'Join the QQ group and ask an admin to help',
  supportGroup: 'QQ group',
  copy: 'Copy number',
  copied: 'Copied',

  invite: {
    title: 'No API key yet?',
    body: 'A few steps to connect Codex or Claude Code.',
    start: 'Start',
    no: 'No thanks'
  }
}
