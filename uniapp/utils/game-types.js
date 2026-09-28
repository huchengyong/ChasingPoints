const ORDERED_GAME_TYPE_META = [
  { value: 3, key: 'chinese_eight', category: 'chinese_eight', label: '中式八球' },
  { value: 2, key: 'nine_ball', category: 'nine_ball', label: '九球追分' },
  { value: 1, key: 'snooker', category: 'snooker', label: '斯诺克' },
  { value: 4, key: 'american_nine', category: 'american_nine', label: '美式九球' }
]

export const ORDERED_GAME_TYPES = ORDERED_GAME_TYPE_META.map((item) => ({ ...item }))

export const GAME_TYPE_OPTIONS = ORDERED_GAME_TYPE_META.map(({ value, label }) => ({ value, label }))

export const GAME_TYPE_TABS = GAME_TYPE_OPTIONS.map((item) => ({ ...item }))

export const GAME_TYPE_FILTER_OPTIONS_WITH_ALL = [
  { value: 0, label: '全部球种' },
  ...GAME_TYPE_OPTIONS.map((item) => ({ ...item }))
]

export const GAME_TYPE_STATS_TABS = ORDERED_GAME_TYPE_META.map(({ key, label }) => ({ key, label }))

export const GAME_TYPE_LABEL_MAP = Object.freeze(
  Object.fromEntries(ORDERED_GAME_TYPE_META.map(({ value, label }) => [value, label]))
)

export const GAME_TYPE_KEY_MAP = Object.freeze(
  Object.fromEntries(ORDERED_GAME_TYPE_META.map(({ value, key }) => [value, key]))
)

export const GAME_TYPE_VALUE_MAP = Object.freeze(
  Object.fromEntries(ORDERED_GAME_TYPE_META.map(({ value, key }) => [key, value]))
)

export const RULE_CATEGORY_LABEL_MAP = Object.freeze(
  Object.fromEntries(ORDERED_GAME_TYPE_META.map(({ category, label }) => [category, label]))
)

export const getGameTypeLabel = (gameType, fallback = '台球') => {
  return GAME_TYPE_LABEL_MAP[Number(gameType)] || fallback
}

export const getRuleCategoryLabel = (category, fallback = '') => {
  return RULE_CATEGORY_LABEL_MAP[category] || fallback || category
}
