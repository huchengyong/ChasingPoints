export const resolveStatsDetailLoadingMode = ({
  hasLoadedOnce = false,
  isFetching = false
} = {}) => {
  if (!isFetching) return 'idle'
  return hasLoadedOnce ? 'refreshing' : 'initial'
}

export const shouldApplyStatsDetailResponse = ({
  requestId = 0,
  latestRequestId = 0
} = {}) => requestId === latestRequestId
