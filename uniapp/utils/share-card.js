// 分享卡片：只依据页面已经加载的数据同步生成，不在分享时发请求。
// 公开对局带比分与球种；私密、已取消或数据未加载时退回通用卡片。
import { getGameTypeLabel } from './game-types.js'

const GENERIC_TITLE = '追分 · 记录每一局真实比分'
const HOME_PATH = '/pages/index/index'
const NAME_LIMIT = 8

const truncateName = (name) => {
  const text = String(name || '').trim() || '球友'
  const chars = Array.from(text)
  return chars.length > NAME_LIMIT ? `${chars.slice(0, NAME_LIMIT).join('')}…` : text
}

export const buildHomeShareCard = () => ({
  title: GENERIC_TITLE,
  path: HOME_PATH
})

export const buildMatchShareCard = (detail = {}) => {
  const matchId = Number(detail.match_id || detail.id || 0)
  if (!matchId || detail.visibility !== 'public' || Number(detail.status) === 3) {
    return buildHomeShareCard()
  }

  // 统一成选手 1/选手 2 的顺序，与公开详情的第三方视角一致，不因分享者身份交换。
  // 公开详情给 player1_*；参与者详情只给 is_player1 + my_*/opponent_*。
  const isPublicShape = detail.player1_name !== undefined || detail.player1_score !== undefined
  const isPlayer1 = detail.is_player1 === true
  const player1Name = truncateName(isPublicShape
    ? detail.player1_name
    : (isPlayer1 ? detail.my_name : detail.opponent_name))
  const player2Name = truncateName(isPublicShape
    ? detail.player2_name
    : (isPlayer1 ? detail.opponent_name : detail.my_name))
  const player1Score = Number(isPublicShape
    ? detail.player1_score
    : (isPlayer1 ? detail.my_score : detail.opponent_score)) || 0
  const player2Score = Number(isPublicShape
    ? detail.player2_score
    : (isPlayer1 ? detail.opponent_score : detail.my_score)) || 0

  const gameType = getGameTypeLabel(detail.game_type)
  const title = Number(detail.status) === 2
    ? `${player1Name} ${player1Score}:${player2Score} ${player2Name} · ${gameType}${detail.match_mode === 'practice' ? '练习赛' : '排位赛'}`
    : `${player1Name} vs ${player2Name} 正在比赛 · ${gameType}`

  return {
    title,
    path: `/subPages/match/matchDetail?match_id=${matchId}&mode=spectate`
  }
}
