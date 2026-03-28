export const buildH2HHistoryParams = ({
  opponentId = 0,
  fallbackOpponentId = 0,
  opponentName = '',
  page = 1,
  pageSize = 20,
  result = 0
} = {}) => {
  const resolvedOpponentId = Number(opponentId) || Number(fallbackOpponentId) || 0
  const params = {
    page,
    page_size: pageSize
  }

  if (resolvedOpponentId > 0) {
    params.opponent_id = resolvedOpponentId
  } else if (opponentName) {
    params.opponent_name = opponentName
  }

  if (result > 0) {
    params.result = result
  }

  return params
}
