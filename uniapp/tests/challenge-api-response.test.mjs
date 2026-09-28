import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('../api/challenge.js', import.meta.url), 'utf8')
  .replace(/^import .*$/gm, '')
  .replaceAll('export const ', 'const ')
const makeApi = (post) => new Function('post', 'get', `${source}; return { enterChallenge, acceptChallenge, cancelChallenge }`)(post, () => {})

test('约球可恢复业务响应交给页面处理，网络与权限错误仍抛出', async () => {
  const confirm = { success: false, action: 'confirm_early', message: '请确认' }
  const api = makeApi(() => Promise.reject({ statusCode: 200, category: 'business', responseData: confirm }))
  assert.deepEqual(await api.enterChallenge({ challenge_id: 1 }), confirm)
  assert.deepEqual(await api.acceptChallenge({ challenge_id: 1 }), confirm)

  for (const error of [
    { statusCode: 401, category: 'session', responseData: confirm },
    { statusCode: 503, category: 'server', responseData: confirm },
    { category: 'network' }
  ]) {
    await assert.rejects(makeApi(() => Promise.reject(error)).cancelChallenge({ challenge_id: 1 }), (received) => received === error)
  }
})
