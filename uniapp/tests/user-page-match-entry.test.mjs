import test, { after } from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'

const appDir = dirname(dirname(fileURLToPath(import.meta.url)))
const PAGE_FILE = 'pages/user/index.vue'
const CUSTOM_ELEMENTS = ['view', 'text', 'image', 'canvas', 'uni-icons']

// 动态桩：行为由每个用例通过 globalThis.__USER_MATCH_ENTRY__ 注入。
const stubSources = {
  '@dcloudio/uni-app': `
export const onShow = () => {}
export const onPullDownRefresh = () => {}
`,
  '@/store/user.js': `
export const useUserStore = () => globalThis.__USER_MATCH_ENTRY__.user
`,
  '@/store/activity.js': `
export const useActivityStore = () => ({ latestSeasonRollover: null })
`,
  '@/store/userOverview.js': `
export const useUserOverviewStore = () => ({})
`,
  '@/store/rank.js': `
export const useRankStore = () => ({ loaded: false, ownerUserId: 0, rankInfoMap: {}, loading: false })
`,
  '@/store/notification.js': `
export const useNotificationStore = () => ({ unreadCount: 0 })
`,
  '@/store/friendRequest.js': `
export const useFriendRequestStore = () => ({ pendingCount: 0 })
`,
  '@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`,
  '@/utils/user-profile.js': `
export const resolveAvatarUrl = () => ''
`,
  '@/api/notification.js': `
export const markAsRead = async () => ({ success: true })
`,
  '@/api/match.js': `
const call = (name) => (...args) => globalThis.__USER_MATCH_ENTRY__.api[name](...args)
export const getMatchQRCode = call('getMatchQRCode')
`,
  '@/utils/use-main-tab-bar.js': `
export const useMainTabBar = (tabKey) => globalThis.__USER_MATCH_ENTRY__.tabBars.push(tabKey)
export const setMainTabBarVisible = (visible) => globalThis.__USER_MATCH_ENTRY__.tabBarVisibility.push(visible)
`,
  '@/utils/local-qrcode.js': `
export const renderLocalQRCode = (options) => {
  globalThis.__USER_MATCH_ENTRY__.renderedQRCodes.push(options)
}
`
}

// 纯逻辑工具直接复用真实实现。
const realModules = [
  '@/utils/user-homepage.js',
  '@/utils/game-types.js',
  '@/utils/honor-wall.js',
  '@/utils/game-type-preference.js',
  '@/utils/favorite-venue-reward.js',
  '@/utils/member-center.js'
]

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

// Vue 的 runtime-dom 在模块加载时抓取 document，因此必须先装好 jsdom 再引入 vue。
const pageDom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
const previousGlobals = installDOMGlobals(pageDom.window)
const vueRuntimePromise = import('vue')
const testUtilsPromise = import('@vue/test-utils')

after(() => {
  restoreDOMGlobals(previousGlobals)
  pageDom.window.close()
})

async function compilePage(tempDir) {
  const source = await readFile(join(appDir, PAGE_FILE), 'utf8')
  const { descriptor, errors } = parse(source, { filename: PAGE_FILE })
  assert.deepEqual(errors, [])
  const compiled = compileScript(descriptor, {
    id: 'user-page-match-entry',
    inlineTemplate: true,
    templateOptions: {
      compilerOptions: {
        isCustomElement: (tag) => CUSTOM_ELEMENTS.includes(tag)
      }
    }
  })
  let moduleSource = compiled.content
  for (const [specifier, content] of Object.entries(stubSources)) {
    if (!moduleSource.includes(`'${specifier}'`)) continue
    const stubName = `stub-${specifier.replace(/\W+/g, '_')}.mjs`
    await writeFile(join(tempDir, stubName), content)
    moduleSource = moduleSource.replaceAll(`from '${specifier}'`, `from './${stubName}'`)
  }
  for (const specifier of realModules) {
    moduleSource = moduleSource.replaceAll(`from '${specifier}'`, `from '../${specifier.replace('@/', '')}'`)
  }
  assert.doesNotMatch(moduleSource, /from '@\//, '页面的 @/ 依赖都必须有桩或真实实现')
  const fileName = 'page-user-index.mjs'
  await writeFile(join(tempDir, fileName), moduleSource)
  return pathToFileURL(join(tempDir, fileName)).href
}

function createUniStub({ actionSheetIndex = 0 } = {}) {
  const calls = []
  const storage = new Map()
  const record = (name) => (options) => { calls.push({ name, options }) }
  return {
    calls,
    callsOf: (name) => calls.filter((item) => item.name === name),
    uni: {
      getStorageSync: (key) => storage.get(key) ?? '',
      setStorageSync: (key, value) => { storage.set(key, value) },
      removeStorageSync: (key) => { storage.delete(key) },
      showActionSheet: (options) => {
        calls.push({ name: 'showActionSheet', options })
        options.success?.({ tapIndex: actionSheetIndex })
      },
      showLoading: record('showLoading'),
      hideLoading: record('hideLoading'),
      showToast: record('showToast'),
      showModal: record('showModal'),
      navigateTo: record('navigateTo'),
      switchTab: record('switchTab'),
      hideTabBar: record('hideTabBar'),
      showTabBar: record('showTabBar')
    }
  }
}

async function mountPage({ api = {}, uniStub = createUniStub() } = {}) {
  const tempDir = await mkdtemp(join(appDir, '.user-match-entry-'))
  const { reactive } = await vueRuntimePromise
  const user = reactive({
    isLoggedIn: true,
    userId: 7,
    authGeneration: 1,
    userInfo: { id: 7, nickname: '测试球友', avatar: '' }
  })
  const renderedQRCodes = []
  const tabBars = []
  const tabBarVisibility = []
  globalThis.uni = uniStub.uni
  globalThis.__USER_MATCH_ENTRY__ = { user, api, renderedQRCodes, tabBars, tabBarVisibility }
  const cleanupGlobals = () => {
    delete globalThis.__USER_MATCH_ENTRY__
    delete globalThis.uni
  }
  try {
    const moduleUrl = await compilePage(tempDir)
    const [{ mount, flushPromises }, componentModule] = await Promise.all([
      testUtilsPromise,
      import(`${moduleUrl}?v=${Date.now()}`)
    ])
    const wrapper = mount(componentModule.default, { global: { stubs: { 'uni-icons': true } } })
    return {
      wrapper,
      user,
      renderedQRCodes,
      tabBars,
      tabBarVisibility,
      flushPromises,
      cleanup: async () => {
        wrapper.unmount()
        cleanupGlobals()
        await rm(tempDir, { recursive: true, force: true })
      }
    }
  } catch (error) {
    cleanupGlobals()
    await rm(tempDir, { recursive: true, force: true })
    throw error
  }
}

test('出示二维码时隐藏 TabBar 并本地渲染，关闭后迟到的二维码不再渲染', async () => {
  const uniStub = createUniStub()
  const lateQRCode = defer()
  let reads = 0
  const page = await mountPage({
    uniStub,
    api: {
      getMatchQRCode: () => {
        reads += 1
        return reads === 1
          ? Promise.resolve({ success: true, qrcode_data: 'my.invite' })
          : lateQRCode.promise
      }
    }
  })
  try {
    const qrButton = page.wrapper.find('[aria-label="出示我的二维码"]')
    await qrButton.trigger('click')
    await page.flushPromises()

    assert.deepEqual(page.tabBars, ['user'], '页面接入主导航')
    assert.equal(page.wrapper.find('.qr-mask').exists(), true)
    assert.deepEqual(page.tabBarVisibility, [false], '打开二维码弹层时隐藏 TabBar')
    assert.deepEqual(
      page.renderedQRCodes.map(({ canvasId, content }) => ({ canvasId, content })),
      [{ canvasId: 'user-match-invite-qrcode', content: 'my.invite' }]
    )

    await page.wrapper.find('.qr-done').trigger('click')
    assert.equal(page.wrapper.find('.qr-mask').exists(), false)
    assert.deepEqual(page.tabBarVisibility, [false, true], '关闭后恢复 TabBar')

    await qrButton.trigger('click')
    await page.wrapper.find('.qr-close').trigger('click')
    lateQRCode.resolve({ success: true, qrcode_data: 'late.invite' })
    await page.flushPromises()

    assert.equal(page.renderedQRCodes.length, 1, '弹窗关闭后迟到的二维码不得渲染')
    assert.equal(page.wrapper.find('.qr-mask').exists(), false)
  } finally {
    await page.cleanup()
  }
})
