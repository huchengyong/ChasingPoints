import test, { mock } from 'node:test'
import assert from 'node:assert/strict'

import { createModuleHarness } from './helpers/page-harness.mjs'

const GLOBAL_KEY = '__MAIN_TAB_BAR__'

const stubSources = {
  '@dcloudio/uni-app': `
export const onShow = (callback) => globalThis.${GLOBAL_KEY}.onShow.push(callback)
export const onHide = (callback) => globalThis.${GLOBAL_KEY}.onHide.push(callback)
`,
  '@/store/theme.js': `export const useThemeStore = () => globalThis.${GLOBAL_KEY}.theme`,
  '@/store/user.js': `export const useUserStore = () => globalThis.${GLOBAL_KEY}.user`,
  '@/store/notification.js': `export const useNotificationStore = () => globalThis.${GLOBAL_KEY}.notification`,
  '@/store/friendRequest.js': `export const useFriendRequestStore = () => globalThis.${GLOBAL_KEY}.friendRequest`,
  '@/store/activity.js': `export const useActivityStore = () => globalThis.${GLOBAL_KEY}.activity`,
  '@/utils/quick-scan-entry.js': `export const startQuickScan = (...args) => globalThis.${GLOBAL_KEY}.scans.push(args)`
}

const harness = createModuleHarness({
  entry: '@/utils/use-main-tab-bar.js',
  globalKey: GLOBAL_KEY,
  stubSources
})
const testUtilsPromise = import('@vue/test-utils')

const ownedStore = (reactive, fields) => {
  const store = reactive({ ownerUserId: 7, authGeneration: 1, ...fields })
  store.matchesIdentity = (identity) => (
    store.ownerUserId === identity.userId && store.authGeneration === identity.authGeneration
  )
  return store
}

const nativeUni = () => {
  const calls = []
  const record = (name) => (options) => calls.push([name, options])
  return {
    calls,
    uni: {
      getSystemInfoSync: () => ({ uniPlatform: 'app-harmony' }),
      setTabBarBadge: record('setTabBarBadge'),
      showTabBarRedDot: record('showTabBarRedDot'),
      hideTabBarRedDot: record('hideTabBarRedDot'),
      removeTabBarBadge: record('removeTabBarBadge'),
      hideTabBar: record('hideTabBar'),
      showTabBar: record('showTabBar')
    }
  }
}

const mountComposable = async ({ tabKey = 'message', uni, counts = {}, userFields = {} } = {}) => {
  const { reactive, defineComponent, h, nextTick } = await harness.vueRuntimePromise
  const { mount } = await testUtilsPromise
  const context = {
    onShow: [],
    onHide: [],
    scans: [],
    theme: reactive({ isDarkMode: false }),
    user: reactive({ isLoggedIn: true, userId: 7, authGeneration: 1, ...userFields }),
    notification: ownedStore(reactive, { unreadCount: counts.unread ?? 0 }),
    friendRequest: ownedStore(reactive, { pendingCount: counts.friends ?? 0 }),
    activity: ownedStore(reactive, { receivedChallengeCount: counts.invites ?? 0 })
  }
  const { module, cleanup } = await harness.load({ context, uni })
  const wrapper = mount(defineComponent({
    setup() {
      module.useMainTabBar(tabKey)
      return () => h('div')
    }
  }))
  return {
    context,
    module,
    nextTick,
    show: () => context.onShow.forEach((callback) => callback()),
    hide: () => context.onHide.forEach((callback) => callback()),
    cleanup: async () => {
      wrapper.unmount()
      await cleanup()
      delete globalThis.getCurrentPages
    }
  }
}

test('native bar: showing the page applies the unread badge and never touches selection or theme', async () => {
  const { uni, calls } = nativeUni()
  const env = await mountComposable({ uni, counts: { unread: 3, friends: 1 } })
  try {
    env.show()

    assert.deepEqual(calls, [
      ['hideTabBarRedDot', { index: 2 }],
      ['setTabBarBadge', { index: 2, text: '3' }]
    ])
  } finally {
    await env.cleanup()
  }
})

test('native bar: badge follows count changes only while the page is visible', async () => {
  const { uni, calls } = nativeUni()
  const env = await mountComposable({ uni, counts: { unread: 0 } })
  try {
    env.show()
    calls.length = 0

    env.context.notification.unreadCount = 5
    await env.nextTick()
    assert.deepEqual(calls.at(-1), ['setTabBarBadge', { index: 2, text: '5' }])

    env.hide()
    calls.length = 0
    env.context.notification.unreadCount = 8
    await env.nextTick()
    assert.deepEqual(calls, [], 'a hidden page must not drive the shared native bar')

    env.show()
    assert.deepEqual(calls.at(-1), ['setTabBarBadge', { index: 2, text: '8' }], 'the next show catches up')
  } finally {
    await env.cleanup()
  }
})

test('native bar: pending friend requests or invites show a dot once the unread count is zero', async () => {
  const { uni, calls } = nativeUni()
  const env = await mountComposable({ uni, counts: { unread: 0, invites: 2 } })
  try {
    env.show()
    assert.deepEqual(calls, [
      ['removeTabBarBadge', { index: 2 }],
      ['showTabBarRedDot', { index: 2 }]
    ])
  } finally {
    await env.cleanup()
  }
})

test('counts of another account never reach the badge', async () => {
  const { uni, calls } = nativeUni()
  const env = await mountComposable({ uni, counts: { unread: 6, friends: 2, invites: 2 } })
  try {
    env.show()
    calls.length = 0

    // 新账号已登录，但各 store 仍属于旧身份，首次读取前不得显示旧计数
    env.context.user.userId = 8
    await env.nextTick()
    assert.deepEqual(calls, [
      ['removeTabBarBadge', { index: 2 }],
      ['hideTabBarRedDot', { index: 2 }]
    ])

    // 同一用户重新登录（认证代次变化）同理
    env.context.user.userId = 7
    await env.nextTick()
    env.context.user.authGeneration = 2
    await env.nextTick()
    assert.deepEqual(calls.at(-2), ['removeTabBarBadge', { index: 2 }])

    // 新身份的数据到位后恢复显示
    env.context.notification.ownerUserId = 7
    env.context.notification.authGeneration = 2
    env.context.notification.unreadCount = 1
    await env.nextTick()
    assert.deepEqual(calls.at(-1), ['setTabBarBadge', { index: 2, text: '1' }])
  } finally {
    await env.cleanup()
  }
})

test('logging out clears the badge immediately', async () => {
  const { uni, calls } = nativeUni()
  const env = await mountComposable({ uni, counts: { unread: 4 } })
  try {
    env.show()
    calls.length = 0

    env.context.user.isLoggedIn = false
    await env.nextTick()
    assert.deepEqual(calls, [
      ['removeTabBarBadge', { index: 2 }],
      ['hideTabBarRedDot', { index: 2 }]
    ])
  } finally {
    await env.cleanup()
  }
})

test('a modal that hid the bar keeps it hidden across re-shows until it is restored', async () => {
  const { uni, calls } = nativeUni()
  const env = await mountComposable({ uni })
  try {
    env.show()
    env.module.setMainTabBarVisible(false)
    assert.deepEqual(calls.at(-1), ['hideTabBar', { animation: true }])

    calls.length = 0
    env.show()
    assert.equal(calls.some(([name]) => name === 'showTabBar'), false, 're-showing the page must not reveal the bar under the modal')

    env.module.setMainTabBarVisible(true)
    assert.deepEqual(calls.at(-1), ['showTabBar', { animation: true }])
  } finally {
    await env.cleanup()
  }
})

const customUni = () => ({ getSystemInfoSync: () => ({ uniPlatform: 'mp-weixin' }) })

const installTabBar = ({ ready = true } = {}) => {
  const state = { setData: [], scanHandlers: [], lookups: 0 }
  const instance = {
    setData: (data) => state.setData.push(data),
    setScanHandler: (handler) => state.scanHandlers.push(handler)
  }
  globalThis.getCurrentPages = () => [{
    route: 'pages/message/index',
    getTabBar: () => {
      state.lookups += 1
      return typeof ready === 'function' ? (ready(state.lookups) ? instance : null) : (ready ? instance : null)
    }
  }]
  return state
}

test('custom bar: showing the page pushes selection, theme, badge and the scan handler', async () => {
  const state = installTabBar()
  const env = await mountComposable({ uni: customUni(), tabKey: 'message', counts: { unread: 120 } })
  try {
    env.context.theme.isDarkMode = true
    env.show()

    assert.deepEqual(state.setData.at(-1), {
      selected: 2,
      dark: true,
      hidden: false,
      badgeType: 'text',
      badgeText: '99+'
    })
    assert.equal(state.scanHandlers.length > 0, true)

    state.scanHandlers.at(-1)()
    assert.deepEqual(env.context.scans, [[]], 'the middle button runs the unified scan flow')
  } finally {
    await env.cleanup()
  }
})

test('custom bar: a theme switch while the page is visible is pushed right away', async () => {
  const state = installTabBar()
  const env = await mountComposable({ uni: customUni(), tabKey: 'user' })
  try {
    env.show()
    assert.equal(state.setData.at(-1).selected, 3)
    assert.equal(state.setData.at(-1).dark, false)

    env.context.theme.isDarkMode = true
    await env.nextTick()
    assert.equal(state.setData.at(-1).dark, true)
  } finally {
    await env.cleanup()
  }
})

test('custom bar: retries a couple of times when the instance is not mounted yet, then stops', async () => {
  mock.timers.enable({ apis: ['setTimeout'] })
  try {
    const state = installTabBar({ ready: (lookup) => lookup >= 3 })
    const env = await mountComposable({ uni: customUni(), tabKey: 'index' })
    try {
      env.show()
      assert.equal(state.setData.length, 0)

      mock.timers.tick(80)
      assert.equal(state.setData.length, 0)
      mock.timers.tick(80)
      assert.equal(state.setData.length, 1, 'the third lookup finds the instance')
      assert.equal(state.setData[0].selected, 0)
    } finally {
      await env.cleanup()
    }

    const never = installTabBar({ ready: false })
    const stuck = await mountComposable({ uni: customUni(), tabKey: 'index' })
    try {
      stuck.show()
      mock.timers.tick(80)
      mock.timers.tick(80)
      mock.timers.tick(80)
      mock.timers.tick(80)
      assert.equal(never.lookups, 3, 'one initial attempt plus two retries, never an endless loop')
    } finally {
      await stuck.cleanup()
    }
  } finally {
    mock.timers.reset()
  }
})

test('custom bar: a hidden page stops pushing and the retry timer does nothing after hide', async () => {
  mock.timers.enable({ apis: ['setTimeout'] })
  try {
    const state = installTabBar({ ready: (lookup) => lookup >= 2 })
    const env = await mountComposable({ uni: customUni(), tabKey: 'index' })
    try {
      env.show()
      env.hide()
      mock.timers.tick(80)
      assert.equal(state.setData.length, 0)
    } finally {
      await env.cleanup()
    }
  } finally {
    mock.timers.reset()
  }
})
