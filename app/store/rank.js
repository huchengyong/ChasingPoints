import { defineStore } from 'pinia'
import { getUserRankInfos } from '@/api/rank.js'
import { createRankCacheActions, createRankCacheState, normalizeUserRankInfos } from '@/utils/rank-cache.js'

export const useRankStore = defineStore('rank', {
  state: createRankCacheState,
  actions: createRankCacheActions({
    requestRankInfos: getUserRankInfos,
    normalizeRankInfos: normalizeUserRankInfos
  })
})
