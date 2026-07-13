export const resolvePlayingViewerUi = ({
  viewerRole = 'player1',
  canScore = true,
  canUndo = true,
  canFinish = true,
  refereeName = ''
} = {}) => {
  if (viewerRole === 'referee') {
    return {
      leftIdentity: '选手1',
      rightIdentity: '选手2',
      subtitleSuffix: '左选手1右选手2',
      roleLabel: '裁判视角',
      showActionPanel: !!canScore,
      showUndoButton: !!canUndo,
      showFinishButton: !!canFinish,
      readonlyHint: ''
    }
  }

  return {
    leftIdentity: '我方',
    rightIdentity: '对手',
    subtitleSuffix: '左我右敌',
    roleLabel: '选手视角',
    showActionPanel: !!canScore,
    showUndoButton: !!canUndo,
    showFinishButton: !!canFinish,
    readonlyHint: !canScore ? `本场由${refereeName || '裁判'}负责记分` : ''
  }
}
