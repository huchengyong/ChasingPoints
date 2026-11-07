import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'

const appDir = dirname(dirname(fileURLToPath(import.meta.url)))
const historyPagePath = join(appDir, 'subPages/season/history.vue')

const stubFileName = (specifier) => `${specifier.replace(/[^a-zA-Z0-9]/g, '_')}.mjs`

// 纯逻辑模块直接使用真实文件，只替换其中的相对依赖；其余外部能力用最小桩替代。
const realModules = {
  '@/utils/season-history.js': 'utils/season-history.js',
  '@/utils/async-page-state.js': 'utils/async-page-state.js'
}

const moduleStubs = {
  '@dcloudio/uni-app': `
export const onLoad = (callback) => globalThis.__HISTORY_MOUNT__.loaded.push(callback)
export const onShow = (callback) => globalThis.__HISTORY_MOUNT__.shown.push(callback)
`,
  '@/api/season.js': `
export const getSeasonHistory = (...args) => globalThis.__HISTORY_MOUNT__.getSeasonHistory(...args)
`,
  '@/store/user.js': `
export const useUserStore = () => ({ userId: 1, authGeneration: 1 })
`,
  '@/utils/game-types.js': `
export const GAME_TYPE_TABS = [{ value: 3, label: '中式八球' }, { value: 2, label: '九球追分' }]
`,
  '@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`,
  '@/utils/request-errors.js': `
export const shouldIgnoreRequestError = () => false
export const createRequestError = ({ message = '请求失败', category = 'business' } = {}) => {
  const error = new Error(message)
  error.category = category
  return error
}
`
}

test('season history switches discipline inline and keeps the page shell', async () => {
  const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
  const previousGlobals = installDOMGlobals(dom.window)
  const tempDir = await mkdtemp(join(appDir, '.season-history-mount-'))
  let wrapper

  try {
    const source = await readFile(historyPagePath, 'utf8')
    const { descriptor, errors } = parse(source, { filename: historyPagePath })
    assert.deepEqual(errors, [])

    const compiled = compileScript(descriptor, {
      id: 'season-history-real-mount',
      inlineTemplate: true,
      templateOptions: {
        compilerOptions: {
          isCustomElement: (tag) => ['view', 'text', 'image', 'uni-icons'].includes(tag)
        }
      }
    })

    let moduleSource = compiled.content
    for (const [specifier, content] of Object.entries(moduleStubs)) {
      await writeFile(join(tempDir, stubFileName(specifier)), content)
    }
    for (const [specifier, relativePath] of Object.entries(realModules)) {
      const realSource = await readFile(join(appDir, relativePath), 'utf8')
      await writeFile(join(tempDir, stubFileName(specifier)), realSource.replaceAll("'./request-errors.js'", `'./${stubFileName('@/utils/request-errors.js')}'`))
    }
    for (const specifier of [...Object.keys(moduleStubs), ...Object.keys(realModules)]) {
      moduleSource = moduleSource
        .replaceAll(`from '${specifier}'`, `from './${stubFileName(specifier)}'`)
        .replaceAll(`from "${specifier}"`, `from "./${stubFileName(specifier)}"`)
    }
    await writeFile(join(tempDir, 'SeasonHistoryPage.mjs'), moduleSource)

    const requests = []
    globalThis.__HISTORY_MOUNT__ = {
      loaded: [],
      shown: [],
      getSeasonHistory: (params) => {
        let resolve
        let reject
        const response = new Promise((res, rej) => {
          resolve = res
          reject = rej
        })
        requests.push({ params, resolve, reject })
        return response
      }
    }

    const [{ mount, flushPromises }, { nextTick }] = await Promise.all([
      import('@vue/test-utils'),
      import('vue')
    ])
    const componentModule = await import(`${pathToFileURL(join(tempDir, 'SeasonHistoryPage.mjs')).href}?v=${Date.now()}`)
    wrapper = mount(componentModule.default, {
      attachTo: document.body,
      global: { stubs: { 'uni-icons': true } }
    })

    globalThis.__HISTORY_MOUNT__.loaded[0]({ game_type: 3 })
    await nextTick()
    assert.equal(requests.length, 1)
    assert.deepEqual(requests[0].params, { game_type: 3, page: 1, page_size: 10 })
    assert.equal(wrapper.find('.loading-state').exists(), true, 'first entry keeps the full-page loading state')
    assert.equal(wrapper.find('.switching-card').exists(), false)

    requests[0].resolve({ success: true, total: 1, list: [historyItem(1, 'S1', 2)] })
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /S1/)
    assert.match(wrapper.text(), /第 2 名/)

    await wrapper.findAll('.game-type-tab')[1].trigger('tap')
    assert.equal(requests.length, 2)
    assert.deepEqual(requests[1].params, { game_type: 2, page: 1, page_size: 10 })
    assert.equal(wrapper.findAll('.game-type-tab').length, 2, 'switching must keep the discipline tabs')
    assert.equal(wrapper.findAll('.game-type-tab')[1].classes('active'), true)
    assert.equal(wrapper.find('.loading-state').exists(), false, 'switching must not show the full-page spinner')
    assert.equal(wrapper.find('.switching-card').exists(), true, 'switching shows the inline loading card')
    assert.doesNotMatch(wrapper.text(), /S1/, 'previous discipline list must not stay under the new tab')

    requests[1].resolve({ success: true, total: 1, list: [historyItem(2, 'S2', 5)] })
    await flushPromises()
    await nextTick()
    assert.equal(wrapper.find('.switching-card').exists(), false)
    assert.match(wrapper.text(), /S2/)
    assert.match(wrapper.text(), /第 5 名/)

    await wrapper.findAll('.game-type-tab')[0].trigger('tap')
    assert.equal(requests.length, 3)
    requests[2].reject(new Error('switch failed'))
    await flushPromises()
    await nextTick()
    assert.equal(wrapper.findAll('.game-type-tab').length, 2, 'failed switch must keep the discipline tabs')
    assert.equal(wrapper.find('.page-error-state').exists(), true, 'failed switch shows the list error')
    assert.equal(wrapper.find('.switching-card').exists(), false)
  } finally {
    wrapper?.unmount()
    delete globalThis.__HISTORY_MOUNT__
    await rm(tempDir, { recursive: true, force: true })
    restoreDOMGlobals(previousGlobals)
    dom.window.close()
  }
})

function historyItem(id, name, finalRank) {
  return {
    season: { id, name, start_date: '2026-01-01', end_date: '2026-01-31' },
    record: { matches_played: 10, wins: 6, final_rank: finalRank }
  }
}

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
