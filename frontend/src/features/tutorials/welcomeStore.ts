import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { User } from '@/types'

const PENDING_KEY = 'tutorial_welcome_pending'
const seenKey = (id: number) => `tutorial_welcome_seen:${id}`
function readPending(): number | null {
  try { return Number(sessionStorage.getItem(PENDING_KEY)) || null } catch { return null }
}

// Persist the prompt across the registration redirect and an accidental refresh.
export const useTutorialWelcomeStore = defineStore('tutorialWelcome', () => {
  const pendingUserId = ref<number | null>(readPending())
  function queue(user: Pick<User, 'id' | 'role'>) {
    if (user.role === 'admin') return
    try { if (localStorage.getItem(seenKey(user.id))) return } catch { /* Optional persistence. */ }
    pendingUserId.value = user.id
    try { sessionStorage.setItem(PENDING_KEY, String(user.id)) } catch { /* Keep the in-memory welcome. */ }
  }
  function queueNewOAuthUser(user: Pick<User, 'id' | 'role' | 'created_at'>, now = Date.now()) {
    // OAuth uses the same login response for existing and newly created accounts.
    const age = now - Date.parse(user.created_at)
    if (Number.isFinite(age) && age >= 0 && age < 120_000) queue(user)
  }
  function dismiss(userId: number) {
    try { localStorage.setItem(seenKey(userId), '1') } catch { /* Dismissal still works for this session. */ }
    if (pendingUserId.value === userId) {
      pendingUserId.value = null
      try { sessionStorage.removeItem(PENDING_KEY) } catch { /* Optional persistence. */ }
    }
  }
  return { pendingUserId, queue, queueNewOAuthUser, dismiss }
})
