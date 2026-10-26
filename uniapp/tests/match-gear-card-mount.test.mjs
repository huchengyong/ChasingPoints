import test from 'node:test'
import assert from 'node:assert/strict'

import { createPageHarness, createUniRecorder, defer } from './helpers/page-harness.mjs'
import { createGearContext, gearRealModules, gearStubSources, sampleItems } from './helpers/gear-stubs.mjs'

const GLOBAL_KEY = '__MATCH_GEAR_CARD__'

const harness = createPageHarness({
  pageFile: 'components/matchGearCard.vue',
  globalKey: GLOBAL_KEY,
  stubSources: gearStubSources(GLOBAL_KEY),
  realModules: gearRealModules
})

const cue = (id, model, extra = {}) => ({ id, category: 'cue', status: 1, brand_name: 'Alpha', model_name: model, cue_style: 'pool', roles: [], ...extra })

const matchResponse = (overrides = {}) => ({
  success: true,
  match_id: 500,
  game_type: 3,
  supports_break: true,
  has_opponent: true,
  me: {
    user_id: 1, recorded: true, source: 'auto', cue: cue(1, 'A-100'),
    tip_product: { id: 21, category: 'tip', brand_name: 'Alpha', name: 'A-Tip' }, chalk_product: null, break_cue: null
  },
  opponent: { user_id: 2, recorded: false },
  editable: true,
  edit_deadline: new Date(Date.now() + 5 * 24 * 3600 * 1000).toISOString(),
  ...overrides
})

const mountCard = async ({ props = { matchId: 500, enabled: true }, getGearMatch, updateGearMatch, user } = {}) => {
  const requests = []
  const updates = []
  const context = createGearContext({
    user: user || { userId: 1, authGeneration: 1, isLoggedIn: true },
    api: {
      getGearMatch: (params, options) => { requests.push({ params, options }); return (getGearMatch || (async () => matchResponse()))(params, options) },
      getMyGear: async () => ({ success: true, max_items: 30, items: sampleItems(), defaults: [] }),
      updateGearMatch: (payload) => { updates.push(payload); return (updateGearMatch || (async () => ({ success: true })))(payload) }
    }
  })
  const listeners = new Map()
  const removed = []
  const recorder = createUniRecorder({
    $once: (name, handler) => { listeners.set(name, handler) },
    $off: (name) => { removed.push(name) }
  })
  const mounted = await harness.mountPage({ context, uni: recorder.uni, props })
  await mounted.flushPromises()
  return { ...mounted, context, recorder, requests, updates, listeners, removed }
}

const text = (wrapper) => wrapper.text().replace(/\s+/g, ' ')
const findText = (wrapper, label) => wrapper.findAll('text').find((node) => node.text() === label)

test('a disabled card (in progress, referee or spectator) renders nothing and never calls the server', async () => {
  const page = await mountCard({ props: { matchId: 500, enabled: false } })
  try {
    assert.deepEqual(page.requests, [])
    assert.equal(page.wrapper.find('.match-gear').exists(), false)
  } finally {
    await page.cleanup()
  }
})

test('an enabled card reads once with a quiet request and shows both sides', async () => {
  const page = await mountCard()
  try {
    assert.equal(page.requests.length, 1)
    assert.deepEqual(page.requests[0].params, { match_id: 500 })
    assert.equal(page.requests[0].options.silent, true, 'a failed read must not pop a global toast')
    const content = text(page.wrapper)
    assert.match(content, /本场用杆/)
    assert.match(content, /Alpha A-100/)
    assert.match(content, /皮头.*Alpha A-Tip/)
    assert.equal(page.wrapper.findAll('.match-gear-side').length, 2)
    assert.match(content, /对手.*未记录/)
    assert.match(content, /月\d+日前可修改/)
    assert.ok(findText(page.wrapper, '修改'), 'the player can edit inside the window')
  } finally {
    await page.cleanup()
  }
})

test('my unrecorded side offers a back-fill link', async () => {
  const page = await mountCard({ getGearMatch: async () => matchResponse({ me: { user_id: 1, recorded: false } }) })
  try {
    assert.match(text(page.wrapper), /我.*未记录/)
    assert.ok(findText(page.wrapper, '补记'))
  } finally {
    await page.cleanup()
  }
})

test('after the edit window the card is read-only', async () => {
  const page = await mountCard({ getGearMatch: async () => matchResponse({ editable: false }) })
  try {
    assert.equal(findText(page.wrapper, '修改'), undefined)
    assert.equal(findText(page.wrapper, '补记'), undefined)
    assert.doesNotMatch(text(page.wrapper), /前可修改/)
  } finally {
    await page.cleanup()
  }
})

test('a match without a registered opponent shows only my side', async () => {
  const page = await mountCard({ getGearMatch: async () => matchResponse({ has_opponent: false }) })
  try {
    assert.equal(page.wrapper.findAll('.match-gear-side').length, 1)
  } finally {
    await page.cleanup()
  }
})

test('refusals that mean "not yours" hide the card instead of showing an error', async () => {
  for (const reason of ['not_participant', 'not_completed', 'not_found']) {
    const page = await mountCard({ getGearMatch: async () => ({ success: false, reason, message: '不是选手' }) })
    try {
      assert.equal(page.wrapper.find('.match-gear').exists(), false, reason)
    } finally {
      await page.cleanup()
    }
  }
})

test('a failed read degrades to a retry inside the card only and recovers', async () => {
  let attempt = 0
  const page = await mountCard({
    getGearMatch: async () => {
      attempt += 1
      if (attempt === 1) throw new Error('网络错误')
      return matchResponse()
    }
  })
  try {
    assert.match(text(page.wrapper), /本场用杆加载失败，请检查网络后重试/)
    await findText(page.wrapper, '重试').trigger('tap')
    await page.flushPromises()
    assert.equal(page.requests.length, 2)
    assert.match(text(page.wrapper), /Alpha A-100/)
  } finally {
    await page.cleanup()
  }
})

test('a server error message is shown with a retry and no crash', async () => {
  const page = await mountCard({ getGearMatch: async () => ({ success: false, message: '读取本场用杆失败' }) })
  try {
    assert.match(text(page.wrapper), /读取本场用杆失败/)
    assert.ok(findText(page.wrapper, '重试'))
  } finally {
    await page.cleanup()
  }
})

test('the card starts reading when the match becomes completed, and stops when it is disabled again', async () => {
  const page = await mountCard({ props: { matchId: 500, enabled: false } })
  try {
    await page.wrapper.setProps({ enabled: true })
    await page.flushPromises()
    assert.equal(page.requests.length, 1)
    assert.match(text(page.wrapper), /Alpha A-100/)
    await page.wrapper.setProps({ enabled: false })
    await page.flushPromises()
    assert.equal(page.wrapper.find('.match-gear').exists(), false)
  } finally {
    await page.cleanup()
  }
})

test('a late response for a previous match id never overwrites the current one', async () => {
  const slow = defer()
  const page = await mountCard({
    props: { matchId: 500, enabled: true },
    getGearMatch: ({ match_id }) => (match_id === 500 ? slow.promise : Promise.resolve(matchResponse({ match_id: 501, me: { user_id: 1, recorded: true, source: 'auto', cue: cue(9, '新对局的杆') } })))
  })
  try {
    await page.wrapper.setProps({ matchId: 501 })
    await page.flushPromises()
    assert.match(text(page.wrapper), /新对局的杆/)
    slow.resolve(matchResponse({ me: { user_id: 1, recorded: true, source: 'auto', cue: cue(1, '旧对局的杆') } }))
    await page.flushPromises()
    assert.doesNotMatch(text(page.wrapper), /旧对局的杆/)
    assert.match(text(page.wrapper), /新对局的杆/)
  } finally {
    await page.cleanup()
  }
})

test('a response that returns after the account switched is dropped', async () => {
  const gate = defer()
  const page = await mountCard({ getGearMatch: () => gate.promise })
  try {
    page.context.user.userId = 2
    page.context.user.authGeneration = 2
    gate.resolve(matchResponse())
    await page.flushPromises()
    assert.doesNotMatch(text(page.wrapper), /Alpha A-100/)
  } finally {
    await page.cleanup()
  }
})

test('the editor loads my gear, prefills tip and chalk when the cue changes, and saves the whole combination', async () => {
  const saved = matchResponse({
    me: { user_id: 1, recorded: true, source: 'manual', cue: cue(3, '我的型号', { status: 2 }), tip_product: null, chalk_product: null, break_cue: null }
  })
  const page = await mountCard({ updateGearMatch: async () => saved })
  try {
    await findText(page.wrapper, '修改').trigger('tap')
    await page.flushPromises()
    // 球杆选项：未删除的球杆（含已退役），外加「不记录」
    const options = page.wrapper.findAll('.match-gear-sheet .option')
    assert.ok(options.length >= 3)
    assert.match(text(page.wrapper), /我的品牌 我的型号/)
    assert.match(text(page.wrapper), /已退役/)

    // 换成退役的那根杆：它没有皮头与巧克，草稿里的皮头被清空
    await options.find((option) => option.text().includes('我的型号')).trigger('tap')
    await page.wrapper.find('.sheet-save').trigger('tap')
    await page.flushPromises()
    assert.deepEqual(page.updates, [{ match_id: 500, cue_item_id: 3, tip_product_id: 0, chalk_product_id: 0, break_item_id: 0 }])
    assert.equal(page.wrapper.find('.match-gear-sheet').exists(), false, 'the editor closes after saving')
    assert.match(text(page.wrapper), /Alpha 我的型号/, 'the card shows the view returned by the save')
    assert.equal(page.requests.length, 1, 'the response already carries the refreshed view')
    assert.equal(page.recorder.lastOf('showToast').options.title, '已保存')
    assert.equal(page.wrapper.emitted('updated').length, 1)
  } finally {
    await page.cleanup()
  }
})

test('choosing a cue that has a tip prefills it, and a rejected save keeps the editor open with the reason', async () => {
  const page = await mountCard({
    getGearMatch: async () => matchResponse({ me: { user_id: 1, recorded: false } }),
    updateGearMatch: async () => ({ success: false, message: '对局完成已超过 7 天，本场用杆不能再修改' })
  })
  try {
    await findText(page.wrapper, '补记').trigger('tap')
    await page.flushPromises()
    await page.wrapper.findAll('.match-gear-sheet .option').find((option) => option.text().includes('A-100')).trigger('tap')
    assert.match(page.wrapper.find('.match-gear-sheet').text(), /Alpha A-Tip/, 'the cue current tip is prefilled')

    await page.wrapper.find('.sheet-save').trigger('tap')
    await page.flushPromises()
    assert.deepEqual(page.updates, [{ match_id: 500, cue_item_id: 1, tip_product_id: 21, chalk_product_id: 0, break_item_id: 0 }])
    assert.equal(page.wrapper.find('.match-gear-sheet').exists(), true)
    assert.equal(page.recorder.lastOf('showToast').options.title, '对局完成已超过 7 天，本场用杆不能再修改')
    assert.equal(page.wrapper.emitted('updated'), undefined)
  } finally {
    await page.cleanup()
  }
})

test('the tip is chosen through the picker page and sent with the save', async () => {
  const page = await mountCard()
  try {
    await findText(page.wrapper, '修改').trigger('tap')
    await page.flushPromises()
    await page.wrapper.findAll('.match-gear-sheet .match-gear-link').find((node) => node.text() === '更换').trigger('tap')
    const url = page.recorder.lastOf('navigateTo').options.url
    const params = new URLSearchParams(url.split('?')[1])
    assert.equal(params.get('mode'), 'pick')
    assert.equal(params.get('category'), 'tip')
    page.listeners.get(`gear:pick:${params.get('token')}`)({ product: { id: 77, category: 'tip', brand_name: 'Beta', name: 'B-Tip' } })
    await page.flushPromises()
    assert.match(page.wrapper.find('.match-gear-sheet').text(), /Beta B-Tip/)
    await page.wrapper.find('.sheet-save').trigger('tap')
    await page.flushPromises()
    assert.equal(page.updates[0].tip_product_id, 77)
    assert.equal(page.updates[0].cue_item_id, 1)
  } finally {
    await page.cleanup()
  }
})

test('the break cue section appears only for games that record one', async () => {
  const withBreak = await mountCard()
  try {
    await findText(withBreak.wrapper, '修改').trigger('tap')
    await withBreak.flushPromises()
    assert.match(withBreak.wrapper.find('.match-gear-sheet').text(), /冲杆/)
    assert.match(withBreak.wrapper.find('.match-gear-sheet').text(), /Alpha A-Break/)
  } finally {
    await withBreak.cleanup()
  }

  const snooker = await mountCard({ getGearMatch: async () => matchResponse({ game_type: 1, supports_break: false }) })
  try {
    await findText(snooker.wrapper, '修改').trigger('tap')
    await snooker.flushPromises()
    assert.doesNotMatch(snooker.wrapper.find('.match-gear-sheet').text(), /冲杆/)
    await snooker.wrapper.find('.sheet-save').trigger('tap')
    await snooker.flushPromises()
    assert.equal(snooker.updates[0].break_item_id, 0, 'snooker never sends a break cue')
  } finally {
    await snooker.cleanup()
  }
})

test('choosing "not recorded" for every field sends an all-zero request that clears the record', async () => {
  const page = await mountCard()
  try {
    await findText(page.wrapper, '修改').trigger('tap')
    await page.flushPromises()
    await page.wrapper.findAll('.match-gear-sheet .option').find((option) => option.text() === '不记录').trigger('tap')
    await page.wrapper.find('.sheet-save').trigger('tap')
    await page.flushPromises()
    assert.deepEqual(page.updates, [{ match_id: 500, cue_item_id: 0, tip_product_id: 0, chalk_product_id: 0, break_item_id: 0 }])
  } finally {
    await page.cleanup()
  }
})

test('unmounting removes a picker listener that never fired', async () => {
  const page = await mountCard()
  await findText(page.wrapper, '修改').trigger('tap')
  await page.flushPromises()
  await page.wrapper.findAll('.match-gear-sheet .match-gear-link').find((node) => node.text() === '更换').trigger('tap')
  const token = new URLSearchParams(page.recorder.lastOf('navigateTo').options.url.split('?')[1]).get('token')
  await page.cleanup()
  assert.ok(page.removed.includes(`gear:pick:${token}`))
})
