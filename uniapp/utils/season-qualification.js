// 赛季达标进度、本季净变化与榜单分页合并。门槛来自服务端赛季信息，旧赛季门槛为 0（全员参与排名）。

const toCount = (value) => Math.max(0, Number(value) || 0)

export const resolveSeasonQualification = (season = null, record = null) => {
  const minMatches = toCount(season?.qualify_min_matches)
  const minOpponents = toCount(season?.qualify_min_opponents)
  const hasThreshold = minMatches > 0
  const matches = toCount(record?.matches_played)
  const opponents = toCount(record?.distinct_opponents)
  return {
    hasThreshold,
    minMatches,
    minOpponents,
    qualified: !hasThreshold || record?.qualified === true,
    progressText: hasThreshold
      ? `场次 ${Math.min(matches, minMatches)}/${minMatches} · 不同对手 ${Math.min(opponents, minOpponents)}/${minOpponents}`
      : '',
    thresholdText: hasThreshold
      ? `本季打满 ${minMatches} 场、遇到 ${minOpponents} 位不同对手即参与赛季排名`
      : ''
  }
}

export const formatSeasonNetChange = (record = null) => {
  if (!record) return ''
  const delta = (Number(record.end_rank_score) || 0) - (Number(record.start_rank_score) || 0)
  return delta > 0 ? `+${delta}` : String(delta)
}

// 分页之间名次可能变化，同一用户只保留首次出现，避免重复行。
export const mergeLeaderboardPage = (current = [], incoming = []) => {
  const seen = new Set(current.map((item) => item.user_id))
  const merged = [...current]
  for (const item of incoming) {
    if (seen.has(item.user_id)) continue
    seen.add(item.user_id)
    merged.push(item)
  }
  return merged
}

// 服务端对未达标记录返回 rank = 0，不显示名次。
export const resolveLeaderboardRank = (item = {}) => {
  const rank = Number(item.rank) || 0
  return {
    rank,
    text: rank > 0 ? String(rank) : '—',
    unqualified: item.qualified === false
  }
}
