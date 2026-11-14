import { existsSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import shots from '../shots.json'
import { buildSteps, type StepContext, type TutorialClient } from '../tutorialSteps'

type Meta = { src: string; w: number; h: number; marks: { n?: number }[] }
const all = shots as unknown as Record<string, Record<string, Meta>>

// 所有引用的截图都必须存在；以后如果有只能站长截的图暂时缺着，把 id 加到这里（缺了会自动不显示）。
const OPTIONAL = new Set<string>([])

const ui = {
  apiKeys: 'API Keys', redeem: 'Redeem', usage: 'Usage', createKey: 'Create API Key', importCcs: 'Import to CCS',
  provider: 'Provider', group: 'Group', name: 'Name', create: 'Create', redeemBtn: 'Redeem', buy: 'Buy plan'
}

function stepsFor(lang: 'zh' | 'en', client: TutorialClient) {
  const ctx: StepContext = { lang, client, ui, siteName: 'Site', endpoint: 'https://example.com/v1', qqGroup: '123456', plaza: true }
  return buildSteps(ctx)
}

const cases = (['zh', 'en'] as const).flatMap((lang) => (['claude-code', 'codex'] as const).map((client) => [lang, client] as const))

describe.each(cases)('tutorial steps (%s, %s)', (lang, client) => {
  const steps = stepsFor(lang, client)

  it('has the seven steps in order, each with at least one action', () => {
    expect(steps.map((s) => s.id)).toEqual(['shop', 'redeem', 'key', 'install', 'import', 'enable', 'verify'])
    for (const step of steps) {
      expect(step.title).toBeTruthy()
      expect(step.lead).toBeTruthy()
      expect(step.actions.length).toBeGreaterThan(0)
      for (const action of step.actions) expect(action.text.trim()).not.toBe('')
    }
  })

  it('only references screenshots that exist (optional ones may be absent)', () => {
    const publicDir = join(__dirname, '..', '..', '..', '..', 'public')
    for (const action of steps.flatMap((s) => s.actions)) {
      if (!action.shot) continue
      const meta = all[lang]?.[action.shot.id] ?? all.shared?.[action.shot.id]
      if (!meta) {
        expect(OPTIONAL.has(action.shot.id), `missing screenshot: ${action.shot.id}`).toBe(true)
        continue
      }
      expect(existsSync(join(publicDir, meta.src)), `file for ${action.shot.id}`).toBe(true)
    }
  })

  it('keeps every numbered callout in a screenshot matched by a legend line', () => {
    for (const action of steps.flatMap((s) => s.actions)) {
      const ref = action.shot
      if (!ref) continue
      const meta = all[lang]?.[ref.id] ?? all.shared?.[ref.id]
      if (!meta) continue
      const numbered = meta.marks.filter((m) => m.n != null).length
      if (numbered > 0) {
        expect(ref.legend?.length, `legend for ${ref.id}`).toBe(numbered)
      } else {
        expect(ref.legend ?? [], `unnumbered shot ${ref.id} should use a caption`).toEqual([])
      }
    }
  })

  it('names the right provider and tool for this client', () => {
    const text = JSON.stringify(steps)
    expect(text).toContain(client === 'codex' ? 'OpenAI' : 'Anthropic')
    expect(text).toContain(client === 'codex' ? 'Codex' : 'Claude Code')
    const form = steps.flatMap((s) => s.actions).find((a) => a.shot?.id.startsWith('keys-form-'))
    expect(form?.shot?.id).toBe(`keys-form-${client}`)
  })

  it('uses the real UI labels it is given instead of hard-coded ones', () => {
    const text = JSON.stringify(steps)
    expect(text).toContain('Import to CCS')
    expect(text).toContain('Buy plan')
  })
})

describe('tutorial step content', () => {
  it('sends Claude Code users to a terminal but not Codex desktop users', () => {
    const claude = stepsFor('zh', 'claude-code').flatMap((s) => s.actions)
    const codex = stepsFor('zh', 'codex').flatMap((s) => s.actions)
    expect(claude.some((a) => a.terminal)).toBe(true)
    expect(codex.some((a) => a.terminal)).toBe(false)
    expect(claude.some((a) => a.code?.text === 'claude')).toBe(true)
  })

  it('tells the reader how to get the code back and who to ask when payment goes wrong', () => {
    const text = JSON.stringify(stepsFor('zh', 'codex')[0])
    expect(text).toContain('订单查询')
    expect(text).toContain('123456')
  })

  it('adds the plaza link only when the plaza is enabled', () => {
    const base = { lang: 'zh', client: 'codex', ui, siteName: 'Site', endpoint: 'x', qqGroup: '1' } as const
    const off = buildSteps({ ...base, plaza: false }).flatMap((s) => s.actions)
    const on = buildSteps({ ...base, plaza: true }).flatMap((s) => s.actions)
    expect(off.some((a) => a.link)).toBe(false)
    expect(on.some((a) => a.link?.to === '/plaza')).toBe(true)
  })
})
