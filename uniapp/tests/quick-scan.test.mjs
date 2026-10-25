import test from 'node:test'
import assert from 'node:assert/strict'

import { SCAN_CONFIRM_ROUTE, createQuickScan } from '../utils/quick-scan.js'

const defer = () => {
  let resolve
  const promise = new Promise((resolvePromise) => { resolve = resolvePromise })
  return { promise, resolve }
}

const startResult = (overrides = {}) => ({
  type: 'start_match',
  inviteToken: 'secret.token',
  opponent: { id: 9, nickname: '对手乙', avatar: 'a.png' },
  ...overrides
})

const refereeResult = () => ({
  type: 'join_referee',
  refereeJoin: { match_id: 77, join_token: 'referee-secret' }
})

const setup = ({ scan, session } = {}) => {
  const currentSession = { isLoggedIn: true, userId: 7, authGeneration: 1, ...session }
  const calls = []
  const uniApi = {
    navigateTo: (options) => calls.push(['navigateTo', options]),
    redirectTo: (options) => calls.push(['redirectTo', options]),
    showToast: (options) => calls.push(['showToast', options])
  }
  const scanCalls = []
  const quickScan = createQuickScan({
    uniApi,
    getSession: () => ({ ...currentSession }),
    previewMatchInvite: async () => ({ success: true }),
    scan: async (options) => {
      scanCalls.push(options)
      return scan ? scan(options) : startResult()
    }
  })
  return { quickScan, session: currentSession, calls, scanCalls, uniApi }
}

const identity = { userId: 7, authGeneration: 1 }

test('a guest is sent to login and the camera is never opened', async () => {
  const { quickScan, calls, scanCalls } = setup({ session: { isLoggedIn: false, userId: 0 } })

  const result = await quickScan.start()

  assert.equal(result.type, 'login')
  assert.deepEqual(calls, [['navigateTo', { url: '/pages/login/login' }]])
  assert.equal(scanCalls.length, 0)
})

test('an opponent code opens the confirm page without putting any credential in the route', async () => {
  const { quickScan, calls, scanCalls, uniApi } = setup()

  const result = await quickScan.start()

  assert.deepEqual(result, { type: 'confirm', mode: 'start' })
  assert.equal(scanCalls.length, 1)
  assert.equal(scanCalls[0].uniApi, uniApi)
  assert.equal(typeof scanCalls[0].previewMatchInvite, 'function')
  assert.equal(calls.length, 1)
  assert.equal(calls[0][0], 'navigateTo')
  assert.equal(calls[0][1].url, SCAN_CONFIRM_ROUTE)
  assert.equal(SCAN_CONFIRM_ROUTE, '/subPages/match/scanConfirm')
  assert.doesNotMatch(JSON.stringify(calls), /secret|token|opponent/i)

  const handoff = quickScan.takeHandoff(identity)
  assert.equal(handoff.mode, 'start')
  assert.equal(handoff.inviteToken, 'secret.token')
  assert.deepEqual(handoff.opponent, { id: 9, nickname: '对手乙', avatar: 'a.png' })
  assert.equal(handoff.preset, null)
  assert.equal(quickScan.takeHandoff(identity), null, 'a handoff can only be taken once')
})

test('a referee code opens the confirm page in referee mode', async () => {
  const { quickScan, calls } = setup({ scan: refereeResult })

  assert.deepEqual(await quickScan.start(), { type: 'confirm', mode: 'referee' })
  assert.equal(calls[0][1].url, SCAN_CONFIRM_ROUTE)
  const handoff = quickScan.takeHandoff(identity)
  assert.equal(handoff.mode, 'referee')
  assert.deepEqual(handoff.refereeJoin, { match_id: 77, join_token: 'referee-secret' })
})

test('cancelling stays silent and an invalid code explains why, neither navigates', async () => {
  const cancelled = setup({ scan: () => ({ type: 'cancelled', reason: 'cancelled' }) })
  assert.equal((await cancelled.quickScan.start()).type, 'cancelled')
  assert.deepEqual(cancelled.calls, [])

  const invalid = setup({ scan: () => ({ type: 'error', message: '无效的二维码' }) })
  assert.equal((await invalid.quickScan.start()).type, 'error')
  assert.deepEqual(invalid.calls, [['showToast', { title: '无效的二维码', icon: 'none' }]])
  assert.equal(invalid.quickScan.takeHandoff(identity), null)

  const noMessage = setup({ scan: () => ({ type: 'error' }) })
  await noMessage.quickScan.start()
  assert.deepEqual(noMessage.calls, [['showToast', { title: '扫码失败', icon: 'none' }]])
})

test('only one scan runs at a time and the flag is released afterwards', async () => {
  const pending = defer()
  const { quickScan, scanCalls } = setup({ scan: () => pending.promise })

  const first = quickScan.start()
  const second = await quickScan.start()
  assert.equal(second.type, 'busy')
  assert.equal(scanCalls.length, 1)

  pending.resolve({ type: 'cancelled' })
  assert.equal((await first).type, 'cancelled')

  const third = quickScan.start()
  assert.equal(scanCalls.length, 2)
  await third
})

test('the busy flag is also released when the scan throws', async () => {
  let shouldThrow = true
  const { quickScan, scanCalls } = setup({
    scan: () => {
      if (shouldThrow) throw new Error('camera crashed')
      return { type: 'cancelled' }
    }
  })

  await assert.rejects(quickScan.start(), /camera crashed/)
  shouldThrow = false
  assert.equal((await quickScan.start()).type, 'cancelled')
  assert.equal(scanCalls.length, 2)
})

test('a result that arrives after the account changed is dropped without navigation or toast', async () => {
  const pending = defer()
  const { quickScan, session, calls } = setup({ scan: () => pending.promise })

  const run = quickScan.start()
  session.authGeneration = 2
  pending.resolve(startResult())

  assert.equal((await run).type, 'stale')
  assert.deepEqual(calls, [])
  assert.equal(quickScan.takeHandoff({ userId: 7, authGeneration: 2 }), null)
  assert.equal(quickScan.takeHandoff(identity), null)
})

test('logging out or switching user during the scan also drops the result', async () => {
  for (const change of [{ isLoggedIn: false }, { userId: 8 }]) {
    const pending = defer()
    const { quickScan, session, calls } = setup({ scan: () => pending.promise })
    const run = quickScan.start()
    Object.assign(session, change)
    pending.resolve(refereeResult())
    assert.equal((await run).type, 'stale')
    assert.deepEqual(calls, [])
  }
})

test('a handoff is only released to the identity that scanned and is discarded otherwise', async () => {
  const { quickScan } = setup()
  await quickScan.start()

  assert.equal(quickScan.takeHandoff({ userId: 8, authGeneration: 1 }), null)
  assert.equal(quickScan.takeHandoff(identity), null, 'a mismatched take must not leave the credential behind')
})

test('a new scan result replaces an untaken handoff', async () => {
  let round = 0
  const { quickScan } = setup({
    scan: () => startResult({ inviteToken: `token-${++round}`, opponent: { id: round, nickname: `对手${round}`, avatar: '' } })
  })

  await quickScan.start()
  await quickScan.start()

  assert.equal(quickScan.takeHandoff(identity).inviteToken, 'token-2')
})

test('rematch only accepts the original opponent and carries the preset to the confirm page', async () => {
  const rematch = { opponent_id: 9, game_type: 1, match_mode: 'ranked', visibility: 'private', opponent_name: '对手乙' }

  const matched = setup()
  assert.deepEqual(await matched.quickScan.start({ rematch }), { type: 'confirm', mode: 'start' })
  const handoff = matched.quickScan.takeHandoff(identity)
  assert.deepEqual(handoff.preset, { gameType: 1, matchMode: 'ranked', visibility: 'public' })
  assert.equal(handoff.rematch, rematch, 'the confirm page needs the original request to rescan')

  const other = setup({ scan: () => startResult({ opponent: { id: 3, nickname: '路人', avatar: '' } }) })
  assert.equal((await other.quickScan.start({ rematch })).type, 'mismatch')
  assert.deepEqual(other.calls, [['showToast', { title: '请扫描指定对手的匹配码', icon: 'none' }]])
  assert.equal(other.quickScan.takeHandoff(identity), null)
})

test('rematch presets default the visibility from the match mode', async () => {
  const practice = setup()
  await practice.quickScan.start({ rematch: { opponent_id: 9, game_type: 3, match_mode: 'practice' } })
  assert.deepEqual(practice.quickScan.takeHandoff(identity).preset, { gameType: 3, matchMode: 'practice', visibility: 'private' })

  const publicPractice = setup()
  await publicPractice.quickScan.start({ rematch: { opponent_id: 9, game_type: 4, match_mode: 'practice', visibility: 'public' } })
  assert.deepEqual(publicPractice.quickScan.takeHandoff(identity).preset, { gameType: 4, matchMode: 'practice', visibility: 'public' })
})

test('a referee code during a rematch is still a referee code', async () => {
  const { quickScan } = setup({ scan: refereeResult })

  assert.deepEqual(await quickScan.start({ rematch: { opponent_id: 9, game_type: 3, match_mode: 'practice' } }), {
    type: 'confirm',
    mode: 'referee'
  })
  const handoff = quickScan.takeHandoff(identity)
  assert.equal(handoff.preset, null)
})

test('rescanning from the confirm page replaces the page instead of stacking another one', async () => {
  const { quickScan, calls } = setup()

  await quickScan.start({ replace: true })

  assert.deepEqual(calls, [['redirectTo', { url: SCAN_CONFIRM_ROUTE, fail: calls[0][1].fail }]])
})

test('failing to open the confirm page drops the credential and tells the user', async () => {
  const { quickScan, calls, uniApi } = setup()
  uniApi.navigateTo = (options) => {
    calls.push(['navigateTo', options.url])
    options.fail?.({ errMsg: 'navigateTo:fail webview count limit exceed' })
  }

  await quickScan.start()

  assert.deepEqual(calls.at(-1), ['showToast', { title: '打开确认页失败，请重试', icon: 'none' }])
  assert.equal(quickScan.takeHandoff(identity), null)
})
