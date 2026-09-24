import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import * as asyncState from '../utils/async-page-state.js'

const { ref } = createRequire(import.meta.url)('vue')
const source = readFileSync(new URL('../subPages/social/challenges.vue', import.meta.url), 'utf8')
const code = ['fetchActive', 'fetchActiveList', 'fetchHistory'].map((name, index) => {
  const next = ['fetchActiveList', 'fetchHistory', 'fetchList'][index]
  return source.slice(source.indexOf(`const ${name} = async`), source.indexOf(`\nconst ${next}`, source.indexOf(`const ${name} = async`)))
}).join('\n')

const makeInputs = (overrides = {}) => ({
  ...asyncState,
  userStore: { userId: 1, authGeneration: 1 },
  activeItems: ref([]),
  historyItems: ref([]),
  nextBeforeId: ref(0),
  challengeState: ref(asyncState.createAsyncPageState({ authGeneration: 1, data: [] })),
  historyState: ref(asyncState.createAsyncPageState({ authGeneration: 1, data: [] })),
  challengesDirty: ref(true),
  historyDirty: ref(true),
  historyDirtySeq: ref(0),
  loading: ref(false),
  tab: ref('received'),
  serverTime: ref(''),
  getPendingChallenges: () => Promise.resolve({ success: true, list: [] }),
  getChallengeSummary: () => Promise.resolve({ success: true, server_time: '2026-08-25 12:00:00' }),
  getChallengeHistory: () => Promise.resolve({ success: true, list: [], next_before_id: 0 }),
  createRequestError: ({ message }) => new Error(message),
  ...overrides
})

const loadFn = (inputs, name) => new Function(...Object.keys(inputs), `${code}; return ${name}`)(...Object.values(inputs))

test('旧账号活动列表返回不得写回新账号页面', async () => {
  let finishOld
  const inputs = makeInputs({
    // 只有旧账号（userId=1）的请求挂起；新账号请求立即返回权威数据。
    getPendingChallenges: () => inputs.userStore.userId === 1
      ? new Promise(resolve => { finishOld = resolve })
      : Promise.resolve({ success: true, list: [{ id: 222 }] })
  })
  const fetchActiveList = loadFn(inputs, 'fetchActiveList')
  const oldRequest = fetchActiveList({ force: true })
  inputs.userStore.userId = 2
  inputs.userStore.authGeneration = 2
  await fetchActiveList({ force: true })
  finishOld({ success: true, list: [{ id: 111 }] })
  await oldRequest
  assert.deepEqual(inputs.activeItems.value.map(item => item.id), [222])
})

test('历史首次读取失败必须进入错误态，不得伪装成空记录', async () => {
  const inputs = makeInputs({
    tab: ref('history'),
    getChallengeHistory: () => Promise.reject(new Error('history down'))
  })
  const fetchHistory = loadFn(inputs, 'fetchHistory')
  await fetchHistory({ force: true })
  assert.equal(inputs.historyState.value.status, asyncState.ASYNC_PAGE_STATUS.ERROR)
  assert.equal(inputs.historyItems.value.length, 0)
})

test('历史已有数据时刷新失败保留列表与游标', async () => {
  let fails = false
  const inputs = makeInputs({
    tab: ref('history'),
    getChallengeHistory: () => fails
      ? Promise.reject(new Error('history down'))
      : Promise.resolve({ success: true, list: [{ id: 7 }], next_before_id: 6 })
  })
  const fetchHistory = loadFn(inputs, 'fetchHistory')
  await fetchHistory({ force: true })
  assert.equal(inputs.historyState.value.status, asyncState.ASYNC_PAGE_STATUS.READY)
  fails = true
  await fetchHistory({ force: true })
  assert.equal(inputs.historyState.value.status, asyncState.ASYNC_PAGE_STATUS.READY)
  assert.ok(inputs.historyState.value.refreshError)
  assert.deepEqual(inputs.historyItems.value.map(item => item.id), [7])
  assert.equal(inputs.nextBeforeId.value, 6)
})
