import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'

const appDir = dirname(dirname(fileURLToPath(import.meta.url)))
const pagePath = join(appDir, 'subPages/match/matchReplay.vue')
const real = (relative) => pathToFileURL(join(appDir, relative)).href

const moduleStubs = {
  '@dcloudio/uni-app': `
const register = (name) => (callback) => globalThis.__REPLAY_PAGE__.hooks[name].push(callback)
export const onLoad = register('onLoad')
export const onShow = register('onShow')
`,
  '@/api/member.js': `
export const getMemberMatchReplay = (params) => globalThis.__REPLAY_PAGE__.request(params)
`,
  '@/store/userDataInvalidation.js': `
export const useUserDataInvalidationStore = () => ({ versionOf: (scope) => globalThis.__REPLAY_PAGE__.versions[scope] || 0 })
`,
  '@/store/user.js': `
export const useUserStore = () => globalThis.__REPLAY_PAGE__.user
`,
  '@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`,
  '@/utils/compliance-mode.js': `
export const canShowPaidMembershipEntry = () => globalThis.__REPLAY_PAGE__.canPurchase
`,
  '@/utils/game-types.js': `export * from '${real('utils/game-types.js')}'`,
  '@/utils/member-replay-view.js': `export * from '${real('utils/member-replay-view.js')}'`,
  '@/utils/request-errors.js': `export * from '${real('utils/request-errors.js')}'`,
  '@/utils/async-page-state.js': `export * from '${real('utils/async-page-state.js')}'`
}

const poolReplay = {
  success: true,
  match_id: 88,
  game_type: 3,
  locked: false,
  frame_count: 3,
  summary: {
    my_frame_wins: 2,
    opponent_frame_wins: 1,
    my_longest_streak: 2,
    opponent_longest_streak: 1,
    win_type_stats: [{ win_type: 'normal', label: '普胜', my_count: 1, opponent_count: 1 }, { win_type: 'break_clear', label: '炸清', my_count: 1, opponent_count: 0 }]
  },
  frames: [
    { frame_no: 1, winner: 1, my_score: 1, opponent_score: 0, win_type: 'break_clear' },
    { frame_no: 2, winner: 2, my_score: 0, opponent_score: 1, win_type: 'normal' },
    { frame_no: 3, winner: 1, my_score: 1, opponent_score: 0, win_type: 'normal', key_tags: ['decider'] }
  ]
}

const snookerReplay = {
  success: true,
  match_id: 99,
  game_type: 1,
  locked: false,
  frame_count: 1,
  summary: { my_frame_wins: 1, opponent_frame_wins: 0, my_max_run: 68, opponent_max_run: 20 },
  frames: [{
    frame_no: 1, winner: 1, my_score: 72, opponent_score: 40, duration_seconds: 1260, max_deficit: 12, key_tags: ['decider', 'comeback'],
    my_max_run: 68, opponent_max_run: 20, my_foul_count: 1, opponent_foul_count: 0, my_foul_points: 4, opponent_foul_points: 0,
    trend: [{ step: 0, diff: 0 }, { step: 1, diff: -12 }, { step: 2, diff: 30 }]
  }]
}

test('replay page: pool, snooker, locked, failure, identity and compliance behavior', async () => {
  const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
  const previousGlobals = installDOMGlobals(dom.window)
  const tempDir = await mkdtemp(join(appDir, '.match-replay-page-mount-'))
  let wrapper

  try {
    const source = await readFile(pagePath, 'utf8')
    const { descriptor, errors } = parse(source, { filename: pagePath })
    assert.deepEqual(errors, [])
    const compiled = compileScript(descriptor, {
      id: 'match-replay-page-mount',
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
    await writeFile(join(tempDir, 'MatchReplayPage.mjs'), moduleSource)

    const requests = []
    const gates = []
    const state = {
      hooks: { onLoad: [], onShow: [] },
      versions: {},
      canPurchase: false,
      user: { isLoggedIn: true, userId: 7, authGeneration: 1 },
      request: (params) => {
        requests.push(params)
        let resolve
        let reject
        const promise = new Promise((res, rej) => { resolve = res; reject = rej })
        gates.push({ resolve, reject })
        return promise
      }
    }
    globalThis.__REPLAY_PAGE__ = state
    const navigations = []
    globalThis.uni = { navigateTo: (options) => navigations.push(options.url) }

    const [{ mount, flushPromises }, { nextTick }] = await Promise.all([import('@vue/test-utils'), import('vue')])
    const componentModule = await import(`${pathToFileURL(join(tempDir, 'MatchReplayPage.mjs')).href}?v=${Date.now()}`)
    wrapper = mount(componentModule.default, { attachTo: document.body, global: { stubs: { 'uni-icons': true } } })
    state.hooks.onLoad.forEach((hook) => hook({ match_id: '88' }))
    state.hooks.onShow.forEach((hook) => hook())
    await nextTick()

    // 首次进入只读一次；重复 onShow 不重复读取。
    assert.deepEqual(requests, [{ match_id: 88 }])
    assert.match(wrapper.text(), /加载单场复盘/)
    state.hooks.onShow.forEach((hook) => hook())
    await nextTick()
    assert.equal(requests.length, 1)

    // 中式八球：逐局胜负序列、胜局方式、最长连胜，每局没有比分、没有走势图。
    gates[0].resolve(poolReplay)
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /中式八球复盘/)
    assert.match(wrapper.text(), /2 : 1/)
    assert.equal(wrapper.findAll('.sequence-item').length, 3)
    assert.deepEqual(wrapper.findAll('.sequence-item').map((item) => item.classes().find((name) => name.startsWith('winner-'))), ['winner-me', 'winner-opponent', 'winner-me'])
    assert.match(wrapper.text(), /最长连胜（局）/)
    assert.match(wrapper.text(), /胜局方式/)
    assert.match(wrapper.text(), /决胜局/)
    assert.equal(wrapper.findAll('.trend-chart').length, 0)
    assert.equal(wrapper.findAll('.frame-score').length, 0)

    // 会员状态变化（member scope）后回源；斯诺克：单杆、犯规、关键局标签与以 0 为轴的走势柱。
    state.versions.member = 1
    state.hooks.onShow.forEach((hook) => hook())
    await nextTick()
    assert.equal(requests.length, 2)
    gates[1].resolve(snookerReplay)
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /斯诺克复盘/)
    assert.match(wrapper.text(), /单杆最高/)
    assert.match(wrapper.text(), /犯规（次 \/ 罚分）/)
    assert.match(wrapper.text(), /逆转局/)
    assert.match(wrapper.text(), /我曾落后 12 分后赢下/)
    assert.equal(wrapper.findAll('.trend-bar.up').length, 1)
    assert.equal(wrapper.findAll('.trend-bar.down').length, 1)
    assert.equal(wrapper.findAll('.sequence-item').length, 0, 'snooker has no win/lose sequence')

    // 截断降级：提示原因，仍显示每局结果。
    state.versions.member = 2
    state.hooks.onShow.forEach((hook) => hook())
    gates[2].resolve({ ...snookerReplay, detail_truncated: true, frame_only: true, frames: [{ frame_no: 1, winner: 1, my_score: 72, opponent_score: 40 }] })
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /记录太长，只显示每局结果/)
    assert.equal(wrapper.findAll('.trend-chart').length, 0)

    // 锁定：只有局数，合规模式下没有开通入口。
    state.versions.member = 3
    state.hooks.onShow.forEach((hook) => hook())
    gates[3].resolve({ success: true, match_id: 88, game_type: 3, locked: true, frame_count: 7 })
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /这场共打了 7 局/)
    assert.match(wrapper.text(), /完整复盘为会员权益/)
    assert.equal(wrapper.findAll('.unlock-action').length, 0)
    for (const selector of ['.sequence', '.frame-block', '.compare-row', '.key-row']) {
      assert.equal(wrapper.find(selector).exists(), false, `locked state must not render ${selector}`)
    }

    state.canPurchase = true
    state.versions.member = 4
    state.hooks.onShow.forEach((hook) => hook())
    gates[4].resolve({ success: true, match_id: 88, game_type: 3, locked: true, frame_count: 7 })
    await flushPromises()
    await nextTick()
    assert.equal(wrapper.findAll('.unlock-action').length, 1)
    await wrapper.find('.unlock-action').trigger('tap')
    assert.equal(navigations.at(-1), '/subPages/user/memberCenter')

    // 服务端拒绝（非选手、未完成、不存在统一 success=false）：中性错误，可重试。
    state.versions.member = 5
    state.hooks.onShow.forEach((hook) => hook())
    gates[5].resolve({ success: false })
    await flushPromises()
    await nextTick()
    // 已有数据时刷新被拒：保留页面并给出横幅；没有数据时（下方的缺少对局编号用例）整页显示错误。
    assert.equal(wrapper.findAll('.refresh-error-banner').length, 1)
    assert.match(wrapper.text(), /请返回上一页选择其他内容/)
    assert.equal(wrapper.findAll('.sequence').length, 0)

    // 切号：旧身份的迟到响应不得回写。
    state.versions.member = 6
    state.hooks.onShow.forEach((hook) => hook())
    const stale = gates.at(-1)
    state.user.userId = 8
    state.user.authGeneration = 2
    state.hooks.onShow.forEach((hook) => hook())
    await nextTick()
    stale.resolve(poolReplay)
    await flushPromises()
    await nextTick()
    assert.doesNotMatch(wrapper.text(), /中式八球复盘/, 'a late response from the previous account must be dropped')
    gates.at(-1).resolve({ success: true, match_id: 88, game_type: 3, locked: true, frame_count: 2 })
    await flushPromises()
    await nextTick()
    assert.match(wrapper.text(), /这场共打了 2 局/)

    // 未登录：不读取，转登录。
    const before = requests.length
    state.user.isLoggedIn = false
    state.hooks.onShow.forEach((hook) => hook())
    assert.equal(requests.length, before)
    assert.equal(navigations.at(-1), '/pages/login/login')
  } finally {
    wrapper?.unmount()
    delete globalThis.__REPLAY_PAGE__
    delete globalThis.uni
    await rm(tempDir, { recursive: true, force: true })
    restoreDOMGlobals(previousGlobals)
    dom.window.close()
  }
})

test('replay page without a match id never calls the API', async () => {
  const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
  const previousGlobals = installDOMGlobals(dom.window)
  const tempDir = await mkdtemp(join(appDir, '.match-replay-page-mount-'))
  let wrapper
  try {
    const source = await readFile(pagePath, 'utf8')
    const { descriptor } = parse(source, { filename: pagePath })
    const compiled = compileScript(descriptor, {
      id: 'match-replay-page-no-id',
      inlineTemplate: true,
      templateOptions: { compilerOptions: { isCustomElement: (tag) => ['view', 'text', 'image', 'uni-icons'].includes(tag) } }
    })
    let moduleSource = compiled.content
    let stubIndex = 0
    for (const [specifier, content] of Object.entries(moduleStubs)) {
      const stubName = `stub-${stubIndex++}.mjs`
      await writeFile(join(tempDir, stubName), content)
      moduleSource = moduleSource.replaceAll(`from '${specifier}'`, `from './${stubName}'`)
    }
    await writeFile(join(tempDir, 'MatchReplayPage.mjs'), moduleSource)
    const calls = []
    globalThis.__REPLAY_PAGE__ = {
      hooks: { onLoad: [], onShow: [] },
      versions: {},
      canPurchase: false,
      user: { isLoggedIn: true, userId: 7, authGeneration: 1 },
      request: (params) => { calls.push(params); return Promise.resolve({ success: false }) }
    }
    globalThis.uni = { navigateTo() {} }
    const [{ mount, flushPromises }] = await Promise.all([import('@vue/test-utils')])
    const componentModule = await import(`${pathToFileURL(join(tempDir, 'MatchReplayPage.mjs')).href}?v=${Date.now()}`)
    wrapper = mount(componentModule.default, { attachTo: document.body, global: { stubs: { 'uni-icons': true } } })
    globalThis.__REPLAY_PAGE__.hooks.onLoad.forEach((hook) => hook({}))
    globalThis.__REPLAY_PAGE__.hooks.onShow.forEach((hook) => hook())
    await flushPromises()
    assert.deepEqual(calls, [])
    assert.match(wrapper.text(), /加载失败|不存在/)
  } finally {
    wrapper?.unmount()
    delete globalThis.__REPLAY_PAGE__
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
