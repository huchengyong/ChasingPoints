/**
 * WebSocket 工具类
 * 用于对局实时同步
 * 兼容 uni-app（含鸿蒙）
 */

const WS_BASE_URL = 'wss://api-tunnel2.kekemate.com'

export const WS_MESSAGE_TYPES = {
  SCORE_UPDATE: 'score_update',
  ROUND_END: 'round_end',
  ROUND_START: 'round_start',
  MATCH_END: 'match_end',
  MATCH_START: 'match_start',
  NOTIFICATION_UPDATE: 'notification_update',
  SYNC: 'sync',
  PING: 'ping',
  PONG: 'pong'
}

const SOCKET_STATES = {
  IDLE: 'idle',
  CONNECTING: 'connecting',
  OPEN: 'open',
  RECONNECTING: 'reconnecting',
  CLOSED: 'closed',
  ERROR: 'error'
}

const logMatchWS = (message, payload) => {
  if (payload === undefined) {
    console.log(`[MatchWS] ${message}`)
    return
  }
  console.log(`[MatchWS] ${message}`, payload)
}

const warnMatchWS = (message, payload) => {
  if (payload === undefined) {
    console.warn(`[MatchWS] ${message}`)
    return
  }
  console.warn(`[MatchWS] ${message}`, payload)
}

const errorMatchWS = (message, payload) => {
  if (payload === undefined) {
    console.error(`[MatchWS] ${message}`)
    return
  }
  console.error(`[MatchWS] ${message}`, payload)
}

const logUserWS = (message, payload) => {
  if (payload === undefined) {
    console.log(`[UserWS] ${message}`)
    return
  }
  console.log(`[UserWS] ${message}`, payload)
}

const errorUserWS = (message, payload) => {
  if (payload === undefined) {
    console.error(`[UserWS] ${message}`)
    return
  }
  console.error(`[UserWS] ${message}`, payload)
}

class MatchWebSocket {
  constructor() {
    this.socket = null
    this.matchId = null
    this.allowAnonymous = false
    this.state = SOCKET_STATES.IDLE
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 5
    this.reconnectDelay = 3000
    this.reconnectTimer = null
    this.heartbeatInterval = null
    this.messageHandlers = new Map()
    this.isConnecting = false
    this.manualDisconnect = false
  }

  connect(matchId, options = {}) {
    const { allowAnonymous = false } = options

    return new Promise((resolve, reject) => {
      if (this.socket && this.matchId === matchId && this.state === SOCKET_STATES.OPEN) {
        this.requestSync()
        resolve()
        return
      }

      if (this.isConnecting) {
        reject(new Error('正在连接中'))
        return
      }

      const token = uni.getStorageSync('token')
      if (!token && !allowAnonymous) {
        reject(new Error('未登录'))
        return
      }

      this.clearReconnectTimer()
      this.stopHeartbeat()
      this.manualDisconnect = false
      this.isConnecting = true
      this.state = this.reconnectAttempts > 0 ? SOCKET_STATES.RECONNECTING : SOCKET_STATES.CONNECTING
      this.matchId = matchId
      this.allowAnonymous = allowAnonymous
      logMatchWS('开始连接', { matchId, allowAnonymous, attempt: this.reconnectAttempts, state: this.state })

      const query = [`match_id=${matchId}`]
      if (token) {
        query.push(`token=${token}`)
      }

      const socketTask = uni.connectSocket({
        url: `${WS_BASE_URL}/api/match/ws?${query.join('&')}`,
        complete: () => {}
      })

      this.socket = socketTask

      let settled = false
      const resolveOnce = () => {
        if (!settled) {
          settled = true
          resolve()
        }
      }
      const rejectOnce = (error) => {
        if (!settled) {
          settled = true
          reject(error)
        }
      }

      socketTask.onOpen(() => {
        this.isConnecting = false
        this.state = SOCKET_STATES.OPEN
        this.reconnectAttempts = 0
        this.startHeartbeat()
        this.requestSync()
        logMatchWS('连接成功', { matchId: this.matchId, state: this.state })
        resolveOnce()
      })

      socketTask.onClose((res) => {
        const shouldReconnect = !this.manualDisconnect
        const closedMatchId = this.matchId
        this.cleanupClosedSocket()
        logMatchWS('连接关闭', { matchId: closedMatchId, shouldReconnect, detail: res })
        if (shouldReconnect) {
          this.handleReconnect()
        }
      })

      socketTask.onError((err) => {
        this.isConnecting = false
        this.state = SOCKET_STATES.ERROR
        errorMatchWS('连接错误', { matchId: this.matchId, detail: err })
        rejectOnce(err)
      })

      socketTask.onMessage((res) => {
        this.handleMessage(res.data)
      })
    })
  }

  disconnect() {
    this.manualDisconnect = true
    this.clearReconnectTimer()
    this.stopHeartbeat()
    this.isConnecting = false
    this.state = SOCKET_STATES.IDLE

    const activeSocket = this.socket
    this.socket = null
    this.matchId = null
    this.allowAnonymous = false
    this.reconnectAttempts = 0

    if (activeSocket) {
      activeSocket.close({
        complete: () => {
          logMatchWS('主动断开完成', { matchId: this.matchId })
        }
      })
    }
  }

  send(message) {
    if (!this.socket || this.state !== SOCKET_STATES.OPEN) {
      return
    }

    this.socket.send({
      data: JSON.stringify(message),
      fail: (err) => {
        errorMatchWS('发送消息失败', { matchId: this.matchId, type: message.type, detail: err })
      }
    })
  }

  requestSync() {
    logMatchWS('请求同步快照', { matchId: this.matchId })
    this.send({ type: WS_MESSAGE_TYPES.SYNC })
  }

  handleMessage(data) {
    try {
      const message = JSON.parse(data)
      logMatchWS('收到消息', { matchId: this.matchId, type: message.type })
      const handlers = this.messageHandlers.get(message.type)
      if (handlers) {
        handlers.forEach(handler => handler(message.data))
      }

      const allHandlers = this.messageHandlers.get('*')
      if (allHandlers) {
        allHandlers.forEach(handler => handler(message))
      }
    } catch (err) {
      errorMatchWS('解析消息失败', { matchId: this.matchId, detail: err, raw: data })
    }
  }

  on(type, handler) {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, [])
    }

    const handlers = this.messageHandlers.get(type)
    if (!handlers.includes(handler)) {
      handlers.push(handler)
    }
  }

  off(type, handler) {
    const handlers = this.messageHandlers.get(type)
    if (!handlers) {
      return
    }

    const index = handlers.indexOf(handler)
    if (index > -1) {
      handlers.splice(index, 1)
    }
  }

  startHeartbeat() {
    this.stopHeartbeat()
    this.heartbeatInterval = setInterval(() => {
      this.send({ type: WS_MESSAGE_TYPES.PING })
    }, 30000)
  }

  stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval)
      this.heartbeatInterval = null
    }
  }

  clearReconnectTimer() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  cleanupClosedSocket() {
    this.stopHeartbeat()
    this.isConnecting = false
    this.socket = null
    this.state = SOCKET_STATES.CLOSED
  }

  handleReconnect() {
    if (this.manualDisconnect || !this.matchId) {
      return
    }

    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      warnMatchWS('重连次数已达上限', { matchId: this.matchId, maxReconnectAttempts: this.maxReconnectAttempts })
      return
    }

    const targetMatchId = this.matchId
    const allowAnonymous = this.allowAnonymous

    this.reconnectAttempts += 1
    this.clearReconnectTimer()
    this.state = SOCKET_STATES.RECONNECTING
    logMatchWS('准备重连', { matchId: targetMatchId, attempt: this.reconnectAttempts, delay: this.reconnectDelay })

    this.reconnectTimer = setTimeout(() => {
      this.connect(targetMatchId, { allowAnonymous }).catch(err => {
        errorMatchWS('重连失败', { matchId: targetMatchId, detail: err })
      })
    }, this.reconnectDelay)
  }

  isConnected() {
    return this.state === SOCKET_STATES.OPEN && this.socket !== null
  }
}

export const matchWS = new MatchWebSocket()

class UserWebSocket {
  constructor() {
    this.socket = null
    this.state = SOCKET_STATES.IDLE
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 5
    this.reconnectDelay = 3000
    this.reconnectTimer = null
    this.heartbeatInterval = null
    this.messageHandlers = new Map()
    this.isConnecting = false
    this.manualDisconnect = false
  }

  connect() {
    return new Promise((resolve, reject) => {
      if (this.socket && this.state === SOCKET_STATES.OPEN) {
        resolve()
        return
      }

      if (this.isConnecting) {
        reject(new Error('正在连接中'))
        return
      }

      const token = uni.getStorageSync('token')
      if (!token) {
        reject(new Error('未登录'))
        return
      }

      this.clearReconnectTimer()
      this.stopHeartbeat()
      this.manualDisconnect = false
      this.isConnecting = true
      this.state = this.reconnectAttempts > 0 ? SOCKET_STATES.RECONNECTING : SOCKET_STATES.CONNECTING
      logUserWS('开始连接', { attempt: this.reconnectAttempts, state: this.state })

      const socketTask = uni.connectSocket({
        url: `${WS_BASE_URL}/api/user/ws?token=${token}`,
        complete: () => {}
      })

      this.socket = socketTask

      let settled = false
      const resolveOnce = () => {
        if (!settled) {
          settled = true
          resolve()
        }
      }
      const rejectOnce = (error) => {
        if (!settled) {
          settled = true
          reject(error)
        }
      }

      socketTask.onOpen(() => {
        this.isConnecting = false
        this.state = SOCKET_STATES.OPEN
        this.reconnectAttempts = 0
        this.startHeartbeat()
        logUserWS('连接成功', { state: this.state })
        resolveOnce()
      })

      socketTask.onClose((res) => {
        const shouldReconnect = !this.manualDisconnect
        this.cleanupClosedSocket()
        logUserWS('连接关闭', { shouldReconnect, detail: res })
        if (shouldReconnect) {
          this.handleReconnect()
        }
      })

      socketTask.onError((err) => {
        this.isConnecting = false
        this.state = SOCKET_STATES.ERROR
        errorUserWS('连接错误', { detail: err })
        rejectOnce(err)
      })

      socketTask.onMessage((res) => {
        this.handleMessage(res.data)
      })
    })
  }

  disconnect() {
    this.manualDisconnect = true
    this.clearReconnectTimer()
    this.stopHeartbeat()
    this.isConnecting = false
    this.state = SOCKET_STATES.IDLE

    const activeSocket = this.socket
    this.socket = null
    this.reconnectAttempts = 0

    if (activeSocket) {
      activeSocket.close({
        complete: () => {
          logUserWS('主动断开完成')
        }
      })
    }
  }

  send(message) {
    if (!this.socket || this.state !== SOCKET_STATES.OPEN) {
      return
    }

    this.socket.send({
      data: JSON.stringify(message),
      fail: (err) => {
        errorUserWS('发送消息失败', { type: message.type, detail: err })
      }
    })
  }

  handleMessage(data) {
    try {
      const message = JSON.parse(data)
      logUserWS('收到消息', { type: message.type })
      const handlers = this.messageHandlers.get(message.type)
      if (handlers) {
        handlers.forEach(handler => handler(message.data))
      }

      const allHandlers = this.messageHandlers.get('*')
      if (allHandlers) {
        allHandlers.forEach(handler => handler(message))
      }
    } catch (err) {
      errorUserWS('解析消息失败', { detail: err, raw: data })
    }
  }

  on(type, handler) {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, [])
    }

    const handlers = this.messageHandlers.get(type)
    if (!handlers.includes(handler)) {
      handlers.push(handler)
    }
  }

  off(type, handler) {
    const handlers = this.messageHandlers.get(type)
    if (!handlers) {
      return
    }

    const index = handlers.indexOf(handler)
    if (index > -1) {
      handlers.splice(index, 1)
    }
  }

  startHeartbeat() {
    this.stopHeartbeat()
    this.heartbeatInterval = setInterval(() => {
      this.send({ type: WS_MESSAGE_TYPES.PING })
    }, 30000)
  }

  stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval)
      this.heartbeatInterval = null
    }
  }

  clearReconnectTimer() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  cleanupClosedSocket() {
    this.stopHeartbeat()
    this.isConnecting = false
    this.socket = null
    this.state = SOCKET_STATES.CLOSED
  }

  handleReconnect() {
    if (this.manualDisconnect) {
      return
    }

    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.warn('[UserWS] 重连次数已达上限', { maxReconnectAttempts: this.maxReconnectAttempts })
      return
    }

    this.reconnectAttempts += 1
    this.clearReconnectTimer()
    this.state = SOCKET_STATES.RECONNECTING
    logUserWS('准备重连', { attempt: this.reconnectAttempts, delay: this.reconnectDelay })

    this.reconnectTimer = setTimeout(() => {
      this.connect().catch(err => {
        errorUserWS('重连失败', { detail: err })
      })
    }, this.reconnectDelay)
  }

  isConnected() {
    return this.state === SOCKET_STATES.OPEN && this.socket !== null
  }
}

export const userWS = new UserWebSocket()

export default matchWS
