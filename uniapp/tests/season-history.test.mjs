import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'

import {
  SEASON_HISTORY_PAGE_SIZE,
  buildSeasonHistoryItem,
  buildSeasonReportUrl,
  hasMoreSeasonHistory
} from '../utils/season-history.js'

const appDir = dirname(dirname(fileURLToPath(import.meta.url)))
const historyPagePath = join(appDir, 'subPages/season/history.vue')

const settledSeason = (id, record = null) => ({
  season: { id, name: `S${id}`, status: 2, start_date: '2026-08-01', end_date: '2026-08-31' },
  record
})

test('season history item exposes rank, stats and report target only when the viewer has a record', () => {
  const ranked = buildSeasonHistoryItem(settledSeason(2, { matches_played: 8, wins: 6, final_rank: 3 }), 1)
  assert.deepEqual(ranked, {
    id: 2,
    name: 'S2',
    dateRange: '2026-08-01 ~ 2026-08-31',
    hasRecord: true,
    rankText: '第 3 名',
    matches: 8,
    wins: 6,
    winRate: 75,
    reportUrl: '/subPages/season/report?id=2&game_type=1'
  })

  const absent = buildSeasonHistoryItem(settledSeason(1), 3)
  assert.equal(absent.hasRecord, false)
  assert.equal(absent.reportUrl, '', 'absent season must not link to a report')
  assert.equal(absent.matches, 0)

  assert.equal(buildSeasonHistoryItem(settledSeason(4, { matches_played: 0, wins: 0, final_rank: 0 }), 3).rankText, '未排名')
  assert.equal(buildSeasonHistoryItem({ season: null }, 3), null)
  assert.equal(buildSeasonReportUrl(9, 4), '/subPages/season/report?id=9&game_type=4')
})

test('season history stops paging at total or after an empty page', () => {
  assert.equal(hasMoreSeasonHistory({ loaded: 10, total: 25, lastPageCount: 10 }), true)
  assert.equal(hasMoreSeasonHistory({ loaded: 25, total: 25, lastPageCount: 5 }), false)
  assert.equal(hasMoreSeasonHistory({ loaded: 10, total: 25, lastPageCount: 0 }), false, 'shrinking data must not loop forever')
  assert.equal(hasMoreSeasonHistory(), false)
})

const realModules = {
  '@/utils/game-types.js': join(appDir, 'utils/game-types.js'),
  '@/utils/season-history.js': join(appDir, 'utils/season-history.js'),
  '@/utils/async-page-state.js': join(appDir, 'utils/async-page-state.js'),
  '@/utils/request-errors.js': join(appDir, 'utils/request-errors.js')
}

const moduleStubs = {
  '@dcloudio/uni-app': `
export const onLoad = (callback) => globalThis.__SEASON_HISTORY__.loaded.push(callback)
export const onShow = (callback) => globalThis.__SEASON_HISTORY__.shown.push(callback)
`,
  '@/api/season.js': `
export const getSeasonHistory = (...args) => globalThis.__SEASON_HISTORY__.getSeasonHistory(...args)
`,
  '@/store/user.js': `
export const useUserStore = () => globalThis.__SEASON_HISTORY__.userStore
`,
  '@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`
}

const deferred = () => {
  let resolve
  let reject
  const promise = new Promise((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

// Vue runtime-dom 在首次加载时绑定 document，同一文件内的挂载测试必须共用一个 JSDOM。
let sharedDom = null
let previousGlobals = null
test.after(() => {
  if (!sharedDom) return
  restoreDOMGlobals(previousGlobals)
  sharedDom.window.close()
})

const mountHistoryPage = async (t) => {
  if (!sharedDom) {
    sharedDom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
    previousGlobals = installDOMGlobals(sharedDom.window)
  }
  const tempDir = await mkdtemp(join(appDir, '.season-history-mount-'))
  const harness = {
    loaded: [],
    shown: [],
    requests: [],
    navigations: [],
    userStore: { userId: 1, authGeneration: 1 },
    getSeasonHistory: (params) => {
      const pending = deferred()
      harness.requests.push({ params, ...pending })
      return pending.promise
    }
  }
  globalThis.__SEASON_HISTORY__ = harness
  globalThis.uni = {
    navigateTo: (options) => harness.navigations.push(options.url),
    navigateBack: () => harness.navigations.push('back')
  }

  const source = await readFile(historyPagePath, 'utf8')
  const { descriptor, errors } = parse(source, { filename: historyPagePath })
  assert.deepEqual(errors, [])
  const compiled = compileScript(descriptor, {
    id: 'season-history-mount',
    inlineTemplate: true,
    templateOptions: {
      compilerOptions: {
        isCustomElement: (tag) => ['view', 'text', 'image', 'uni-icons', 'button'].includes(tag)
      }
    }
  })
  let moduleSource = compiled.content
  for (const [specifier, path] of Object.entries(realModules)) {
    moduleSource = moduleSource.replaceAll(`'${specifier}'`, `'${pathToFileURL(path).href}'`)
  }
  let stubIndex = 0
  for (const [specifier, content] of Object.entries(moduleStubs)) {
    const stubName = `stub-${stubIndex++}.mjs`
    await writeFile(join(tempDir, stubName), content)
    moduleSource = moduleSource.replaceAll(`'${specifier}'`, `'./${stubName}'`)
  }
  await writeFile(join(tempDir, 'SeasonHistoryPage.mjs'), moduleSource)

  const [{ mount, flushPromises }, { nextTick }] = await Promise.all([import('@vue/test-utils'), import('vue')])
  const componentModule = await import(`${pathToFileURL(join(tempDir, 'SeasonHistoryPage.mjs')).href}?v=${Date.now()}`)
  const wrapper = mount(componentModule.default, { attachTo: document.body })
  const settle = async () => {
    await flushPromises()
    await nextTick()
  }

  t.after(async () => {
    wrapper.unmount()
    delete globalThis.__SEASON_HISTORY__
    delete globalThis.uni
    await rm(tempDir, { recursive: true, force: true })
    document.body.innerHTML = ''
  })
  return { wrapper, harness, settle }
}

const historyResponse = (ids, total, records = {}) => ({
  success: true,
  total,
  list: ids.map(id => settledSeason(id, records[id] || null))
})

test('season history page loads once on entry, links ranked seasons and pages until total', async (t) => {
  const { wrapper, harness, settle } = await mountHistoryPage(t)
  harness.loaded[0]({ game_type: '1' })
  harness.shown[0]()
  await settle()
  assert.equal(harness.requests.length, 1, 'onLoad and first onShow must share one request')
  assert.deepEqual(harness.requests[0].params, { game_type: 1, page: 1, page_size: SEASON_HISTORY_PAGE_SIZE })
  assert.match(wrapper.text(), /加载中/)

  const firstPage = Array.from({ length: 10 }, (_, index) => 12 - index)
  harness.requests[0].resolve(historyResponse(firstPage, 12, { 12: { matches_played: 4, wins: 3, final_rank: 2 } }))
  await settle()
  assert.match(wrapper.text(), /S12/)
  assert.match(wrapper.text(), /第 2 名/)
  assert.match(wrapper.text(), /未参加该球种排位/)
  assert.match(wrapper.text(), /加载更多/)

  const items = wrapper.findAll('.history-item')
  await items[1].trigger('tap')
  assert.deepEqual(harness.navigations, [], 'season without record must not open a report')
  await items[0].trigger('tap')
  assert.deepEqual(harness.navigations, ['/subPages/season/report?id=12&game_type=1'])

  harness.shown[0]()
  await settle()
  assert.equal(harness.requests.length, 1, 'repeated onShow with the same identity must not reload')

  await wrapper.find('.load-more').trigger('tap')
  await settle()
  assert.equal(harness.requests.length, 2)
  assert.deepEqual(harness.requests[1].params, { game_type: 1, page: 2, page_size: SEASON_HISTORY_PAGE_SIZE })
  harness.requests[1].resolve(historyResponse([2, 1], 12))
  await settle()
  assert.equal(wrapper.findAll('.history-item').length, 12)
  assert.match(wrapper.text(), /没有更多了/)
  assert.equal(wrapper.find('.load-more').exists(), false)
})

test('season history page shows empty state and retries a failed first load', async (t) => {
  const { wrapper, harness, settle } = await mountHistoryPage(t)
  harness.loaded[0]({})
  await settle()
  assert.equal(harness.requests[0].params.game_type, 3, 'missing game type must default to 中式八球')
  harness.requests[0].reject(Object.assign(new Error('服务暂不可用'), { category: 'server' }))
  await settle()
  assert.match(wrapper.text(), /服务暂不可用/)
  assert.doesNotMatch(wrapper.text(), /暂无已结束的赛季/, 'first load failure must not look empty')

  await wrapper.find('.retry-btn').trigger('tap')
  await settle()
  assert.equal(harness.requests.length, 2)
  harness.requests[1].resolve(historyResponse([], 0))
  await settle()
  assert.match(wrapper.text(), /暂无已结束的赛季/)
})

test('season history page keeps loaded items when load more fails and retries the same page', async (t) => {
  const { wrapper, harness, settle } = await mountHistoryPage(t)
  harness.loaded[0]({ game_type: '3' })
  await settle()
  harness.requests[0].resolve(historyResponse(Array.from({ length: 10 }, (_, index) => 20 - index), 15))
  await settle()

  await wrapper.find('.load-more').trigger('tap')
  await settle()
  harness.requests[1].reject(Object.assign(new Error('网络异常'), { category: 'network' }))
  await settle()
  assert.equal(wrapper.findAll('.history-item').length, 10, 'load more failure must keep existing items')
  assert.match(wrapper.text(), /请检查网络后重试/)

  await wrapper.find('.load-error-action').trigger('tap')
  await settle()
  assert.equal(harness.requests[2].params.page, 2, 'retry must request the failed page again')
  harness.requests[2].resolve(historyResponse([10, 9, 8, 7, 6], 15))
  await settle()
  assert.equal(wrapper.findAll('.history-item').length, 15)
})

test('season history page drops late responses after game type or identity changes', async (t) => {
  const { wrapper, harness, settle } = await mountHistoryPage(t)
  harness.loaded[0]({ game_type: '3' })
  await settle()

  const snookerTab = wrapper.findAll('.game-type-tab').find(tab => tab.text() === '斯诺克')
  await snookerTab.trigger('tap')
  await settle()
  assert.equal(harness.requests.length, 2)
  assert.equal(harness.requests[1].params.game_type, 1)
  harness.requests[0].resolve(historyResponse([99], 1, { 99: { matches_played: 1, wins: 1, final_rank: 1 } }))
  await settle()
  assert.doesNotMatch(wrapper.text(), /S99/, 'late response of the previous game type must be dropped')
  harness.requests[1].resolve(historyResponse([5], 1, { 5: { matches_played: 2, wins: 1, final_rank: 4 } }))
  await settle()
  assert.match(wrapper.text(), /S5/)

  harness.userStore.userId = 2
  harness.userStore.authGeneration = 2
  harness.shown[0]()
  await settle()
  assert.equal(harness.requests.length, 3, 'identity change must reload on show')
  assert.doesNotMatch(wrapper.text(), /S5/, 'previous identity items must be cleared')
  harness.requests[2].resolve(historyResponse([6], 1))
  await settle()
  assert.match(wrapper.text(), /S6/)

  harness.userStore.authGeneration = 3
  await wrapper.find('.game-type-tab').trigger('tap')
  await settle()
  const pending = harness.requests.at(-1)
  harness.userStore.userId = 3
  harness.userStore.authGeneration = 4
  pending.resolve(historyResponse([77], 1))
  await settle()
  assert.doesNotMatch(wrapper.text(), /S77/, 'response of a replaced identity must not be written')
})

function installDOMGlobals(window) {
  const keys = ['window', 'document', 'navigator', 'Node', 'Element', 'HTMLElement', 'SVGElement', 'Event']
  const previous = new Map()
  for (const key of keys) {
    previous.set(key, Object.getOwnPropertyDescriptor(globalThis, key))
    Object.defineProperty(globalThis, key, { configurable: true, writable: true, value: window[key] })
  }
  return previous
}

function restoreDOMGlobals(previous) {
  for (const [key, descriptor] of previous) {
    if (descriptor) {
      Object.defineProperty(globalThis, key, descriptor)
    } else {
      delete globalThis[key]
    }
  }
}
