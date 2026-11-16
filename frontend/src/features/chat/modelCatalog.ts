/**
 * 聊天页给小白看的模型名单：只显示这里列出的最新模型，按用途写一句大白话。
 * 有新模型时只改这一处；分组里没有的模型不会显示，所以可以提前写上还没上架的。
 * 第一项是默认模型（分组里没有它时，退到名单里第一个可用的）。
 */
export interface ChatModelEntry {
  id: string
  name: string
  vendor: 'claude' | 'gpt' | 'gemini'
  tag: { zh: string; en: string }
  /** 引导里讲能力用的一句话。 */
  pitch: { zh: string; en: string }
}

export const CHAT_MODEL_CATALOG: ChatModelEntry[] = [
  {
    id: 'claude-opus-5-5',
    name: 'Claude Opus 5.5',
    vendor: 'claude',
    tag: { zh: '全能主力', en: 'Best all-rounder' },
    pitch: { zh: '聊天、写作、分析、写代码都在行，日常首选。', en: 'Great at chat, writing, analysis and code. The everyday pick.' }
  },
  {
    id: 'claude-fable-5-2',
    name: 'Claude Fable 5.2',
    vendor: 'claude',
    tag: { zh: '最聪明 · 较贵', en: 'Smartest · pricier' },
    pitch: { zh: '目前最聪明的模型，适合难题、长文和复杂推理，单价较高。', en: 'The smartest model today, for hard problems, long documents and deep reasoning. Costs more.' }
  },
  {
    id: 'gpt-6-astra',
    name: 'GPT-6 Astra',
    vendor: 'gpt',
    tag: { zh: 'OpenAI 最强', en: "OpenAI's strongest" },
    pitch: { zh: 'OpenAI 最强的模型，擅长推理和专业问题。', en: "OpenAI's most capable model, strong at reasoning and expert questions." }
  },
  {
    id: 'gpt-6.1-sol',
    name: 'GPT-6.1 Sol',
    vendor: 'gpt',
    tag: { zh: '又快又省', en: 'Fast and affordable' },
    pitch: { zh: 'OpenAI 最新的模型，反应快、价格低。', en: "OpenAI's newest model: quick and inexpensive." }
  },
  {
    id: 'gemini-4-carbon',
    name: 'Gemini 4 Carbon',
    vendor: 'gemini',
    tag: { zh: '谷歌最强', en: "Google's strongest" },
    pitch: { zh: '谷歌最强的模型，知识面广。', en: "Google's most capable model, with broad knowledge." }
  },
  {
    id: 'gemini-4-hydrogen',
    name: 'Gemini 4 Hydrogen',
    vendor: 'gemini',
    tag: { zh: '反应最快', en: 'Fastest' },
    pitch: { zh: '速度最快，适合随手问。', en: 'The fastest, good for quick questions.' }
  }
]

/** 模型 id 对上名单项：完全相同，或是带日期的快照（如 claude-opus-5-5-20260922）。 */
export function chatModelEntry(id: string): ChatModelEntry | undefined {
  return CHAT_MODEL_CATALOG.find((entry) => id === entry.id || id.startsWith(`${entry.id}-20`))
}

/**
 * 把网关返回的模型列表收成名单里的最新模型，按名单顺序排。
 * 分组里一个名单模型都没有时（比如只开了旧模型的分组），原样返回，免得下拉框是空的。
 */
export function chatModelsFrom(models: string[]): string[] {
  const picked: string[] = []
  for (const entry of CHAT_MODEL_CATALOG) {
    const hit = models.find((id) => chatModelEntry(id) === entry)
    if (hit) picked.push(hit)
  }
  return picked.length ? picked : models
}

export function chatModelLabel(id: string, zh: boolean): string {
  const entry = chatModelEntry(id)
  if (!entry) return id
  return `${entry.name} · ${zh ? entry.tag.zh : entry.tag.en}`
}
