/**
 * 持久化会话恢复校验（纯逻辑，依赖注入以便单测）
 */

export const isRetryableSessionCheckError = (error = {}) => {
  const category = error.category || ''
  return category === 'network' || category === 'server'
}

export const canApplySessionRecoveryResult = ({
  expectedGeneration,
  currentGeneration,
  expectedLifecycleGeneration,
  currentLifecycleGeneration,
  isLoggedIn = false,
  isForeground = false
} = {}) => {
  if (!isLoggedIn || !isForeground || expectedGeneration !== currentGeneration) {
    return false
  }
  if (
    expectedLifecycleGeneration !== undefined &&
    currentLifecycleGeneration !== undefined &&
    expectedLifecycleGeneration !== currentLifecycleGeneration
  ) {
    return false
  }
  return true
}

/**
 * 创建按 auth generation 隔离的会话校验器：同一会话并发调用共享结果，
 * 不同账号世代各自请求。helper 只返回资料，由 App 在复核当前会话后应用。
 */
export const createSessionRecovery = ({ getUserInfo, onSessionInvalid } = {}) => {
  const inFlightByGeneration = new Map()

  const validate = (generation = 'default') => {
    const generationKey = String(generation)
    const existing = inFlightByGeneration.get(generationKey)
    if (existing) return existing

    const promise = (async () => {
      try {
        const response = await getUserInfo(generation)
        if (!response?.success || !response?.user_info || typeof response.user_info !== 'object') {
          const error = new Error(response?.message || '会话校验失败')
          error.category = 'business'
          throw error
        }
        return {
          valid: true,
          retryable: false,
          userInfo: response.user_info
        }
      } catch (error) {
        if (isRetryableSessionCheckError(error)) {
          return { valid: false, retryable: true }
        }
        if ((error?.category === 'session' || error?._isHandled) && typeof onSessionInvalid === 'function') {
          await onSessionInvalid(error)
        }
        return { valid: false, retryable: false }
      }
    })().finally(() => {
      if (inFlightByGeneration.get(generationKey) === promise) {
        inFlightByGeneration.delete(generationKey)
      }
    })

    inFlightByGeneration.set(generationKey, promise)
    return promise
  }

  return { validate }
}
