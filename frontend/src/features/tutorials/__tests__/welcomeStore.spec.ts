import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useTutorialWelcomeStore } from '../welcomeStore'

const newUser = { id: 42, role: 'user' as const, created_at: new Date().toISOString() }
beforeEach(() => {
  localStorage.clear()
  sessionStorage.clear()
  setActivePinia(createPinia())
})

describe('new-user tutorial welcome', () => {
  it('persists a successful registration prompt across a page refresh', () => {
    useTutorialWelcomeStore().queue(newUser)
    setActivePinia(createPinia())
    expect(useTutorialWelcomeStore().pendingUserId).toBe(42)
  })
  it('remembers dismissal for this account without suppressing another account', () => {
    const store = useTutorialWelcomeStore()
    store.queue(newUser)
    store.dismiss(newUser.id)
    store.queue(newUser)
    expect(store.pendingUserId).toBeNull()
    setActivePinia(createPinia())
    const refreshed = useTutorialWelcomeStore()
    refreshed.queue(newUser)
    expect(refreshed.pendingUserId).toBeNull()
    refreshed.queue({ ...newUser, id: 43 })
    expect(refreshed.pendingUserId).toBe(43)
  })
  it('does not let dismissal of a different account clear a pending prompt', () => {
    const store = useTutorialWelcomeStore()
    store.queue(newUser)
    store.dismiss(43)
    expect(store.pendingUserId).toBe(42)
  })
  it('queues a just-created OAuth account', () => {
    const now = Date.now()
    const store = useTutorialWelcomeStore()
    store.queueNewOAuthUser({ ...newUser, created_at: new Date(now - 1000).toISOString() }, now)
    expect(store.pendingUserId).toBe(42)
  })
  it.each(['2020-01-01', 'invalid', new Date(Date.now() + 60_000).toISOString()])('does not queue an existing or invalid OAuth account (%s)', created_at => {
    const store = useTutorialWelcomeStore()
    store.queueNewOAuthUser({ ...newUser, created_at })
    expect(store.pendingUserId).toBeNull()
  })
  it('does not prompt administrators', () => {
    const store = useTutorialWelcomeStore()
    store.queue({ ...newUser, role: 'admin' })
    expect(store.pendingUserId).toBeNull()
  })
})
