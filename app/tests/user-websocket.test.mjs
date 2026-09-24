import test from 'node:test'
import assert from 'node:assert/strict'

const createSocketTask = () => {
  const handlers = {}
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
      complete?.()
    },
    send() {},
    emitOpen() {
      handlers.open?.()
    },
    emitClose(detail = {}) {
      handlers.close?.(detail)
    }
  }
}

test('user websocket reuses an in-flight connection and ignores an obsolete close event', async (t) => {
  let token = 'first-token'
  const sockets = []
  globalThis.uni = {
    getStorageSync(key) {
      return key === 'token' ? token : ''
    },
    connectSocket() {
      const socket = createSocketTask()
      sockets.push(socket)
      return socket
    }
  }

  const { userWS } = await import('../utils/websocket.js')
  t.after(() => {
    userWS.disconnect()
    delete globalThis.uni
  })

  const first = userWS.connect()
  const shared = userWS.connect()
  assert.equal(shared, first)
  assert.equal(sockets.length, 1)
  sockets[0].emitOpen()
  await first

  userWS.disconnect()
  token = 'second-token'
  const second = userWS.connect()
  assert.equal(sockets.length, 2)
  sockets[0].emitClose()
  sockets[1].emitOpen()
  await second

  assert.equal(userWS.isConnected(), true)
})
