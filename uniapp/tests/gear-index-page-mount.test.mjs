import test from 'node:test'
import assert from 'node:assert/strict'

import { createPageHarness, createUniRecorder, defer } from './helpers/page-harness.mjs'
import { createGearContext, gearRealModules, gearStubSources, sampleItems } from './helpers/gear-stubs.mjs'

const GLOBAL_KEY = '__GEAR_INDEX_PAGE__'

const harness = createPageHarness({
  pageFile: 'subPages/gear/index.vue',
  globalKey: GLOBAL_KEY,
  stubSources: gearStubSources(GLOBAL_KEY),
  realModules: gearRealModules
})

const mine = (items = sampleItems(), defaults = []) => ({ success: true, max_items: 30, items, defaults })

const mountIndex = async ({ getMyGear, setGearDefault, user } = {}) => {
  const requests = []
  const context = createGearContext({
    user: user || { userId: 1, authGeneration: 1, isLoggedIn: true },
    api: {
      getMyGear: (...args) => { requests.push('mine'); return (getMyGear || (async () => mine()))(...args) },
      setGearDefault: (...args) => (setGearDefault || (async () => ({ success: true })))(...args)
    }
  })
  const recorder = createUniRecorder()
  const mounted = await harness.mountPage({ context, uni: recorder.uni })
  const show = async () => {
    context.hooks.shown.forEach((callback) => callback())
    await mounted.flushPromises()
  }
  return { ...mounted, context, recorder, requests, show }
}

const text = (wrapper) => wrapper.text().replace(/\s+/g, ' ')

test('first entry reads my gear once and renders defaults, groups and the count', async () => {
  const page = await mountIndex()
  try {
    await page.show()
    assert.deepEqual(page.requests, ['mine'])
    const content = text(page.wrapper)
    for (const label of ['中式八球', '九球追分', '斯诺克', '美式九球']) assert.match(content, new RegExp(label))
    assert.match(content, /3\/30/)
    assert.match(content, /球杆/)
    assert.match(content, /冲杆 \/ 跳杆/)
    assert.match(content, /已退役/)
    // 三件装备各自一张卡片
    assert.equal(page.wrapper.findAll('.gear-card-stub').length, 3)
    // 只有球杆槽位有「未设置」，且斯诺克没有冲杆槽位：4 个球种 4 个球杆槽 + 3 个冲杆槽
    assert.equal(page.wrapper.findAll('.slot').length, 7)
  } finally {
    await page.cleanup()
  }
})

test('repeated onShow inside the cache window does not read again', async () => {
  const page = await mountIndex()
  try {
    await page.show()
    await page.show()
    await page.show()
    assert.equal(page.requests.length, 1)
    assert.match(text(page.wrapper), /3\/30/)
  } finally {
    await page.cleanup()
  }
})

test('an empty account sees the empty state with an add button', async () => {
  const page = await mountIndex({ getMyGear: async () => mine([], []) })
  try {
    await page.show()
    assert.match(text(page.wrapper), /还没有装备/)
    assert.equal(page.wrapper.findAll('.gear-card-stub').length, 0)
    const addButtons = page.wrapper.findAll('button').filter((button) => button.text().includes('添加球杆'))
    assert.equal(addButtons.length, 1)
    await addButtons[0].trigger('tap')
    assert.equal(page.recorder.lastOf('navigateTo').options.url, '/subPages/gear/add?category=cue')
  } finally {
    await page.cleanup()
  }
})

test('a failed first read shows a retry that reads again', async () => {
  let attempt = 0
  const page = await mountIndex({
    getMyGear: async () => {
      attempt += 1
      if (attempt === 1) throw Object.assign(new Error('boom'), { category: 'network' })
      return mine()
    }
  })
  try {
    await page.show()
    assert.match(text(page.wrapper), /网络连接失败/)
    assert.equal(page.wrapper.findAll('.gear-card-stub').length, 0)
    const retry = page.wrapper.findAll('button').find((button) => button.text().includes('重试'))
    await retry.trigger('tap')
    await page.flushPromises()
    assert.equal(page.requests.length, 2)
    assert.equal(page.wrapper.findAll('.gear-card-stub').length, 3)
  } finally {
    await page.cleanup()
  }
})

test('logged-out visitors are sent to login without reading anything', async () => {
  const page = await mountIndex({ user: { userId: 0, authGeneration: 0, isLoggedIn: false } })
  try {
    await page.show()
    assert.deepEqual(page.requests, [])
    assert.equal(page.recorder.lastOf('navigateTo').options.url, '/pages/login/login')
  } finally {
    await page.cleanup()
  }
})

test('the main-gear sheet lists only eligible items and saves the choice, then reads again', async () => {
  const saved = []
  let current = mine()
  const page = await mountIndex({
    getMyGear: async () => current,
    setGearDefault: async (payload) => {
      saved.push(payload)
      current = mine(sampleItems(), [{ game_type: 3, slot: 'cue', item_id: 1 }])
      return { success: true }
    }
  })
  try {
    await page.show()
    // 第一行是中式八球，第一个槽位是球杆
    await page.wrapper.findAll('.slot')[0].trigger('tap')
    const rows = page.wrapper.findAll('.picker-row')
    assert.equal(rows.length, 1, '只有在用的球杆能做主力球杆，退役的和冲杆都不出现')
    assert.match(rows[0].text(), /Alpha A-100/)
    assert.match(text(page.wrapper), /中式八球 · 主力球杆/)

    await rows[0].trigger('tap')
    await page.flushPromises()
    assert.deepEqual(saved, [{ game_type: 3, slot: 'cue', item_id: 1 }])
    assert.equal(page.requests.length, 2, 'a successful write must read the list again')
    assert.equal(page.wrapper.findAll('.picker-row').length, 0, 'the sheet closes after saving')
    assert.match(text(page.wrapper), /Alpha A-100/)
  } finally {
    await page.cleanup()
  }
})

test('the break slot offers only active break cues and snooker has no break slot', async () => {
  const page = await mountIndex()
  try {
    await page.show()
    // 槽位顺序：八球球杆、八球冲杆、九球球杆、九球冲杆、斯诺克球杆、美式球杆、美式冲杆
    await page.wrapper.findAll('.slot')[1].trigger('tap')
    const rows = page.wrapper.findAll('.picker-row')
    assert.equal(rows.length, 1)
    assert.match(rows[0].text(), /A-Break/)
    assert.match(text(page.wrapper), /中式八球 · 主力冲杆/)
  } finally {
    await page.cleanup()
  }
})

test('a rejected save keeps the sheet open and tells the user why', async () => {
  const page = await mountIndex({ setGearDefault: async () => ({ success: false, message: '已退役的装备不能设为主力' }) })
  try {
    await page.show()
    await page.wrapper.findAll('.slot')[0].trigger('tap')
    await page.wrapper.findAll('.picker-row')[0].trigger('tap')
    await page.flushPromises()
    assert.equal(page.wrapper.findAll('.picker-row').length, 1, 'the sheet stays open so the user can pick again')
    assert.equal(page.recorder.lastOf('showToast').options.title, '已退役的装备不能设为主力')
    assert.equal(page.requests.length, 1, 'a rejected write must not trigger a re-read')
  } finally {
    await page.cleanup()
  }
})

test('a late response from the previous account never reaches the new account page', async () => {
  const first = defer()
  const responses = [first, null]
  let call = 0
  const page = await mountIndex({
    getMyGear: () => {
      const gate = responses[call]
      call += 1
      return gate ? gate.promise : Promise.resolve(mine([{ ...sampleItems()[0], id: 99, model_name: '新账号的杆' }], []))
    }
  })
  try {
    await page.show()
    // 旧账号的读取还没返回，账号已经切换
    page.context.user.userId = 2
    page.context.user.authGeneration = 2
    await page.show()
    assert.match(text(page.wrapper), /新账号的杆/)
    first.resolve(mine([{ ...sampleItems()[0], id: 1, model_name: '旧账号的杆' }], []))
    await page.flushPromises()
    assert.doesNotMatch(text(page.wrapper), /旧账号的杆/)
    assert.match(text(page.wrapper), /新账号的杆/)
  } finally {
    await page.cleanup()
  }
})
