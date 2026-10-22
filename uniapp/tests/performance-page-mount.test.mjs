import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'

const appDir = dirname(dirname(fileURLToPath(import.meta.url)))
const pagePath = join(appDir, 'subPages/user/performance.vue')
const real = (relative) => pathToFileURL(join(appDir, relative)).href

// 纯逻辑模块直接用真实实现；页面依赖的 store、框架钩子和主题用桩，行为由用例通过 globalThis.__PERFORMANCE_PAGE__ 注入。
const moduleStubs = {
  '@dcloudio/uni-app': `
const register = (name) => (callback) => globalThis.__PERFORMANCE_PAGE__.hooks[name].push(callback)
export const onLoad = register('onLoad')
export const onShow = register('onShow')
`,
  '@/store/memberPerformance.js': `
import { createPerformanceCacheActions, createPerformanceCacheState } from '${real('utils/member-performance-cache.js')}'
let store
export const useMemberPerformanceStore = () => {
  if (!store) {
    store = Object.assign(createPerformanceCacheState(), createPerformanceCacheActions({
      requestPerformance: (params) => globalThis.__PERFORMANCE_PAGE__.request(params)
    }))
  }
  return store
}
`,
  '@/store/userDataInvalidation.js': `
export const useUserDataInvalidationStore = () => ({ versionOf: (scope) => globalThis.__PERFORMANCE_PAGE__.versions[scope] || 0 })
`,
  '@/store/user.js': `
export const useUserStore = () => globalThis.__PERFORMANCE_PAGE__.user
`,
  '@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`,
  '@/utils/compliance-mode.js': `
export const canShowPaidMembershipEntry = () => globalThis.__PERFORMANCE_PAGE__.canPurchase
`,
  '@/utils/game-types.js': `export * from '${real('utils/game-types.js')}'`,
  '@/utils/member-performance-view.js': `export * from '${real('utils/member-performance-view.js')}'`,
  '@/utils/async-page-state.js': `export * from '${real('utils/async-page-state.js')}'`
}

const memberResponse = (marker, extra = {}) => ({
  success: true,
  game_type: 3,
  locked: false,
  sample_count: 40,
  summary: { wins: 22, expected_wins: 18.4, wins_diff: 3.6, rating: { rating: marker } },
  months: [{ month: '2026-10', sample_count: 6, rating: { rating: marker }, end_rank_score: 1650 }],
  ...extra
})

const deferred = () => {
  let resolve
  let reject
  const promise = new Promise((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

test('performance page: first entry, repeated onShow, game switch, late responses, locked state and compliance', async () => {
  const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
  const previousGlobals = installDOMGlobals(dom.window)
  const tempDir = await mkdtemp(join(appDir, '.performance-page-mount-'))
  let wrapper

  try {
    const source = await readFile(pagePath, 'utf8')
    const { descriptor, errors } = parse(source, { filename: pagePath })
    assert.deepEqual(errors, [])
    const compiled = compileScript(descriptor, {
      id: 'performance-page-mount',
      inlineTemplate: true,
      templateOptions: { compilerOptions: { isCustomElement: (tag) => ['view', 'text', 'image', 'uni-icons'].includes(tag) } }
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
    await writeFile(join(tempDir, 'PerformancePage.mjs'), moduleSource)

    const requests = []
    const gates = []
    globalThis.__PERFORMANCE_PAGE__ = {
      hooks: { onLoad: [], onShow: [] },
      versions: {},
      canPurchase: false,
      user: { isLoggedIn: true, userId: 7, authGeneration: 1 },
      request: (params) => {
        requests.push(params)
        const gate = deferred()
        gates.push(gate)
        return gate.promise
      }
    }
    const state = globalThis.__PERFORMANCE_PAGE__
    const navigations = []
    globalThis.uni = { navigateTo: (options) => navigations.push(options.url) }

    const [{ mount, flushPromises }, { nextTick }] = await Promise.all([import('@vue/test-utils'), import('vue')])
    const componentModule = await import(`${pathToFileURL(join(tempDir, 'PerformancePage.mjs')).href}?v=${Date.now()}`)
    wrapper = mount(componentModule.default, {
      attachTo: document.body,
      global: { stubs: { 'uni-icons': true } }
    })
    state.hooks.onLoad.forEach((hook) => hook({ game_type: '2' }))
    state.hooks.onShow.forEach((hook) => hook())
    await nextTick()

    // 首次进入：按链接里的球种只读一次，期间显示加载态。
    assert.deepEqual(requests, [{ game_type: 2 }])
    assert.match(wrapper.text(), /加载表现分析/)
    assert.equal(wrapper.findAll('.tab-item')[1].classes('active'), true, 'game_type=2 selects the nine-ball tab')

    // 重复 onShow：复用进行中的请求，不重复读取。
    state.hooks.onShow.forEach((hook) => hook())
    await nextTick()
    assert.equal(requests.length, 1, 'repeated onShow must reuse the in-flight read')

    gates[0].resolve(memberResponse(1720))
    await flushPromises()
    await nextTick()
    assert.doesNotMatch(wrapper.text(), /加载表现分析/)
    assert.match(wrapper.text(), /1720/)
    assert.match(wrapper.text(), /表现分只反映这段时间打出的水平，不改变排位分/)
    assert.doesNotMatch(wrapper.text(), /开通会员/)

    state.hooks.onShow.forEach((hook) => hook())
    await flushPromises()
    assert.equal(requests.length, 1, 'an unchanged page keeps its data on re-show')

    // 对局结算（rank scope）使版本变化，下一次进入回源，期间仍展示旧数据。
    state.versions.rank = 1
    state.hooks.onShow.forEach((hook) => hook())
    await nextTick()
    assert.equal(requests.length, 2)
    assert.match(wrapper.text(), /1720/, 'cached data keeps the page steady while refreshing')
    gates[1].resolve(memberResponse(1750))
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /1750/)

    // 快速切球种：先发的请求后到，不能覆盖当前球种的结果。
    await wrapper.findAll('.tab-item')[0].trigger('tap')
    assert.deepEqual(requests.at(-1), { game_type: 3 })
    const slowType3 = gates.length - 1
    await wrapper.findAll('.tab-item')[2].trigger('tap')
    assert.deepEqual(requests.at(-1), { game_type: 1 })
    const fastType1 = gates.length - 1
    gates[fastType1].resolve(memberResponse(1111, { game_type: 1 }))
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /1111/)
    gates[slowType3].resolve(memberResponse(3333))
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /1111/, 'a late response for another game type must not replace the current one')
    assert.doesNotMatch(wrapper.text(), /3333/)

    // 失败：没有旧数据时显示错误与重试，重试成功后恢复。
    await wrapper.findAll('.tab-item')[3].trigger('tap')
    gates.at(-1).reject(new Error('boom'))
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /表现分析加载失败/)
    await wrapper.find('.state-action').trigger('tap')
    gates.at(-1).resolve(memberResponse(1444, { game_type: 4 }))
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /1444/)

    // 锁定态：只显示服务端的样本数与本月预览，合规模式下没有开通入口。
    state.versions.member = 1
    state.hooks.onShow.forEach((hook) => hook())
    gates.at(-1).resolve({ success: true, game_type: 4, locked: true, sample_count: 200, month_preview: { month: '2026-10', sample_count: 6, rating: { rating: 1678 } } })
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /近 90 天已有 200\+ 场可分析的排位/)
    assert.match(wrapper.text(), /10月表现分预览/)
    assert.match(wrapper.text(), /1678/)
    assert.match(wrapper.text(), /完整分析为会员权益/)
    assert.equal(wrapper.findAll('.unlock-action').length, 0, 'compliance mode must not show a purchase entry')
    // 功能清单会写出这些名称，所以按区块是否渲染来判断，而不是按文字。
    for (const selector of ['.summary-card', '.wins-row', '.notable-row', '.month-chart']) {
      assert.equal(wrapper.find(selector).exists(), false, `locked state must not render ${selector}`)
    }

    // 非合规模式：开通入口只由合规开关决定。
    state.canPurchase = true
    state.versions.member = 2
    state.hooks.onShow.forEach((hook) => hook())
    gates.at(-1).resolve({ success: true, game_type: 4, locked: true, sample_count: 3, month_preview: { month: '2026-10', sample_count: 3, rating: null } })
    await flushPromises()
    await nextTick()
    assert.equal(wrapper.findAll('.unlock-action').length, 1)
    await wrapper.find('.unlock-action').trigger('tap')
    assert.deepEqual(navigations.at(-1), '/subPages/user/memberCenter')

    // 切号：旧身份的迟到响应不得回写；新身份重新读取。
    state.versions.member = 3
    state.hooks.onShow.forEach((hook) => hook())
    const stale = gates.at(-1)
    state.user.userId = 8
    state.user.authGeneration = 2
    state.hooks.onShow.forEach((hook) => hook())
    await nextTick()
    stale.resolve(memberResponse(9999))
    await flushPromises()
    await nextTick()
    assert.doesNotMatch(wrapper.text(), /9999/, 'a late response from the previous account must be dropped')
    gates.at(-1).resolve(memberResponse(1888))
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /1888/)

    // 未登录：不读取，转登录。
    const before = requests.length
    state.user.isLoggedIn = false
    state.hooks.onShow.forEach((hook) => hook())
    assert.equal(requests.length, before)
    assert.equal(navigations.at(-1), '/pages/login/login')
  } finally {
    wrapper?.unmount()
    delete globalThis.__PERFORMANCE_PAGE__
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
    Object.defineProperty(globalThis, key, { configurable: true, writable: true, value: window[key] })
  }
  return previous
}

function restoreDOMGlobals(previous) {
  for (const [key, descriptor] of previous) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor)
    else delete globalThis[key]
  }
}
