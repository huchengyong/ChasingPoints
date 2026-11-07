import { defineStore } from 'pinia'
import { getMyGear } from '@/api/gear.js'
import { createGearCacheActions, createGearCacheState } from '@/utils/gear-cache.js'

// 只缓存本人装备；好友装备每次进入都回源（好友关系可能随时变化），不进这个 Store。
export const useGearStore = defineStore('gear', {
  state: createGearCacheState,
  actions: createGearCacheActions({
    requestMine: () => getMyGear()
  })
})
