const STATIC_READ_SCHEMA_VERSION = 1
const STATIC_READ_PREFIX = 'cp:static-read:'

const storage = () => (typeof uni !== 'undefined' ? uni : null)

export const readStaticReadCache = (key, version = STATIC_READ_SCHEMA_VERSION) => {
  const client = storage()
  if (!client || !key) return null
  try {
    const cached = client.getStorageSync(`${STATIC_READ_PREFIX}${key}`)
    if (!cached || Number(cached.schemaVersion) !== Number(version) || !cached.loadedAt) {
      return null
    }
    return { value: cached.value, loadedAt: Number(cached.loadedAt) || 0 }
  } catch (_) {
    return null
  }
}

export const writeStaticReadCache = (key, value, version = STATIC_READ_SCHEMA_VERSION, loadedAt = Date.now()) => {
  const client = storage()
  if (!client || !key) return
  try {
    client.setStorageSync(`${STATIC_READ_PREFIX}${key}`, {
      schemaVersion: Number(version),
      loadedAt,
      value
    })
  } catch (_) {}
}

export const removeStaticReadCache = (key) => {
  const client = storage()
  if (!client || !key) return
  try {
    client.removeStorageSync(`${STATIC_READ_PREFIX}${key}`)
  } catch (_) {}
}

export { STATIC_READ_SCHEMA_VERSION }
