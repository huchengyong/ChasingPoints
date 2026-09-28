/**
 * WebSocket 工具类
 * 用于对局实时同步
 * 兼容 uni-app（含鸿蒙）
 */

import { NETWORK_CONFIG } from './runtime-config.js'
import {
  calculateReconnectDelay,
  classifyWebSocketFailure,
  isCurrentConnectionGeneration,
  nextConnectionGeneration,
  shouldReconnectWebSocket
} from './websocket-reconnect.js'

const WS_BASE_URL = NETWORK_CONFIG.wsBaseUrl
const DEFAULT_RECONNECT_DELAY = 3000

export const WS_MESSAGE_TYPES = {
  SCORE_UPDATE: 'score_update',
  ROUND_END: 'round_end',
  ROUND_START: 'round_start',
  MATCH_END: 'match_end',
  MATCH_START: 'match_start',
  MATCH_ROLE_CHANGED: 'match_role_changed',
  MATCH_FINISH_REQUEST: 'match_finish_request',
  MATCH_FINISH_CONFIRM: 'match_finish_confirm',
  MATCH_FINISH_DISPUTE: 'match_finish_dispute',
  MATCH_FINISH_WITHDRAW: 'match_finish_withdraw',
  MATCH_FINISH_EXPIRED: 'match_finish_expired',
  NOTIFICATION_UPDATE: 'notification_update',
  RANK_INFO_UPDATED: 'rank_info_updated',
  USER_DATA_UPDATED: 'user_data_updated',
  SESSION_INVALID: 'session_invalid',
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
    console.log('[MatchWS] ' + message)
    return
  }
  console.log('[MatchWS] ' + message, payload)
}

const errorMatchWS = (message, payload) => {
  if (payload === undefined) {
    console.error('[MatchWS] ' + message)
    return
  }
  console.error('[MatchWS] ' + message, payload)
}

const logUserWS = (message, payload) => {
  if (payload === undefined) {
    console.log('[UserWS] ' + message)
    return
  }
  console.log('[UserWS] ' + message, payload)
}

const errorUserWS = (message, payload) => {
  if (payload === undefined) {
    console.error('[UserWS] ' + message)
    return
  }
  console.error('[UserWS] ' + message, payload)
}

const summarizeSocketError = (error) => ({
  category: classifyWebSocketFailure(error),
  statusCode: Number(error?.statusCode || error?.status || 0) || undefined
})

class ManagedWebSocket {
  constructor({ log, errorLog }) {
    this.log = log
    this.errorLog = errorLog
    this.socket = null
    this.state = SOCKET_STATES.IDLE
    this.reconnectAttempts = 0
    this.reconnectDelay = DEFAULT_RECONNECT_DELAY
    this.reconnectTimer = null
    this.heartbeatInterval = null
    this.messageHandlers = new Map()
    this.isConnecting = false
    this.manualDisconnect = false
    this.connectionGeneration = 0
    this.connectPromise = null
    this.connectReject = null
    this.ticketIssuer = null
    this.target = null
    this.targetActive = false
    this.foreground = true
    this.online = true
    this.sessionInvalid = false
    this.permissionDenied = false
  }

  setTicketIssuer(ticketIssuer) {
    this.ticketIssuer = typeof ticketIssuer === 'function' ? ticketIssuer : null
  }

  isSameTarget(target) {
    return Boolean(target && this.target && target.key === this.target.key)
  }

  isCurrentGeneration(generation) {
    return isCurrentConnectionGeneration(this.connectionGeneration, generation)
  }

  isCurrentSocket(socketTask, generation) {
    return this.socket === socketTask && this.isCurrentGeneration(generation)
  }

  reconnectDecision(error = null) {
    return shouldReconnectWebSocket({
      manualDisconnect: this.manualDisconnect,
      foreground: this.foreground,
      online: this.online,
      sessionInvalid: this.sessionInvalid,
      permissionDenied: this.permissionDenied,
      targetActive: this.targetActive && Boolean(this.target),
      error
    })
  }

  connectTarget(target, options = {}) {
    const sameTarget = this.isSameTarget(target)

    if (this.socket && this.state === SOCKET_STATES.OPEN && sameTarget) {
      this.onReuse()
      return Promise.resolve()
    }
    if (this.isConnecting && sameTarget && this.connectPromise) {
      return this.connectPromise
    }
    if (this.isConnecting || this.socket) {
      this.cancelConnection({ clearTarget: false, message: '实时连接已替换' })
    }

    if (!sameTarget || options.force) {
      this.sessionInvalid = false
      this.permissionDenied = false
    }
    if (!options.reconnect) {
      this.manualDisconnect = false
    }

    this.target = target
    this.targetActive = true
    const decision = this.reconnectDecision()
    if (!decision.allow) {
      this.state = SOCKET_STATES.CLOSED
      return Promise.reject(new Error('实时连接暂不可恢复：' + decision.reason))
    }

    this.clearReconnectTimer()
    this.stopHeartbeat()
    this.isConnecting = true
    this.state = this.reconnectAttempts > 0 ? SOCKET_STATES.RECONNECTING : SOCKET_STATES.CONNECTING
    this.connectionGeneration = nextConnectionGeneration(this.connectionGeneration)
    const generation = this.connectionGeneration
    this.log('开始连接', {
      ...this.describeTarget(target),
      attempt: this.reconnectAttempts,
      state: this.state
    })

    const promise = this.openConnection(target, generation, options)
    this.connectPromise = promise
    return promise
  }

  openConnection(target, generation, options) {
    let settled = false
    let resolveOnce
    let rejectOnce
    const promise = new Promise((resolve, reject) => {
      const clearPendingConnect = () => {
        if (this.isCurrentGeneration(generation)) {
          this.connectPromise = null
          this.connectReject = null
        }
      }
      resolveOnce = () => {
        if (settled) return
        settled = true
        clearPendingConnect()
        resolve()
      }
      rejectOnce = (error) => {
        if (settled) return
        settled = true
        clearPendingConnect()
        reject(error)
      }
    })
    this.connectReject = rejectOnce

    Promise.resolve()
      .then(() => this.resolveTicket(target, options))
      .then((ticket) => {
        if (!this.isCurrentGeneration(generation) || this.manualDisconnect) {
          rejectOnce(new Error('实时连接已替换'))
          return
        }

        const socketTask = uni.connectSocket({
          url: this.buildSocketUrl(target, ticket),
          complete: () => {}
        })
        this.socket = socketTask

        socketTask.onOpen(() => {
          if (!this.isCurrentSocket(socketTask, generation)) {
            rejectOnce(new Error('实时连接已替换'))
            return
          }
          this.isConnecting = false
          this.state = SOCKET_STATES.OPEN
          this.reconnectAttempts = 0
          this.startHeartbeat()
          this.onConnected()
          this.log('连接成功', {
            ...this.describeTarget(target),
            state: this.state
          })
          resolveOnce()
        })

        socketTask.onClose((detail) => {
          if (!this.isCurrentSocket(socketTask, generation)) return
          this.stopHeartbeat()
          this.isConnecting = false
          this.socket = null
          this.state = SOCKET_STATES.CLOSED
          rejectOnce(new Error('实时连接已关闭'))
          this.log('连接关闭', {
            ...this.describeTarget(target),
            code: Number(detail?.code || detail?.statusCode || 0) || undefined
          })
          this.scheduleReconnect()
        })

        socketTask.onError((error) => {
          if (!this.isCurrentSocket(socketTask, generation)) return
          this.socket = null
          this.safeClose(socketTask)
          this.handleConnectionFailure(error, generation)
          rejectOnce(error)
        })

        socketTask.onMessage((res) => {
          if (this.isCurrentSocket(socketTask, generation)) {
            this.handleMessage(res.data)
          }
        })
      })
      .catch((error) => {
        if (this.isCurrentGeneration(generation)) {
          this.handleConnectionFailure(error, generation)
        }
        rejectOnce(error)
      })

    return promise
  }

  async resolveTicket(target, options = {}) {
    const token = String(uni.getStorageSync('token') || '')
    if (!token && target.allowAnonymous) return ''
    if (!token) {
      throw new Error('未登录')
    }

    const issuer = typeof options.ticketIssuer === 'function'
      ? options.ticketIssuer
      : (typeof target.ticketIssuer === 'function' ? target.ticketIssuer : this.ticketIssuer)
    if (!issuer) {
      throw new Error('实时连接票据服务未初始化')
    }

    const response = await this.issueTicket(issuer, target)
    const ticket = String(response?.ticket || '').trim()
    if (!ticket) {
      throw new Error('实时连接票据无效')
    }
    return ticket
  }

  handleConnectionFailure(error, generation) {
    if (!this.isCurrentGeneration(generation)) return
    this.stopHeartbeat()
    this.isConnecting = false
    this.socket = null
    this.state = SOCKET_STATES.ERROR

    const category = classifyWebSocketFailure(error)
    this.sessionInvalid = category === 'session-invalid'
    this.permissionDenied = category === 'permission-denied'
    this.errorLog('连接失败', {
      ...this.describeTarget(this.target),
      ...summarizeSocketError(error)
    })
    this.scheduleReconnect(error)
  }

  scheduleReconnect(error = null) {
    const decision = this.reconnectDecision(error)
    if (!decision.allow || this.reconnectTimer || this.isConnecting) {
      return
    }

    const target = this.target
    const generation = this.connectionGeneration
    this.reconnectAttempts += 1
    this.state = SOCKET_STATES.RECONNECTING
    const delay = calculateReconnectDelay({
      attempt: this.reconnectAttempts,
      baseDelayMs: this.reconnectDelay
    })
    this.log('准备重连', {
      ...this.describeTarget(target),
      attempt: this.reconnectAttempts,
      delay
    })

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      if (!this.isCurrentGeneration(generation) || this.target !== target) return
      if (!this.reconnectDecision().allow) return
      this.connectTarget(target, { reconnect: true }).catch((reconnectError) => {
        this.errorLog('重连失败', {
          ...this.describeTarget(target),
          ...summarizeSocketError(reconnectError)
        })
      })
    }, delay)
  }

  setForeground(foreground) {
    const nextForeground = Boolean(foreground)
    if (this.foreground === nextForeground) return
    this.foreground = nextForeground
    if (!nextForeground) {
      this.pauseConnection()
      return
    }
    this.resumeConnection()
  }

  setNetworkOnline(online) {
    const nextOnline = Boolean(online)
    if (this.online === nextOnline) return
    this.online = nextOnline
    if (!nextOnline) {
      this.pauseConnection()
      return
    }
    this.resumeConnection()
  }

  setTargetActive(active) {
    const nextActive = Boolean(active)
    if (this.targetActive === nextActive) return
    this.targetActive = nextActive
    if (!nextActive) {
      this.pauseConnection()
      return
    }
    this.resumeConnection()
  }

  pauseConnection() {
    this.cancelConnection({
      clearTarget: false,
      message: '实时连接已暂停'
    })
  }

  resumeConnection() {
    if (!this.target || !this.reconnectDecision().allow) return
    this.connectTarget(this.target, { reconnect: true }).catch((error) => {
      this.errorLog('恢复连接失败', {
        ...this.describeTarget(this.target),
        ...summarizeSocketError(error)
      })
    })
  }

  disconnect() {
    this.manualDisconnect = true
    this.targetActive = false
    this.reconnectAttempts = 0
    this.sessionInvalid = false
    this.permissionDenied = false
    this.cancelConnection({
      clearTarget: true,
      state: SOCKET_STATES.IDLE,
      message: '实时连接已断开'
    })
  }

  cancelConnection({
    clearTarget = false,
    state = SOCKET_STATES.CLOSED,
    message = '实时连接已替换'
  } = {}) {
    const activeSocket = this.socket
    const rejectPendingConnect = this.connectReject
    this.connectionGeneration = nextConnectionGeneration(this.connectionGeneration)
    this.socket = null
    this.connectPromise = null
    this.connectReject = null
    this.clearReconnectTimer()
    this.stopHeartbeat()
    this.isConnecting = false
    this.state = state

    if (clearTarget) {
      this.target = null
      this.targetActive = false
    }
    if (rejectPendingConnect) {
      rejectPendingConnect(new Error(message))
    }
    this.safeClose(activeSocket)
  }

  safeClose(socketTask) {
    if (!socketTask || typeof socketTask.close !== 'function') return
    socketTask.close({ complete: () => {} })
  }

  send(message) {
    if (!this.socket || this.state !== SOCKET_STATES.OPEN) {
      return
    }

    this.socket.send({
      data: JSON.stringify(message),
      fail: (error) => {
        this.errorLog('发送消息失败', {
          ...this.describeTarget(this.target),
          type: message.type,
          ...summarizeSocketError(error)
        })
      }
    })
  }

  handleMessage(data) {
    try {
      const message = JSON.parse(data)
      this.log('收到消息', {
        ...this.describeTarget(this.target),
        type: message.type
      })
      if (message.type === WS_MESSAGE_TYPES.SESSION_INVALID && message.data?.reason === 'SESSION_INVALID') {
        this.sessionInvalid = true
        this.clearReconnectTimer()
        this.stopHeartbeat()
        this.state = SOCKET_STATES.CLOSED
        if (typeof uni.$emit === 'function') {
          uni.$emit('session-invalid', message.data)
        }
      }
      const handlers = this.messageHandlers.get(message.type)
      if (handlers) {
        handlers.forEach(handler => handler(message.data))
      }

      const allHandlers = this.messageHandlers.get('*')
      if (allHandlers) {
        allHandlers.forEach(handler => handler(message))
      }
    } catch (error) {
      this.errorLog('解析消息失败', {
        ...this.describeTarget(this.target),
        ...summarizeSocketError(error)
      })
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
    if (!handlers) return

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

  isConnected() {
    return this.state === SOCKET_STATES.OPEN && this.socket !== null
  }

  describeTarget() {
    return {}
  }

  onConnected() {}

  onReuse() {}
}

class MatchWebSocket extends ManagedWebSocket {
  constructor() {
    super({ log: logMatchWS, errorLog: errorMatchWS })
    this.matchId = null
    this.allowAnonymous = false
  }

  connect(matchId, options = {}) {
    const allowAnonymous = Boolean(options.allowAnonymous)
    this.matchId = matchId
    this.allowAnonymous = allowAnonymous
    return this.connectTarget({
      key: 'match:' + String(matchId) + ':' + String(allowAnonymous),
      matchId,
      allowAnonymous,
      ticketIssuer: options.ticketIssuer
    }, options)
  }

  issueTicket(issuer, target) {
    return issuer(target.matchId)
  }

  buildSocketUrl(target, ticket) {
    const query = ['match_id=' + encodeURIComponent(target.matchId)]
    if (ticket) {
      query.push('ticket=' + encodeURIComponent(ticket))
    }
    return WS_BASE_URL + '/api/match/ws?' + query.join('&')
  }

  describeTarget(target = this.target) {
    return target ? {
      matchId: target.matchId,
      allowAnonymous: target.allowAnonymous
    } : {}
  }

  onConnected() {
    this.requestSync()
  }

  onReuse() {
    this.requestSync()
  }

  requestSync() {
    this.log('请求同步快照', this.describeTarget())
    this.send({ type: WS_MESSAGE_TYPES.SYNC })
  }

  disconnect() {
    super.disconnect()
    this.matchId = null
    this.allowAnonymous = false
  }
}

export const matchWS = new MatchWebSocket()

class UserWebSocket extends ManagedWebSocket {
  constructor() {
    super({ log: logUserWS, errorLog: errorUserWS })
    this.authGeneration = null
  }

  connect(options = {}) {
    const requestedGeneration = Number(options.authGeneration)
    if (Number.isFinite(requestedGeneration) && this.authGeneration !== null && this.authGeneration !== requestedGeneration) {
      this.disconnect()
    }
    if (Number.isFinite(requestedGeneration)) {
      this.authGeneration = requestedGeneration
    }

    return this.connectTarget({
      key: 'user',
      allowAnonymous: false,
      ticketIssuer: options.ticketIssuer
    }, options)
  }

  issueTicket(issuer) {
    return issuer()
  }

  buildSocketUrl(_target, ticket) {
    return WS_BASE_URL + '/api/user/ws?ticket=' + encodeURIComponent(ticket)
  }

  disconnect() {
    super.disconnect()
    this.authGeneration = null
  }
}

export const userWS = new UserWebSocket()

export const configureWebSocketTickets = ({ issueMatchWSTicket, issueUserWSTicket } = {}) => {
  matchWS.setTicketIssuer(issueMatchWSTicket)
  userWS.setTicketIssuer(issueUserWSTicket)
}

export default matchWS
