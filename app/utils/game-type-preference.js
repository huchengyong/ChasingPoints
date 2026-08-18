import { GAME_TYPE_OPTIONS } from './game-types.js'

const STORAGE_PREFIX = 'default-game-type'
const supportedTypes = new Set(GAME_TYPE_OPTIONS.map((item) => Number(item.value)))

export const DEFAULT_GAME_TYPE_OPTIONS = [
  { value: 0, label: '每次询问' },
  ...GAME_TYPE_OPTIONS.map((item) => ({ ...item }))
]

export const normalizeDefaultGameType = (value) => {
  const normalized = Number(value)
  return supportedTypes.has(normalized) ? normalized : 0
}

export const getDefaultGameTypeStorageKey = (userId) => {
  const normalizedUserId = Number(userId) || 0
  return normalizedUserId > 0 ? `${STORAGE_PREFIX}:${normalizedUserId}` : ''
}

export const readDefaultGameType = (storage, userId) => {
  const key = getDefaultGameTypeStorageKey(userId)
  if (!key || typeof storage?.getStorageSync !== 'function') return 0
  return normalizeDefaultGameType(storage.getStorageSync(key))
}

export const saveDefaultGameType = (storage, userId, gameType) => {
  const key = getDefaultGameTypeStorageKey(userId)
  if (!key) return 0
  const normalized = normalizeDefaultGameType(gameType)
  if (normalized > 0 && typeof storage?.setStorageSync === 'function') {
    storage.setStorageSync(key, normalized)
    return normalized
  }
  if (typeof storage?.removeStorageSync === 'function') storage.removeStorageSync(key)
  return 0
}
