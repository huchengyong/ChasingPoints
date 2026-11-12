// 与后端 adminEventNewsPlayerNameZhMaxRunes 保持一致
export const PLAYER_NAME_ZH_MAX_LENGTH = 32

export const normalizePlayerNameZh = (value: string | undefined | null): string => String(value ?? '').trim()

// 去掉首尾空白后与已保存的值比较；清空也算修改（表示清除中文名）
export const isPlayerNameZhChanged = (saved: string, draft: string | undefined | null): boolean =>
  normalizePlayerNameZh(draft) !== normalizePlayerNameZh(saved)
