import test from 'node:test'
import assert from 'node:assert/strict'

import { createPageHarness, createUniRecorder, defer } from './helpers/page-harness.mjs'
import { createGearContext, gearRealModules, gearStubSources, sampleItems } from './helpers/gear-stubs.mjs'

const GLOBAL_KEY = '__GEAR_FRIEND_PAGE__'

const harness = createPageHarness({
  pageFile: 'subPages/gear/friend.vue',
  globalKey: GLOBAL_KEY,
  stubSources: gearStubSources(GLOBAL_KEY),
  realModules: gearRealModules
})

const friendResponse = () => ({
  success: true,
  items: sampleItems().filter((item) => item.status === 1),
  defaults: [{ game_type: 3, slot: 'cue', item_id: 1 }]
})

const mountFriend = async ({ getUserGear, options = { user_id: '7', name: encodeURIComponent('老王') }, user } = {}) => {
  const requests = []
  const context = createGearContext({
    user: user || { userId: 1, authGeneration: 1, isLoggedIn: true },
    api: {
      getUserGear: (params) => { requests.push(params); return (getUserGear || (async () => friendResponse()))(params) }
    }
  })
  const recorder = createUniRecorder()
  const mounted = await harness.mountPage({ context, uni: recorder.uni })
  context.hooks.loaded.forEach((callback) => callback(options))
  await mounted.flushPromises()
  return { ...mounted, context, recorder, requests }
}

const text = (wrapper) => wrapper.text().replace(/\s+/g, ' ')

test('a friend page reads the target user once and shows the owner, main gear and active items only', async () => {
  const page = await mountFriend()
  try {
    assert.deepEqual(page.requests, [{ user_id: 7 }])
    const content = text(page.wrapper)
    assert.match(content, /老王的装备/)
    assert.doesNotMatch(content, /老王 的装备/)
    assert.match(content, /只展示在用的装备/)
    assert.match(content, /主力装备/)
    assert.match(content, /球杆：Alpha A-100/)
    assert.equal(page.wrapper.findAll('.gear-card-stub').length, 2)
    assert.doesNotMatch(content, /已退役/)
    assert.equal(page.wrapper.findAll('.gear-card-stub-badge').length, 1, 'the main badge is shown on the main cue')
  } finally {
    await page.cleanup()
  }
})

test('the main gear block is hidden when the friend has set none', async () => {
  const page = await mountFriend({ getUserGear: async () => ({ ...friendResponse(), defaults: [] }) })
  try {
    assert.doesNotMatch(text(page.wrapper), /主力装备/)
    assert.equal(page.wrapper.findAll('.gear-card-stub').length, 2)
  } finally {
    await page.cleanup()
  }
})

test('a friend with no gear shown gets a friendly empty state', async () => {
  const page = await mountFriend({ getUserGear: async () => ({ success: true, items: [], defaults: [] }) })
  try {
    assert.match(text(page.wrapper), /TA 还没有展示装备/)
  } finally {
    await page.cleanup()
  }
})

test('a non-friend sees the friends-only state and no gear at all', async () => {
  const page = await mountFriend({ getUserGear: async () => ({ success: false, reason: 'not_friend', message: '仅好友可以查看装备', items: [], defaults: [] }) })
  try {
    assert.match(text(page.wrapper), /仅好友可以查看/)
    assert.equal(page.wrapper.findAll('.gear-card-stub').length, 0)
    const back = page.wrapper.findAll('button').find((button) => button.text() === '返回')
    assert.ok(back, 'a non-retryable refusal offers a way back instead of retry')
    await back.trigger('tap')
    assert.equal(page.recorder.callsOf('navigateBack').length, 1)
  } finally {
    await page.cleanup()
  }
})

test('a failed read can be retried and then renders', async () => {
  let attempt = 0
  const page = await mountFriend({
    getUserGear: async () => {
      attempt += 1
      if (attempt === 1) throw Object.assign(new Error('x'), { category: 'network' })
      return friendResponse()
    }
  })
  try {
    assert.match(text(page.wrapper), /网络连接失败/)
    await page.wrapper.findAll('button').find((button) => button.text() === '重试').trigger('tap')
    await page.flushPromises()
    assert.equal(page.requests.length, 2, 'every entry reads again; friend gear is never cached')
    assert.equal(page.wrapper.findAll('.gear-card-stub').length, 2)
  } finally {
    await page.cleanup()
  }
})

test('a missing user id never reads and says the user does not exist', async () => {
  const page = await mountFriend({ options: {} })
  try {
    assert.deepEqual(page.requests, [])
    assert.match(text(page.wrapper), /用户不存在/)
  } finally {
    await page.cleanup()
  }
})

test('logged-out visitors are sent to login', async () => {
  const page = await mountFriend({ user: { userId: 0, authGeneration: 0, isLoggedIn: false } })
  try {
    assert.deepEqual(page.requests, [])
    assert.equal(page.recorder.lastOf('navigateTo').options.url, '/pages/login/login')
  } finally {
    await page.cleanup()
  }
})

test('a response that returns after the account switched is dropped', async () => {
  const gate = defer()
  const page = await mountFriend({ getUserGear: () => gate.promise })
  try {
    page.context.user.userId = 2
    page.context.user.authGeneration = 2
    gate.resolve(friendResponse())
    await page.flushPromises()
    assert.equal(page.wrapper.findAll('.gear-card-stub').length, 0, 'the previous account must not see the data')
  } finally {
    await page.cleanup()
  }
})
