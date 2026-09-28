import { scanQrCode } from './permission-helper.js'

const inviteTokenPattern = /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/

export const parseMatchScanPayload = (rawValue = '') => {
  const value = String(rawValue || '').trim()
  if (!value) {
    return { type: 'error', message: '无效的二维码' }
  }

  try {
    const payload = JSON.parse(value)
    if (payload?.type === 'match_referee' && payload?.match_id && payload?.join_token) {
      return {
        type: 'join_referee',
        refereeJoin: {
          match_id: Number(payload.match_id),
          join_token: String(payload.join_token)
        }
      }
    }
    return {
      type: 'error',
      message: '该二维码已不再支持，请让对方刷新匹配二维码'
    }
  } catch {
    if (!inviteTokenPattern.test(value)) {
      return { type: 'error', message: '无效的二维码' }
    }
    return {
      type: 'preview_match_invite',
      inviteToken: value
    }
  }
}

export const resolveMatchScanPayload = async ({
  rawValue,
  previewMatchInvite
} = {}) => {
  const parsed = parseMatchScanPayload(rawValue)
  if (parsed.type !== 'preview_match_invite') return parsed
  if (typeof previewMatchInvite !== 'function') {
    return { type: 'error', message: '匹配二维码服务暂不可用' }
  }

  try {
    const response = await previewMatchInvite(parsed.inviteToken)
    if (!response?.success || !response.preview?.opponent_id) {
      return {
        type: 'error',
        message: response?.message || '匹配二维码已失效，请让对方刷新二维码'
      }
    }
    return {
      type: 'start_match',
      inviteToken: parsed.inviteToken,
      opponent: {
        id: Number(response.preview.opponent_id),
        nickname: response.preview.opponent_name || '对手',
        avatar: response.preview.opponent_avatar || ''
      }
    }
  } catch {
    return { type: 'error', message: '匹配二维码服务暂不可用，请稍后重试' }
  }
}

export const scanAndResolveMatchCode = async ({
  uniApi = globalThis.uni,
  previewMatchInvite
} = {}) => {
  const scanResult = await scanQrCode({ uniApi })
  if (!scanResult.success) {
    return {
      type: scanResult.reason === 'cancelled' ? 'cancelled' : 'error',
      reason: scanResult.reason,
      message: scanResult.reason === 'permission-denied'
        ? '需要相机权限才能扫码'
        : scanResult.reason === 'unsupported'
          ? '当前端暂不支持扫码'
          : scanResult.message || '扫码失败'
    }
  }
  return resolveMatchScanPayload({
    rawValue: scanResult.result?.result,
    previewMatchInvite
  })
}
