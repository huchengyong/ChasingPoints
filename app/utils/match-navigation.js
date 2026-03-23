export const getMatchHistoryTabUrl = () => '/pages/user/index'

export const getMatchHistoryPageUrl = () => '/subPages/user/matchHistory'

export const consumeResultNavigationGuard = (state = {}) => {
  if (state.hasNavigatedToResult) {
    return false
  }
  state.hasNavigatedToResult = true
  return true
}

export const shouldLeavePlayingPage = ({
  pageMatchId,
  detailStatus,
  currentMatchId
} = {}) => {
  const normalizedPageMatchId = Number(pageMatchId) || 0
  if (!normalizedPageMatchId) return false

  const normalizedDetailStatus = Number(detailStatus) || 0
  if (normalizedDetailStatus && normalizedDetailStatus !== 1) {
    return true
  }

  if (normalizedDetailStatus === 1) {
    if (currentMatchId === undefined || currentMatchId === null) {
      return false
    }

    return (Number(currentMatchId) || 0) !== normalizedPageMatchId
  }

  return false
}
