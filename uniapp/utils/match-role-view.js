export const resolvePlayingViewerUi = ({
  viewerRole = 'player1',
  canScore = true,
  canUndo = true,
  canFinish = true,
  refereeName = '',
  canRequestFinish,
  canConfirmFinish,
  canDisputeFinish,
  canWithdrawFinish,
  finishState = 'none',
  lastAction = null
} = {}) => {
  const hasFinishCapabilities = [
    canRequestFinish,
    canConfirmFinish,
    canDisputeFinish,
    canWithdrawFinish,
    finishState !== 'none',
    lastAction !== null
  ].some(value => value !== undefined && value !== false)

  const withFinishCapabilities = (base) => {
    if (!hasFinishCapabilities) return base
    return {
      ...base,
      readonlyHint: finishState === 'pending_confirmation' ? '等待对手处理结束确认' : base.readonlyHint,
      showFinishRequestButton: !!canRequestFinish,
      showFinishConfirmButton: !!canConfirmFinish,
      showFinishDisputeButton: !!canDisputeFinish,
      showFinishWithdrawButton: !!canWithdrawFinish,
      finishState,
      lastAction
    }
  }

  if (viewerRole === 'referee') {
    return withFinishCapabilities({
      leftIdentity: '选手1',
      rightIdentity: '选手2',
      subtitleSuffix: '左选手1右选手2',
      roleLabel: '裁判视角',
      showActionPanel: !!canScore,
      showUndoButton: !!canUndo,
      showFinishButton: !!canFinish,
      readonlyHint: ''
    })
  }

  return withFinishCapabilities({
    leftIdentity: '我方',
    rightIdentity: '对手',
    subtitleSuffix: '左我右敌',
    roleLabel: '选手视角',
    showActionPanel: !!canScore,
    showUndoButton: !!canUndo,
    showFinishButton: !!canFinish,
    readonlyHint: !canScore ? `本场由${refereeName || '裁判'}负责记分` : ''
  })
}
