import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'

const appDir = dirname(dirname(fileURLToPath(import.meta.url)))
const seasonPagePath = join(appDir, 'subPages/season/index.vue')

const moduleStubs = {
  '@dcloudio/uni-app': `
export const onShow = (callback) => globalThis.__SEASON_PAGE_MOUNT__.shown.push(callback)
`,
  '@/api/season.js': `
export const getSeasonOverview = (...args) => globalThis.__SEASON_PAGE_MOUNT__.getSeasonOverview(...args)
export const getSeasonLeaderboard = (...args) => globalThis.__SEASON_PAGE_MOUNT__.getSeasonLeaderboard(...args)
`,
  '@/store/userDataInvalidation.js': `
export const useUserDataInvalidationStore = () => ({
  versionOf: (scope) => globalThis.__SEASON_PAGE_MOUNT__.versionOf(scope)
})
`,
  '@/store/user.js': `
export const useUserStore = () => ({ userId: 1, authGeneration: 1 })
`,
  '@/utils/game-types.js': `
export const GAME_TYPE_TABS = [{ value: 3, label: '中式八球' }, { value: 2, label: '九球追分' }]
`,
  '@/utils/honor-wall.js': `
export const resolveCurrentSeasonState = () => ({ title: '暂无赛季', description: '请稍后再试' })
`,
  '@/utils/season-lifecycle.js': `
export const resolveSeasonTimeline = () => ({ remainDays: 10, progressPercent: 50 })
`,
  '@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`,
  '@/utils/user-profile.js': `
export const resolveAvatarUrl = () => ''
`,
  '@/utils/async-page-state.js': `
export const ASYNC_PAGE_STATUS = {
  IDLE: 'idle',
  LOADING: 'loading',
  READY: 'ready',
  EMPTY: 'empty',
  ERROR: 'error',
  REFRESHING: 'refreshing'
}
export const createAsyncPageState = ({ authGeneration = -1, data = null } = {}) => ({
  status: ASYNC_PAGE_STATUS.IDLE,
  authGeneration,
  requestId: 0,
  data,
  hasData: data !== null,
  error: null,
  refreshError: null
})
export const beginAsyncPageLoad = (state, { authGeneration } = {}) => ({
  ...state,
  status: state.hasData ? ASYNC_PAGE_STATUS.REFRESHING : ASYNC_PAGE_STATUS.LOADING,
  authGeneration,
  requestId: state.requestId + 1,
  error: null,
  refreshError: null
})
export const getAsyncPageRequest = (state) => ({
  requestId: state.requestId,
  authGeneration: state.authGeneration
})
export const resolveAsyncPageLoad = (state, request, { data } = {}) => ({
  ...state,
  status: ASYNC_PAGE_STATUS.READY,
  data,
  hasData: true,
  error: null,
  refreshError: null
})
export const rejectAsyncPageLoad = (state, request, error) => ({
  ...state,
  status: state.hasData ? ASYNC_PAGE_STATUS.READY : ASYNC_PAGE_STATUS.ERROR,
  error: state.hasData ? null : error,
  refreshError: state.hasData ? error : null
})
export const resolveAsyncPageErrorFeedback = (error, { resource = '内容' } = {}) => ({
  title: resource,
  description: error?.message || '加载失败',
  actionText: '重试'
})
`,
  '@/utils/request-errors.js': `
export const createRequestError = ({ message = '请求失败', category = 'business' } = {}) => {
  const error = new Error(message)
  error.category = category
  return error
}
`
}

test('season page compiles and mounts with one overview request on first entry', async () => {
  const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
  const previousGlobals = installDOMGlobals(dom.window)
  const tempDir = await mkdtemp(join(appDir, '.season-page-mount-'))
  let wrapper

  try {
    const source = await readFile(seasonPagePath, 'utf8')
    const { descriptor, errors } = parse(source, { filename: seasonPagePath })
    assert.deepEqual(errors, [])

    const compiled = compileScript(descriptor, {
      id: 'season-page-real-mount',
      inlineTemplate: true,
      templateOptions: {
        compilerOptions: {
          isCustomElement: (tag) => ['view', 'text', 'image', 'uni-icons'].includes(tag)
        }
      }
    })

    let moduleSource = compiled.content
    let stubIndex = 0
    for (const [specifier, content] of Object.entries(moduleStubs)) {
      const stubName = `stub-${stubIndex++}.mjs`
      await writeFile(join(tempDir, stubName), content)
      moduleSource = moduleSource
        .replaceAll(`from '${specifier}'`, `from './${stubName}'`)
        .replaceAll(`from "${specifier}"`, `from "./${stubName}"`)
    }
    await writeFile(join(tempDir, 'SeasonPage.mjs'), moduleSource)

    let overviewCalls = 0
    let scopeVersion = 0
    const navigations = []
    const activeSeason = { id: 1, name: '真实挂载测试赛季', status: 1, start_date: '2026-08-01', end_date: '2026-09-01' }
    let resolveSwitchOverview
    const switchOverview = new Promise((resolve) => {
      resolveSwitchOverview = resolve
    })
    let rejectSwitchOverview
    const failedSwitchOverview = new Promise((_, reject) => {
      rejectSwitchOverview = reject
    })
    const overviewParams = []
    const followUpOverviews = [
      {
        success: true,
        season: activeSeason,
        season_state: 'active',
        availability: { season: true, record: false, leaderboard: false, my_rank: false },
        record: null,
        my_rank: 0,
        leaderboard: []
      },
      {
        success: true,
        season: activeSeason,
        season_state: 'active',
        availability: { season: true, record: true, leaderboard: false, my_rank: true },
        record: { matches_played: 3, wins: 2, peak_rank_score: 1260, final_rank: 0 },
        my_rank: 5,
        leaderboard: []
      },
      {
        success: true,
        season: activeSeason,
        season_state: 'active',
        availability: { season: true, record: true, leaderboard: true, my_rank: true },
        record: null,
        my_rank: 0,
        leaderboard: []
      },
      switchOverview,
      failedSwitchOverview,
      {
        success: true,
        season: null,
        season_state: 'not_started',
        availability: { season: true },
        record: null,
        leaderboard: []
      }
    ]
    let resolveOverview
    const firstOverview = new Promise((resolve) => {
      resolveOverview = resolve
    })
    globalThis.__SEASON_PAGE_MOUNT__ = {
      shown: [],
      versionOf: () => scopeVersion,
      getSeasonOverview: (params) => {
        overviewCalls += 1
        overviewParams.push(params)
        if (overviewCalls === 1) return firstOverview
        return Promise.resolve(followUpOverviews[overviewCalls - 2])
      },
      getSeasonLeaderboard: async () => ({ success: true, list: [], total: 0 })
    }
    globalThis.uni = { navigateTo: (options) => navigations.push(options.url) }

    const [{ mount, flushPromises }, { nextTick }] = await Promise.all([
      import('@vue/test-utils'),
      import('vue')
    ])
    const componentModule = await import(`${pathToFileURL(join(tempDir, 'SeasonPage.mjs')).href}?v=${Date.now()}`)
    wrapper = mount(componentModule.default, {
      attachTo: document.body,
      global: { stubs: { 'uni-icons': true } }
    })

    await nextTick()
    assert.equal(overviewCalls, 1)
    assert.match(wrapper.text(), /加载中/)
    assert.equal(globalThis.__SEASON_PAGE_MOUNT__.shown.length, 1)

    globalThis.__SEASON_PAGE_MOUNT__.shown[0]()
    await nextTick()
    assert.equal(overviewCalls, 1, 'onShow must reuse the in-flight mounted request')

    resolveOverview({
      success: true,
      season: {
        id: 1,
        name: '真实挂载测试赛季',
        status: 1,
        start_date: '2026-08-01',
        end_date: '2026-09-01'
      },
      season_state: 'active',
      availability: { record: true, leaderboard: true, my_rank: true },
      record: { matches_played: 2, wins: 1, peak_rank_score: 1200, final_rank: 0 },
      my_rank: 2,
      leaderboard: []
    })
    await flushPromises()
    await nextTick()

    assert.doesNotMatch(wrapper.text(), /加载中/)
    assert.match(wrapper.text(), /真实挂载测试赛季/)
    assert.match(wrapper.text(), /当前排名：第 2 名/, 'active season must show live rank instead of final_rank')
    assert.match(wrapper.text(), /历届赛季/)
    assert.doesNotMatch(wrapper.text(), /赛季报告/, 'unreachable report button must be removed')
    assert.equal(overviewCalls, 1)

    globalThis.__SEASON_PAGE_MOUNT__.shown[0]()
    await flushPromises()
    assert.equal(overviewCalls, 1, 'unchanged scope must not reload a retained page')

    scopeVersion = 1
    globalThis.__SEASON_PAGE_MOUNT__.shown[0]()
    await flushPromises()
    assert.equal(overviewCalls, 2, 'season scope invalidation must trigger one refresh')
    await nextTick()
    assert.match(wrapper.text(), /真实挂载测试赛季/, 'partial overview must keep the season card')
    assert.match(wrapper.text(), /赛季数据暂时没加载出来/)
    assert.equal(wrapper.findAll('.section-notice').length, 1, 'record and leaderboard failures must collapse into one notice')
    assert.doesNotMatch(wrapper.text(), /赛季排行榜|暂无排行数据/, 'unavailable leaderboard must not look empty')
    assert.doesNotMatch(wrapper.text(), /当前排名|本赛季还没有排位记录/, 'unavailable rank must not look unranked')

    scopeVersion = 2
    globalThis.__SEASON_PAGE_MOUNT__.shown[0]()
    await flushPromises()
    await nextTick()
    assert.equal(overviewCalls, 3)
    assert.match(wrapper.text(), /当前排名：第 5 名/, 'record must stay visible when only the leaderboard fails')
    assert.match(wrapper.text(), /赛季排行榜/)
    assert.match(wrapper.text(), /榜单暂时没加载出来/)
    assert.equal(wrapper.findAll('.section-notice').length, 1)
    assert.doesNotMatch(wrapper.text(), /赛季数据暂时没加载出来|暂无排行数据/)

    scopeVersion = 3
    globalThis.__SEASON_PAGE_MOUNT__.shown[0]()
    await flushPromises()
    await nextTick()
    assert.equal(overviewCalls, 4)
    assert.match(wrapper.text(), /本赛季还没有排位记录/)
    assert.match(wrapper.text(), /完成一场中式八球排位即可上榜/)
    assert.doesNotMatch(wrapper.text(), /当前排名/)
    assert.equal(wrapper.findAll('.section-notice').length, 0)

    // 切换球种：只替换下方数据卡片，赛季信息与球种标签保持可见
    await wrapper.findAll('.game-type-tab')[1].trigger('tap')
    assert.equal(overviewCalls, 5)
    assert.deepEqual(overviewParams.at(-1), { game_type: 2 })
    assert.match(wrapper.text(), /真实挂载测试赛季/, 'switching must keep the season card')
    assert.equal(wrapper.find('.loading-state').exists(), false, 'switching must not replace the whole page')
    assert.match(wrapper.text(), /加载中/)
    assert.equal(wrapper.findAll('.game-type-tab')[1].classes('active'), true)
    assert.doesNotMatch(wrapper.text(), /当前排名/, 'previous discipline rank must not leak into the new tab')

    resolveSwitchOverview({
      success: true,
      season: activeSeason,
      season_state: 'active',
      availability: { season: true, record: true, leaderboard: true, my_rank: true },
      record: null,
      my_rank: 0,
      leaderboard: []
    })
    await flushPromises()
    await nextTick()
    assert.doesNotMatch(wrapper.text(), /加载中/)
    assert.match(wrapper.text(), /完成一场九球追分排位即可上榜/)

    await wrapper.findAll('.game-type-tab')[0].trigger('tap')
    assert.equal(overviewCalls, 6)
    assert.deepEqual(overviewParams.at(-1), { game_type: 3 })
    rejectSwitchOverview(new Error('switch failed'))
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /真实挂载测试赛季/, 'failed switch must keep the season card')
    assert.match(wrapper.text(), /赛季数据暂时没加载出来/, 'failed switch must show the notice instead of an empty state')
    assert.doesNotMatch(wrapper.text(), /本赛季还没有排位记录/, 'failed switch must not look unranked')
    assert.doesNotMatch(wrapper.text(), /加载中/)

    scopeVersion = 4
    globalThis.__SEASON_PAGE_MOUNT__.shown[0]()
    await flushPromises()
    await nextTick()
    assert.equal(overviewCalls, 7)
    assert.match(wrapper.text(), /暂无赛季/)
    assert.match(wrapper.text(), /历届赛季/, 'history entry must stay reachable without an active season')
    await wrapper.find('.history-entry').trigger('tap')
    assert.deepEqual(navigations, ['/subPages/season/history?game_type=3'])
  } finally {
    wrapper?.unmount()
    delete globalThis.__SEASON_PAGE_MOUNT__
    delete globalThis.uni
    await rm(tempDir, { recursive: true, force: true })
    restoreDOMGlobals(previousGlobals)
    dom.window.close()
  }
})

function installDOMGlobals(window) {
  const keys = ['window', 'document', 'navigator', 'Node', 'Element', 'HTMLElement', 'SVGElement', 'Event']
  const previous = new Map()
  for (const key of keys) {
    previous.set(key, Object.getOwnPropertyDescriptor(globalThis, key))
    Object.defineProperty(globalThis, key, {
      configurable: true,
      writable: true,
      value: window[key]
    })
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
