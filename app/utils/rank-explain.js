export const resolveRankExplainLoadingMode = ({
  hasLoadedOnce = false,
  isFetching = false
} = {}) => {
  if (!isFetching) return 'idle'
  return hasLoadedOnce ? 'refreshing' : 'initial'
}

export const shouldApplyRankExplainResponse = ({
  requestId = 0,
  latestRequestId = 0
} = {}) => requestId === latestRequestId
