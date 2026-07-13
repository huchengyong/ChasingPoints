import test from 'node:test'
import assert from 'node:assert/strict'

const createStorageUni = () => {
  const storage = new Map()
  const downloadCalls = []
  const saveCalls = []

  return {
    storage,
    downloadCalls,
    saveCalls,
    uni: {
      getStorageSync(key) {
        return storage.get(key)
      },
      setStorageSync(key, value) {
        storage.set(key, value)
      },
      removeStorageSync(key) {
        storage.delete(key)
      },
      downloadFile({ url, success, fail }) {
        downloadCalls.push(url)
        success({
          statusCode: 200,
          tempFilePath: `/tmp/${url.split('/').pop()}`
        })
      },
      saveFile({ tempFilePath, success, fail }) {
        saveCalls.push(tempFilePath)
        success({
          savedFilePath: `/saved/${tempFilePath.split('/').pop()}`
        })
      },
      removeSavedFile({ filePath, success }) {
        success?.()
      }
    }
  }
}

const importImageCache = async (suffix) => {
  return import(`../utils/image-cache.js?case=${suffix}`)
}

test('getCachedImage saves and reuses local file paths', async () => {
  const fixture = createStorageUni()
  global.uni = fixture.uni

  const { getCachedImage } = await importImageCache(`save-${Date.now()}`)
  const url = 'https://images.gc.wstservices.co.uk/fit-in/200x300/player-a.png'

  const first = await getCachedImage(url)
  const second = await getCachedImage(url)

  assert.equal(first, '/saved/player-a.png')
  assert.equal(second, '/saved/player-a.png')
  assert.deepEqual(fixture.downloadCalls, [url])
  assert.deepEqual(fixture.saveCalls, ['/tmp/player-a.png'])
})

test('getCachedImage falls back to remote url when cache capability is unavailable', async () => {
  global.uni = {
    getStorageSync() {
      return ''
    }
  }

  const { getCachedImage } = await importImageCache(`fallback-${Date.now()}`)
  const url = 'https://images.gc.wstservices.co.uk/fit-in/200x300/player-b.png'

  const result = await getCachedImage(url)

  assert.equal(result, url)
})

test('cacheSaiXunMatchAvatars rewrites player avatars with cached local paths', async () => {
  const fixture = createStorageUni()
  global.uni = fixture.uni

  const { cacheSaiXunMatchAvatars } = await importImageCache(`rounds-${Date.now()}`)
  const rounds = [
    {
      key: '1',
      matches: [
        {
          id: 1,
          homePlayerAvatar: 'https://images.gc.wstservices.co.uk/fit-in/200x300/player-c.png',
          awayPlayerAvatar: 'https://images.gc.wstservices.co.uk/fit-in/200x300/player-d.png'
        }
      ]
    }
  ]

  const cachedRounds = await cacheSaiXunMatchAvatars(rounds)

  assert.notEqual(cachedRounds, rounds)
  assert.equal(cachedRounds[0].matches[0].homePlayerAvatar, '/saved/player-c.png')
  assert.equal(cachedRounds[0].matches[0].awayPlayerAvatar, '/saved/player-d.png')
})

