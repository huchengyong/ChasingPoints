const resolveSnookerFrameSettleAction = ({
  currentFrameStarted,
  myFrameScore,
  opponentFrameScore,
  continueOnZeroTie = false,
  successAction,
  blockedMessage
} = {}) => {
  if (!currentFrameStarted) {
    return { action: successAction }
  }

  const myScore = Number(myFrameScore) || 0
  const opponentScore = Number(opponentFrameScore) || 0

  if (myScore === opponentScore) {
    if (continueOnZeroTie && myScore === 0) {
      return { action: successAction }
    }
    return {
      action: 'blocked',
      message: blockedMessage
    }
  }

  return {
    action: `settle_and_${successAction}`,
    winner: myScore > opponentScore ? 1 : 2
  }
}

export const resolveSnookerNextFrameAction = (payload = {}) => resolveSnookerFrameSettleAction({
  ...payload,
  successAction: 'start_next_round',
  blockedMessage: '当前局比分相同，无法自动判定胜负，请继续记分或结束本场对局'
})

export const resolveSnookerFinishMatchAction = (payload = {}) => resolveSnookerFrameSettleAction({
  ...payload,
  continueOnZeroTie: true,
  successAction: 'finish_match',
  blockedMessage: '当前局比分相同，无法自动判定胜负，请继续记分后再结束本场对局'
})
