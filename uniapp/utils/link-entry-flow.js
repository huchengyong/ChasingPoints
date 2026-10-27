// 入口链接承接页的状态流转：先匿名预览，登录后兑换一次并交接给确认页。
// 纯逻辑（只用相对路径引用），会话、接口和交接都由调用方注入。
import { SCAN_CONFIRM_ROUTE } from './quick-scan.js'

const expiredMessage = '匹配二维码已失效，请让对方刷新二维码'
const unavailableMessage = '匹配二维码服务暂不可用，请稍后重试'

export const createLinkEntryFlow = ({
  getSession,
  previewPublicMatchInvite,
  resolveMatchInviteCode,
  handoffResolved
}) => {
  let inviteCode = ''
  let started = false
  let resolving = false
  let settled = false

  const readIdentity = () => {
    const session = getSession()
    return session.isLoggedIn
      ? { userId: Number(session.userId) || 0, authGeneration: Number(session.authGeneration) }
      : null
  }

  const isCurrent = (identity) => {
    const current = readIdentity()
    return Boolean(current) && current.userId === identity.userId && current.authGeneration === identity.authGeneration
  }

  const preview = async () => {
    try {
      const response = await previewPublicMatchInvite(inviteCode)
      if (!response?.success || !response.preview) {
        settled = true
        return { type: 'expired', message: response?.message || expiredMessage }
      }
      return {
        type: 'await_login',
        inviter: {
          name: response.preview.inviter_name || '球友',
          avatar: response.preview.inviter_avatar || ''
        },
        expiresAt: response.preview.expires_at || ''
      }
    } catch {
      // 网络或服务异常不代表短码失效：不结束流程，页面可以重试。
      return { type: 'error', message: unavailableMessage }
    }
  }

  const exchange = async (identity) => {
    if (resolving || settled) return { type: 'ignore' }
    resolving = true
    try {
      const response = await resolveMatchInviteCode(inviteCode)
      // 迟到响应：登录期间换了账号或登出，直接丢弃，不交接凭证。
      if (!isCurrent(identity)) return { type: 'ignore' }
      if (!response?.success || !response.invite_token || !response.preview?.opponent_id) {
        settled = true
        return { type: 'expired', message: response?.message || expiredMessage }
      }
      const handed = handoffResolved({
        identity,
        inviteToken: response.invite_token,
        opponent: {
          id: Number(response.preview.opponent_id),
          nickname: response.preview.opponent_name || '对手',
          avatar: response.preview.opponent_avatar || ''
        }
      })
      if (!handed) return { type: 'ignore' }
      settled = true
      return { type: 'confirm', route: SCAN_CONFIRM_ROUTE }
    } catch {
      if (!isCurrent(identity)) return { type: 'ignore' }
      return { type: 'error', message: unavailableMessage }
    } finally {
      resolving = false
    }
  }

  // 进入承接页时调用：已登录直接兑换，未登录先展示邀请人。
  const begin = async (code) => {
    inviteCode = String(code || '')
    started = true
    const identity = readIdentity()
    if (identity) return exchange(identity)
    return preview()
  }

  // 登录页返回或页面重新显示时调用；每次只兑换一次。
  const resume = async () => {
    if (!started || settled || resolving) return { type: 'ignore' }
    const identity = readIdentity()
    if (!identity) return { type: 'ignore' }
    return exchange(identity)
  }

  // 网络失败后由页面「重试」触发：重新走一遍进入流程，短码已确认失效或已交接时不再请求。
  const retry = async () => {
    if (!started || settled || resolving) return { type: 'ignore' }
    return begin(inviteCode)
  }

  return { begin, resume, retry }
}
