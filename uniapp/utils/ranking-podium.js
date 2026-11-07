const PODIUM_PLACEMENTS = ['second', 'first', 'third']
const PODIUM_SOURCE_INDEX_MAP = {
  second: 1,
  first: 0,
  third: 2
}

export const buildLeaderboardPodiumSlots = (topThree = []) => {
  const normalizedTopThree = Array.isArray(topThree) ? topThree.slice(0, 3) : []

  return PODIUM_PLACEMENTS.map((placement) => ({
    key: placement,
    placement,
    rank: PODIUM_SOURCE_INDEX_MAP[placement] + 1,
    user: normalizedTopThree[PODIUM_SOURCE_INDEX_MAP[placement]] || null
  }))
}
