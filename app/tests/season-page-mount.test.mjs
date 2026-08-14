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
  '@/utils/game-types.js': `
export const GAME_TYPE_TABS = [{ value: 3, label: '中式八球' }]
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
    let resolveOverview
    const firstOverview = new Promise((resolve) => {
      resolveOverview = resolve
    })
    globalThis.__SEASON_PAGE_MOUNT__ = {
      shown: [],
      versionOf: () => scopeVersion,
      getSeasonOverview: () => {
        overviewCalls += 1
        return firstOverview
      },
      getSeasonLeaderboard: async () => ({ success: true, list: [], total: 0 })
    }
    globalThis.uni = { navigateTo: () => {} }

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
      availability: { record: true, leaderboard: true },
      record: { matches_played: 2, wins: 1, peak_rank_score: 1200, final_rank: 0 },
      leaderboard: []
    })
    await flushPromises()
    await nextTick()

    assert.doesNotMatch(wrapper.text(), /加载中/)
    assert.match(wrapper.text(), /真实挂载测试赛季/)
    assert.equal(overviewCalls, 1)

    globalThis.__SEASON_PAGE_MOUNT__.shown[0]()
    await flushPromises()
    assert.equal(overviewCalls, 1, 'unchanged scope must not reload a retained page')

    scopeVersion = 1
    globalThis.__SEASON_PAGE_MOUNT__.shown[0]()
    await flushPromises()
    assert.equal(overviewCalls, 2, 'season scope invalidation must trigger one refresh')
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
