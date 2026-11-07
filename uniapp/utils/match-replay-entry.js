// 「单场复盘」入口：只对已完成对局的选手展示；资格最终由服务端判断（非选手、未完成的对局一律拒绝）。
const PLAYER_ROLES = ['player1', 'player2']

export const buildMatchReplayUrl = (matchId) => `/subPages/match/matchReplay?match_id=${Number(matchId) || 0}`

/**
 * @param {{ status?: number, viewerRole?: string, source?: string, perspectiveUserId?: number, currentUserId?: number }} input
 * - 本人对局详情带 viewer_role，player1 / player2 即选手，裁判没有入口。
 * - 战绩列表进入的历史详情走公开接口，没有 viewer_role：视角用户就是当前登录用户时，才认为是本人的对局。
 */
export const canOpenMatchReplay = ({ status = 0, viewerRole = '', source = '', perspectiveUserId = 0, currentUserId = 0 } = {}) => {
  if (Number(status) !== 2) return false
  if (PLAYER_ROLES.includes(viewerRole)) return true
  if (viewerRole === 'referee') return false
  const perspective = Number(perspectiveUserId) || 0
  return source === 'history' && perspective > 0 && perspective === (Number(currentUserId) || 0)
}
