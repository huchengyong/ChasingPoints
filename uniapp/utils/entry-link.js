// 入口链接：https://www.zhuifen.cn/go/?j=<短码> / ?m=<对局ID> / 无参数首页。
// 纯函数，不读写全局状态，便于 node 直接测试。
const ENTRY_BASE = 'https://www.zhuifen.cn/go/'
const entryLinkPattern = /^https:\/\/www\.zhuifen\.cn\/go\/?(?:\?([^#]*))?(?:#.*)?$/i
const inviteCodePattern = /^[0-9A-Za-z]{12}$/

const home = () => ({ type: 'home' })

export const buildEntryLink = ({ type = 'home', value } = {}) => {
  if (type === 'invite') return `${ENTRY_BASE}?j=${value}`
  if (type === 'match') return `${ENTRY_BASE}?m=${value}`
  return ENTRY_BASE
}

const readQuery = (query) => {
  const params = {}
  String(query || '').split('&').forEach((pair) => {
    if (!pair) return
    const index = pair.indexOf('=')
    const key = index < 0 ? pair : pair.slice(0, index)
    if (key in params) return
    params[key] = index < 0 ? '' : pair.slice(index + 1)
  })
  return params
}

export const parseEntryLink = (raw) => {
  const matched = entryLinkPattern.exec(String(raw == null ? '' : raw).trim())
  if (!matched) return { type: 'other' }

  const params = readQuery(matched[1])
  const hasCode = Object.prototype.hasOwnProperty.call(params, 'j')
  const hasMatchId = Object.prototype.hasOwnProperty.call(params, 'm')
  if (hasCode && hasMatchId) return home()

  if (hasCode) {
    return inviteCodePattern.test(params.j) ? { type: 'invite', inviteCode: params.j } : home()
  }
  if (hasMatchId) {
    const matchId = Number(params.m)
    return /^[1-9][0-9]*$/.test(params.m) && Number.isSafeInteger(matchId)
      ? { type: 'match', matchId }
      : home()
  }
  return home()
}

// 微信启动参数 q 是原链接的 URL 编码，少数场景会编码两次；缺失或解码失败时原样返回。
export const decodeLaunchQuery = (q) => {
  let value = String(q == null ? '' : q)
  for (let round = 0; round < 2; round += 1) {
    try {
      const decoded = decodeURIComponent(value)
      if (decoded === value) break
      value = decoded
    } catch {
      break
    }
  }
  return value
}
