export const ACHIEVEMENT_CATEGORY_TABS = [
  { key: 'all', label: '全部' },
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

export const filterAchievementsByCategory = (list, category) => {
  if (!Array.isArray(list)) return []
  if (!category || category === 'all') return list
  return list.filter(item => item.category === category)
}

export const getAchievementCategoryLabel = (category) => {
  return categoryLabelMap[category] || category || '其他'
}

export const getAchievementCategoryEmoji = (category) => {
  return categoryEmojiMap[category] || '🎯'
}

export const getTitleSourceLabel = (source) => {
  return titleSourceMap[source] || source || '其他'
}

export const getTitleSourceClass = (source) => {
  return titleSourceClassMap[source] || 'default'
}
