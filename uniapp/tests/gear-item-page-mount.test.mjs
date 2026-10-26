import test from 'node:test'
import assert from 'node:assert/strict'

import { createPageHarness, createUniRecorder } from './helpers/page-harness.mjs'
import { createGearContext, gearRealModules, gearStubSources, sampleItems } from './helpers/gear-stubs.mjs'

const GLOBAL_KEY = '__GEAR_ITEM_PAGE__'

const harness = createPageHarness({
  pageFile: 'subPages/gear/item.vue',
  globalKey: GLOBAL_KEY,
  stubSources: gearStubSources(GLOBAL_KEY),
  realModules: gearRealModules
})

const mine = (defaults = []) => ({ success: true, max_items: 30, items: sampleItems(), defaults })

const mountItem = async ({ options = { id: '1' }, defaults = [], api = {}, confirm = true } = {}) => {
  const writes = { updates: [], deletes: [], defaults: [] }
  let reads = 0
  const context = createGearContext({
    api: {
      getMyGear: async () => { reads += 1; return mine(defaults) },
      updateGearItem: async (payload) => { writes.updates.push(payload); return (api.updateGearItem || (async () => ({ success: true })))(payload) },
      deleteGearItem: async (payload) => { writes.deletes.push(payload); return (api.deleteGearItem || (async () => ({ success: true })))(payload) },
      setGearDefault: async (payload) => { writes.defaults.push(payload); return (api.setGearDefault || (async () => ({ success: true })))(payload) }
    }
  })
  const listeners = new Map()
  const removed = []
  const modals = []
  const recorder = createUniRecorder({
    $once: (name, handler) => { listeners.set(name, handler) },
    $off: (name) => { removed.push(name) },
    showModal: (modal) => { modals.push(modal); modal.success?.({ confirm }) }
  })
  const mounted = await harness.mountPage({ context, uni: recorder.uni })
  context.hooks.loaded.forEach((callback) => callback(options))
  context.hooks.shown.forEach((callback) => callback())
  await mounted.flushPromises()
  return { ...mounted, context, recorder, writes, listeners, removed, modals, reads: () => reads }
}

const text = (wrapper) => wrapper.text().replace(/\s+/g, ' ')
const findByText = (wrapper, selector, label) => wrapper.findAll(selector).find((node) => node.text() === label)

test('a catalog cue shows its official specs, tip, empty chalk and the added hint', async () => {
  const page = await mountItem({ options: { id: '1', added: '1' } })
  try {
    const content = text(page.wrapper)
    assert.match(content, /Alpha A-100/)
    assert.match(content, /官方规格/)
    assert.match(content, /1470 mm/)
    assert.match(content, /540 g（19\.0 oz）/)
    assert.match(content, /12\.9 mm/)
    assert.match(content, /皮头.*Alpha A-Tip/)
    assert.match(content, /巧克.*未设置/)
    assert.match(content, /已添加。给这根杆设置/)
    assert.equal(page.reads(), 1)
  } finally {
    await page.cleanup()
  }
})

test('a cue offers all four game types as main gear and toggling sends the right request then reads again', async () => {
  const page = await mountItem()
  try {
    const chips = page.wrapper.findAll('.chip')
    assert.deepEqual(chips.map((chip) => chip.text()), ['中式八球', '九球追分', '斯诺克', '美式九球'])
    await chips[0].trigger('tap')
    await page.flushPromises()
    assert.deepEqual(page.writes.defaults, [{ game_type: 3, slot: 'cue', item_id: 1 }])
    assert.equal(page.reads(), 2)
  } finally {
    await page.cleanup()
  }
})

test('an active main chip clears the main gear when tapped again', async () => {
  const page = await mountItem({ defaults: [{ game_type: 3, slot: 'cue', item_id: 1 }] })
  try {
    const first = page.wrapper.findAll('.chip')[0]
    assert.ok(first.classes().includes('active'))
    await first.trigger('tap')
    await page.flushPromises()
    assert.deepEqual(page.writes.defaults, [{ game_type: 3, slot: 'cue', item_id: 0 }])
  } finally {
    await page.cleanup()
  }
})

test('a break cue has no snooker chip', async () => {
  const page = await mountItem({ options: { id: '2' } })
  try {
    assert.deepEqual(page.wrapper.findAll('.chip').map((chip) => chip.text()), ['中式八球', '九球追分', '美式九球'])
    assert.equal(page.wrapper.findAll('.part-row').length, 0, '冲杆没有皮头与巧克')
    assert.doesNotMatch(text(page.wrapper), /官方规格/, '没有规格的型号不展示规格卡片')
  } finally {
    await page.cleanup()
  }
})

test('a retired item explains why it cannot be main gear and can be restored', async () => {
  const page = await mountItem({ options: { id: '3' } })
  try {
    assert.equal(page.wrapper.findAll('.chip').length, 0)
    assert.match(text(page.wrapper), /退役的装备不能设为主力/)
    assert.match(text(page.wrapper), /未收录型号/)
    assert.match(text(page.wrapper), /你填写的名称只有你自己能看到/)
    await findByText(page.wrapper, 'button', '恢复使用').trigger('tap')
    await page.flushPromises()
    assert.deepEqual(page.writes.updates, [{ item_id: 3, status: 1 }])
  } finally {
    await page.cleanup()
  }
})

test('retiring asks for confirmation first', async () => {
  const declined = await mountItem({ confirm: false })
  try {
    await findByText(declined.wrapper, 'button', '退役').trigger('tap')
    await declined.flushPromises()
    assert.deepEqual(declined.writes.updates, [])
  } finally {
    await declined.cleanup()
  }

  const accepted = await mountItem()
  try {
    await findByText(accepted.wrapper, 'button', '退役').trigger('tap')
    await accepted.flushPromises()
    assert.deepEqual(accepted.writes.updates, [{ item_id: 1, status: 2 }])
    assert.match(accepted.modals[0].content, /取消它的主力设置/)
  } finally {
    await accepted.cleanup()
  }
})

test('deleting confirms, deletes and goes back without reading the list again', async () => {
  const page = await mountItem()
  try {
    await findByText(page.wrapper, 'button', '删除装备').trigger('tap')
    await page.flushPromises()
    assert.deepEqual(page.writes.deletes, [{ item_id: 1 }])
    assert.equal(page.recorder.callsOf('navigateBack').length, 1)
    assert.equal(page.reads(), 1, 'no flash of the "item not found" state after deleting')
    assert.doesNotMatch(text(page.wrapper), /装备不存在/)
  } finally {
    await page.cleanup()
  }
})

test('a rejected write shows the server message and does not read again', async () => {
  const page = await mountItem({ api: { updateGearItem: async () => ({ success: false, message: '装备不存在' }) } })
  try {
    await findByText(page.wrapper, 'text', '清除').trigger('tap')
    await page.flushPromises()
    assert.equal(page.recorder.lastOf('showToast').options.title, '装备不存在')
    assert.equal(page.reads(), 1)
  } finally {
    await page.cleanup()
  }
})

test('picking a tip opens the picker with a token and applies the result handed back through that token', async () => {
  const page = await mountItem()
  try {
    await findByText(page.wrapper, 'text', '更换').trigger('tap')
    const url = page.recorder.lastOf('navigateTo').options.url
    const params = new URLSearchParams(url.split('?')[1])
    assert.equal(url.split('?')[0], '/subPages/gear/add')
    assert.equal(params.get('mode'), 'pick')
    assert.equal(params.get('category'), 'tip')
    const token = params.get('token')
    assert.ok(token)

    page.listeners.get(`gear:pick:${token}`)({ product: { id: 22, category: 'tip', name: 'B-Tip' } })
    await page.flushPromises()
    assert.deepEqual(page.writes.updates, [{ item_id: 1, update_tip: true, tip_product_id: 22 }])
    assert.equal(page.reads(), 2)
  } finally {
    await page.cleanup()
  }
})

test('choosing nothing in the picker clears the part, and clearing from the row does the same', async () => {
  const page = await mountItem()
  try {
    await findByText(page.wrapper, 'text', '选择').trigger('tap') // 巧克一行目前没有设置
    const token = new URLSearchParams(page.recorder.lastOf('navigateTo').options.url.split('?')[1]).get('token')
    page.listeners.get(`gear:pick:${token}`)({ product: null })
    await page.flushPromises()
    assert.deepEqual(page.writes.updates.at(-1), { item_id: 1, update_chalk: true, chalk_product_id: 0 })

    await findByText(page.wrapper, 'text', '清除').trigger('tap') // 皮头一行有「清除」
    await page.flushPromises()
    assert.deepEqual(page.writes.updates.at(-1), { item_id: 1, update_tip: true, tip_product_id: 0 })
  } finally {
    await page.cleanup()
  }
})

test('leaving the page removes a picker listener that never fired', async () => {
  const page = await mountItem()
  await findByText(page.wrapper, 'text', '更换').trigger('tap')
  const token = new URLSearchParams(page.recorder.lastOf('navigateTo').options.url.split('?')[1]).get('token')
  await page.cleanup()
  assert.ok(page.removed.includes(`gear:pick:${token}`))
})

test('an unknown item id shows the not-found state with a way back', async () => {
  const page = await mountItem({ options: { id: '404' } })
  try {
    assert.match(text(page.wrapper), /装备不存在/)
    await findByText(page.wrapper, 'button', '返回').trigger('tap')
    assert.equal(page.recorder.callsOf('navigateBack').length, 1)
  } finally {
    await page.cleanup()
  }
})
