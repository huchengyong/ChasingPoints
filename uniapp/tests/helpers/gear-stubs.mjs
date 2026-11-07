import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

const appDir = dirname(dirname(dirname(fileURLToPath(import.meta.url))))
const realFile = (relative) => pathToFileURL(join(appDir, relative)).href

// 装备页面与组件挂载测试共用的桩。行为由 globalThis[globalKey] 注入：
//   ctx.api.<name>(...)      装备 API
//   ctx.user                 { userId, authGeneration, isLoggedIn }，可在测试中途修改来模拟切号
//   ctx.hooks.shown/loaded/reachBottom   页面生命周期回调，由测试手动触发
// 装备缓存 store 使用真实的 utils/gear-cache.js，请求函数转给 ctx.api.getMyGear。
export const gearStubSources = (globalKey) => ({
  '@dcloudio/uni-app': `
export const onShow = (callback) => globalThis.${globalKey}.hooks.shown.push(callback)
export const onLoad = (callback) => globalThis.${globalKey}.hooks.loaded.push(callback)
export const onReachBottom = (callback) => globalThis.${globalKey}.hooks.reachBottom.push(callback)
`,
  '@/api/gear.js': `
const api = () => globalThis.${globalKey}.api
export const getGearBrands = (...args) => api().getGearBrands(...args)
export const getGearProducts = (...args) => api().getGearProducts(...args)
export const getMyGear = (...args) => api().getMyGear(...args)
export const createGearItem = (...args) => api().createGearItem(...args)
export const updateGearItem = (...args) => api().updateGearItem(...args)
export const deleteGearItem = (...args) => api().deleteGearItem(...args)
export const setGearDefault = (...args) => api().setGearDefault(...args)
export const getUserGear = (...args) => api().getUserGear(...args)
export const getGearMatch = (...args) => api().getGearMatch(...args)
export const updateGearMatch = (...args) => api().updateGearMatch(...args)
`,
  '@/store/gear.js': `
import { createGearCacheActions, createGearCacheState } from '${realFile('utils/gear-cache.js')}'
const store = Object.assign(createGearCacheState(), createGearCacheActions({
  requestMine: () => globalThis.${globalKey}.api.getMyGear()
}))
export const useGearStore = () => store
`,
  '@/store/user.js': `
export const useUserStore = () => globalThis.${globalKey}.user
`,
  '@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`,
  '@/components/gearImage.vue': `
import { h } from 'vue'
export default {
  props: ['subject', 'size', 'fill'],
  render() { return h('view', { class: 'gear-image-stub' }) }
}
`,
  '@/components/gearItemCard.vue': `
import { h } from 'vue'
export default {
  props: ['item', 'badges'],
  render() {
    return h('view', { class: 'gear-card-stub' }, [
      h('text', { class: 'gear-card-stub-title' }, this.item.model_name),
      ...(this.badges || []).map((badge) => h('text', { class: 'gear-card-stub-badge' }, badge))
    ])
  }
}
`
})

export const gearRealModules = [
  '@/utils/gear-display.js',
  '@/utils/gear-form.js',
  '@/utils/gear-match.js',
  '@/utils/async-page-state.js',
  '@/utils/game-types.js'
]

export const createGearContext = (overrides = {}) => ({
  user: { userId: 1, authGeneration: 1, isLoggedIn: true },
  hooks: { shown: [], loaded: [], reachBottom: [] },
  api: {},
  ...overrides
})

// 常用的测试数据
export const sampleItems = () => ([
  { id: 1, category: 'cue', status: 1, unlisted: false, house: false, brand_name: 'Alpha', model_name: 'A-100', cue_style: 'pool', roles: [], created_at: '', product: { id: 11, category: 'cue', name: 'A-100', brand_name: 'Alpha', length_mm: 1470, weight_g: 540, tip_diameter_mm: 12.9, cue_style: 'pool' }, tip_product: { id: 21, category: 'tip', brand_name: 'Alpha', name: 'A-Tip' }, chalk_product: null },
  { id: 2, category: 'break_jump', status: 1, unlisted: false, house: false, brand_name: 'Alpha', model_name: 'A-Break', cue_style: '', roles: ['break'], created_at: '', product: { id: 12, category: 'break_jump', name: 'A-Break', brand_name: 'Alpha', roles: ['break'] }, tip_product: null, chalk_product: null },
  { id: 3, category: 'cue', status: 2, unlisted: true, house: false, brand_name: '我的品牌', model_name: '我的型号', cue_style: 'snooker', roles: [], created_at: '', tip_product: null, chalk_product: null }
])
