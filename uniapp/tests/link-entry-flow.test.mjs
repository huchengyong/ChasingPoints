import test from 'node:test'
import assert from 'node:assert/strict'

import { createLinkEntryFlow } from '../utils/link-entry-flow.js'
import { SCAN_CONFIRM_ROUTE } from '../utils/quick-scan.js'

const code = 'Ab3dEf6hIj9k'
const previewResponse = {
  success: true,
  preview: { inviter_name: '邀请人', inviter_avatar: 'a.png', expires_at: '2030-01-01T00:00:00Z' }
}
const exchangeResponse = {
  success: true,
  invite_token: 'signed.token',
  preview: { opponent_id: 9, opponent_name: '邀请人', opponent_avatar: 'a.png' }
}

const defer = () => {
  let resolve
  const promise = new Promise((resolvePromise) => { resolve = resolvePromise })
  return { promise, resolve }
}

const setup = ({ loggedIn = false, userId = 7, preview, exchange } = {}) => {
  const session = { isLoggedIn: loggedIn, userId, authGeneration: 1 }
  const calls = { preview: 0, exchange: 0, handoff: [] }
  const flow = createLinkEntryFlow({
    getSession: () => ({ ...session }),
    previewPublicMatchInvite: async () => {
      calls.preview += 1
      return preview ? preview() : previewResponse
    },
    resolveMatchInviteCode: async () => {
      calls.exchange += 1
      return exchange ? exchange() : exchangeResponse
    },
    handoffResolved: (payload) => {
      calls.handoff.push(payload)
      return true
    }
  })
  return { flow, session, calls }
}

test('a guest sees the inviter first and exchanges only after logging in', async () => {
  const { flow, session, calls } = setup({ loggedIn: false })

  const first = await flow.begin(code)
  assert.equal(first.type, 'await_login')
  assert.equal(first.inviter.name, '邀请人')
  assert.equal(calls.exchange, 0)
  assert.equal(calls.preview, 1)

  // 放弃登录直接返回：仍然留在预览态，不兑换。
  assert.equal((await flow.resume()).type, 'ignore')
  assert.equal(calls.exchange, 0)

  session.isLoggedIn = true
  const afterLogin = await flow.resume()
  assert.equal(afterLogin.type, 'confirm')
  assert.equal(afterLogin.route, SCAN_CONFIRM_ROUTE)
  assert.equal(calls.exchange, 1)
  assert.equal(calls.handoff[0].inviteToken, 'signed.token')
  assert.deepEqual(calls.handoff[0].opponent, { id: 9, nickname: '邀请人', avatar: 'a.png' })
})

test('an already logged in visitor exchanges immediately without previewing', async () => {
  const { flow, calls } = setup({ loggedIn: true })

  const action = await flow.begin(code)

  assert.equal(action.type, 'confirm')
  assert.equal(calls.preview, 0)
  assert.equal(calls.exchange, 1)
})

test('the code expires while the guest is logging in', async () => {
  const { flow, session, calls } = setup({
    loggedIn: false,
    exchange: () => ({ success: false, message: '匹配二维码已失效，请让对方刷新二维码' })
  })

  await flow.begin(code)
  session.isLoggedIn = true
  const action = await flow.resume()

  assert.equal(action.type, 'expired')
  assert.equal(action.message, '匹配二维码已失效，请让对方刷新二维码')
  assert.equal(calls.handoff.length, 0)
})

test('repeated resume calls do not exchange twice', async () => {
  const { flow, session, calls } = setup({ loggedIn: true })

  await flow.begin(code)
  const again = await flow.resume()

  assert.equal(again.type, 'ignore')
  assert.equal(calls.exchange, 1)
  assert.equal(calls.handoff.length, 1)
  assert.equal(session.isLoggedIn, true)
})

test('a late exchange response is dropped after the account changed', async () => {
  const pending = defer()
  const { flow, session, calls } = setup({
    loggedIn: true,
    exchange: () => pending.promise
  })

  const running = flow.begin(code)
  session.userId = 8
  pending.resolve(exchangeResponse)
  const action = await running

  assert.equal(action.type, 'ignore')
  assert.equal(calls.handoff.length, 0)
})

test('a failed preview reports an expired invitation', async () => {
  const { flow, calls } = setup({
    loggedIn: false,
    preview: () => ({ success: false, message: '匹配二维码已失效，请让对方刷新二维码' })
  })

  const action = await flow.begin(code)

  assert.equal(action.type, 'expired')
  assert.equal(calls.exchange, 0)
})

test('a network failure never hands over a credential', async () => {
  const { flow, calls } = setup({
    loggedIn: true,
    exchange: () => { throw new Error('network') }
  })

  const action = await flow.begin(code)

  assert.equal(action.type, 'error')
  assert.equal(calls.handoff.length, 0)
})

test('a network failure can be retried and then hands over once', async () => {
  let failing = true
  const { flow, calls } = setup({
    loggedIn: true,
    exchange: () => {
      if (failing) throw new Error('network')
      return exchangeResponse
    }
  })

  assert.equal((await flow.begin(code)).type, 'error')
  failing = false
  const action = await flow.retry()

  assert.equal(action.type, 'confirm')
  assert.equal(calls.handoff.length, 1)
  assert.equal((await flow.retry()).type, 'ignore')
})

test('a guest preview network failure is an error, not an expired code', async () => {
  let failing = true
  const { flow } = setup({
    preview: () => {
      if (failing) throw new Error('network')
      return previewResponse
    }
  })

  const failed = await flow.begin(code)
  assert.equal(failed.type, 'error')

  failing = false
  const action = await flow.retry()
  assert.equal(action.type, 'await_login')
})

test('an expired code cannot be retried', async () => {
  const { flow, calls } = setup({ preview: () => ({ success: false, message: '匹配二维码已失效，请让对方刷新二维码' }) })

  assert.equal((await flow.begin(code)).type, 'expired')
  assert.equal((await flow.retry()).type, 'ignore')
  assert.equal(calls.preview, 1)
})
