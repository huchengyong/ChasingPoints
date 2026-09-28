const normalizeActor = (actor) => actor === 1 ? 1 : 2

export const resolveMatchApiActor = ({ viewerRole = 'player1', isPlayer1 = true, uiActor = 2 } = {}) => {
  const actor = normalizeActor(uiActor)
  if (viewerRole === 'referee' || isPlayer1) return actor
  return actor === 1 ? 2 : 1
}

export const resolveScoringParticipantLabel = ({ viewerRole = 'player1', uiActor = 2 } = {}) => {
  const actor = normalizeActor(uiActor)
  if (viewerRole === 'referee') return actor === 1 ? '选手1' : '选手2'
  return actor === 1 ? '我方' : '对手'
}

export const buildRoundWinActionOptions = ({ gameType, viewerRole = 'player1', uiActor = 2 } = {}) => {
  const label = resolveScoringParticipantLabel({ viewerRole, uiActor })
  if (gameType === 4) {
    return [
      { type: 'normal', score: 1, title: `${label}普胜`, desc: `本局计给${label}` },
      { type: 'small_gold', score: 1, title: `${label}小金`, desc: `${label}金球直接制胜` },
      { type: 'big_gold', score: 1, title: `${label}大金`, desc: `${label}开球直接制胜` }
    ]
  }
  return [
    { type: 'normal', score: 1, title: `${label}普胜`, desc: `本局计给${label}` },
    { type: 'break_clear', score: 1, title: `${label}炸清`, desc: `${label}直接清台获胜` },
    { type: 'continue_clear', score: 1, title: `${label}接清`, desc: `${label}连续清台获胜` }
  ]
}

export const buildJiuqiuWinOptions = ({ viewerRole = 'player1', uiActor = 2 } = {}) => {
  const label = resolveScoringParticipantLabel({ viewerRole, uiActor })
  return [
    { type: 'normal', score: 4, title: `${label}普胜`, desc: `${label} +4 分` },
    { type: 'small_gold', score: 7, title: `${label}小金`, desc: `${label} +7 分` },
    { type: 'big_gold', score: 10, title: `${label}大金`, desc: `${label} +10 分` }
  ]
}

export const resolveScoreFeedback = ({ viewerRole = 'player1', uiActor = 2, score = 0 } = {}) =>
  `已给${resolveScoringParticipantLabel({ viewerRole, uiActor })} +${score} 分`

export const resolveWinFeedback = ({ viewerRole = 'player1', uiActor = 2 } = {}) =>
  `已判给${resolveScoringParticipantLabel({ viewerRole, uiActor })}`

export const resolveFoulFeedback = ({ viewerRole = 'player1', foulingActor = 2, score = 1 } = {}) => {
  const foulActor = normalizeActor(foulingActor)
  const scoringActor = foulActor === 1 ? 2 : 1
  const foulLabel = resolveScoringParticipantLabel({ viewerRole, uiActor: foulActor })
  const scoringLabel = resolveScoringParticipantLabel({ viewerRole, uiActor: scoringActor })
  return `${foulLabel}犯规，${scoringLabel} +${score} 分`
}
