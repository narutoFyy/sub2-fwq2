import type { ChatMode } from './types'

export interface ChatSuggestion {
  title: string
  prompt: string
}

/** 空对话里的快捷问题：点一下填进输入框，用户补完再发。 */
const SUGGESTIONS: Record<'zh' | 'en', Record<ChatMode, ChatSuggestion[]>> = {
  zh: {
    chat: [
      { title: '帮我写一封邮件', prompt: '帮我写一封礼貌的邮件，向领导请两天假，语气正式但不生硬。' },
      { title: '解释一个概念', prompt: '用通俗的例子给我解释一下什么是大语言模型的「上下文窗口」。' },
      { title: '翻译成英文', prompt: '把下面这段话翻译成地道的英文：\n' },
      { title: '做个旅行计划', prompt: '帮我规划一个三天两晚的杭州周末行程，预算适中，喜欢美食和散步。' }
    ],
    fortune: [
      { title: '八字命盘', prompt: '帮我排一下八字命盘，分析五行和性格。我的出生信息是：' },
      { title: '今年运势', prompt: '帮我看看今年的整体运势（事业、财运、感情、健康）。我的出生信息是：' },
      { title: '塔罗三张牌', prompt: '用塔罗牌帮我抽三张牌（过去、现在、未来），我想问：' },
      { title: '星座配对', prompt: '帮我看看这两个星座的配对：我是' },
      { title: '周公解梦', prompt: '帮我解一个梦：' },
      { title: '今日宜忌', prompt: '帮我看看今天的宜忌、幸运色和幸运数字。' }
    ]
  },
  en: {
    chat: [
      { title: 'Draft an email', prompt: 'Write a polite email asking my manager for two days off. Formal but warm.' },
      { title: 'Explain a concept', prompt: 'Explain what a context window is in large language models, with an everyday example.' },
      { title: 'Translate', prompt: 'Translate the following into natural Chinese:\n' },
      { title: 'Plan a trip', prompt: 'Plan a relaxed three-day weekend in Kyoto with good food and walks.' }
    ],
    fortune: [
      { title: 'BaZi chart', prompt: 'Please cast my BaZi chart and read my five elements and personality. My birth details: ' },
      { title: 'This year', prompt: 'How does this year look for me (career, money, love, health)? My birth details: ' },
      { title: 'Three-card tarot', prompt: 'Draw three tarot cards for me (past, present, future). My question: ' },
      { title: 'Zodiac match', prompt: 'How compatible are these two signs? I am ' },
      { title: 'Dream reading', prompt: 'Please interpret my dream: ' },
      { title: "Today's luck", prompt: "What are today's dos and don'ts, lucky color and lucky number for me?" }
    ]
  }
}

export function suggestionsFor(locale: string, mode: ChatMode): ChatSuggestion[] {
  return SUGGESTIONS[locale.startsWith('zh') ? 'zh' : 'en'][mode]
}
