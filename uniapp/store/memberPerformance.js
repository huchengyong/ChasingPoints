import { defineStore } from 'pinia'
import { getMemberPerformance } from '@/api/member.js'
import { createPerformanceCacheActions, createPerformanceCacheState } from '@/utils/member-performance-cache.js'

export const useMemberPerformanceStore = defineStore('memberPerformance', {
  state: createPerformanceCacheState,
  actions: createPerformanceCacheActions({
    requestPerformance: getMemberPerformance
  })
})
