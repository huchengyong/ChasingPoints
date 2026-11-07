import test, { mock } from 'node:test'
import assert from 'node:assert/strict'

import { createPageHarness, createUniRecorder, defer } from './helpers/page-harness.mjs'

const GLOBAL_KEY = '__MESSAGE_PAGE__'

const stubSources = {
  '@dcloudio/uni-app': `
export const onShow = (callback) => globalThis.${GLOBAL_KEY}.onShow.push(callback)
export const onHide = (callback) => globalThis.${GLOBAL_KEY}.onHide.push(callback)
export const onPullDownRefresh = (callback) => globalThis.${GLOBAL_KEY}.onPullDown.push(callback)
`,
  '@/api/notification.js': `
const call = (name) => (...args) => globalThis.${GLOBAL_KEY}.api[name](...args)
export const getNotificationList = call('getNotificationList')
export const markAsRead = call('markAsRead')
export const markAllAsRead = call('markAllAsRead')
export const deleteNotification = call('deleteNotification')
`,
  '@/store/user.js': `export const useUserStore = () => globalThis.${GLOBAL_KEY}.user`,
  '@/store/activity.js': `export const useActivityStore = () => globalThis.${GLOBAL_KEY}.activity`,
  '@/store/notification.js': `export const useNotificationStore = () => globalThis.${GLOBAL_KEY}.notification`,
  '@/store/friendRequest.js': `export const useFriendRequestStore = () => globalThis.${GLOBAL_KEY}.friendRequest`,
  '@/store/userDataInvalidation.js': `export const useUserDataInvalidationStore = () => globalThis.${GLOBAL_KEY}.invalidation`,
  '@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`,
  '@/utils/use-main-tab-bar.js': `export const useMainTabBar = (tabKey) => globalThis.${GLOBAL_KEY}.tabBars.push(tabKey)`,
  '@/utils/use-message-counts.js': `
export const useMessageCounts = () => ({
  friendRequestCount: globalThis.${GLOBAL_KEY}.counts.friends,
  challengeCount: globalThis.${GLOBAL_KEY}.counts.invites
})
`
}

const realModules = [
  '@/utils/message-center.js',
  '@/utils/async-page-state.js',
  '@/utils/request-errors.js'
]

const harness = createPageHarness({
  pageFile: 'pages/message/index.vue',
  globalKey: GLOBAL_KEY,
  stubSources,
  realModules
})

const note = (id, extra = {}) => ({
  id,
  type: 'system',
  title: `通知${id}`,
  content: `内容${id}`,
  created_at: '2026-10-24 10:00',
  is_read: 0,
  data: '',
  ...extra
})

const listResponse = (list, extra = {}) => ({ success: true, list, unread_count: list.filter((item) => !item.is_read).length, ...extra })

const mountMessage = async ({ api = {}, loggedIn = true, friends = 0, invites = 0 } = {}) => {
  const { reactive, ref } = await harness.vueRuntimePromise
  const recorder = createUniRecorder()
  const calls = { list: [], activityFetch: [], unread: [], read: [], friendCount: [] }
  const context = {
    onShow: [],
    onHide: [],
    onPullDown: [],
    tabBars: [],
    api: {
      getNotificationList: async (params) => { calls.list.push(params); return listResponse([note(1), note(2, { is_read: 1 })]) },
      markAsRead: async (payload) => { calls.read.push(payload); return { success: true } },
      markAllAsRead: async () => ({ success: true }),
      deleteNotification: async () => ({ success: true }),
      ...api
    },
    user: reactive({ isLoggedIn: loggedIn, userId: loggedIn ? 7 : 0, authGeneration: 1 }),
    activity: { fetch: (...args) => { calls.activityFetch.push(args); return Promise.resolve({}) } },
    notification: {
      setUnreadCount: (...args) => calls.unread.push(['set', ...args]),
      decrementUnread: (...args) => calls.unread.push(['decrement', ...args]),
      clearUnread: (...args) => calls.unread.push(['clear', ...args])
    },
    friendRequest: { fetchPendingCount: (...args) => calls.friendCount.push(args) },
    versions: reactive({ notification: 0 }),
    counts: { friends: ref(friends), invites: ref(invites) }
  }
  context.invalidation = { versionOf: (scope) => context.versions[scope] ?? 0 }
  const mounted = await harness.mountPage({ context, uni: recorder.uni })
  const show = async () => {
    context.onShow.forEach((callback) => callback())
    await mounted.flushPromises()
  }
  return { ...mounted, context, calls, recorder, show }
}

test('a guest sees the login prompt and nothing is requested', async () => {
  const { wrapper, flushPromises, context, calls, recorder, show, cleanup } = await mountMessage({ loggedIn: false })
  try {
    await show()

    assert.match(wrapper.text(), /登录后查看消息/)
    assert.equal(wrapper.find('.entry-card').exists(), false)
    assert.equal(calls.list.length, 0)
    assert.equal(calls.activityFetch.length, 0)
    assert.deepEqual(context.tabBars, ['message'])

    await wrapper.find('.login-btn').trigger('tap')
    await flushPromises()
    assert.deepEqual(recorder.lastOf('navigateTo').options, { url: '/pages/login/login' })

    for (const callback of context.onPullDown) await callback()
    assert.equal(recorder.callsOf('stopPullDownRefresh').length, 1)
    assert.equal(calls.list.length, 0)
  } finally {
    await cleanup()
  }
})

test('first show loads the first notification page once and takes counts from the shared snapshot', async () => {
  const { wrapper, calls, show, cleanup } = await mountMessage({ friends: 2, invites: 1 })
  try {
    await show()

    assert.deepEqual(calls.list, [{ page: 1, page_size: 20 }])
    assert.equal(wrapper.findAll('.notification-item').length, 2)
    assert.deepEqual(calls.unread, [['set', 1, { userId: 7, authGeneration: 1 }]], 'the unread count comes from the list response itself')

    // 两个入口的数量来自共享活动快照，页面不另发计数请求
    const entries = wrapper.findAll('.entry-item')
    assert.equal(entries.length, 2)
    assert.match(entries[0].text(), /好友申请[\s\S]*2 条待处理/)
    assert.match(entries[1].text(), /约球邀请[\s\S]*1 条待回应/)
    assert.deepEqual(calls.activityFetch, [[{ userId: 7, authGeneration: 1 }, { silent: true }]], 'not forced: the store decides by TTL and dirty')
  } finally {
    await cleanup()
  }
})

test('showing the tab again without a change does not reload the list', async () => {
  const { calls, show, cleanup } = await mountMessage()
  try {
    await show()
    await show()
    await show()

    assert.equal(calls.list.length, 1)
  } finally {
    await cleanup()
  }
})

test('a notification invalidation that happened elsewhere reloads the first page on the next show', async () => {
  const { context, calls, show, cleanup } = await mountMessage()
  try {
    await show()
    context.onHide.forEach((callback) => callback())

    context.versions.notification += 1
    await show()

    assert.equal(calls.list.length, 2)
    assert.deepEqual(calls.list[1], { page: 1, page_size: 20 })
  } finally {
    await cleanup()
  }
})

test('a burst of invalidations while the tab is visible triggers a single refresh', async () => {
  mock.timers.enable({ apis: ['setTimeout'] })
  try {
    const { context, calls, show, cleanup } = await mountMessage()
    try {
      await show()
      calls.list.length = 0

      context.versions.notification += 1
      await Promise.resolve()
      mock.timers.tick(200)
      context.versions.notification += 1
      await Promise.resolve()
      mock.timers.tick(200)
      context.versions.notification += 1
      await Promise.resolve()
      mock.timers.tick(499)
      assert.equal(calls.list.length, 0, 'still inside the debounce window')

      mock.timers.tick(1)
      assert.equal(calls.list.length, 1)
    } finally {
      await cleanup()
    }
  } finally {
    mock.timers.reset()
  }
})

test('invalidations while the tab is hidden never fire a refresh by themselves', async () => {
  mock.timers.enable({ apis: ['setTimeout'] })
  try {
    const { context, calls, show, cleanup } = await mountMessage()
    try {
      await show()
      calls.list.length = 0
      context.onHide.forEach((callback) => callback())

      context.versions.notification += 1
      await Promise.resolve()
      mock.timers.tick(5000)

      assert.equal(calls.list.length, 0)
    } finally {
      await cleanup()
    }
  } finally {
    mock.timers.reset()
  }
})

test('a refresh requested while a load is running is queued and runs right after it', async () => {
  const first = defer()
  let requests = 0
  const { context, calls, flushPromises, show, cleanup } = await mountMessage({
    api: {
      getNotificationList: (params) => {
        requests += 1
        calls.list.push(params)
        return requests === 1 ? first.promise : Promise.resolve(listResponse([note(1), note(2), note(3)]))
      }
    }
  })
  mock.timers.enable({ apis: ['setTimeout'] })
  try {
    await show()
    assert.equal(requests, 1)

    context.versions.notification += 1
    await Promise.resolve()
    mock.timers.tick(500)
    assert.equal(requests, 1, 'the running load is not duplicated')

    first.resolve(listResponse([note(1)]))
    await flushPromises()
    assert.equal(requests, 2, 'the queued refresh runs after the first load')
  } finally {
    mock.timers.reset()
    await cleanup()
  }
})

test('switching account clears the list and drops a late response of the old account', async () => {
  const pending = defer()
  let requests = 0
  const { wrapper, context, calls, flushPromises, show, cleanup } = await mountMessage({
    api: {
      getNotificationList: () => {
        requests += 1
        return requests === 1 ? pending.promise : Promise.resolve(listResponse([note(9, { title: '新账号的通知' })]))
      }
    }
  })
  try {
    await show()
    context.user.userId = 8
    context.user.authGeneration = 2
    await show()

    assert.equal(requests, 2, 'the new identity loads on its own')
    assert.match(wrapper.text(), /新账号的通知/)

    pending.resolve(listResponse([note(1, { title: '旧账号的通知' })], { unread_count: 5 }))
    await flushPromises()

    assert.doesNotMatch(wrapper.text(), /旧账号的通知/)
    assert.equal(calls.unread.some(([kind, count]) => kind === 'set' && count === 5), false, 'the old account unread count is not written')
  } finally {
    await cleanup()
  }
})

test('logging out clears what was loaded and shows the login prompt', async () => {
  const { wrapper, context, show, cleanup } = await mountMessage()
  try {
    await show()
    assert.equal(wrapper.findAll('.notification-item').length, 2)

    context.user.isLoggedIn = false
    await show()

    assert.match(wrapper.text(), /登录后查看消息/)
    assert.equal(wrapper.findAll('.notification-item').length, 0)
  } finally {
    await cleanup()
  }
})

test('opening an unread notification marks it read, updates the shared counts and navigates', async () => {
  const { wrapper, flushPromises, calls, recorder, show, cleanup } = await mountMessage({
    api: {
      getNotificationList: async () => listResponse([
        note(1, { type: 'friend_request', title: '收到好友申请' }),
        note(2, { type: 'match_result', data: JSON.stringify({ target: '/subPages/match/matchResult', match_id: 66 }) })
      ])
    }
  })
  try {
    await show()
    calls.unread.length = 0

    await wrapper.findAll('.notification-item')[0].trigger('tap')
    await flushPromises()

    assert.deepEqual(calls.read, [{ notification_id: 1 }])
    assert.deepEqual(calls.unread, [['decrement', { userId: 7, authGeneration: 1 }]])
    assert.equal(calls.friendCount.length, 1, 'a friend request notification refreshes the pending request count')
    assert.deepEqual(recorder.lastOf('navigateTo').options, { url: '/subPages/social/friendRequests' })

    await wrapper.findAll('.notification-item')[1].trigger('tap')
    await flushPromises()
    assert.deepEqual(recorder.lastOf('navigateTo').options, { url: '/subPages/match/matchResult?match_id=66&from=history' })
  } finally {
    await cleanup()
  }
})

test('mark all read and delete keep the unread counter consistent', async () => {
  const { wrapper, flushPromises, calls, show, cleanup } = await mountMessage()
  try {
    await show()
    calls.unread.length = 0

    await wrapper.find('.item-delete').trigger('tap')
    await flushPromises()
    assert.deepEqual(calls.unread, [['decrement', { userId: 7, authGeneration: 1 }]], 'deleting an unread item lowers the count')
    assert.equal(wrapper.findAll('.notification-item').length, 1)

    calls.unread.length = 0
    await wrapper.find('.read-all-btn').trigger('tap')
    await flushPromises()
    assert.deepEqual(calls.unread, [['clear', { userId: 7, authGeneration: 1 }]])
  } finally {
    await cleanup()
  }
})

test('entries open the friend requests and the received invitations', async () => {
  const { wrapper, flushPromises, recorder, show, cleanup } = await mountMessage({ friends: 1, invites: 3 })
  try {
    await show()
    const entries = wrapper.findAll('.entry-item')

    await entries[0].trigger('tap')
    await flushPromises()
    assert.deepEqual(recorder.lastOf('navigateTo').options, { url: '/subPages/social/friendRequests' })

    await entries[1].trigger('tap')
    await flushPromises()
    assert.deepEqual(recorder.lastOf('navigateTo').options, { url: '/subPages/social/challenges?tab=received' })
  } finally {
    await cleanup()
  }
})

test('pull down refreshes the list and forces the shared counts, then stops the spinner', async () => {
  const { context, calls, flushPromises, recorder, show, cleanup } = await mountMessage()
  try {
    await show()
    calls.list.length = 0
    calls.activityFetch.length = 0

    for (const callback of context.onPullDown) await callback()
    await flushPromises()

    assert.equal(calls.list.length, 1)
    assert.deepEqual(calls.activityFetch, [[{ userId: 7, authGeneration: 1 }, { force: true, silent: true }]])
    assert.equal(recorder.callsOf('stopPullDownRefresh').length, 1)
  } finally {
    await cleanup()
  }
})

test('a failed first load shows a retryable error and a failed refresh keeps the list with a banner', async () => {
  let mode = 'fail'
  const { wrapper, context, flushPromises, show, cleanup } = await mountMessage({
    api: {
      getNotificationList: async () => {
        if (mode === 'fail') throw new Error('offline')
        return listResponse([note(1)])
      }
    }
  })
  const originalError = console.error
  console.error = () => {}
  try {
    await show()
    assert.equal(wrapper.find('.error-state').exists(), true)

    mode = 'ok'
    await wrapper.find('.error-action').trigger('tap')
    await flushPromises()
    assert.equal(wrapper.findAll('.notification-item').length, 1)

    mode = 'fail'
    context.versions.notification += 1
    context.onHide.forEach((callback) => callback())
    await show()
    assert.equal(wrapper.findAll('.notification-item').length, 1, 'the previous list stays visible')
    assert.equal(wrapper.find('.refresh-error-banner').exists(), true)
  } finally {
    console.error = originalError
    await cleanup()
  }
})

test('an empty list shows the empty state and "load more" appears only for a full page', async () => {
  const empty = await mountMessage({ api: { getNotificationList: async () => listResponse([]) } })
  try {
    await empty.show()
    assert.match(empty.wrapper.text(), /暂无消息/)
    assert.equal(empty.wrapper.find('.load-more').exists(), false)
  } finally {
    await empty.cleanup()
  }

  const full = await mountMessage({
    api: {
      getNotificationList: async (params) => listResponse(
        Array.from({ length: 20 }, (_, index) => note(params.page * 100 + index))
      )
    }
  })
  try {
    await full.show()
    assert.equal(full.wrapper.find('.load-more').exists(), true)
    await full.wrapper.find('.load-more').trigger('tap')
    await full.flushPromises()
    assert.equal(full.wrapper.findAll('.notification-item').length, 40)
  } finally {
    await full.cleanup()
  }
})
