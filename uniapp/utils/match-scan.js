import { scanQrCode } from './permission-helper.js'
import { parseEntryLink } from './entry-link.js'

const inviteTokenPattern = /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/
const urlPattern = /^https?:\/\//i

export const parseMatchScanPayload = (rawValue = '') => {
  const value = String(rawValue || '').trim()
  if (!value) {
    return { type: 'error', message: '无效的二维码' }
  }

  // 入口链接先分流：只有匹配码和对局两种能进业务，其他网址一律无效。
  if (urlPattern.test(value)) {
    const entry = parseEntryLink(value)
    if (entry.type === 'invite') {
      return { type: 'resolve_invite_code', inviteCode: entry.inviteCode }
    }
    if (entry.type === 'match') {
      return { type: 'open_match', matchId: entry.matchId }
    }
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

const buildStartMatch = (inviteToken, preview) => ({
  type: 'start_match',
  inviteToken,
  opponent: {
    id: Number(preview.opponent_id),
    nickname: preview.opponent_name || '对手',
    avatar: preview.opponent_avatar || ''
  }
})

export const resolveMatchScanPayload = async ({
  rawValue,
  previewMatchInvite,
  resolveMatchInviteCode
} = {}) => {
  const parsed = parseMatchScanPayload(rawValue)

  if (parsed.type === 'resolve_invite_code') {
    if (typeof resolveMatchInviteCode !== 'function') {
      return { type: 'error', message: '匹配二维码服务暂不可用' }
    }
    try {
      const response = await resolveMatchInviteCode(parsed.inviteCode)
      if (!response?.success || !response.invite_token || !response.preview?.opponent_id) {
        return {
          type: 'error',
          message: response?.message || '匹配二维码已失效，请让对方刷新二维码'
        }
      }
      return buildStartMatch(response.invite_token, response.preview)
    } catch {
      return { type: 'error', message: '匹配二维码服务暂不可用，请稍后重试' }
    }
  }

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
    return buildStartMatch(parsed.inviteToken, response.preview)
  } catch {
    return { type: 'error', message: '匹配二维码服务暂不可用，请稍后重试' }
  }
}

export const scanAndResolveMatchCode = async ({
  uniApi = globalThis.uni,
  previewMatchInvite,
  resolveMatchInviteCode
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
    previewMatchInvite,
    resolveMatchInviteCode
  })
}
