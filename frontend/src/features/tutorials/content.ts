import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

export type TutorialClient = 'claude-code' | 'codex'
export const clientName = (client: TutorialClient) => client === 'codex' ? 'Codex' : 'Claude Code'
export const tutorialClients: TutorialClient[] = ['claude-code', 'codex']

export function useTutorialCopy() {
  const { locale } = useI18n()
  return computed(() => locale.value.startsWith('zh') ? zh : en)
}

const zh = {
  nav: '接入教程', home: '返回首页', title: '从这里，开始第一轮对话。',
  intro: '充值、创建密钥、导入 CC Switch。跟着图解，把你常用的编程工具接上来。',
  entryTitle: '第一次接入？跟着做就好。', entryDesc: '从充值到第一轮对话，两份完整图文教程。',
  read: '查看图文教程', claudeDesc: '用 Claude 理解代码、编写功能、解决问题。', codexDesc: '让 Codex 进入你的项目，开始协作。',
  guideSuffix: '接入指南', guideIntro: '通过 CC Switch 导入本站配置，一步步完成准备、接入与验证。',
  switchClient: '切换教程', contents: '本页步骤', prereq: '开始前',
  prereqText: '准备一个本站账号和一台电脑。CC Switch 负责管理连接配置，Claude Code / Codex 是实际使用的命令行工具，需要分别安装。本教程以 CLI 为例。',
  steps: ['充值到账', '创建 API Key', '安装工具', '一键导入', '启用与启动', '验证接入'],
  figure: '操作示意', figureNote: '示意图仅展示操作位置；名称、金额和可用分组以你的页面为准。',
  actionRecharge: '前往充值', actionKeys: '前往 API 密钥', actionUsage: '查看调用记录', actionModels: '查看可用渠道',
  register: '注册账号', loginHint: '还没登录？可以先看完整教程，操作时再登录。',
  rechargeTitle: '先让账户有可用额度',
  rechargeBody: ['登录控制台，进入「充值 / 购买」，选择余额充值，填写金额并选择页面支持的支付方式。', '核对实际支付金额和到账额度，创建订单后按页面提示完成支付。回到控制台，确认订单显示支付成功、余额已更新。', '如果使用兑换码，进入「兑换」完成兑换。如果选择订阅套餐，确认套餐已生效，并记住对应分组，下一步创建密钥时要选它。'],
  rechargeNote: '到账额度、汇率、最低充值金额和套餐范围均以结算页面为准。支付后未到账，先在「我的订单」检查状态，避免重复付款。',
  keyTitle: '为你的工具创建一把密钥',
  keyBody: ['进入「API 密钥」，点击「创建密钥」，填写一个容易识别的名称。', '选择下方所示的供应商类型，再选择支持目标模型、且账户有使用权限的分组。订阅用户选择套餐对应分组；余额用户选择可用的按量分组。', '按需设置额度和有效期，点击创建。确认密钥状态为启用，并保留这条记录，稍后直接从这里导入。两种工具建议分别建 Key，便于管理。'],
  claudeGroup: 'Claude Code：选择 Claude / Anthropic 类型分组。', codexGroup: 'Codex：选择 OpenAI 类型分组。',
  keyNote: 'API Key 相当于访问凭证，请勿公开、截图分享或提交到代码仓库。泄露后立即禁用并重新创建。',
  installTitle: '安装 CC Switch 和命令行工具',
  installBody: ['从 CC Switch 官方发布页下载与你电脑系统、芯片架构匹配的安装包，安装后打开一次。macOS 可下载 .dmg，Windows 可选 .msi，Linux 按发行版选择 .deb / .rpm / AppImage。', '安装下方对应的命令行工具。npm 方式需要先安装 Node.js 22 或以上的 LTS 版本；如果没有 npm，可打开官方安装文档选择其他方式。安装结束后新开终端，运行版本检查。', '如果工具运行在 WSL 或另一台电脑中，CC Switch 必须管理同一环境的配置文件；Windows 原生配置不会自动写入 WSL。首次接入建议在同一台电脑的原生终端完成。'],
  downloads: '下载 CC Switch', officialInstall: '命令行工具安装文档', nodeDownload: '安装 Node.js LTS（22+）', versionCheck: '版本检查',
  importTitle: '回到密钥页，一键导入',
  importBody: ['在本站「API 密钥」页面找到刚创建的密钥，点击该行的「导入到 CC Switch」。', '浏览器提示打开外部应用时，允许打开 CC Switch。在导入确认窗口核对工具类型、供应商名称和 API 地址，再确认导入。', '工具类型由密钥所属分组决定。若导入到了错误的工具页，回到本站重新创建正确类型分组的密钥，再导入。'],
  importHidden: '如果页面没有导入按钮，可能是站点关闭了该入口，可按下方参数在 CC Switch 中添加自定义供应商。',
  endpoint: 'API 地址', apiKey: 'API Key', keyPlaceholder: '填写你在本站创建的完整 Key',
  endpointNote: '优先使用一键导入生成的地址。手动添加时，Claude 填 API 根地址；Codex 的 base_url 通常需要 /v1，以 CC Switch 最终配置为准，避免重复 /v1/v1。',
  activateTitle: '启用供应商，启动新会话',
  activateBody: ['打开 CC Switch，切到对应工具页，找到刚导入的本站供应商并点击「启用」。确认它显示为当前使用的供应商。', '关闭原先运行的 CLI 会话，新开终端，进入你要使用的项目目录，再执行下方启动命令。首次启动时阅读提示，只信任你确认安全的工作目录。', '如果仍然跳到官方账号登录，先检查是否启用了正确供应商，再检查终端中是否残留旧的 API 地址或密钥环境变量。不要用本站 Key 登录 Claude / ChatGPT 官方网页。'],
  activateNote: '导入只会添加配置，仍需点击「启用」。Codex 切换后需重启 CLI；Claude Code 也建议用新会话验证。',
  verifyTitle: '收到回复，再核对一次调用记录',
  verifyBody: ['在工具中发送下面这条测试消息。收到正常回答，说明工具已经可以交互。', '回到本站「使用记录」，检查是否出现这把密钥的新请求，核对时间、模型和费用。记录可能稍有延迟，可手动刷新。', '工具有回复，且本站有对应的成功记录，才算完整接入。之后就可以在项目里开始日常使用。'],
  prompt: '请只回复：连接成功。', success: '接入完成', successDesc: '新会话有回复，本站有对应的成功调用记录。',
  faqTitle: '卡住了？从这里排查。',
  faqs: [
    { q: '点击导入，没有打开 CC Switch', a: '先确认已安装并打开过 CC Switch，然后允许浏览器打开外部应用。仍然无反应时，重新安装以注册 ccswitch:// 协议，或在 CC Switch 中手动添加供应商。不要把含密钥的导入链接发给别人。' },
    { q: '提示 401 / API Key 无效', a: '检查密钥是否完整、启用且未过期，并确认当前工具使用的是刚导入的供应商。手动输入时不要带多余空格；必要时在本站重新创建密钥并导入。' },
    { q: '提示 403、没有可用分组或余额不足', a: '检查账户余额、密钥额度、订阅有效期及分组权限。Claude Code 需要 Claude 分组，Codex 需要 OpenAI 分组。没有合适分组时联系站点支持。' },
    { q: '模型不存在、404、429 或上游繁忙', a: '模型名称以所选渠道支持列表为准，不要随意填写。404 时检查地址路径，尤其是重复 /v1；429 或上游繁忙时等待后重试，并查看渠道状态。' },
    { q: '命令找不到，或仍在使用旧配置', a: '新开终端运行版本检查；命令找不到时检查安装和 PATH。确认 CC Switch 与 CLI 处于同一系统环境，排查旧环境变量、项目内覆盖配置及其他代理工具。' },
  ],
  copy: '复制', copied: '已复制', copyFailed: '复制失败，请选中文本手动复制',
  nextGuide: '也要接入另一种工具？', start: '开始接入',
  welcomeTitle: '欢迎加入，先接上你的工具。', welcomeDesc: '我们准备了完整图文教程，从充值、创建密钥，到用 CC Switch 导入和开始使用。',
  later: '稍后再看', welcomeFoot: '之后随时可以从首页「接入教程」找到。',
  ui: { recharge: '余额充值', amount: '充值金额', pay: '创建订单并支付', paid: '确认余额已到账', keys: 'API 密钥', create: '创建密钥', provider: '供应商类型', group: '选择可用分组', keyName: '密钥名称', import: '导入到 CC Switch', allow: '允许打开 CC Switch', confirm: '确认导入', enable: '启用', enabled: '当前使用', reply: '连接成功。', record: '使用记录', success: '成功', install: '安装完成后检查版本' },
}

// Kept in the same shape so both language versions exercise the same interface.
const en: typeof zh = {
  nav: 'Setup guides', home: 'Back to home', title: 'Start your first conversation.',
  intro: 'Add credit, create a key, and import with CC Switch. Follow the illustrated steps to connect your coding tool.',
  entryTitle: 'Your first connection, step by step.', entryDesc: 'Two illustrated guides, from credit to your first reply.',
  read: 'Read the guide', claudeDesc: 'Understand code, build features, and solve problems with Claude.', codexDesc: 'Bring Codex into your project and start working together.',
  guideSuffix: 'setup guide', guideIntro: 'Import this site’s configuration through CC Switch, then verify your first request.',
  switchClient: 'Choose a guide', contents: 'On this page', prereq: 'Before you start',
  prereqText: 'You need an account on this site and a computer. CC Switch manages connection settings; Claude Code and Codex are separate CLI tools that you also need to install. This guide covers the CLI.',
  steps: ['Add credit', 'Create an API key', 'Install the tools', 'Import settings', 'Enable and launch', 'Verify the connection'],
  figure: 'Illustration', figureNote: 'These diagrams show where to act. Names, amounts, and groups depend on your account.',
  actionRecharge: 'Add credit', actionKeys: 'Open API keys', actionUsage: 'View usage records', actionModels: 'View available channels', register: 'Create an account', loginHint: 'Not signed in? Read the full guide now and sign in when you are ready.',
  rechargeTitle: 'Make sure your account has credit',
  rechargeBody: ['Sign in, open the purchase page, choose balance recharge, enter an amount, and select an available payment method.', 'Review the payment and credited amounts. Create the order, complete payment, then confirm the order is paid and your balance has updated.', 'If you have a redeem code, use the Redeem page. If you purchase a subscription, confirm it is active and note its group for the next step.'],
  rechargeNote: 'Amounts, exchange rates, minimum payments, and plan coverage follow the checkout page. If credit is missing after payment, check My Orders before paying again.',
  keyTitle: 'Create a key for your coding tool',
  keyBody: ['Open API Keys, select Create Key, and give it a recognizable name.', 'Choose the provider type below, then a group that supports your model and is available to your account. Subscription users should select the group linked to their plan; balance users should select an available pay-as-you-go group.', 'Set an optional quota and expiry, then create the key. Check that it is enabled. Keep this record for the import step. Separate keys make the two tools easier to manage.'],
  claudeGroup: 'Claude Code: choose a Claude / Anthropic group.', codexGroup: 'Codex: choose an OpenAI group.',
  keyNote: 'Your API key grants access to your account. Keep it out of shared screenshots and source control. Disable and replace it immediately if exposed.',
  installTitle: 'Install CC Switch and the CLI',
  installBody: ['Download CC Switch from its official release page for your operating system and CPU. Install it and open it once. Choose .dmg for macOS, .msi for Windows, or the appropriate Linux package.', 'Install the matching CLI using the command below. The npm method requires Node.js LTS (22 or later). If npm is unavailable, see the official installation guide for alternatives. Open a new terminal and check the version.', 'If your CLI runs in WSL or on another computer, CC Switch must manage that same environment’s configuration. Windows settings are not automatically written into WSL. For the first setup, use the native terminal on the same computer.'],
  downloads: 'Download CC Switch', officialInstall: 'CLI installation guide', nodeDownload: 'Install Node.js LTS (22+)', versionCheck: 'Check the version',
  importTitle: 'Import directly from the key page',
  importBody: ['Find the key you created on the API Keys page and click Import to CC Switch in its row.', 'Allow your browser to open CC Switch. Review the tool type, provider name, and API address in the import dialog, then confirm.', 'The key’s group determines the target tool. If it imports into the wrong tool, create a key in the correct provider group and import again.'],
  importHidden: 'If the import button is hidden by the site, add a custom provider in CC Switch with the values below.', endpoint: 'API address', apiKey: 'API key', keyPlaceholder: 'Your complete API key from this site',
  endpointNote: 'Prefer the address generated by one-click import. For manual setup, Claude uses the API root; Codex base_url usually needs /v1. Check the final CC Switch configuration and avoid /v1/v1.',
  activateTitle: 'Enable the provider and start a new session',
  activateBody: ['Open the matching tool page in CC Switch, find this site’s provider, and click Enable. Confirm it is the active provider.', 'Close existing CLI sessions. Open a new terminal, enter your project directory, and run the launch command below. Read the first-run prompts and only trust a directory you recognize.', 'If the CLI still requests an official account login, check the active provider and look for old API address or key environment variables. This site’s key is not a login for the official Claude or ChatGPT website.'],
  activateNote: 'Importing adds a provider; you still need to enable it. Restart Codex after switching. A fresh Claude Code session also makes verification clearer.',
  verifyTitle: 'Check both the reply and the usage record',
  verifyBody: ['Send the test message below in your tool and wait for a normal reply.', 'Open Usage Records on this site. Look for a new request from this key and check its time, model, and cost. Records may take a moment to appear; refresh if needed.', 'A reply in the tool and a matching successful request on this site confirm the connection. You can now start working in your project.'],
  prompt: 'Reply only with: Connection successful.', success: 'You are connected', successDesc: 'Your new session responds and this site shows the matching successful request.',
  faqTitle: 'Need a hand?',
  faqs: [
    { q: 'CC Switch does not open after importing', a: 'Install and open CC Switch first, then allow external apps in your browser. Reinstall to register the ccswitch:// handler, or add a custom provider manually. Never share an import link containing a key.' },
    { q: '401 or invalid API key', a: 'Check that the key is complete, active, and not expired. Make sure the newly imported provider is selected. Remove extra spaces or create and import a replacement key.' },
    { q: '403, missing groups, or insufficient balance', a: 'Check your balance, key quota, subscription expiry, and group permissions. Claude Code needs a Claude group; Codex needs an OpenAI group. Contact site support if no suitable group is available.' },
    { q: 'Unknown model, 404, 429, or busy upstream', a: 'Use a model supported by the selected channel. For 404, check the address, especially duplicate /v1 paths. For 429 or upstream overload, wait and retry, and check channel status.' },
    { q: 'Command not found or old settings still in use', a: 'Open a new terminal and check the version. Check installation and PATH if the command is missing. Ensure CC Switch and the CLI use the same environment, and inspect old environment variables, project overrides, and other proxy tools.' },
  ],
  copy: 'Copy', copied: 'Copied', copyFailed: 'Copy failed. Select and copy the text manually.', nextGuide: 'Connecting the other tool too?', start: 'Start setup',
  welcomeTitle: 'Welcome. Let’s connect your tool.', welcomeDesc: 'Follow the illustrated guide from adding credit and creating a key to importing with CC Switch and getting your first reply.', later: 'Maybe later', welcomeFoot: 'Find these guides anytime under Setup guides on the home page.',
  ui: { recharge: 'Add credit', amount: 'Amount', pay: 'Create order and pay', paid: 'Confirm credit received', keys: 'API keys', create: 'Create key', provider: 'Provider type', group: 'Choose an available group', keyName: 'Key name', import: 'Import to CC Switch', allow: 'Allow opening CC Switch', confirm: 'Confirm import', enable: 'Enable', enabled: 'Active provider', reply: 'Connection successful.', record: 'Usage records', success: 'Success', install: 'Check the installed version' },
}
