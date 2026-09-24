import test, { after } from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'
import * as challengeTime from '../utils/challenge-time.js'

const appDir = dirname(dirname(fileURLToPath(import.meta.url)))
const CUSTOM_ELEMENTS = ['view', 'text', 'image', 'picker', 'uni-icons']

// 动态桩：行为由每个用例通过 globalThis.__QUICK_MATCH_FLOW__ 注入。
const stubSources = {
  '@dcloudio/uni-app': `
const register = (name) => (callback) => {
  globalThis.__QUICK_MATCH_FLOW__.hooks[name].push(callback)
}
export const onLoad = register('onLoad')
export const onShow = register('onShow')
export const onHide = register('onHide')
export const onUnload = register('onUnload')
`,
  '@/api/challenge.js': `
const call = (name) => (...args) => {
  const handler = globalThis.__QUICK_MATCH_FLOW__.api[name]
  return handler ? handler(...args) : Promise.resolve({ success: true })
}
export const getPendingChallenges = call('getPendingChallenges')
export const getChallengeSummary = call('getChallengeSummary')
export const getChallengeHistory = call('getChallengeHistory')
export const getChallengeDetail = call('getChallengeDetail')
export const acceptChallenge = call('acceptChallenge')
export const rejectChallenge = call('rejectChallenge')
export const cancelChallenge = call('cancelChallenge')
export const abandonChallenge = call('abandonChallenge')
export const enterChallenge = call('enterChallenge')
export const leaveChallenge = call('leaveChallenge')
export const sendChallenge = call('sendChallenge')
`,
  '@/store/user.js': `
export const useUserStore = () => globalThis.__QUICK_MATCH_FLOW__.user
`,
  '@/store/activity.js': `
export const useActivityStore = () => globalThis.__QUICK_MATCH_FLOW__.activity
`,
  '@/utils/game-types.js': `
export const GAME_TYPE_OPTIONS = [{ value: 3, label: '中式八球' }]
export const GAME_TYPE_LABEL_MAP = { 3: '中式八球' }
`,
  '@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`,
  '@/utils/user-profile.js': `
export const resolveAvatarUrl = () => ''
`,
  '@/utils/challenge-view.js': `
export const formatChallengeSchedule = () => ''
export const formatChallengeExpiry = () => ''
`
}

// 纯逻辑工具直接复用真实实现，避免在测试里复写业务规则。
const realModules = ['@/utils/async-page-state.js', '@/utils/request-errors.js', '@/utils/challenge-time.js']

const defer = () => {
  let resolve
  const promise = new Promise((resolvePromise) => { resolve = resolvePromise })
  return { promise, resolve }
}

function installDOMGlobals(window) {
  const keys = ['window', 'document', 'navigator', 'Node', 'Element', 'HTMLElement', 'HTMLInputElement', 'HTMLButtonElement', 'SVGElement', 'Event', 'Document', 'DocumentFragment', 'ShadowRoot', 'CustomEvent']
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

// Vue 的 runtime-dom 在模块加载时抓取 document，因此必须先装好 jsdom 再引入 vue；
// 整个文件共用一个 DOM，避免后续用例挂载到已关闭的文档。
const flowDom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
const previousGlobals = installDOMGlobals(flowDom.window)
const vueRuntimePromise = import('vue')
const testUtilsPromise = import('@vue/test-utils')

after(() => {
  restoreDOMGlobals(previousGlobals)
  flowDom.window.close()
})

async function compilePage(file, tempDir) {
  const source = await readFile(join(appDir, file), 'utf8')
  const { descriptor, errors } = parse(source, { filename: file })
  assert.deepEqual(errors, [])
  const compiled = compileScript(descriptor, {
    id: `quick-match-flow-${file.replace(/\W+/g, '-')}`,
    inlineTemplate: true,
    templateOptions: {
      compilerOptions: {
        isCustomElement: (tag) => CUSTOM_ELEMENTS.includes(tag)
      }
    }
  })
  let moduleSource = compiled.content
  for (const [specifier, content] of Object.entries(stubSources)) {
    if (!moduleSource.includes(`'${specifier}'`) && !moduleSource.includes(`"${specifier}"`)) continue
    const stubName = `stub-${specifier.replace(/\W+/g, '_')}.mjs`
    await writeFile(join(tempDir, stubName), content)
    moduleSource = moduleSource
      .replaceAll(`from '${specifier}'`, `from './${stubName}'`)
      .replaceAll(`from "${specifier}"`, `from "./${stubName}"`)
  }
  for (const specifier of realModules) {
    if (!moduleSource.includes(`'${specifier}'`) && !moduleSource.includes(`"${specifier}"`)) continue
    const relative = `../${specifier.replace('@/', '')}`
    moduleSource = moduleSource
      .replaceAll(`from '${specifier}'`, `from '${relative}'`)
      .replaceAll(`from "${specifier}"`, `from "${relative}"`)
  }
  const fileName = `page-${file.replace(/\W+/g, '-')}.mjs`
  await writeFile(join(tempDir, fileName), moduleSource)
  return pathToFileURL(join(tempDir, fileName)).href
}

async function mountPage({ file, user, api = {}, activityDirty = () => {}, globals = {} }) {
  const tempDir = await mkdtemp(join(appDir, '.quick-match-flow-'))
  const hooks = { onLoad: [], onShow: [], onHide: [], onUnload: [] }
  const reactiveUser = (await vueRuntimePromise).reactive(user)
  Object.assign(globalThis, globals)
  globalThis.__QUICK_MATCH_FLOW__ = { hooks, user: reactiveUser, api, activity: { markDirty: activityDirty } }
  try {
    const moduleUrl = await compilePage(file, tempDir)
    const [{ mount, flushPromises }, componentModule] = await Promise.all([
      testUtilsPromise,
      import(`${moduleUrl}?v=${Date.now()}`)
    ])
    const wrapper = mount(componentModule.default, { global: { stubs: { 'uni-icons': true } } })
    return {
      wrapper,
      user: reactiveUser,
      flushPromises,
      runHooks: (name, ...args) => hooks[name].forEach((hook) => hook(...args)),
      cleanup: async () => {
        wrapper.unmount()
        delete globalThis.__QUICK_MATCH_FLOW__
        for (const key of Object.keys(globals)) delete globalThis[key]
        await rm(tempDir, { recursive: true, force: true })
      }
    }
  } catch (error) {
    delete globalThis.__QUICK_MATCH_FLOW__
    for (const key of Object.keys(globals)) delete globalThis[key]
    await rm(tempDir, { recursive: true, force: true })
    throw error
  }
}

test('发起页被替换的时间读取不得初始化默认时段', async () => {
  let reads = 0
  let posted
  const first = defer()
  const second = defer()
  const page = await mountPage({
    file: 'subPages/match/challengeCompose.vue',
    user: { userId: 1, authGeneration: 1 },
    api: {
      getChallengeSummary: () => (++reads === 1 ? first.promise : second.promise),
      sendChallenge: async (payload) => {
        posted = payload
        return { success: false }
      }
    },
    globals: { uni: { showToast: () => {} } }
  })
  try {
    page.runHooks('onLoad', { opponent_id: 2 })
    page.runHooks('onShow')
    page.runHooks('onHide')
    page.runHooks('onShow')
    first.resolve({ success: true, server_time: '2026-09-24 15:19:50' })
    await page.flushPromises()
    second.resolve({ success: true, server_time: '2026-09-24 15:20:00' })
    await page.flushPromises()

    await page.wrapper.find('.submit-btn').trigger('tap')
    await page.flushPromises()

    assert.ok(posted, '两次时间读取结束后表单必须可以发送')
    assert.equal(posted.scheduled_date, '2026-09-24')
    assert.equal(posted.start_hour, 16)
  } finally {
    await page.cleanup()
  }
})

test('旧历史响应不得消费新写入的失效标记', async () => {
  let status = 0
  let reads = 0
  const stale = defer()
  const item = { id: 12, from_user_id: 1, to_user_id: 2, status: 0, game_type: 3 }
  const page = await mountPage({
    file: 'subPages/social/challenges.vue',
    user: { userId: 1, authGeneration: 1 },
    api: {
      getPendingChallenges: async () => ({ success: true, list: status === 0 ? [item] : [] }),
      getChallengeSummary: async () => ({ success: true, server_time: '2026-09-24 12:00:00' }),
      getChallengeHistory: () => (++reads === 1 ? stale.promise : Promise.resolve({ success: true, list: [{ ...item, status }] })),
      cancelChallenge: async () => {
        status = 4
        return { success: true }
      }
    },
    globals: {
      getCurrentPages: () => [{ options: { tab: 'sent' } }],
      uni: { showToast: () => {}, showModal: (options) => options.success({ confirm: true }) }
    }
  })
  try {
    await page.flushPromises()
    await page.wrapper.findAll('.tab-item')[2].trigger('tap')
    await page.flushPromises()
    assert.equal(reads, 1)

    await page.wrapper.findAll('.tab-item')[1].trigger('tap')
    await page.wrapper.find('.ghost-btn').trigger('tap')
    await page.flushPromises()
    assert.equal(status, 4)

    stale.resolve({ success: true, list: [item] })
    await page.flushPromises()

    await page.wrapper.findAll('.tab-item')[2].trigger('tap')
    await page.flushPromises()
    assert.match(page.wrapper.text(), /已取消/)
    assert.equal(reads, 2, '写入后的历史读取不得复用旧响应')
  } finally {
    await page.cleanup()
  }
})

test('回前台恢复必须让历史缓存回源', async () => {
  let status = 0
  let reads = 0
  const item = { id: 12, from_user_id: 1, to_user_id: 2, status: 0, game_type: 3 }
  const page = await mountPage({
    file: 'subPages/social/challenges.vue',
    user: { userId: 1, authGeneration: 1 },
    api: {
      getPendingChallenges: async () => ({ success: true, list: status === 0 ? [item] : [] }),
      getChallengeSummary: async () => ({ success: true, server_time: '2026-09-24 12:00:00' }),
      getChallengeHistory: async () => {
        reads += 1
        return { success: true, list: [{ ...item, status }] }
      }
    },
    globals: {
      getCurrentPages: () => [{ options: { tab: 'history' } }],
      uni: { showToast: () => {} }
    }
  })
  try {
    await page.flushPromises()
    assert.equal(reads, 1)

    await page.wrapper.findAll('.tab-item')[1].trigger('tap')
    status = 4
    page.runHooks('onShow')
    await page.flushPromises()

    await page.wrapper.findAll('.tab-item')[2].trigger('tap')
    await page.flushPromises()
    assert.match(page.wrapper.text(), /已取消/, '前台恢复后历史必须回源，不得永久复用旧缓存')
    assert.equal(reads, 2)
  } finally {
    await page.cleanup()
  }
})

test('列表页写响应在离开页面后不得导航', async () => {
  const late = defer()
  const routes = []
  const page = await mountPage({
    file: 'subPages/social/challenges.vue',
    user: { userId: 2, authGeneration: 1 },
    api: {
      getPendingChallenges: async () => ({ success: true, list: [{ id: 12, from_user_id: 1, to_user_id: 2, status: 0, game_type: 3 }] }),
      getChallengeSummary: async () => ({ success: true, server_time: '2026-09-24 12:00:00' }),
      getChallengeHistory: async () => ({ success: true, list: [] }),
      acceptChallenge: () => late.promise
    },
    globals: {
      getCurrentPages: () => [{ options: { tab: 'received' } }],
      uni: { showToast: () => {}, navigateTo: ({ url }) => routes.push(url) }
    }
  })
  try {
    await page.flushPromises()
    await page.wrapper.find('.accept-btn').trigger('tap')
    page.runHooks('onHide')
    page.runHooks('onUnload')

    late.resolve({ success: true, challenge_id: 12 })
    await page.flushPromises()
    assert.deepEqual(routes, [], '离开页面后的旧接受响应不得导航')
  } finally {
    await page.cleanup()
  }
})

test('发起页待处理弹窗确认必须校验原始身份', async () => {
  const modals = []
  const routes = []
  const page = await mountPage({
    file: 'subPages/match/challengeCompose.vue',
    user: { userId: 1, authGeneration: 1 },
    api: {
      getChallengeSummary: async () => ({ success: true, server_time: '2026-09-24 12:00:00' }),
      sendChallenge: async () => ({ success: false, pending_challenge_id: 77, message: '旧账号待回应邀请' })
    },
    globals: {
      uni: { showToast: () => {}, showModal: (options) => modals.push(options), navigateTo: ({ url }) => routes.push(url) }
    }
  })
  try {
    page.runHooks('onLoad', { opponent_id: 2 })
    page.runHooks('onShow')
    await page.flushPromises()
    await page.wrapper.find('.submit-btn').trigger('tap')
    await page.flushPromises()
    assert.equal(modals.length, 1)

    page.runHooks('onHide')
    // 切号后同一页面恢复：旧弹窗确认不得导航旧账号的邀请。
    page.user.userId = 3
    page.user.authGeneration = 2
    page.runHooks('onShow')
    await page.flushPromises()
    modals[0].success({ confirm: true })
    await page.flushPromises()
    assert.deepEqual(routes, [], '切号后的旧弹窗确认不得导航')
  } finally {
    await page.cleanup()
  }
})


test('首次历史读取途中恢复前台，旧批次结束后去重补读', async () => {
  let status = 0
  let reads = 0
  const stale = defer()
  const item = { id: 12, from_user_id: 1, to_user_id: 2, status: 0, game_type: 3 }
  const page = await mountPage({
    file: 'subPages/social/challenges.vue',
    user: { userId: 1, authGeneration: 1 },
    api: {
      getPendingChallenges: async () => ({ success: true, list: status === 0 ? [item] : [] }),
      getChallengeSummary: async () => ({ success: true, server_time: '2026-09-24 12:00:00' }),
      getChallengeHistory: () => (++reads === 1 ? stale.promise : Promise.resolve({ success: true, list: [{ ...item, status }] })),
      cancelChallenge: async () => {
        status = 4
        return { success: true }
      }
    },
    globals: {
      getCurrentPages: () => [{ options: { tab: 'history' } }],
      uni: { showToast: () => {} }
    }
  })
  try {
    await page.flushPromises()
    assert.equal(reads, 1)
    assert.ok(page.wrapper.find('.loading-state').exists())

    // 读取挂起期间离开又恢复：对方已取消邀请。
    page.runHooks('onHide')
    status = 4
    page.runHooks('onShow')
    await page.flushPromises()

    stale.resolve({ success: true, list: [item] })
    await page.flushPromises()

    assert.match(page.wrapper.text(), /已取消/, '旧批次返回后必须消费恢复期间的新失效，不能停在待回应缓存')
    assert.equal(reads, 2, '恢复所需的补读不能被旧 loading 吞掉，也不得无限自我重试')
  } finally {
    await page.cleanup()
  }
})

test('发起页持续停留跨午夜，提交与重试使用当前日期基准', async () => {
  let serverNow = '2026-09-24 23:59:50'
  let timeReads = 0
  const posted = []
  const page = await mountPage({
    file: 'subPages/match/challengeCompose.vue',
    user: { userId: 1, authGeneration: 1 },
    api: {
      getChallengeSummary: async () => {
        timeReads += 1
        return { success: true, server_time: serverNow }
      },
      // 模拟服务端：以提交时刻的北京时间校验相对日与实际日期一致。
      sendChallenge: async (payload) => {
        posted.push({ ...payload })
        return {
          success: payload.day_offset === challengeTime.dayOffsetOfScheduledDate(serverNow, payload.scheduled_date),
          message: '预约日期与所选相对日期不一致'
        }
      }
    },
    globals: {
      uni: { showToast: () => {}, switchTab: () => {} }
    }
  })
  try {
    page.runHooks('onLoad', { opponent_id: 2, scheduled_date: '2026-09-25', start_hour: 20, end_hour: 22 })
    page.runHooks('onShow')
    await page.flushPromises()

    // 页面保持打开跨过午夜：服务端日期前进，旧基准的相对日被拒绝。
    serverNow = '2026-09-25 00:01:00'
    await page.wrapper.find('.submit-btn').trigger('tap')
    await page.flushPromises()
    await page.wrapper.find('.submit-btn').trigger('tap')
    await page.flushPromises()

    assert.ok(posted.some((payload) => payload.day_offset === 0), '提交前刷新时间基准后，重试必须使用当前相对日')
    assert.equal(timeReads, 3, '两次提交各自刷新一次时间基准')
  } finally {
    // 等待成功路径的 600ms 延迟导航触发完毕，避免清理 uni 后产生未捕获异常。
    await new Promise((resolve) => setTimeout(resolve, 650))
    await page.cleanup()
  }
})


test('发起页提交链路：切号/卸载/重复点击不得发出写请求', async () => {
  const user = { userId: 1, authGeneration: 1 }
  let summaryReads = 0
  const writes = []
  let pendingSummary
  const page = await mountPage({
    file: 'subPages/match/challengeCompose.vue',
    user,
    api: {
      getChallengeSummary: () => {
        summaryReads += 1
        // 首读立即完成；提交预读挂起，模拟跨请求切号/卸载窗口。
        return summaryReads === 1
          ? Promise.resolve({ success: true, server_time: '2026-09-24 12:00:00' })
          : new Promise((resolve) => { pendingSummary = resolve })
      },
      sendChallenge: async (payload) => {
        writes.push({ ...payload })
        return { success: true, challenge_id: 77 }
      }
    },
    globals: {
      uni: { showToast: () => {}, switchTab: () => {} }
    }
  })
  try {
    page.runHooks('onLoad', { opponent_id: 2, message: 'A账号附言' })
    page.runHooks('onShow')
    await page.flushPromises()

    // 第一击进入挂起的提交前预读。
    await page.wrapper.find('.submit-btn').trigger('tap')
    await page.flushPromises()
    assert.equal(page.wrapper.find('.submit-btn').attributes('disabled'), '', '提交锁必须在时间预读前禁用按钮')
    assert.equal(summaryReads, 2)

    // 重复点击：不得再发起时间读取或 POST。
    await page.wrapper.find('.submit-btn').trigger('tap')
    await page.flushPromises()
    assert.equal(summaryReads, 2, '重复点击不得绕过提交锁')
    assert.equal(writes.length, 0)

    // 卸载页面并切号后，旧预读返回：不得用新账号 Token 发出 A 的表单。
    page.runHooks('onUnload')
    page.user.userId = 3
    page.user.authGeneration = 2
    pendingSummary({ success: true, server_time: '2026-09-24 12:00:01' })
    await page.flushPromises()
    assert.equal(writes.length, 0, '预读期间切号/卸载后不得继续发送写请求')
  } finally {
    await new Promise((resolve) => setTimeout(resolve, 650))
    await page.cleanup()
  }
})

test('发起页预读失败可重试且只发送一次', async () => {
  const user = { userId: 1, authGeneration: 1 }
  let summaryReads = 0
  const writes = []
  const page = await mountPage({
    file: 'subPages/match/challengeCompose.vue',
    user,
    api: {
      getChallengeSummary: () => {
        summaryReads += 1
        // 首读成功；提交预读第一次失败，重试成功。
        if (summaryReads === 1) return Promise.resolve({ success: true, server_time: '2026-09-24 12:00:00' })
        return summaryReads === 2 ? Promise.reject(new Error('time read failed')) : Promise.resolve({ success: true, server_time: '2026-09-24 12:00:00' })
      },
      sendChallenge: async (payload) => {
        writes.push({ ...payload })
        return { success: true, challenge_id: 78 }
      }
    },
    globals: {
      uni: { showToast: () => {}, switchTab: () => {} }
    }
  })
  try {
    page.runHooks('onLoad', { opponent_id: 2 })
    page.runHooks('onShow')
    await page.flushPromises()

    // 预读失败：不发送 POST，提交锁释放后可重试。
    await page.wrapper.find('.submit-btn').trigger('tap')
    await page.flushPromises()
    assert.equal(writes.length, 0, '预读失败不得发送 POST')
    assert.notEqual(page.wrapper.find('.submit-btn').attributes('disabled'), '', '预读失败后提交锁必须释放')

    // 重试成功：恰好一次 POST。
    await page.wrapper.find('.submit-btn').trigger('tap')
    await page.flushPromises()
    assert.equal(writes.length, 1, '重试应恰好发送一次')
    assert.equal(writes[0].day_offset, 0)
  } finally {
    await new Promise((resolve) => setTimeout(resolve, 650))
    await page.cleanup()
  }
})
