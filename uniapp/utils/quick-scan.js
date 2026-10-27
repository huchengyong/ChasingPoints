// 统一扫码流程：中间扫码按钮、首页「扫描对手二维码」和赛果页「再来一局」共用。
// 先扫码，再按码类型进入确认页（subPages/match/scanConfirm）：对手匹配码确认球种与对局类型后开局，裁判码预览后确认执裁。
// 纯逻辑（只用相对路径引用）：会话、预览接口和 uni API 都由调用方注入，接线见 quick-scan-entry.js。
import { scanAndResolveMatchCode } from './match-scan.js'
import { normalizeStartMatchOptions } from './start-match.js'

export const SCAN_CONFIRM_ROUTE = '/subPages/match/scanConfirm'

const LOGIN_ROUTE = '/pages/login/login'

const sameIdentity = (left, right) => (
  left.userId === right.userId && left.authGeneration === right.authGeneration
)

const buildRematchPreset = (rematch) => {
  const options = normalizeStartMatchOptions({ matchMode: rematch.match_mode, visibility: rematch.visibility })
  return {
    gameType: Number(rematch.game_type) || 0,
    matchMode: options.match_mode,
    visibility: options.visibility
  }
}

export const createQuickScan = ({
  uniApi,
  getSession,
  previewMatchInvite,
  resolveMatchInviteCode,
  scan = scanAndResolveMatchCode
}) => {
  let scanning = false
  // 凭证只在内存里交给确认页使用一次，不进路由、本地存储或日志。
  let handoff = null

  const readIdentity = () => {
    const session = getSession()
    return session.isLoggedIn
      ? { userId: Number(session.userId) || 0, authGeneration: Number(session.authGeneration) }
      : null
  }

  const isCurrent = (identity) => {
    const current = readIdentity()
    return Boolean(current) && sameIdentity(identity, current)
  }

  const toast = (title) => uniApi.showToast({ title, icon: 'none' })

  const openConfirmPage = (replace) => {
    const options = {
      url: SCAN_CONFIRM_ROUTE,
      fail: () => {
        handoff = null
        toast('打开确认页失败，请重试')
      }
    }
    if (replace) uniApi.redirectTo(options)
    else uniApi.navigateTo(options)
  }

  // rematch：赛果页「再来一局」传入上一场的上下文，只接受原对手的匹配码；replace：确认页重新扫码时替换自身。
  const start = async ({ rematch = null, replace = false } = {}) => {
    if (scanning) return { type: 'busy' }
    const identity = readIdentity()
    if (!identity) {
      uniApi.navigateTo({ url: LOGIN_ROUTE })
      return { type: 'login' }
    }

    scanning = true
    try {
      const result = await scan({ uniApi, previewMatchInvite, resolveMatchInviteCode })
      if (!isCurrent(identity)) return { type: 'stale' }
      if (result.type === 'cancelled') return result
      if (result.type === 'error') {
        toast(result.message || '扫码失败')
        return result
      }

      if (result.type === 'open_match') {
        uniApi.navigateTo({ url: `/subPages/match/matchDetail?match_id=${result.matchId}&mode=spectate` })
        return { type: 'spectate' }
      }

      if (result.type === 'join_referee') {
        handoff = { identity, mode: 'referee', refereeJoin: result.refereeJoin, preset: null, rematch: null }
        openConfirmPage(replace)
        return { type: 'confirm', mode: 'referee' }
      }

      const expectedOpponentId = Number(rematch?.opponent_id) || 0
      const scannedOpponentId = Number(result.opponent?.id) || 0
      if (expectedOpponentId > 0 && scannedOpponentId > 0 && scannedOpponentId !== expectedOpponentId) {
        toast('请扫描指定对手的匹配码')
        return { type: 'mismatch' }
      }

      handoff = {
        identity,
        mode: 'start',
        inviteToken: result.inviteToken,
        opponent: result.opponent,
        preset: rematch ? buildRematchPreset(rematch) : null,
        rematch
      }
      openConfirmPage(replace)
      return { type: 'confirm', mode: 'start' }
    } finally {
      scanning = false
    }
  }

  // 只交给扫码时的身份；取走或身份不符都会清掉，凭证不会滞留。
  const takeHandoff = (identity) => {
    const pending = handoff
    handoff = null
    if (!pending || !identity) return null
    const current = { userId: Number(identity.userId) || 0, authGeneration: Number(identity.authGeneration) }
    return sameIdentity(pending.identity, current) ? pending : null
  }

  // 承接页（入口链接）兑换成功后交接凭证，不走扫码；身份已变化时丢弃。
  const handoffResolved = ({ identity, inviteToken, opponent } = {}) => {
    if (!inviteToken || !isCurrent(identity)) return false
    handoff = {
      identity: { userId: Number(identity.userId) || 0, authGeneration: Number(identity.authGeneration) },
      mode: 'start',
      inviteToken,
      opponent,
      preset: null,
      rematch: null
    }
    return true
  }

  return { start, takeHandoff, handoffResolved }
}
