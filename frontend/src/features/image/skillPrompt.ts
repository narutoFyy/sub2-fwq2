/**
 * 发给 Codex / Claude Code 的一段话：让智能体自己装好 my-image skill、写入本站地址和密钥、再画一张测试图。
 * 引导、图文教程、密钥页的轻提示、生图页底部共用这一份。
 */

export const MY_IMAGE_REPO = 'https://github.com/less55093-collab/my-image'
export const MY_IMAGE_DEFAULT_MODEL = 'gpt-image-2.5'

/** 生图接口的 Base URL：站点地址 + /v1（my-image 要求 OpenAI 兼容地址）。 */
export function imageBaseUrl(apiBaseUrl: string | null | undefined): string {
  const base = (apiBaseUrl || window.location.origin).replace(/\/+$/, '')
  return base.endsWith('/v1') ? base : `${base}/v1`
}

export function buildSkillPrompt(options: { lang: 'zh' | 'en'; baseUrl: string; apiKey: string; model?: string | null }): string {
  const { baseUrl, apiKey } = options
  const model = options.model || MY_IMAGE_DEFAULT_MODEL
  if (options.lang === 'en') {
    return [
      'Please install and set up the image skill "my-image" for me, all by yourself:',
      '',
      `1. Install: clone ${MY_IMAGE_REPO}.git into your skills folder (Codex: ~/.codex/skills/my-image, Claude Code: ~/.claude/skills/my-image). If the folder already exists, run git pull inside it.`,
      '2. Configure: read its SKILL.md and write the config through stdin, the way it describes for a user who already gave a Base URL and API key. Never put the key in command arguments or print it.',
      `   - Base URL: ${baseUrl}`,
      `   - API Key: ${apiKey}`,
      `   - Model: ${model} (if the server does not have it, use the first image model it suggests)`,
      '3. Test: once it is configured, generate one test image and show it to me: an orange cat lying on a sunny windowsill in the afternoon, film look.',
      '',
      'From now on, whenever I ask you to create or edit an image, use this skill.'
    ].join('\n')
  }
  return [
    '请帮我安装并配置生图 skill「my-image」，全程你自己完成：',
    '',
    `1. 安装：把 ${MY_IMAGE_REPO}.git 克隆到你的 skills 目录（Codex 用 ~/.codex/skills/my-image，Claude Code 用 ~/.claude/skills/my-image）。目录已存在就在里面 git pull。`,
    '2. 配置：读一下它的 SKILL.md，按「用户已经给出 Base URL 和 API Key」的方式，通过 stdin 写入配置。不要把 Key 写进命令行参数，也不要打印出来。',
    `   - Base URL：${baseUrl}`,
    `   - API Key：${apiKey}`,
    `   - 模型：${model}（上游没有的话，用它推荐的第一个图像模型）`,
    '3. 测试：配置好后生成一张测试图并展示给我：一只趴在午后阳光窗台上的橘猫，胶片质感。',
    '',
    '以后我让你生图或改图，就用这个 skill。'
  ].join('\n')
}

/** 展示用：密钥只露前后几位，复制时用完整版本。 */
export function maskKey(value: string): string {
  return value.length <= 12 ? '••••••••' : `${value.slice(0, 5)}••••••••${value.slice(-4)}`
}

/** 能生图的密钥：生效中，且主分组或任一附加分组开了生图。 */
export function keyCanGenerateImages(
  key: { status: string; group?: { allow_image_generation?: boolean } | null; group_ids?: number[] },
  imageGroupIds?: Set<number>
): boolean {
  if (key.status !== 'active') return false
  if (key.group?.allow_image_generation) return true
  return !!imageGroupIds && !!key.group_ids?.some((id) => imageGroupIds.has(id))
}
