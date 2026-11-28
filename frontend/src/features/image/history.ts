/**
 * 生图记录只存在这台设备的浏览器里（IndexedDB，按账号分开），刷新后还在。
 * 存储不可用（隐私模式、被禁用）时本次会话内照常可用，只是不保存。
 * 最多留 24 张图，超出时删掉最早的记录。
 */
import type { ImageRatio } from './api'

export interface ImageRecord {
  id: string
  userId: number
  prompt: string
  ratio: ImageRatio
  model: string
  createdAt: number
  /** data URL 或上游 URL。 */
  images: string[]
}

const DB_NAME = 'sub2api-image'
const STORE = 'records'
/** 一张原图的 data URL 有 2～5 MB：按图片张数而不是记录条数限量，最多占一百来 MB。 */
const KEEP_IMAGES = 24

function open(): Promise<IDBDatabase | null> {
  return new Promise((resolve) => {
    try {
      if (typeof indexedDB === 'undefined') return resolve(null)
      const req = indexedDB.open(DB_NAME, 1)
      req.onupgradeneeded = () => {
        const store = req.result.createObjectStore(STORE, { keyPath: 'id' })
        store.createIndex('user', 'userId')
      }
      req.onsuccess = () => resolve(req.result)
      req.onerror = () => resolve(null)
    } catch {
      resolve(null)
    }
  })
}

export async function loadRecords(userId: number): Promise<ImageRecord[]> {
  const db = await open()
  if (!db) return []
  return new Promise((resolve) => {
    try {
      const req = db.transaction(STORE, 'readonly').objectStore(STORE).index('user').getAll(userId)
      req.onsuccess = () => resolve((req.result as ImageRecord[]).sort((a, b) => b.createdAt - a.createdAt))
      req.onerror = () => resolve([])
    } catch {
      resolve([])
    }
  })
}

export async function saveRecord(record: ImageRecord): Promise<void> {
  const db = await open()
  if (!db) return
  const all = await loadRecords(record.userId)
  try {
    const store = db.transaction(STORE, 'readwrite').objectStore(STORE)
    store.put(record)
    let kept = record.images.length
    for (const old of all.filter((r) => r.id !== record.id)) {
      kept += old.images.length
      if (kept > KEEP_IMAGES) store.delete(old.id)
    }
  } catch {
    /* 存不下就算了 */
  }
}

export async function deleteRecord(id: string): Promise<void> {
  const db = await open()
  if (!db) return
  try {
    db.transaction(STORE, 'readwrite').objectStore(STORE).delete(id)
  } catch {
    /* ignore */
  }
}

export async function clearRecords(userId: number): Promise<void> {
  const all = await loadRecords(userId)
  await Promise.all(all.map((r) => deleteRecord(r.id)))
}
