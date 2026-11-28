import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GuideClient } from './config'

export function useGuideCopy() {
  const { locale } = useI18n()
  return computed(() => (locale.value.startsWith('zh') ? guideCopyZh : guideCopyEn))
}

export const guideCopyZh = {
  stages: ['选用法', '兑换额度', '创建密钥', '开始使用'],
  stageDone: '全部完成',
  clientName: (c: GuideClient) => ({ codex: 'Codex', 'claude-code': 'Claude Code', image: '生图', chat: '纯聊天', fortune: '玄学测算' })[c],

  welcome: {
    title: '几分钟，把 AI 用起来',
    body: '跟着引导走：兑换额度、创建密钥，然后接上编程工具、在网页里生图，或者直接聊天。每一步都有指引，随时可以跳过。',
    trialTitle: '1 元好友体验额度已到账',
    trialBody: '有效期 7 天、体验期限速 3 次/分钟、任意首充即解除，无需先充值，建好密钥就能试用。',
    start: '开始',
    skip: '跳过引导'
  },
  client: {
    title: '你想怎么用？',
    body: '选好后，后面的步骤会按你的选择来。',
    codex: 'OpenAI 的编程助手，有桌面版，装好就能用。',
    claude: 'Anthropic 的命令行编程助手。',
    image: '写一句话出图，也能装进 Codex / Claude Code。',
    chat: '不用装软件，直接在网页里和 AI 对话。',
    fortune: '八字、塔罗、星座、解梦，网页里直接问。'
  },
  imageMode: {
    title: '你想在哪里生图？',
    body: '两种都可以，之后随时能换。',
    web: '在网页上生图',
    webDesc: '什么都不用装，写一句话、点一下就出图。',
    agent: '在智能体里生图',
    agentDesc: '复制一段话发给 Codex / Claude Code，它自己装好 my-image。'
  },
  imageGo: {
    title: '密钥好了，去生图吧',
    body: '生图页会自动用上你刚建的密钥。写一句描述，或者点一个例子，再点「生成」。',
    open: '打开生图页'
  },
  skill: {
    title: '把这段话发给你的智能体',
    body: '复制后粘贴到 Codex 或 Claude Code 的对话框里发送。它会自己装好 my-image、配置好密钥，再画一张测试图。',
    heading: '复制这段话',
    note: '这段话里带着你的密钥，只发给自己电脑上的智能体。',
    loading: '正在读取密钥…',
    sent: '我发过去了',
    noAgent: '还没装 Codex / Claude Code？'
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
  keyName: {
    title: '给密钥起个名字',
    body: (client: string) => `随便起一个方便你认出来的名字，比如「我的${/^[A-Za-z0-9]/.test(client) ? ' ' : ''}${client || '密钥'}」。填好后按回车，或点「下一步」。`,
    next: '下一步'
  },
  keyForm: {
    title: '选择分组',
    body: (client: string) => `已按你选的 ${client} 选好类型。点开下拉框选一个分组，倍率越低越便宜。`,
    bodyChat: '点开下拉框选一个分组，倍率越低越便宜。',
    bodyImage: '点开下拉框，选一个支持生图的分组（名字里一般带「生图」或 image）。'
  },
  keySubmit: {
    title: '点「创建」',
    body: '分组选好了，点弹窗右下角的「创建」就完成了。',
    multi: '还可以在下面多勾几个分组，一把密钥用所有模型（可忽略）。'
  },
  keySteps: ['起名字', '选分组', '点创建'],
  chatGo: {
    title: '密钥好了，去聊天吧',
    body: '聊天页会自动用上你刚建的密钥，选个模型、发一句话就行。',
    bodyFortune: '已为你备好「玄学测算」：报上出生日期和时辰，或者点一个快捷问题就能开始。',
    open: '打开聊天页',
    modelsHeading: '推荐模型（已默认选好第一个）',
    modelsHeadingFortune: '测算推荐用 Claude Opus 5.5（已默认选好），想看得更深可以换 Fable 5.2',
    modelsHint: '在聊天页顶部的「模型」里随时切换。',
    defaultTag: '默认'
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
    title: (label: string) => `点「${label}」`,
    body: '这是你刚创建的密钥那一行的按钮。',
    hint: '浏览器会问是否打开 CC Switch，选「打开」，再在 CC Switch 弹出的窗口里点「导入」。'
  },
  imported: {
    title: 'CC Switch 弹出确认窗口了吗？',
    body: 'CC Switch 会弹出「确认导入供应商配置」，点「导入」，这个密钥就配置好了。',
    hint: '浏览器如果问「是否打开 CC Switch」，请选择打开。',
    yes: '已导入成功',
    no: '没反应',
    retry: '重新点一次'
  },
  manual: {
    title: '手动配置也很快',
    body: '先确认 CC Switch 已安装并能打开，点它右上角的「添加供应商」，选自定义，填入下面两项。',
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
    stepsCodex: ['在 CC Switch 左边选「Codex」，点刚导入的供应商右边的 ▶，变成「使用中」', '打开 Codex 桌面版，新建一个对话', '随便问一句，能回复就接通了'],
    stepsClaude: ['在 CC Switch 左边选「Claude Code」，点刚导入的供应商右边的 ▶，变成「使用中」', '打开终端，进入项目文件夹，运行 claude', '随便问一句，能回复就接通了'],
    waiting: '正在等待你的第一次调用…试一句，这里会自动提示',
    titleChat: '发一句话试试',
    bodyChat: '在聊天页发出第一条消息，收到回复就说明一切正常。',
    stepsChat: ['在顶部确认密钥和模型', '在输入框里随便说一句，按回车发送', '看到回复就接通了'],
    stepsFortune: ['在顶部确认密钥和模型', '点一个快捷问题，或者报上你的出生日期和时辰', '看到回复就接通了'],
    openChat: '打开聊天页',
    titleImage: '生成第一张图',
    bodyImage: '在生图页写一句描述，点「生成」。图出来，就说明一切正常。',
    stepsImage: ['写一句你想要的画面，或点一个例子', '选好比例，点「生成」', '等几十秒，图片出现就成功了'],
    openImage: '打开生图页',
    titleSkill: '等它画出测试图',
    bodySkill: '智能体装好、配置好之后会画一张橘猫测试图，图出来这里会自动提示。',
    stepsSkill: ['在 Codex / Claude Code 里发送刚复制的那段话', '它会自己克隆 my-image 并写好配置', '看到橘猫测试图就成功了，之后说「帮我画……」就行'],
    pill: '等待第一次调用 · 点这里查看',
    pillImage: '生成第一张图 · 点这里看步骤',
    tutorial: '查看图文教程',
    done: '完成'
  },
  success: {
    title: '接通了！',
    body: '第一次调用成功。之后每次调用都能在「使用记录」里看到，余额消耗也会实时更新。',
    affiliateHint: (rate?: string | null) =>
      rate
        ? `觉得顺手？去「邀请返利」领链接和扫码海报，好友领 1 元体验，你最高赚 8 元 + ${rate}% 返利。`
        : '觉得顺手？去「邀请返利」领链接和扫码海报，好友领 1 元体验，你最高赚 8 元 + 永久返利。',
    usage: '查看使用记录',
    done: '完成',
    titleImage: '第一张图出好了',
    bodyImage: '以后在「生图」页写一句话就能出图。每次的消耗都能在「使用记录」里看到。',
    keepGoing: '继续生图',
    titleSkill: '智能体会生图了',
    bodySkill: '以后跟 Codex / Claude Code 说「帮我画……」或「把这张图改成……」就行，图片会保存在项目的 outputs/my-image 文件夹里。'
  },
  reopen: {
    title: '新手引导',
    desc: '重新走一遍：兑换额度、创建密钥、接入工具、生图或开始聊天'
  },
  flyHome: {
    text: '忘了怎么操作？点头像，随时再来一遍',
    ok: '知道了'
  },
  haveKey: '我已有密钥，跳过这步',
  importAny: '点你要用的那把密钥的「导入到 CC Switch」。',
  done: { redeem: '兑换成功', key: '密钥已创建' },

  back: '上一步',
  backArrow: '返回上一步',
  tutorial: '图文教程',
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
    body: '几步就能接上 Codex / Claude Code、在网页里生图，或直接聊天。',
    start: '开始',
    no: '不用了'
  }
}

export type GuideCopy = typeof guideCopyZh

export const guideCopyEn: GuideCopy = {
  stages: ['Pick a use', 'Add credit', 'Create key', 'Get going'],
  stageDone: 'All done',
  clientName: (c: GuideClient) => ({ codex: 'Codex', 'claude-code': 'Claude Code', image: 'Images', chat: 'Chat', fortune: 'Fortune reading' })[c],

  welcome: {
    title: 'Put AI to work in minutes',
    body: 'Follow along: add credit, create a key, then connect a coding tool or just chat in the browser. Every step is guided, and you can skip any time.',
    trialTitle: '1.00 CNY friend trial credit is ready',
    trialBody: 'Valid for 7 days (3 requests/min during trial, lifted after any top-up). No recharge needed — create a key and try right away.',
    start: 'Start',
    skip: 'Skip guide'
  },
  client: {
    title: 'How do you want to use it?',
    body: 'The next steps follow your choice.',
    codex: "OpenAI's coding assistant, with a desktop app.",
    claude: "Anthropic's command-line coding assistant.",
    image: 'Describe it, get a picture — here or in Codex / Claude Code.',
    chat: 'Nothing to install — chat with AI right in your browser.',
    fortune: 'BaZi, tarot, astrology and dreams — ask in the browser.'
  },
  imageMode: {
    title: 'Where do you want to make images?',
    body: 'Either works, and you can switch later.',
    web: 'In the browser',
    webDesc: 'Nothing to install — write a sentence, click, get a picture.',
    agent: 'In your agent',
    agentDesc: 'Copy a message to Codex / Claude Code and it installs my-image itself.'
  },
  imageGo: {
    title: 'Key ready — let\'s make an image',
    body: 'The image page uses the key you just created. Write a description or tap an example, then click Generate.',
    open: 'Open the image page'
  },
  skill: {
    title: 'Send this message to your agent',
    body: 'Copy it, paste it into Codex or Claude Code and send. It installs my-image, sets up your key and draws a test image.',
    heading: 'Copy this message',
    note: 'The message contains your key. Only send it to an agent on your own computer.',
    loading: 'Loading your key…',
    sent: 'I sent it',
    noAgent: 'No Codex / Claude Code yet?'
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
  keyName: {
    title: 'Name your key',
    body: (client: string) => `Pick any name you will recognize, e.g. “My ${client || 'key'}”. Then press Enter or click Next.`,
    next: 'Next'
  },
  keyForm: {
    title: 'Pick a group',
    body: (client: string) => `We preselected the type for ${client}. Open the dropdown and pick a group (lower rate = cheaper).`,
    bodyChat: 'Open the dropdown and pick a group (lower rate = cheaper).',
    bodyImage: 'Open the dropdown and pick a group that supports images (usually named with “image”).'
  },
  keySubmit: {
    title: 'Click Create',
    body: 'Group picked. Click Create at the bottom right of the dialog and you are done.',
    multi: 'You can also tick more groups below so one key works with every model (optional).'
  },
  keySteps: ['Name it', 'Pick a group', 'Create'],
  chatGo: {
    title: 'Key ready — let\'s chat',
    body: 'The chat page uses the key you just created. Pick a model and send a message.',
    bodyFortune: 'A fortune-reading chat is ready: share your birth date and time, or tap a quick question to begin.',
    open: 'Open chat',
    modelsHeading: 'Recommended models (the first is preselected)',
    modelsHeadingFortune: 'Readings work best with Claude Opus 5.5 (preselected); switch to Fable 5.2 for deeper answers',
    modelsHint: 'Switch any time from Model at the top of the chat page.',
    defaultTag: 'Default'
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
    title: (label: string) => `Click “${label}”`,
    body: 'It is the button on the row of the key you just created.',
    hint: 'Your browser will ask to open CC Switch — choose Open, then click “Import” in the window CC Switch shows.'
  },
  imported: {
    title: 'Did CC Switch ask you to confirm?',
    body: 'CC Switch shows a “Confirm import” window — click “Import” and the key is configured.',
    hint: 'If the browser asks whether to open CC Switch, choose Open.',
    yes: 'Imported',
    no: 'Nothing happened',
    retry: 'Try again'
  },
  manual: {
    title: 'Manual setup is quick too',
    body: 'Make sure CC Switch is installed and opens, click “Add provider” at its top right, choose custom, and fill in the two values below.',
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
    stepsCodex: ['In CC Switch pick “Codex” on the left and click ▶ next to the imported provider until it says “In use”', 'Open the Codex desktop app and start a chat', 'Ask anything — if it answers, you are connected'],
    stepsClaude: ['In CC Switch pick “Claude Code” on the left and click ▶ next to the imported provider until it says “In use”', 'Open a terminal, go to your project folder and run claude', 'Ask anything — if it answers, you are connected'],
    waiting: 'Waiting for your first call… send one message and this will notice',
    titleChat: 'Send a message',
    bodyChat: 'Send your first message on the chat page. A reply means everything works.',
    stepsChat: ['Check the key and model at the top', 'Type anything and press Enter', 'If it answers, you are connected'],
    stepsFortune: ['Check the key and model at the top', 'Tap a quick question, or share your birth date and time', 'If it answers, you are connected'],
    openChat: 'Open chat',
    titleImage: 'Make your first image',
    bodyImage: 'Write a description on the image page and click Generate. When a picture shows up, everything works.',
    stepsImage: ['Describe what you want, or tap an example', 'Pick a shape and click Generate', 'Wait under a minute — the picture means you are done'],
    openImage: 'Open the image page',
    titleSkill: 'Wait for its test image',
    bodySkill: 'Once your agent has installed and set up my-image it draws a cat as a test. This card notices when it does.',
    stepsSkill: ['Send the message you copied in Codex / Claude Code', 'It clones my-image and writes the config itself', 'Seeing the cat means it works — after that just ask it to draw'],
    pill: 'Waiting for first call · click to view',
    pillImage: 'Make your first image · click for steps',
    tutorial: 'Open the illustrated guide',
    done: 'Done'
  },
  success: {
    title: "You're connected!",
    body: 'Your first call went through. Every call shows up in Usage records, and your balance updates in real time.',
    affiliateHint: (rate?: string | null) =>
      rate
        ? `Enjoying it? Grab your invite link & QR poster on Affiliate Rebates — friends get 1 CNY to try, and you earn up to 8 CNY + ${rate}% rebate.`
        : 'Enjoying it? Grab your invite link & QR poster on Affiliate Rebates — friends get 1 CNY to try, and you earn up to 8 CNY + lifetime rebate.',
    usage: 'View usage records',
    done: 'Done',
    titleImage: 'Your first image is ready',
    bodyImage: 'From now on, write a sentence on the Images page to get a picture. Every charge shows up in Usage records.',
    keepGoing: 'Keep making images',
    titleSkill: 'Your agent can make images now',
    bodySkill: 'Just ask Codex / Claude Code to draw something or edit an image. Pictures are saved in your project’s outputs/my-image folder.'
  },
  reopen: {
    title: 'Getting started guide',
    desc: 'Walk through it again: add credit, create a key, connect a tool, make images or start chatting'
  },
  flyHome: {
    text: 'Forgot a step? Click your avatar to run the guide again',
    ok: 'Got it'
  },
  haveKey: 'I already have a key — skip',
  importAny: 'Click “Import to CC Switch” on the key you want to use.',
  done: { redeem: 'Redeemed', key: 'Key created' },

  back: 'Back',
  backArrow: 'Back to previous step',
  tutorial: 'Step-by-step guide',
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
    body: 'A few steps to connect Codex or Claude Code, make images, or just chat in the browser.',
    start: 'Start',
    no: 'No thanks'
  }
}
