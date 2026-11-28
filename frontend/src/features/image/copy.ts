import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

export function useImageCopy() {
  const { locale } = useI18n()
  return computed(() => (locale.value.startsWith('zh') ? imageCopyZh : imageCopyEn))
}

export const imageCopyZh = {
  title: '生图',
  lead: '写一句话描述画面，几十秒后拿到图片',
  promptLabel: '描述画面',
  promptPlaceholder: '比如：雨后的上海街头，霓虹倒映在积水里，电影感',
  promptPlaceholderEdit: '说说要怎么改，比如：把天空换成星空，其他保持不变',
  examplesLabel: '不知道写什么？点一个试试',
  examples: [
    { label: '窗台橘猫', prompt: '一只橘猫趴在洒满午后阳光的木质窗台上，窗外是虚化的绿植，柔和自然光，胶片质感' },
    { label: '产品海报', prompt: '一瓶透明玻璃香水立在湿润的黑色岩石上，背后是清晨的薄雾，商业摄影，干净高级' },
    { label: '水墨江南', prompt: '水墨风格的江南小镇，石桥、乌篷船和细雨，大面积留白，淡雅配色' },
    { label: '3D 头像', prompt: '一个戴耳机的年轻人侧脸，3D 卡通风格，柔和灯光，干净的浅色背景，适合做头像' }
  ],
  ratioLabel: '比例',
  ratios: { square: '方形', landscape: '横版', portrait: '竖版' },
  ratioHint: { square: '1:1', landscape: '3:2', portrait: '2:3' },
  referenceLabel: '参考图',
  referenceOptional: '可选',
  referenceAdd: '上传一张图，按描述修改它',
  referenceRemove: '移除参考图',
  referenceTooBig: '图片太大了，请换一张 20 MB 以内的',
  keyLabel: '密钥',
  modelLabel: '模型',
  qualityLabel: '质量',
  qualities: { auto: '自动', low: '草稿（快）', high: '高清（慢）' },
  more: '更多设置',
  generate: '生成',
  generating: '生成中',
  stop: '停止',
  stopHint: '不再等这次的结果；已经发出的请求可能仍会计费',
  elapsed: (s: number) => `已等待 ${s} 秒 · 通常 30 到 90 秒`,
  stageEmpty: '图片会出现在这里',
  stageEmptyHint: '写好描述，点「生成」',
  download: '下载',
  enlarge: '放大查看',
  useAsReference: '在这张基础上改',
  again: '再来一次',
  retry: '重试',
  toKeys: '去换个分组',
  close: '关闭',
  historyLabel: '最近生成',
  historyNote: '只保存在这台设备的浏览器里',
  historyClear: '清空',
  historyClearConfirm: '清空这台设备上的生图记录？',
  historyDelete: '删除这条',
  loadingKeys: '正在读取密钥…',
  noKeyTitle: '还没有能生图的密钥',
  noKeyBody: '创建密钥时选一个支持生图的分组就行，一分钟搞定',
  noKeyAction: '去创建密钥',
  noModel: '这把密钥的分组里没有生图模型，换一把试试',
  errors: {
    invalid_key: '密钥无效或已停用，换一把试试',
    billing: '余额不足，或这个分组没有生图权限',
    noPermission: '这把密钥所在的分组不能生图，换一个支持生图的分组',
    rate_limit: '请求太频繁了，稍等一会儿再试',
    upstream: (message: string) => `没生成出来：${message}`,
    empty: '没有拿到图片，换个描述再试一次',
    network: '网络断了一下，再点一次「生成」',
    timeout: '等了 100 秒还没出图，连接被断开了。这次可能已经扣费。可以在「更多设置」里把质量换成「草稿」再试'
  },
  agentTitle: '在 Codex / Claude Code 里也能生图',
  agentAction: '复制一段话发给它',

  skill: {
    title: '在智能体里生图',
    body: '复制下面这段话，发给你的 Codex 或 Claude Code。它会自己装好 my-image、配置好密钥，再画一张测试图。之后跟它说「帮我画……」就行。',
    keyLabel: '用哪把密钥',
    copy: '复制这段话',
    copied: '已复制，去粘贴发送吧',
    note: '这段话里带着你的密钥，只发给自己电脑上的智能体',
    repo: '查看 my-image 源码',
    loading: '正在读取密钥…'
  },

  keysHint: {
    text: (name: string) => `「${name}」能生图，也可以装进 Codex / Claude Code 里用`,
    action: '复制一段话发给它',
    dismiss: '不再提示'
  }
}

export type ImageCopy = typeof imageCopyZh

export const imageCopyEn: ImageCopy = {
  title: 'Images',
  lead: 'Describe a picture in a sentence and get it back in under a minute',
  promptLabel: 'Describe the picture',
  promptPlaceholder: 'e.g. A Shanghai street after rain, neon reflected in puddles, cinematic',
  promptPlaceholderEdit: 'Say what to change, e.g. turn the sky into a starry night, keep everything else',
  examplesLabel: 'Not sure what to write? Try one',
  examples: [
    { label: 'Cat on a windowsill', prompt: 'An orange cat lying on a sunlit wooden windowsill, blurred green plants outside, soft natural light, film look' },
    { label: 'Product poster', prompt: 'A clear glass perfume bottle standing on wet black rock, morning mist behind it, commercial photography, clean and refined' },
    { label: 'Ink-wash town', prompt: 'An ink-wash painting of a small water town with a stone bridge, a boat and light rain, lots of empty space, muted colors' },
    { label: '3D avatar', prompt: 'Side profile of a young person wearing headphones, 3D cartoon style, soft lighting, clean light background, avatar-ready' }
  ],
  ratioLabel: 'Shape',
  ratios: { square: 'Square', landscape: 'Landscape', portrait: 'Portrait' },
  ratioHint: { square: '1:1', landscape: '3:2', portrait: '2:3' },
  referenceLabel: 'Reference',
  referenceOptional: 'optional',
  referenceAdd: 'Upload an image to edit it with your description',
  referenceRemove: 'Remove reference',
  referenceTooBig: 'That image is too large — pick one under 20 MB',
  keyLabel: 'Key',
  modelLabel: 'Model',
  qualityLabel: 'Quality',
  qualities: { auto: 'Auto', low: 'Draft (fast)', high: 'High (slow)' },
  more: 'More settings',
  generate: 'Generate',
  generating: 'Generating',
  stop: 'Stop',
  stopHint: 'Stop waiting for this result. A request already sent may still be charged',
  elapsed: (s: number) => `${s}s so far · usually 30 to 90s`,
  stageEmpty: 'Your images appear here',
  stageEmptyHint: 'Write a description and click Generate',
  download: 'Download',
  enlarge: 'View larger',
  useAsReference: 'Edit from this one',
  again: 'Run again',
  retry: 'Try again',
  toKeys: 'Switch group',
  close: 'Close',
  historyLabel: 'Recent',
  historyNote: 'Saved only in this browser',
  historyClear: 'Clear',
  historyClearConfirm: 'Clear the image history on this device?',
  historyDelete: 'Delete',
  loadingKeys: 'Loading your keys…',
  noKeyTitle: 'No key that can make images yet',
  noKeyBody: 'Create a key and pick a group that supports images — it takes a minute',
  noKeyAction: 'Create a key',
  noModel: 'This key’s group has no image model. Try another key',
  errors: {
    invalid_key: 'This key is invalid or disabled. Try another one',
    billing: 'Not enough balance, or this group cannot make images',
    noPermission: 'This key’s group cannot make images. Switch to a group that supports images',
    rate_limit: 'Too many requests. Wait a moment and try again',
    upstream: (message: string) => `It didn’t work: ${message}`,
    empty: 'No image came back. Try a different description',
    network: 'The connection dropped. Click Generate again',
    timeout: 'No image after 100 seconds, so the connection was closed. This one may still be charged. Try Draft quality under More settings'
  },
  agentTitle: 'Make images in Codex or Claude Code too',
  agentAction: 'Copy a message for it',

  skill: {
    title: 'Make images in your agent',
    body: 'Copy the message below and send it to Codex or Claude Code. It installs my-image, sets up your key and draws a test image. After that, just ask it to draw something.',
    keyLabel: 'Key to use',
    copy: 'Copy the message',
    copied: 'Copied — paste it and send',
    note: 'The message contains your key. Only send it to an agent on your own computer',
    repo: 'my-image source',
    loading: 'Loading your key…'
  },

  keysHint: {
    text: (name: string) => `“${name}” can make images — you can also use it in Codex or Claude Code`,
    action: 'Copy a message for it',
    dismiss: 'Don’t show again'
  }
}
