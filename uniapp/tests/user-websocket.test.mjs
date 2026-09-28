import test from 'node:test'
import assert from 'node:assert/strict'

const createSocketTask = () => {
  const handlers = {}
  const sent = []
  let closeCalls = 0
  return {
    onOpen(handler) {
      handlers.open = handler
    },
    onClose(handler) {
      handlers.close = handler
    },
    onError(handler) {
      handlers.error = handler
    },
    onMessage(handler) {
      handlers.message = handler
    },
    close({ complete } = {}) {
      closeCalls += 1
      complete?.()
    },
    send({ data } = {}) {
      sent.push(data)
    },
    emitOpen() {
      handlers.open?.()
    },
    emitClose(detail = {}) {
      handlers.close?.(detail)
    },
    emitError(detail = {}) {
      handlers.error?.(detail)
    },
    emitMessage(data) {
      handlers.message?.({ data })
    },
    get sent() {
      return sent
    },
    get closeCalls() {
      return closeCalls
    }
  }
}

const flushMicrotasks = async () => {
  await Promise.resolve()
  await Promise.resolve()
  await Promise.resolve()
}

test('user websocket reuses an in-flight connection and ignores an obsolete close event', async (t) => {
  let token = 'first-token'
  let ticketIndex = 0
  const sockets = []
  const urls = []
  globalThis.uni = {
    getStorageSync(key) {
      return key === 'token' ? token : ''
    },
    connectSocket(options) {
      const socket = createSocketTask()
      sockets.push(socket)
      urls.push(options.url)
      return socket
    }
  }

  const { userWS } = await import('../utils/websocket.js')
  userWS.setTicketIssuer(async () => ({ ticket: `ticket-${++ticketIndex}` }))
  t.after(() => {
    userWS.disconnect()
    delete globalThis.uni
  })

  const first = userWS.connect()
  const shared = userWS.connect()
  assert.equal(shared, first)
  await flushMicrotasks()
  assert.equal(sockets.length, 1)
  sockets[0].emitOpen()
  await first

  userWS.disconnect()
  token = 'second-token'
  const second = userWS.connect()
  await flushMicrotasks()
  assert.equal(sockets.length, 2)
  sockets[0].emitClose()
  sockets[1].emitOpen()
  await second

  assert.equal(userWS.isConnected(), true)
  assert.deepEqual(urls.map((url) => url.includes('token=')), [false, false])
  assert.deepEqual(urls.map((url) => url.match(/ticket=([^&]+)/)?.[1]), ['ticket-1', 'ticket-2'])
})

test('match websocket connects with a fresh ticket instead of query access token', async (t) => {
  const sockets = []
  const urls = []
  globalThis.uni = {
    getStorageSync(key) {
      return key === 'token' ? 'match-access-token' : ''
    },
    connectSocket(options) {
      const socket = createSocketTask()
      sockets.push(socket)
      urls.push(options.url)
      return socket
    }
  }

  const { matchWS } = await import('../utils/websocket.js')
  matchWS.setTicketIssuer(async (matchId) => ({ ticket: `match-ticket-${matchId}` }))
  t.after(() => {
    matchWS.disconnect()
    delete globalThis.uni
  })

  const connecting = matchWS.connect(42)
  await flushMicrotasks()
  assert.equal(sockets.length, 1)
  sockets[0].emitOpen()
  await connecting

  assert.equal(matchWS.isConnected(), true)
  assert.equal(urls[0].includes('token='), false)
  assert.match(urls[0], /match_id=42/)
  assert.match(urls[0], /ticket=match-ticket-42/)
})

test('user websocket keeps retrying beyond five failures and recovers with a new ticket', async (t) => {
  const sockets = []
  let ticketAttempts = 0
  globalThis.uni = {
    getStorageSync(key) {
      return key === 'token' ? 'access-token' : ''
    },
    connectSocket() {
      const socket = createSocketTask()
      sockets.push(socket)
      return socket
    }
  }

  const { userWS } = await import('../utils/websocket.js')
  userWS.setTicketIssuer(async () => {
    ticketAttempts += 1
    if (ticketAttempts === 1) {
      const error = new Error('network unavailable')
      error.category = 'network'
      throw error
    }
    return { ticket: 'ticket-' + String(ticketAttempts) }
  })
  t.after(() => {
    userWS.disconnect()
    delete globalThis.uni
  })

  await assert.rejects(userWS.connect(), /network unavailable/)
  assert.equal(userWS.reconnectAttempts, 1)
  userWS.clearReconnectTimer()

  userWS.reconnectAttempts = 5
  userWS.scheduleReconnect()
  assert.equal(userWS.reconnectAttempts, 6)
  userWS.clearReconnectTimer()

  const recovering = userWS.connect({ reconnect: true })
  await flushMicrotasks()
  assert.equal(sockets.length, 1)
  sockets[0].emitOpen()
  await recovering
  assert.equal(userWS.isConnected(), true)
  assert.equal(ticketAttempts, 2)
})

test('ticket session and permission failures stop automatic reconnect', async (t) => {
  globalThis.uni = {
    getStorageSync(key) {
      return key === 'token' ? 'access-token' : ''
    },
    connectSocket() {
      throw new Error('terminal ticket failure must not connect')
    }
  }

  const { matchWS, userWS } = await import('../utils/websocket.js')
  userWS.setTicketIssuer(async () => {
    const error = new Error('session invalid')
    error.category = 'session'
    error.statusCode = 401
    throw error
  })
  matchWS.setTicketIssuer(async () => {
    const error = new Error('permission denied')
    error.category = 'forbidden'
    error.statusCode = 403
    throw error
  })
  t.after(() => {
    userWS.disconnect()
    matchWS.disconnect()
    delete globalThis.uni
  })

  await assert.rejects(userWS.connect(), /session invalid/)
  assert.equal(userWS.sessionInvalid, true)
  assert.equal(userWS.reconnectTimer, null)

  await assert.rejects(matchWS.connect(42), /permission denied/)
  assert.equal(matchWS.permissionDenied, true)
  assert.equal(matchWS.reconnectTimer, null)
})

test('server session-invalid message stops reconnect and emits global session cleanup signal', async (t) => {
  const sockets = []
  const events = []
  globalThis.uni = {
    getStorageSync(key) {
      return key === 'token' ? 'access-token' : ''
    },
    connectSocket() {
      const socket = createSocketTask()
      sockets.push(socket)
      return socket
    },
    $emit(name, payload) {
      events.push({ name, payload })
    }
  }

  const { userWS } = await import('../utils/websocket.js')
  userWS.setTicketIssuer(async () => ({ ticket: 'user-ticket' }))
  t.after(() => {
    userWS.disconnect()
    delete globalThis.uni
  })

  const connecting = userWS.connect()
  await flushMicrotasks()
  sockets[0].emitOpen()
  await connecting

  sockets[0].emitMessage(JSON.stringify({ type: 'session_invalid', data: { reason: 'SESSION_INVALID' } }))
  assert.equal(userWS.sessionInvalid, true)
  assert.equal(userWS.reconnectTimer, null)
  assert.equal(userWS.state, 'closed')
  assert.deepEqual(events, [{ name: 'session-invalid', payload: { reason: 'SESSION_INVALID' } }])

  sockets[0].emitClose()
  assert.equal(userWS.reconnectTimer, null)
})

test('foreground, network, account switch, target exit, and manual disconnect invalidate old callbacks', async (t) => {
  const sockets = []
  let ticketIndex = 0
  globalThis.uni = {
    getStorageSync(key) {
      return key === 'token' ? 'access-token' : ''
    },
    connectSocket() {
      const socket = createSocketTask()
      sockets.push(socket)
      return socket
    }
  }

  const { matchWS, userWS } = await import('../utils/websocket.js')
  userWS.setTicketIssuer(async () => ({ ticket: 'user-ticket-' + String(++ticketIndex) }))
  matchWS.setTicketIssuer(async () => ({ ticket: 'match-ticket-' + String(++ticketIndex) }))
  t.after(() => {
    userWS.disconnect()
    matchWS.disconnect()
    delete globalThis.uni
  })

  const firstUserConnect = userWS.connect({ authGeneration: 1 })
  await flushMicrotasks()
  sockets[0].emitOpen()
  await firstUserConnect

  userWS.setForeground(false)
  assert.equal(sockets[0].closeCalls, 1)
  assert.equal(userWS.reconnectTimer, null)
  assert.equal(userWS.heartbeatInterval, null)

  userWS.setForeground(true)
  await flushMicrotasks()
  sockets[1].emitOpen()
  assert.equal(userWS.isConnected(), true)

  userWS.setNetworkOnline(false)
  assert.equal(sockets[1].closeCalls, 1)
  userWS.setNetworkOnline(true)
  await flushMicrotasks()
  sockets[2].emitOpen()

  const switchedAccount = userWS.connect({ authGeneration: 2 })
  await flushMicrotasks()
  sockets[2].emitClose()
  sockets[3].emitOpen()
  await switchedAccount
  assert.equal(userWS.isConnected(), true)

  const matchConnect = matchWS.connect(42)
  await flushMicrotasks()
  sockets[4].emitOpen()
  await matchConnect
  assert.match(sockets[4].sent[0], /"type":"sync"/)

  matchWS.setTargetActive(false)
  assert.equal(sockets[4].closeCalls, 1)
  assert.equal(matchWS.reconnectTimer, null)
  matchWS.setTargetActive(true)
  await flushMicrotasks()
  sockets[5].emitOpen()
  assert.equal(matchWS.isConnected(), true)

  matchWS.disconnect()
  sockets[5].emitClose()
  assert.equal(matchWS.reconnectTimer, null)
  assert.equal(matchWS.manualDisconnect, true)
})
