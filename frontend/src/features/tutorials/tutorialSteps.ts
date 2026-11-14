/**
 * 图文教程的内容：每一步拆成「一个动作一行」，配真实截图。
 * 界面文字（按钮名、菜单名）由 UiLabels 传入，不在这里写死。
 */
import type { UiLabels } from './uiLabels'

export type TutorialClient = 'claude-code' | 'codex'

export interface ShotRef {
  /** shots.json 里的截图 id；没有对应截图时这一块自动不显示。 */
  id: string
  /** 带编号标注时，每个编号对应的说明（按编号顺序）。 */
  legend?: string[]
  /** 只有一个标注点时，用一句话说明图里要看哪里。 */
  caption?: string
}

export interface TutorialAction {
  text: string
  /** 紧跟在这一行下面的补充说明（灰字）。 */
  hint?: string
  shot?: ShotRef
  code?: { label: string; text: string }
  /** 渲染和引导里同一套的「下载」块（按系统自动识别）。 */
  downloads?: boolean
  /** 展开式的「怎么打开终端」。 */
  terminal?: boolean
  link?: { to: string; label: string }
}

export interface TutorialDetail {
  summary: string
  lines: string[]
  fields?: { label: string; value: string }[]
}

export interface TutorialStep {
  id: string
  title: string
  lead: string
  actions: TutorialAction[]
  note?: string
  detail?: TutorialDetail
}

export interface StepContext {
  lang: 'zh' | 'en'
  client: TutorialClient
  ui: UiLabels
  siteName: string
  /** 手动配置时填的 API 地址（Codex 已带 /v1）。 */
  endpoint: string
  qqGroup: string
  plaza: boolean
}

const TOOL: Record<TutorialClient, string> = { 'claude-code': 'Claude Code', codex: 'Codex' }
const PROVIDER: Record<TutorialClient, string> = { 'claude-code': 'Anthropic', codex: 'OpenAI' }
export const TEST_PROMPT = { zh: '请只回复：连接成功。', en: 'Reply only with: Connection successful.' }

export function buildSteps(c: StepContext): TutorialStep[] {
  return c.lang === 'zh' ? zh(c) : en(c)
}

function zh(c: StepContext): TutorialStep[] {
  const { ui: u, siteName: site, qqGroup } = c
  const tool = TOOL[c.client]
  const provider = PROVIDER[c.client]
  const isCodex = c.client === 'codex'
  return [
    {
      id: 'shop',
      title: '去小铺买兑换码',
      lead: '本站通过「小铺」售卖额度。付款后你会拿到一个兑换码，下一步用它给账户充值。',
      actions: [
        { text: `登录本站，点击左侧菜单里的「${u.buy}」。小铺会在新标签页打开。`, shot: { id: 'sidebar-buy', caption: `左侧菜单最下面的「${u.buy}」` } },
        {
          text: '在小铺里选「额度卡」分类，再点一个面额。',
          hint: '面额里的「刀」就是美元：10 刀 = 10 美元额度。',
          shot: { id: 'shop-list', legend: ['先选「额度卡」这个分类', '再点一个面额'] }
        },
        {
          text: '在「订单确认」里填联系方式，选支付宝或微信，点「去支付」，按页面提示付款。',
          hint: '联系方式用来以后查订单，请填你常用的。价格里含少量支付手续费。',
          shot: { id: 'shop-order', legend: ['联系方式（必填）', '选支付方式', '点「去支付」'] }
        },
        {
          text: '付款成功后，页面会显示一串兑换码（也叫卡密），完整复制下来。',
          hint: '不小心关了页面？回到小铺，点右上角「订单查询」，输入刚才填的联系方式就能找回。',
          shot: { id: 'shop-code', legend: ['这一串就是兑换码（卡密）', '点「复制」，或点上面的「一键复制」'] }
        }
      ],
      note: `付款后没拿到兑换码，请加 QQ 群 ${qqGroup} 找管理员，不要重复付款。`
    },
    {
      id: 'redeem',
      title: '兑换成余额',
      lead: '回到本站，把兑换码换成可用额度。',
      actions: [
        { text: `点击左侧菜单的「${u.redeem}」。`, shot: { id: 'sidebar-redeem', caption: `左侧菜单里的「${u.redeem}」` } },
        {
          text: `把兑换码粘贴进输入框，点「${u.redeemBtn}」。`,
          hint: '图中是演示用的兑换码，你的会不一样。',
          shot: { id: 'redeem-form', legend: ['粘贴兑换码', `点「${u.redeemBtn}」`] }
        },
        {
          text: '看到「兑换成功」，页面上的余额变多了，就可以进入下一步。',
          shot: { id: 'redeem-ok', caption: '兑换成功后，余额和提示都会更新' }
        }
      ],
      note: '兑换码区分大小写，复制时别带上前后的空格。'
    },
    {
      id: 'key',
      title: '创建 API 密钥',
      lead: '密钥是你的工具连接本站的凭证，相当于一张「通行证」。',
      actions: [
        { text: `点击左侧菜单的「${u.apiKeys}」，再点右上角的「${u.createKey}」。`, shot: { id: 'keys-create', caption: `右上角的「${u.createKey}」` } },
        {
          text: '在弹出的窗口里这样填：',
          shot: {
            id: `keys-form-${c.client}`,
            legend: [
              `${u.name}：随便起个名字，比如 ${tool}`,
              `${u.provider}：${tool} 选「${provider}」`,
              `${u.group}：选一个分组。倍率越低越便宜，不确定就选倍率最低的`,
              `点「${u.create}」`
            ]
          },
          link: c.plaza ? { to: '/plaza', label: '不知道分组怎么选？去「广场」看各分组的价格' } : undefined
        },
        { text: '创建成功后，列表里会出现这把密钥，先别关页面，下一步马上用到。' }
      ],
      note: '密钥相当于账户的钥匙：不要发给别人，也不要截图公开。万一泄露了，立刻禁用它再重新创建一把。'
    },
    {
      id: 'install',
      title: `安装 CC Switch 和 ${tool}`,
      lead: `这两个软件都装在你自己的电脑上（手机不行）。CC Switch 是个小工具，负责把密钥一键配置进 ${tool}，省得手动改配置文件。`,
      actions: [
        { text: '按下面的步骤下载并安装（会按你的系统自动挑选合适的版本）：', downloads: true },
        ...(isCodex
          ? [{ text: '都装好后，各自打开一次，确认能正常启动。' }]
          : [
              { text: `${tool} 要在终端里安装和运行。不会打开终端？点开看看：`, terminal: true },
              { text: '都装好后，在终端里运行 claude --version，能看到版本号就行。' }
            ])
      ]
    },
    {
      id: 'import',
      title: '一键导入到 CC Switch',
      lead: '回到本站，把刚创建的密钥导入 CC Switch，不用自己填地址和密钥。',
      actions: [
        {
          text: `打开本站的「${u.apiKeys}」，找到刚创建的密钥，点这一行的「${u.importCcs}」。`,
          shot: { id: `keys-import-${c.client}`, caption: `这一行最右边的「${u.importCcs}」` }
        },
        { text: '浏览器会问「要打开 CC Switch 吗」，选「打开」。第一次可能还要勾选「始终允许」。' },
        {
          text: 'CC Switch 会弹出「确认导入供应商配置」窗口，核对一下，点「导入」。',
          hint: '窗口里那一大段脚本代码是用来在 CC Switch 里显示余额的，不用管它。',
          shot: {
            id: 'ccswitch-import',
            legend: [
              isCodex ? '应用类型：这里会显示 Codex（图中是 Claude 的例子）' : '应用类型：显示 Claude',
              `供应商名称：「${site}」`,
              'API 端点：本站的地址',
              '点「导入」'
            ]
          }
        },
        { text: `回到 CC Switch：左边选「${tool}」，供应商列表里会多出一个「${site}」，就是刚导入的。` }
      ],
      detail: {
        summary: '点了没反应，或者找不到这个按钮？手动添加也行',
        lines: [
          `先确认 CC Switch 已经装好并且能打开。`,
          `在 CC Switch 左边选「${tool}」，点右上角的「添加供应商」，选择自定义，填下面两项：`
        ],
        fields: [
          { label: 'API 地址', value: c.endpoint },
          { label: 'API Key', value: `本站「${u.apiKeys}」页里你创建的那把完整密钥` }
        ]
      }
    },
    {
      id: 'enable',
      title: '启用并开始使用',
      lead: '导入只是把配置添加进去，还要「启用」才会生效。',
      actions: [
        {
          text: `在 CC Switch 左边选「${tool}」，找到「${site}」，点它右边的 ▶ 按钮。`,
          hint: '变成「使用中」就是启用了。上面保持「直连」即可。',
          shot: { id: 'ccswitch-enable', legend: [`左边选「${tool}」`, '点要用的供应商右边的 ▶', '变成「使用中」，就生效了'] }
        },
        ...(isCodex
          ? [{ text: '打开 Codex，选一个项目文件夹，新建一个对话。' }]
          : [
              { text: '打开终端，进入你的项目文件夹。还没有项目？先新建一个空文件夹练手：', code: { label: '新建并进入一个练习文件夹', text: 'mkdir my-project && cd my-project' } },
              { text: '然后运行：', code: { label: tool, text: 'claude' } }
            ]),
        { text: '第一次启动如果问「是否信任这个文件夹」，确认是你自己的文件夹再选信任。' }
      ],
      note: `在 CC Switch 里切换或启用之后，已经打开的 ${tool} 要关掉重开，才会读到新的配置。`
    },
    {
      id: 'verify',
      title: '发一句话，确认接通',
      lead: '最后确认一下，工具和本站都正常。',
      actions: [
        { text: `在 ${tool} 里发送这句话：`, code: { label: tool, text: TEST_PROMPT.zh } },
        { text: '收到回复「连接成功」，说明已经接通。' },
        {
          text: `再回本站的「${u.usage}」，能看到刚才这次调用的记录，说明计费也正常。`,
          shot: { id: 'usage', caption: '使用记录里出现一条新的调用' }
        }
      ]
    }
  ]
}

function en(c: StepContext): TutorialStep[] {
  const { ui: u, siteName: site, qqGroup } = c
  const tool = TOOL[c.client]
  const provider = PROVIDER[c.client]
  const isCodex = c.client === 'codex'
  return [
    {
      id: 'shop',
      title: 'Get a redeem code from the shop',
      lead: 'Credit is sold through our shop. After you pay you get a redeem code, which you use in the next step.',
      actions: [
        { text: `Sign in and click “${u.buy}” in the left menu. The shop opens in a new tab.`, shot: { id: 'sidebar-buy', caption: `“${u.buy}” at the bottom of the left menu` } },
        {
          text: 'In the shop, pick the credit-card category, then click an amount.',
          hint: 'The amount is in US dollars: “10” means $10 of credit.',
          shot: { id: 'shop-list', legend: ['Pick the credit-card category first', 'Then click an amount'] }
        },
        {
          text: 'In the order dialog, enter a contact, choose Alipay or WeChat, click “Pay”, and follow the payment prompts.',
          hint: 'The contact is how you look the order up later. The price includes a small payment fee.',
          shot: { id: 'shop-order', legend: ['Contact (required)', 'Payment method', 'Click “Pay”'] }
        },
        {
          text: 'After payment the page shows a redeem code (also called a card key). Copy all of it.',
          hint: 'Closed the page by mistake? Go back to the shop, click “Order lookup” at the top right, and enter the same contact.',
          shot: { id: 'shop-code', legend: ['This string is the redeem code (card key)', 'Click “Copy”, or “Copy all” above it'] }
        }
      ],
      note: `If you paid but got no code, join QQ group ${qqGroup} and ask an admin — do not pay again.`
    },
    {
      id: 'redeem',
      title: 'Redeem it for credit',
      lead: 'Back on this site, turn the code into usable credit.',
      actions: [
        { text: `Click “${u.redeem}” in the left menu.`, shot: { id: 'sidebar-redeem', caption: `“${u.redeem}” in the left menu` } },
        {
          text: `Paste the code into the box and click “${u.redeemBtn}”.`,
          hint: 'The code in the picture is a demo; yours will differ.',
          shot: { id: 'redeem-form', legend: ['Paste the code', `Click “${u.redeemBtn}”`] }
        },
        { text: 'You will see a success message and your balance goes up. Then move on.', shot: { id: 'redeem-ok', caption: 'After redeeming, the balance and message update' } }
      ],
      note: 'Codes are case-sensitive. Do not copy extra spaces around them.'
    },
    {
      id: 'key',
      title: 'Create an API key',
      lead: 'A key is the credential your tool uses to connect to this site — like a pass.',
      actions: [
        { text: `Click “${u.apiKeys}” in the left menu, then “${u.createKey}” at the top right.`, shot: { id: 'keys-create', caption: `“${u.createKey}” at the top right` } },
        {
          text: 'Fill in the dialog like this:',
          shot: {
            id: `keys-form-${c.client}`,
            legend: [
              `${u.name}: anything you like, e.g. ${tool}`,
              `${u.provider}: for ${tool} choose “${provider}”`,
              `${u.group}: pick one. A lower rate is cheaper; if unsure, take the lowest`,
              `Click “${u.create}”`
            ]
          },
          link: c.plaza ? { to: '/plaza', label: 'Not sure which group? Compare prices on the Plaza' } : undefined
        },
        { text: 'Once created, the key appears in the list. Keep the page open — you need it right away.' }
      ],
      note: 'A key is like the key to your account: never share it or post screenshots. If it leaks, disable it and create a new one.'
    },
    {
      id: 'install',
      title: `Install CC Switch and ${tool}`,
      lead: `Both go on your own computer (not a phone). CC Switch is a small tool that puts your key into ${tool} in one click, so you never edit config files.`,
      actions: [
        { text: 'Download and install using the steps below (the right build for your system is picked for you):', downloads: true },
        ...(isCodex
          ? [{ text: 'When both are installed, open each once to make sure it starts.' }]
          : [
              { text: `${tool} is installed and run from a terminal. Not sure how to open one? Expand this:`, terminal: true },
              { text: 'When everything is installed, run claude --version in the terminal; a version number means it worked.' }
            ])
      ]
    },
    {
      id: 'import',
      title: 'Import into CC Switch',
      lead: 'Back on this site, import the key you just created — no typing addresses or keys by hand.',
      actions: [
        {
          text: `Open “${u.apiKeys}”, find the key you created and click “${u.importCcs}” on its row.`,
          shot: { id: `keys-import-${c.client}`, caption: `“${u.importCcs}” at the right end of the row` }
        },
        { text: 'Your browser asks whether to open CC Switch — choose Open. The first time you may also tick “Always allow”.' },
        {
          text: 'CC Switch shows a “Confirm import” window. Check it, then click “Import”.',
          hint: 'The long script in that window just shows your balance inside CC Switch — ignore it.',
          shot: {
            id: 'ccswitch-import',
            legend: [
              isCodex ? 'App type: this will say Codex (the picture shows Claude as an example)' : 'App type: shows Claude',
              `Provider name: “${site}”`,
              'API endpoint: this site’s address',
              'Click “Import”'
            ]
          }
        },
        { text: `Back in CC Switch, pick “${tool}” on the left. A new provider named “${site}” appears in the list — that is the one you just imported.` }
      ],
      detail: {
        summary: 'Nothing happened, or no such button? Add it manually',
        lines: [
          'First make sure CC Switch is installed and opens.',
          `In CC Switch pick “${tool}” on the left, click “Add provider” at the top right, choose custom, and fill in:`
        ],
        fields: [
          { label: 'API address', value: c.endpoint },
          { label: 'API key', value: `The full key you created on this site’s “${u.apiKeys}” page` }
        ]
      }
    },
    {
      id: 'enable',
      title: 'Enable it and start using it',
      lead: 'Importing only adds the settings. You still have to enable them.',
      actions: [
        {
          text: `In CC Switch pick “${tool}” on the left, find “${site}”, and click the ▶ button on its right.`,
          hint: 'When it says “In use”, it is enabled. Keep the tab above on “Direct”.',
          shot: { id: 'ccswitch-enable', legend: [`Pick “${tool}” on the left`, 'Click ▶ on the provider you want', 'It turns to “In use” — now it is live'] }
        },
        ...(isCodex
          ? [{ text: 'Open Codex, choose a project folder, and start a new chat.' }]
          : [
              { text: 'Open a terminal and go to your project folder. No project yet? Make an empty practice folder:', code: { label: 'Create and enter a practice folder', text: 'mkdir my-project && cd my-project' } },
              { text: 'Then run:', code: { label: tool, text: 'claude' } }
            ]),
        { text: 'If the first launch asks whether to trust the folder, only choose Trust for a folder you recognize.' }
      ],
      note: `After you switch or enable a provider in CC Switch, restart ${tool} so it picks up the new settings.`
    },
    {
      id: 'verify',
      title: 'Send one message to confirm',
      lead: 'A last check that the tool and this site both work.',
      actions: [
        { text: `Send this in ${tool}:`, code: { label: tool, text: TEST_PROMPT.en } },
        { text: 'A reply saying the connection succeeded means you are connected.' },
        { text: `Then open “${u.usage}” on this site: you should see a record for that call, which confirms billing works too.`, shot: { id: 'usage', caption: 'A new call shows up in the usage records' } }
      ]
    }
  ]
}
