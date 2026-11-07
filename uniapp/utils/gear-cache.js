// 本人装备的用户级读取缓存：按 userId + authGeneration 隔离。
// - 退出、切号或认证代次变化：整体清空，旧身份的迟到响应不得写入。
// - 同一份数据在 TTL 内复用，并发读取共用同一个请求；写操作成功后调用 invalidate，下一次读取回源。
// - 失效发生在请求进行中时，旧响应既不写缓存，也不当作最新结果。
export const GEAR_CACHE_TTL_MS = 60 * 1000

const normalizeIdentity = (identity = {}) => ({
  userId: Number(identity?.userId) || 0,
  authGeneration: Number.isFinite(Number(identity?.authGeneration)) ? Number(identity.authGeneration) : -1
})

export const createGearCacheState = () => ({
  ownerUserId: 0,
  authGeneration: -1,
  data: null,
  // 缓存数据对应的失效版本与写入时间
  dataVersion: -1,
  fetchedAt: 0,
  // 每次 invalidate 递增；缓存只在「版本一致且未过期」时复用
  version: 0,
  inFlight: null,
  inFlightVersion: -1
})

export const createGearCacheActions = ({ requestMine, now = () => Date.now(), ttlMs = GEAR_CACHE_TTL_MS }) => ({
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

  // 读取已有缓存而不触发请求；身份不一致时返回 null，调用方不得展示别人的数据。
  peek(identity) {
    return this.isCurrent(identity) ? this.data : null
  },

  /**
   * 读取我的装备。force 为 true 时跳过 TTL 直接回源。
   * 返回 null 表示未登录或身份已变化，调用方不得再使用结果；请求失败会抛出，缓存保持不变。
   */
  async load(identity, { force = false } = {}) {
    const expected = this.ensureOwner(identity)
    if (!expected.userId) return null

    const fresh = this.data && this.dataVersion === this.version && now() - this.fetchedAt < ttlMs
    if (!force && fresh) return this.data
    if (!force && this.inFlight && this.inFlightVersion === this.version) return this.inFlight

    const startVersion = this.version
    const request = (async () => {
      try {
        const response = await requestMine()
        if (!this.isCurrent(expected)) return null
        // 请求期间已有更新的读取：旧响应不写缓存，交还最新结果。
        if (this.inFlight !== request) return this.inFlight || this.data || null
        if (!response?.success) throw new Error(response?.message || '获取装备失败')
        // 请求期间发生了写操作（失效）：结果只给本次调用展示，不当作新鲜缓存。
        if (this.version === startVersion) {
          this.data = response
          this.dataVersion = startVersion
          this.fetchedAt = now()
        }
        return response
      } finally {
        if (this.inFlight === request) {
          this.inFlight = null
          this.inFlightVersion = -1
        }
      }
    })()
    this.inFlight = request
    this.inFlightVersion = startVersion
    return request
  },

  invalidate() {
    this.version += 1
  },

  clear() {
    this.ownerUserId = 0
    this.authGeneration = -1
    this.data = null
    this.dataVersion = -1
    this.fetchedAt = 0
    this.version = 0
    this.inFlight = null
    this.inFlightVersion = -1
  }
})
