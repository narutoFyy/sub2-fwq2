import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GuideClient } from './config'

export function useGuideCopy() {
  const { locale } = useI18n()
  return computed(() => (locale.value.startsWith('zh') ? guideCopyZh : guideCopyEn))
}

export const guideCopyZh = {
  stages: ['选工具', '兑换额度', '创建密钥', '接入工具'],
  stageDone: '全部完成',
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
  ccs: {
    title: (tool: string) => `${tool} 和 CC Switch 都装好了吗？`,
    body: (tool: string) => `CC Switch 负责把密钥一键配置进 ${tool}，两个都要装在电脑上。`,
    yes: '都装好了',
    no: '还没有，带我装'
  },
  download: {
    title: '先把软件装好',
    body: '按顺序来：下载 → 安装 → 回到这里点「继续」。',
    cont: '我装好了，继续',
    collapse: '收起',
    pill: '引导进行中 · 装好后点这里继续',
    preparing: '正在准备下载链接…',
    ccsHeading: '下载 CC Switch',
    ccsAsset: { 'win-x64': 'Windows（64 位）', 'win-arm64': 'Windows（ARM）', mac: 'macOS', 'linux-deb': 'Linux（.deb）' },
    ccsMacNote: '首次打开如果提示「无法验证开发者」，到「系统设置 → 隐私与安全性」点「仍要打开」。',
    ccsGithubNote: '该下载来自 GitHub，国内可能较慢；打不开可用下面的备用链接，或找客服。',
    ccsBackup: 'GitHub 发布页（全部版本）',
    codexHeading: '下载 Codex 桌面版',
    codexAsset: { 'win-x64': 'Windows（64 位）', 'win-arm64': 'Windows（ARM）', 'mac-arm64': 'macOS · Apple 芯片（M 系列）', 'mac-intel': 'macOS · Intel 芯片' },
    codexWinNote: '文件较大，约 1 GB，建议连 Wi-Fi。下载后双击安装。',
    codexBigNote: '文件较大，约 900 MB，建议连 Wi-Fi。下载后打开安装包，把 Codex 拖进「应用程序」。',
    macChipHint: '不确定芯片？点屏幕左上角苹果 →「关于本机」：显示 Apple M 系列选「Apple 芯片」，显示 Intel 选「Intel 芯片」。文件约 900 MB，建议连 Wi-Fi。',
    codexNoDesktop: 'Codex 桌面版目前只有 Windows 和 macOS 版，你的系统可以用命令行版：',
    codexCliHeading: '安装 Codex 命令行版',
    nodeNeeded: '需要先安装 Node.js 22 或更高版本。',
    verify: '校验文件完整性（SHA-256）',
    nodeHeading: '安装 Node.js（已装 22 或更高版本可跳过）',
    nodeNote: '用来运行下面的安装命令，装完需要重新打开终端。',
    nodeLinux: '请先用系统自带的包管理器安装 Node.js 22 或更高版本，已装可跳过。',
    nodeWin: 'Windows（64 位）',
    nodeWinArm: 'Windows（ARM）',
    nodeMac: 'macOS',
    claudeHeading: '安装 Claude Code',
    claudeCheck: '检查是否装好',
    claudeCheckHint: '能看到版本号就说明装好了。',
    shellWin: '在「PowerShell」或「命令提示符」里运行；如提示权限不足，用「以管理员身份运行」重开。',
    shellUnix: '在「终端」里运行；如提示权限不足，在命令前面加 sudo。',
    mobile: 'CC Switch 和 Codex / Claude Code 都要装在电脑上，请用电脑打开本站，继续这个引导。',
    siteUrl: '本站网址',
    copy: '复制',
    copied: '已复制',
    get: '下载',
    other: '其他版本',
    backupLabel: '备用'
  },
  importNav: {
    title: '回到密钥页',
    body: (label: string) => `点击左侧的「${label}」，找到刚创建的密钥。`
  },
  import: {
    title: '点「导入到 CC Switch」',
    body: '这是你刚创建的密钥那一行的按钮。',
    hint: '浏览器会问是否打开 CC Switch，选「打开」，再在 CC Switch 里确认导入。'
  },
  imported: {
    title: 'CC Switch 弹出确认窗口了吗？',
    body: '在 CC Switch 里确认导入，这个密钥就配置好了。',
    hint: '浏览器如果问「是否打开 CC Switch」，请选择打开。',
    yes: '已导入成功',
    no: '没反应',
    retry: '重新点一次'
  },
  manual: {
    title: '手动配置也很快',
    body: '先确认 CC Switch 已安装并能打开，再在里面添加一个自定义供应商，填入下面两项。',
    endpoint: 'API 地址',
    key: 'API Key',
    loading: '正在读取密钥…',
    done: '我配置好了',
    toDownload: '回到下载页',
    tutorial: '看图文教程'
  },
  finish: {
    title: '导入完成，最后一步',
    body: '启用刚导入的供应商，就可以开始用了。',
    stepsCodex: ['在 CC Switch 里启用刚导入的供应商', '打开 Codex 桌面版，新建一个对话', '随便问一句，能回复就接通了'],
    stepsClaude: ['在 CC Switch 里启用刚导入的供应商', '打开终端，运行 claude', '随便问一句，能回复就接通了'],
    waiting: '正在等待你的第一次调用…试一句，这里会自动提示',
    pill: '等待第一次调用 · 点这里查看',
    tutorial: '查看图文教程',
    done: '完成'
  },
  success: {
    title: '接通了！',
    body: '第一次调用成功。之后每次调用都能在「使用记录」里看到，余额消耗也会实时更新。',
    usage: '查看使用记录',
    done: '完成'
  },
  reopen: {
    title: '新手引导',
    desc: '重新走一遍：兑换额度、创建密钥、接入工具'
  },
  haveKey: '我已有密钥，跳过这步',
  importAny: '点你要用的那把密钥的「导入到 CC Switch」。',
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

export type GuideCopy = typeof guideCopyZh

export const guideCopyEn: GuideCopy = {
  stages: ['Pick a tool', 'Add credit', 'Create key', 'Connect'],
  stageDone: 'All done',
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
  ccs: {
    title: (tool: string) => `Are ${tool} and CC Switch installed?`,
    body: (tool: string) => `CC Switch configures your key into ${tool} in one click. You need both on your computer.`,
    yes: 'Both installed',
    no: 'Not yet, help me install'
  },
  download: {
    title: 'Install the software first',
    body: 'In order: download → install → come back and click Continue.',
    cont: "I've installed it, continue",
    collapse: 'Minimize',
    pill: 'Guide in progress · click to continue',
    preparing: 'Preparing download links…',
    ccsHeading: 'Download CC Switch',
    ccsAsset: { 'win-x64': 'Windows (64-bit)', 'win-arm64': 'Windows (ARM)', mac: 'macOS', 'linux-deb': 'Linux (.deb)' },
    ccsMacNote: 'If macOS says it cannot verify the developer, open System Settings → Privacy & Security and click Open Anyway.',
    ccsGithubNote: 'This download comes from GitHub and may be slow in some regions. Use the backup link below or ask support if it fails.',
    ccsBackup: 'GitHub releases (all versions)',
    codexHeading: 'Download the Codex desktop app',
    codexAsset: { 'win-x64': 'Windows (64-bit)', 'win-arm64': 'Windows (ARM)', 'mac-arm64': 'macOS · Apple silicon (M series)', 'mac-intel': 'macOS · Intel' },
    codexWinNote: 'Large file, about 1 GB — use Wi-Fi. Double-click to install after downloading.',
    codexBigNote: 'Large file, about 900 MB — use Wi-Fi. Open the installer and drag Codex into Applications.',
    macChipHint: 'Not sure which chip? Click the Apple menu → About This Mac: “Apple M…” means Apple silicon, “Intel” means Intel. About 900 MB — use Wi-Fi.',
    codexNoDesktop: 'The Codex desktop app is only available for Windows and macOS. On your system, use the command-line version:',
    codexCliHeading: 'Install the Codex CLI',
    nodeNeeded: 'Requires Node.js 22 or newer.',
    verify: 'Verify file integrity (SHA-256)',
    nodeHeading: 'Install Node.js (skip if you already have 22 or newer)',
    nodeNote: 'It runs the install command below. Reopen your terminal after installing.',
    nodeLinux: 'Install Node.js 22 or newer with your system package manager first (skip if you already have it).',
    nodeWin: 'Windows (64-bit)',
    nodeWinArm: 'Windows (ARM)',
    nodeMac: 'macOS',
    claudeHeading: 'Install Claude Code',
    claudeCheck: 'Check it worked',
    claudeCheckHint: 'If you see a version number, it is installed.',
    shellWin: 'Run it in PowerShell or Command Prompt; if you get a permissions error, reopen as administrator.',
    shellUnix: 'Run it in your terminal; if you get a permissions error, prefix the command with sudo.',
    mobile: 'CC Switch and Codex / Claude Code must be installed on a computer. Open this site on your computer to continue the guide.',
    siteUrl: 'Site address',
    copy: 'Copy',
    copied: 'Copied',
    get: 'Download',
    other: 'Other versions',
    backupLabel: 'Backup'
  },
  importNav: {
    title: 'Back to your keys',
    body: (label: string) => `Click “${label}” on the left and find the key you just created.`
  },
  import: {
    title: 'Click “Import to CC Switch”',
    body: 'It is the button on the row of the key you just created.',
    hint: 'Your browser will ask to open CC Switch — choose Open, then confirm the import inside CC Switch.'
  },
  imported: {
    title: 'Did CC Switch ask you to confirm?',
    body: 'Confirm the import in CC Switch and the key is configured.',
    hint: 'If the browser asks whether to open CC Switch, choose Open.',
    yes: 'Imported',
    no: 'Nothing happened',
    retry: 'Try again'
  },
  manual: {
    title: 'Manual setup is quick too',
    body: 'Make sure CC Switch is installed and opens, then add a custom provider in it with the two values below.',
    endpoint: 'API address',
    key: 'API key',
    loading: 'Loading your key…',
    done: "I've set it up",
    toDownload: 'Back to downloads',
    tutorial: 'Illustrated guide'
  },
  finish: {
    title: 'Imported — one last step',
    body: 'Enable the provider you just imported and you can start using it.',
    stepsCodex: ['Enable the imported provider in CC Switch', 'Open the Codex desktop app and start a chat', 'Ask anything — if it answers, you are connected'],
    stepsClaude: ['Enable the imported provider in CC Switch', 'Open a terminal and run claude', 'Ask anything — if it answers, you are connected'],
    waiting: 'Waiting for your first call… send one message and this will notice',
    pill: 'Waiting for first call · click to view',
    tutorial: 'Open the illustrated guide',
    done: 'Done'
  },
  success: {
    title: "You're connected!",
    body: 'Your first call went through. Every call shows up in Usage records, and your balance updates in real time.',
    usage: 'View usage records',
    done: 'Done'
  },
  reopen: {
    title: 'Getting started guide',
    desc: 'Walk through it again: add credit, create a key, connect your tool'
  },
  haveKey: 'I already have a key — skip',
  importAny: 'Click “Import to CC Switch” on the key you want to use.',
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
