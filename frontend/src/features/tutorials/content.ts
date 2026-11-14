import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TutorialClient } from './tutorialSteps'

export type { TutorialClient } from './tutorialSteps'
export const clientName = (client: TutorialClient) => client === 'codex' ? 'Codex' : 'Claude Code'
export const tutorialClients: TutorialClient[] = ['claude-code', 'codex']

export function useTutorialCopy() {
  const { locale } = useI18n()
  return computed(() => locale.value.startsWith('zh') ? zh : en)
}

/** 页面框架、目录、常见问题等文案；每一步的具体内容在 tutorialSteps.ts。 */
const zh = {
  nav: '接入教程', home: '返回首页', title: '从这里，开始第一轮对话。',
  intro: '买兑换码、创建密钥、装好软件、一键导入。照着真实界面的截图，一步一步把你的编程工具接上来。',
  entryTitle: '第一次接入？跟着做就好。', entryDesc: '从买兑换码到第一轮对话，两份带真实截图的教程。',
  read: '查看图文教程', claudeDesc: '用 Claude 理解代码、编写功能、解决问题。', codexDesc: '让 Codex 进入你的项目，开始协作。',
  guideSuffix: '接入指南', guideIntro: '大约 10～15 分钟。每一步都配了真实界面的截图，照着点就行。',
  switchClient: '切换教程', contents: '本页步骤', prereq: '开始前',
  prereqText: '你需要：一台电脑（Windows 或 macOS）和一个本站账号。整个过程会用到三样东西：本站（买额度、创建密钥）、CC Switch（把密钥一键配置进工具的小软件）、以及你要用的编程工具。',
  steps: ['去小铺买兑换码', '兑换成余额', '创建 API 密钥', '安装软件', '一键导入', '启用并开始使用', '发一句话验证'],
  actionKeys: '前往 API 密钥', register: '注册账号',
  guideCta: '想让网站一步步带你做？', guideCtaAction: '开始站内引导',
  figureNote: '截图来自真实界面；名称、金额和可用分组以你的页面为准。',
  detailSteps: '手动添加',
  terminalSummary: '怎么打开终端？',
  terminalLines: ['Windows：按键盘上的 Win 键，输入 PowerShell，按回车。', 'macOS：按 ⌘ + 空格，输入「终端」，按回车。', '打开后，把要运行的命令复制进去（Windows 在窗口里点右键即可粘贴，macOS 按 ⌘ + V），再按回车。'],
  success: '接入完成', successDesc: '工具有回复，本站也有对应的调用记录。之后就可以在项目里日常使用了。',
  faqTitle: '卡住了？从这里排查。',
  faqs: [
    { q: '付款了，但没拿到兑换码', a: '先回到小铺，点右上角「订单查询」，输入下单时填的联系方式，就能看到订单和兑换码。还是找不到，加 QQ 群联系管理员，不要重复付款。' },
    { q: '兑换时提示兑换码无效', a: '检查有没有多复制空格、漏了字符，兑换码区分大小写。一个兑换码只能用一次，如果已经兑换过，余额会直接增加，不用再兑换。' },
    { q: '点了「导入到 CCS」没反应，或找不到这个按钮', a: '先确认 CC Switch 已经装好并且打开过一次，浏览器问是否打开时选「打开」。还是不行，就用第 5 步里的「手动添加」，自己填地址和密钥。不要把含密钥的导入链接发给别人。' },
    { q: '提示 401 或「API Key 无效」', a: '检查密钥有没有复制完整、前后有没有空格，并确认 CC Switch 里启用的是刚导入的那个供应商。必要时在本站重新创建一把密钥再导入。' },
    { q: '提示 403、没有可用分组，或余额不足', a: '先看本站余额是否还有；再看创建密钥时选的厂商和分组是否对：Claude Code 要选 Anthropic，Codex 要选 OpenAI。没有合适的分组，加群联系管理员。' },
    { q: '提示模型不存在、404、429，或「上游繁忙」', a: '模型名称以所选分组支持的为准，不要自己乱填。429 或「上游繁忙」是当前用的人多，等一会儿再试，或换一个分组。' },
    { q: '命令找不到，或还在用旧配置、还让登录官方账号', a: '新开一个终端再试；命令找不到说明没装好，回到第 4 步。确认 CC Switch 里那个供应商显示「使用中」，然后把工具关掉重开。不要用本站的密钥去登录官方网页。' },
  ],
  copy: '复制', copied: '已复制', copyFailed: '复制失败，请选中文本手动复制',
  nextGuide: '也要接入另一种工具？',
}

// Kept in the same shape so both language versions exercise the same interface.
const en: typeof zh = {
  nav: 'Setup guides', home: 'Back to home', title: 'Start your first conversation.',
  intro: 'Get a redeem code, create a key, install the software, import in one click. Follow the real screenshots to connect your coding tool.',
  entryTitle: 'Your first connection, step by step.', entryDesc: 'Two guides with real screenshots, from redeem code to first reply.',
  read: 'Read the guide', claudeDesc: 'Understand code, build features, and solve problems with Claude.', codexDesc: 'Bring Codex into your project and start working together.',
  guideSuffix: 'setup guide', guideIntro: 'About 10–15 minutes. Every step has a screenshot of the real interface — just follow along.',
  switchClient: 'Choose a guide', contents: 'On this page', prereq: 'Before you start',
  prereqText: 'You need a computer (Windows or macOS) and an account on this site. You will use three things: this site (buy credit, create a key), CC Switch (a small app that puts the key into your tool in one click), and the coding tool itself.',
  steps: ['Get a redeem code', 'Redeem it', 'Create an API key', 'Install the software', 'Import in one click', 'Enable and start', 'Send a test message'],
  actionKeys: 'Open API keys', register: 'Create an account',
  guideCta: 'Want the site to walk you through it?', guideCtaAction: 'Start the in-app guide',
  figureNote: 'Screenshots are from the real interface; names, amounts and groups depend on your account.',
  detailSteps: 'Manual setup',
  terminalSummary: 'How do I open a terminal?',
  terminalLines: ['Windows: press the Win key, type PowerShell, press Enter.', 'macOS: press ⌘ + Space, type “Terminal”, press Enter.', 'Then paste the command you want to run (on Windows, right-click in the window; on macOS press ⌘ + V) and press Enter.'],
  success: 'You are connected', successDesc: 'Your tool replies and this site shows the matching call. You can now use it in your projects.',
  faqTitle: 'Need a hand?',
  faqs: [
    { q: 'I paid but did not get a redeem code', a: 'Go back to the shop, click “Order lookup” at the top right, and enter the contact you used when ordering — your order and code are there. Still nothing? Ask an admin in the QQ group and do not pay again.' },
    { q: 'Redeeming says the code is invalid', a: 'Check for extra spaces or missing characters; codes are case-sensitive. A code works once — if you already redeemed it, your balance went up and you do not need to do it again.' },
    { q: 'Nothing happens when I click “Import to CCS”, or I cannot find the button', a: 'Make sure CC Switch is installed and has been opened once, and choose Open when the browser asks. If it still fails, use “Manual setup” in step 5 and fill in the address and key yourself. Never share an import link that contains a key.' },
    { q: '401 or “invalid API key”', a: 'Check that the key is complete with no extra spaces, and that CC Switch is using the provider you just imported. If needed, create a new key on this site and import it again.' },
    { q: '403, no available group, or insufficient balance', a: 'Check your balance first, then that you chose the right provider and group when creating the key: Anthropic for Claude Code, OpenAI for Codex. If no suitable group exists, ask an admin in the group.' },
    { q: 'Unknown model, 404, 429, or “upstream busy”', a: 'Use a model your group supports and do not make names up. 429 or “upstream busy” means it is busy right now — wait a bit, or try another group.' },
    { q: 'Command not found, old settings, or it still asks for an official login', a: 'Open a fresh terminal and try again; “command not found” means it is not installed — go back to step 4. Make sure the provider shows “In use” in CC Switch, then fully restart the tool. Never use this site’s key to sign in on an official website.' },
  ],
  copy: 'Copy', copied: 'Copied', copyFailed: 'Copy failed. Select and copy the text manually.',
  nextGuide: 'Connecting the other tool too?',
}
