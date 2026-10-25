import test, { beforeEach } from 'node:test'
import assert from 'node:assert/strict'

import {
  MAIN_TABS,
  getMainTabBarHidden,
  isMainTabRoute,
  resetMainTabBarState,
  resolveMainTabBarPlatform,
  resolveMainTabBarState,
  resolveMainTabIndex,
  resolveMessageBadge,
  setMainTabBarVisible,
  syncMainTabBar
} from '../utils/main-tab-bar.js'
import { readPagesJson } from './helpers/pages-json.mjs'

beforeEach(() => {
  resetMainTabBarState()
})

const createCustomEnv = ({ tabBar, route = 'pages/message/index' } = {}) => {
  const dataCalls = []
  const scanHandlers = []
  const instance = tabBar === undefined
    ? {
        setData: (data) => dataCalls.push(data),
        setScanHandler: (handler) => scanHandlers.push(handler)
      }
    : tabBar
  return {
    dataCalls,
    scanHandlers,
    uniApi: { getSystemInfoSync: () => ({ uniPlatform: 'mp-weixin' }) },
    getPages: () => [
      { route: 'pages/login/login', getTabBar: () => null },
      { route, getTabBar: () => instance }
    ]
  }
}

const createNativeEnv = () => {
  const calls = []
  const record = (name) => (options) => calls.push([name, options])
  return {
    calls,
    uniApi: {
      getSystemInfoSync: () => ({ uniPlatform: 'app-harmony' }),
      setTabBarBadge: record('setTabBarBadge'),
      showTabBarRedDot: record('showTabBarRedDot'),
      hideTabBarRedDot: record('hideTabBarRedDot'),
      removeTabBarBadge: record('removeTabBarBadge'),
      hideTabBar: record('hideTabBar'),
      showTabBar: record('showTabBar')
    },
    getPages: () => []
  }
}

test('main tabs follow the pages.json tabBar list and keep the message tab at the badge index', () => {
  const tabBar = readPagesJson().tabBar
  assert.deepEqual(MAIN_TABS.map((tab) => tab.route), tabBar.list.map((item) => item.pagePath))
  assert.deepEqual(MAIN_TABS.map((tab) => tab.key), ['index', 'tournament', 'message', 'user'])
  // 鸿蒙原生 TabBar 的 midButton 不占下标，角标下标就是 list 中的位置。
  assert.equal(MAIN_TABS.findIndex((tab) => tab.key === 'message'), 2)
  // midButton 位于 list.length / 2 项之后，也就是赛讯与消息之间。
  assert.equal(tabBar.list.length / 2, 2)
})

test('route helpers accept leading slashes and query strings', () => {
  assert.equal(isMainTabRoute('/pages/message/index?x=1'), true)
  assert.equal(isMainTabRoute('pages/match/index'), false)
  assert.equal(resolveMainTabIndex('pages/user/index'), 3)
  assert.equal(resolveMainTabIndex('subPages/user/settings'), -1)
  assert.equal(resolveMainTabIndex(''), -1)
})

test('message badge shows unread count, then a dot for pending items, and nothing when logged out', () => {
  assert.deepEqual(
    resolveMessageBadge({ isLoggedIn: true, unreadCount: 3, friendRequestCount: 1, challengeCount: 2 }),
    { type: 'text', text: '3' }
  )
  assert.deepEqual(resolveMessageBadge({ isLoggedIn: true, unreadCount: 120 }), { type: 'text', text: '99+' })
  assert.deepEqual(resolveMessageBadge({ isLoggedIn: true, unreadCount: 99 }), { type: 'text', text: '99' })
  assert.deepEqual(resolveMessageBadge({ isLoggedIn: true, friendRequestCount: 1 }), { type: 'dot', text: '' })
  assert.deepEqual(resolveMessageBadge({ isLoggedIn: true, challengeCount: 2 }), { type: 'dot', text: '' })
  assert.deepEqual(resolveMessageBadge({ isLoggedIn: true }), { type: 'none', text: '' })
  assert.deepEqual(
    resolveMessageBadge({ isLoggedIn: false, unreadCount: 5, friendRequestCount: 2, challengeCount: 1 }),
    { type: 'none', text: '' }
  )
})

test('message badge treats invalid counts as zero', () => {
  assert.deepEqual(
    resolveMessageBadge({ isLoggedIn: true, unreadCount: -2, friendRequestCount: Number.NaN, challengeCount: 'x' }),
    { type: 'none', text: '' }
  )
  assert.deepEqual(resolveMessageBadge({ isLoggedIn: true, unreadCount: '7' }), { type: 'text', text: '7' })
})

test('resolveMainTabBarState combines selection, theme, visibility and the message badge', () => {
  assert.deepEqual(
    resolveMainTabBarState({
      tabKey: 'message',
      isDarkMode: true,
      hidden: true,
      badge: { type: 'text', text: '9' }
    }),
    { selected: 2, dark: true, hidden: true, badgeType: 'text', badgeText: '9' }
  )
  assert.deepEqual(
    resolveMainTabBarState({ tabKey: 'index' }),
    { selected: 0, dark: false, hidden: false, badgeType: 'none', badgeText: '' }
  )
  assert.equal(resolveMainTabBarState({ tabKey: 'unknown' }).selected, -1)
})

test('platform: only the WeChat mini program uses the custom tab bar component', () => {
  const withPlatform = (uniPlatform) => ({ getSystemInfoSync: () => ({ uniPlatform }) })
  assert.equal(resolveMainTabBarPlatform(withPlatform('mp-weixin')), 'custom')
  assert.equal(resolveMainTabBarPlatform(withPlatform('app-harmony')), 'native')
  assert.equal(resolveMainTabBarPlatform(withPlatform('web')), 'native')
  assert.equal(resolveMainTabBarPlatform({}), 'native')
  assert.equal(resolveMainTabBarPlatform({ getSystemInfoSync: () => { throw new Error('boom') } }), 'native')
})

test('custom tab bar: pushes state and the scan handler to the current page instance', () => {
  const env = createCustomEnv()
  const onScan = () => {}
  const state = { selected: 2, dark: true, hidden: false, badgeType: 'dot', badgeText: '' }

  assert.equal(syncMainTabBar({ state, onScan, ...env }), true)
  assert.deepEqual(env.dataCalls, [state])
  assert.deepEqual(env.scanHandlers, [onScan])
})

test('custom tab bar: a partial patch only sets the given keys', () => {
  const env = createCustomEnv()

  assert.equal(syncMainTabBar({ state: { hidden: true }, ...env }), true)
  assert.deepEqual(env.dataCalls, [{ hidden: true }])
  assert.deepEqual(env.scanHandlers, [])
})

test('custom tab bar: reports false instead of throwing when the instance is not ready', () => {
  assert.equal(syncMainTabBar({ state: { selected: 0 }, ...createCustomEnv({ tabBar: null }) }), false)
  const noMethod = createCustomEnv()
  const pages = [{ route: 'subPages/match/playing' }]
  assert.equal(syncMainTabBar({ state: { selected: 0 }, uniApi: noMethod.uniApi, getPages: () => pages }), false)
  assert.equal(syncMainTabBar({ state: { selected: 0 }, uniApi: noMethod.uniApi, getPages: () => [] }), false)
})

test('native tab bar: badge calls reset first and use the message tab index', () => {
  const env = createNativeEnv()

  syncMainTabBar({ state: { badgeType: 'text', badgeText: '12' }, ...env })
  assert.deepEqual(env.calls, [
    ['hideTabBarRedDot', { index: 2 }],
    ['setTabBarBadge', { index: 2, text: '12' }]
  ])

  env.calls.length = 0
  syncMainTabBar({ state: { badgeType: 'dot', badgeText: '' }, ...env })
  assert.deepEqual(env.calls, [
    ['removeTabBarBadge', { index: 2 }],
    ['showTabBarRedDot', { index: 2 }]
  ])

  env.calls.length = 0
  syncMainTabBar({ state: { badgeType: 'none', badgeText: '' }, ...env })
  assert.deepEqual(env.calls, [
    ['removeTabBarBadge', { index: 2 }],
    ['hideTabBarRedDot', { index: 2 }]
  ])
})

test('native tab bar: repeated identical state does not call the native bar again', () => {
  const env = createNativeEnv()
  const state = { badgeType: 'text', badgeText: '5', hidden: true }

  syncMainTabBar({ state, ...env })
  const firstRound = env.calls.length
  assert.ok(firstRound > 0)

  syncMainTabBar({ state, ...env })
  assert.equal(env.calls.length, firstRound)

  syncMainTabBar({ state: { badgeType: 'text', badgeText: '6', hidden: true }, ...env })
  assert.deepEqual(env.calls.slice(firstRound), [
    ['hideTabBarRedDot', { index: 2 }],
    ['setTabBarBadge', { index: 2, text: '6' }]
  ])
})

test('native tab bar: hides and shows only when visibility actually changes', () => {
  const env = createNativeEnv()

  syncMainTabBar({ state: { hidden: false }, ...env })
  assert.deepEqual(env.calls, [])

  syncMainTabBar({ state: { hidden: true }, ...env })
  syncMainTabBar({ state: { hidden: true }, ...env })
  syncMainTabBar({ state: { hidden: false }, ...env })
  assert.deepEqual(env.calls, [
    ['hideTabBar', { animation: true }],
    ['showTabBar', { animation: true }]
  ])
})

test('native tab bar ignores selection and theme keys because the runtime owns them', () => {
  const env = createNativeEnv()

  syncMainTabBar({ state: { selected: 2, dark: true }, ...env })
  assert.deepEqual(env.calls, [])
})

test('setMainTabBarVisible remembers the requested visibility and applies it on both platforms', () => {
  const custom = createCustomEnv()
  assert.equal(getMainTabBarHidden(), false)
  setMainTabBarVisible(false, custom)
  assert.equal(getMainTabBarHidden(), true)
  assert.deepEqual(custom.dataCalls, [{ hidden: true }])

  setMainTabBarVisible(true, custom)
  assert.equal(getMainTabBarHidden(), false)
  assert.deepEqual(custom.dataCalls, [{ hidden: true }, { hidden: false }])

  const native = createNativeEnv()
  setMainTabBarVisible(false, native)
  setMainTabBarVisible(true, native)
  assert.deepEqual(native.calls, [
    ['hideTabBar', { animation: true }],
    ['showTabBar', { animation: true }]
  ])
})
