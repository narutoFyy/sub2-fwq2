import { describe, expect, it } from 'vitest'
import { chatModelLabel, chatModelsFrom } from '../modelCatalog'

describe('chat model catalog', () => {
  it('keeps only the latest catalog models, in catalog order, default first', () => {
    const gateway = ['gpt-5.5', 'grok-4.7', 'gemini-4-hydrogen', 'claude-fable-5-2', 'claude-sonnet-4-6', 'claude-opus-5-5', 'x-ai/grok', 'gpt-6.1-sol']
    expect(chatModelsFrom(gateway)).toEqual(['claude-opus-5-5', 'claude-fable-5-2', 'gpt-6.1-sol', 'gemini-4-hydrogen'])
  })

  it('accepts dated snapshots of a catalog model', () => {
    expect(chatModelsFrom(['claude-opus-5-5-20260922'])).toEqual(['claude-opus-5-5-20260922'])
  })

  it('falls back to the raw list when no catalog model is available', () => {
    expect(chatModelsFrom(['claude-sonnet-4-6', 'gpt-5.5'])).toEqual(['claude-sonnet-4-6', 'gpt-5.5'])
  })

  it('labels catalog models in plain words and leaves others as ids', () => {
    expect(chatModelLabel('claude-fable-5-2', true)).toBe('Claude Fable 5.2 · 最聪明 · 较贵')
    expect(chatModelLabel('gpt-6-astra', false)).toBe("GPT-6 Astra · OpenAI's strongest")
    expect(chatModelLabel('mystery-1', true)).toBe('mystery-1')
  })
})
