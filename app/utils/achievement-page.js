export const ACHIEVEMENT_CATEGORY_GROUPS = [
  { key: 'wins', label: '胜场' },
  { key: 'streak', label: '连胜' },
  { key: 'special', label: '特殊' },
  { key: 'match', label: '对局' },
  { key: 'tournament', label: '赛事' }
]

const categoryLabelMap = {
  wins: '胜场',
  streak: '连胜',
  special: '特殊',
  match: '对局',
  tournament: '赛事',
  social: '社交'
}

const categoryEmojiMap = {
  wins: '🏅',
  streak: '🔥',
  special: '⭐',
  match: '🎱',
  tournament: '🏆',
  social: '👥'
}

const titleSourceMap = {
  achievement: '成就',
  season: '赛季',
  tournament: '赛事',
  成就: '成就',
  赛季: '赛季',
  赛事: '赛事'
}

const titleSourceClassMap = {
  achievement: 'achievement',
  season: 'season',
  tournament: 'tournament',
  成就: 'achievement',
  赛季: 'season',
  赛事: 'tournament'
}

const titleSourceDescriptionMap = {
  achievement: '生涯成就',
  season: '赛季荣誉',
  tournament: '赛事荣誉',
  成就: '生涯成就',
  赛季: '赛季荣誉',
  赛事: '赛事荣誉'
}

export const getAchievementCategoryLabel = (category) => {
  return categoryLabelMap[category] || category || '其他'
}

export const getAchievementCategoryEmoji = (category) => {
  return categoryEmojiMap[category] || '🎯'
}

export const groupAchievementsByCategory = (list) => {
  if (!Array.isArray(list)) return []

  const bucket = new Map()
  list.forEach((item) => {
    const key = item && item.category ? item.category : 'other'
    if (!bucket.has(key)) bucket.set(key, [])
    bucket.get(key).push(item)
  })

  const knownKeys = new Set(ACHIEVEMENT_CATEGORY_GROUPS.map(group => group.key))
  const groups = ACHIEVEMENT_CATEGORY_GROUPS
    .filter(group => bucket.has(group.key))
    .map(group => ({
      ...group,
      list: bucket.get(group.key)
    }))

  bucket.forEach((items, key) => {
    if (knownKeys.has(key)) return
    const label = getAchievementCategoryLabel(key)
    groups.push({ key, label, list: items })
  })

  return groups
}

export const getTitleSourceLabel = (source) => {
  return titleSourceMap[source] || source || '其他'
}

export const getTitleSourceClass = (source) => {
  return titleSourceClassMap[source] || 'default'
}

export const getTitleSourceDescription = (title = {}) => {
  const source = title.source_type || title.source
  const label = titleSourceDescriptionMap[source] || getTitleSourceLabel(source)
  const sourceName = String(title.source_ref_name || '').trim()
  return sourceName ? `${label} · ${sourceName}` : label
}

export const sortTitleOptions = (list) => {
  if (!Array.isArray(list)) return []

  const result = [...list]
  const equippedIndex = result.findIndex(item => item?.equipped)
  if (equippedIndex <= 0) return result

  const [equipped] = result.splice(equippedIndex, 1)
  return [equipped, ...result]
}

export const resolveTitleSelection = ({ currentTitleId = 0, selectedTitleId = 0 } = {}) => {
  const currentId = Number(currentTitleId) > 0 ? Number(currentTitleId) : 0
  const selectedId = Number(selectedTitleId) > 0 ? Number(selectedTitleId) : 0

  if (currentId === selectedId) return { action: 'close' }
  if (selectedId > 0) return { action: 'equip', titleId: selectedId }
  return currentId > 0 ? { action: 'unequip', titleId: currentId } : { action: 'close' }
}
