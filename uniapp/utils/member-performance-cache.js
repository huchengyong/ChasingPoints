// 表现分析的用户级读取缓存：按 userId + authGeneration + gameType 隔离。
// 数据由服务端按请求时刻的会员状态与最新结算记录生成，所以缓存只在「同一身份、同一球种、同一失效版本」内复用：
// - 退出、切号或认证代次变化：整体清空，旧身份的迟到响应不得写入。
// - 失效版本变化（对局结算、会员状态变化）：下一次读取回源，进行中的旧请求不得覆盖新版本。
export const SUPPORTED_PERFORMANCE_GAME_TYPES = [1, 2, 3, 4]

const normalizeIdentity = (identity = {}) => ({
  userId: Number(identity?.userId) || 0,
  authGeneration: Number.isFinite(Number(identity?.authGeneration)) ? Number(identity.authGeneration) : -1
})

const normalizeGameType = (gameType) => {
  const value = Number(gameType)
  return SUPPORTED_PERFORMANCE_GAME_TYPES.includes(value) ? value : 3
}

export const createPerformanceCacheState = () => ({
  ownerUserId: 0,
  authGeneration: -1,
  // { [gameType]: { data, version, inFlight, inFlightVersion } }
  entries: {}
})

export const createPerformanceCacheActions = ({ requestPerformance }) => ({
  ensureOwner(identity) {
    const expected = normalizeIdentity(identity)
    if (this.ownerUserId !== expected.userId || this.authGeneration !== expected.authGeneration) {
      this.clear()
      this.ownerUserId = expected.userId
      this.authGeneration = expected.authGeneration
    }
    return expected
  },

  isCurrent(identity) {
    const expected = normalizeIdentity(identity)
    return this.ownerUserId === expected.userId && this.authGeneration === expected.authGeneration
  },

  peek(identity, gameType) {
    if (!this.isCurrent(identity)) return null
    return this.entries[normalizeGameType(gameType)]?.data || null
  },

  /**
   * 读取表现分析。version 是调用方给出的失效版本（例如 rank 与 member 两个 scope 的版本拼接），
   * 与缓存版本相同则直接复用，否则回源；同版本的并发读取共用同一个请求。
   * 返回 null 表示身份已变化，调用方不得再使用结果。请求失败会抛出，缓存保持不变。
   */
  async load(identity, gameType, version = '') {
    const expected = this.ensureOwner(identity)
    if (!expected.userId) return null
    const type = normalizeGameType(gameType)
    const key = String(version)
    const entry = this.entries[type] || { data: null, version: null, inFlight: null, inFlightVersion: null }
    this.entries[type] = entry

    if (entry.data && entry.version === key) return entry.data
    if (entry.inFlight && entry.inFlightVersion === key) return entry.inFlight

    const request = (async () => {
      try {
        const response = await requestPerformance({ game_type: type })
        if (!this.isCurrent(expected)) return null
        const current = this.entries[type]
        // 请求期间版本又变了、已经有更新的读取：旧响应既不写缓存，也不返回给调用方，改为交还最新结果。
        if (!current) return null
        if (current.inFlight !== request) return current.inFlight || current.data || null
        if (!response?.success) throw new Error(response?.message || '获取表现分析失败')
        current.data = response
        current.version = key
        return response
      } finally {
        const current = this.entries[type]
        if (current && current.inFlight === request) {
          current.inFlight = null
          current.inFlightVersion = null
        }
      }
    })()
    entry.inFlight = request
    entry.inFlightVersion = key
    return request
  },

  clear() {
    this.ownerUserId = 0
    this.authGeneration = -1
    this.entries = {}
  }
})
