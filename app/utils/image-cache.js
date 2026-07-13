const IMAGE_CACHE_INDEX_KEY = 'image_cache:index'
const IMAGE_CACHE_TTL = 7 * 24 * 60 * 60 * 1000
const IMAGE_CACHE_LIMIT = 80

const pendingDownloads = new Map()

const isRemoteImage = (url) => /^https?:\/\//i.test(String(url || '').trim())

const getUniApi = () => {
  if (typeof uni === 'undefined' || !uni) return null
  return uni
}

const readCacheIndex = (uniApi) => {
  try {
    const raw = uniApi.getStorageSync(IMAGE_CACHE_INDEX_KEY)
    if (!raw || typeof raw !== 'object') return {}
    return raw
  } catch (error) {
    return {}
  }
}

const writeCacheIndex = (uniApi, index) => {
  try {
    uniApi.setStorageSync(IMAGE_CACHE_INDEX_KEY, index)
  } catch (error) {
    // ignore cache write failures and keep the remote url fallback
  }
}

const removeSavedFile = (uniApi, filePath) => {
  if (!filePath || typeof uniApi.removeSavedFile !== 'function') return Promise.resolve()
  return new Promise((resolve) => {
    uniApi.removeSavedFile({
      filePath,
      complete: () => resolve()
    })
  })
}

const cleanupExpiredEntry = async (uniApi, index, url, entry, persist = true) => {
  if (entry?.savedFilePath) {
    await removeSavedFile(uniApi, entry.savedFilePath)
  }
  delete index[url]
  if (persist) writeCacheIndex(uniApi, index)
}

const ensureCacheLimit = async (uniApi, index) => {
  const entries = Object.entries(index)
  if (entries.length <= IMAGE_CACHE_LIMIT) return

  const overflow = entries
    .sort((left, right) => Number(left[1]?.updatedAt || 0) - Number(right[1]?.updatedAt || 0))
    .slice(0, entries.length - IMAGE_CACHE_LIMIT)

  for (const [url, entry] of overflow) {
    await cleanupExpiredEntry(uniApi, index, url, entry, false)
  }

  writeCacheIndex(uniApi, index)
}

const downloadImage = (uniApi, url) => {
  return new Promise((resolve, reject) => {
    uniApi.downloadFile({
      url,
      success: resolve,
      fail: reject
    })
  })
}

const saveImage = (uniApi, tempFilePath) => {
  return new Promise((resolve, reject) => {
    uniApi.saveFile({
      tempFilePath,
      success: resolve,
      fail: reject
    })
  })
}

export const getCachedImage = async (url) => {
  const normalizedUrl = String(url || '').trim()
  if (!normalizedUrl) return ''
  if (!isRemoteImage(normalizedUrl)) return normalizedUrl

  const uniApi = getUniApi()
  if (!uniApi || typeof uniApi.downloadFile !== 'function' || typeof uniApi.saveFile !== 'function' || typeof uniApi.getStorageSync !== 'function' || typeof uniApi.setStorageSync !== 'function') {
    return normalizedUrl
  }

  const now = Date.now()
  const index = readCacheIndex(uniApi)
  const cachedEntry = index[normalizedUrl]
  if (cachedEntry?.savedFilePath && Number(cachedEntry.expiresAt || 0) > now) {
    return cachedEntry.savedFilePath
  }
  if (cachedEntry) {
    await cleanupExpiredEntry(uniApi, index, normalizedUrl, cachedEntry)
  }

  if (pendingDownloads.has(normalizedUrl)) {
    return pendingDownloads.get(normalizedUrl)
  }

  const task = (async () => {
    try {
      const downloadRes = await downloadImage(uniApi, normalizedUrl)
      if (Number(downloadRes?.statusCode || 0) !== 200 || !downloadRes?.tempFilePath) {
        return normalizedUrl
      }

      const saveRes = await saveImage(uniApi, downloadRes.tempFilePath)
      if (!saveRes?.savedFilePath) {
        return normalizedUrl
      }

      const nextIndex = readCacheIndex(uniApi)
      nextIndex[normalizedUrl] = {
        savedFilePath: saveRes.savedFilePath,
        updatedAt: now,
        expiresAt: now + IMAGE_CACHE_TTL
      }
      await ensureCacheLimit(uniApi, nextIndex)
      writeCacheIndex(uniApi, nextIndex)
      return saveRes.savedFilePath
    } catch (error) {
      return normalizedUrl
    } finally {
      pendingDownloads.delete(normalizedUrl)
    }
  })()

  pendingDownloads.set(normalizedUrl, task)
  return task
}

export const cacheSaiXunMatchAvatars = async (rounds = []) => {
  const nextRounds = await Promise.all(
    (Array.isArray(rounds) ? rounds : []).map(async (round) => {
      const nextMatches = await Promise.all(
        (Array.isArray(round?.matches) ? round.matches : []).map(async (match) => {
          const [homePlayerAvatar, awayPlayerAvatar] = await Promise.all([
            getCachedImage(match?.homePlayerAvatar || ''),
            getCachedImage(match?.awayPlayerAvatar || '')
          ])

          return {
            ...match,
            homePlayerAvatar,
            awayPlayerAvatar
          }
        })
      )

      return {
        ...round,
        matches: nextMatches
      }
    })
  )

  return nextRounds
}

