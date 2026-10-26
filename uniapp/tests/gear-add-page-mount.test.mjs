import test from 'node:test'
import assert from 'node:assert/strict'

import { createPageHarness, createUniRecorder, defer } from './helpers/page-harness.mjs'
import { createGearContext, gearRealModules, gearStubSources } from './helpers/gear-stubs.mjs'

const GLOBAL_KEY = '__GEAR_ADD_PAGE__'

const harness = createPageHarness({
  pageFile: 'subPages/gear/add.vue',
  globalKey: GLOBAL_KEY,
  stubSources: gearStubSources(GLOBAL_KEY),
  realModules: gearRealModules
})

const product = (id, name, extra = {}) => ({
  id, category: 'cue', brand_id: 10, brand_name: 'Alpha', name, house: false, cue_style: 'pool', roles: [], ...extra
})

const defaultApi = () => ({
  getGearBrands: async () => ({ success: true, list: [{ id: 10, name: 'Alpha' }] }),
  getGearProducts: async (params) => {
    if (params.house) return { success: true, total: 1, list: [product(2, '球房公杆', { house: true, brand_id: 0, brand_name: '', cue_style: '' })] }
    return { success: true, total: 3, list: [product(2, '球房公杆', { house: true, brand_id: 0, brand_name: '', cue_style: '' }), product(11, 'A-100', { weight_g: 540 }), product(12, 'A-200')] }
  },
  createGearItem: async () => ({ success: true, item: { id: 7 } })
})

const mountAdd = async ({ api = {}, options = { category: 'cue' }, uniOverrides = {} } = {}) => {
  const calls = { brands: [], products: [], created: [] }
  const merged = { ...defaultApi(), ...api }
  const context = createGearContext({
    api: {
      getGearBrands: (params) => { calls.brands.push(params); return merged.getGearBrands(params) },
      getGearProducts: (params) => { calls.products.push(params); return merged.getGearProducts(params) },
      createGearItem: (payload) => { calls.created.push(payload); return merged.createGearItem(payload) }
    }
  })
  const emitted = []
  const modals = []
  const recorder = createUniRecorder({
    $emit: (name, payload) => { emitted.push({ name, payload }) },
    showModal: (modal) => { modals.push(modal); modal.success?.({ confirm: true }) },
    ...uniOverrides
  })
  const mounted = await harness.mountPage({ context, uni: recorder.uni })
  context.hooks.loaded.forEach((callback) => callback(options))
  await mounted.flushPromises()
  return { ...mounted, context, recorder, calls, emitted, modals }
}

const text = (wrapper) => wrapper.text().replace(/\s+/g, ' ')
const inputEvent = (value) => ({ detail: { value } })

test('create mode reads brands, the house cue and the first page of products, and hides the house row inside the list', async () => {
  const page = await mountAdd()
  try {
    assert.deepEqual(page.calls.brands, [{ category: 'cue' }])
    const houseRequest = page.calls.products.find((params) => params.house)
    assert.deepEqual(houseRequest, { category: 'cue', house: true, page: 1, page_size: 1 })
    const listRequest = page.calls.products.find((params) => !params.house)
    // 严格相等：不带 brand_id / keyword 这些值为 undefined 的键，小程序会把它们拼成 "undefined" 字符串导致后端 400。
    assert.deepEqual(listRequest, { category: 'cue', page: 1, page_size: 20 })
    assert.equal(page.wrapper.findAll('.house-row').length, 1)
    const rows = page.wrapper.findAll('.product-row')
    assert.equal(rows.length, 2, 'the house cue has its own quick row and is not repeated in the list')
    assert.match(text(page.wrapper), /Alpha A-100/)
    assert.match(text(page.wrapper), /美式杆 · 540 g/)
    assert.match(text(page.wrapper), /找不到我的型号/)
  } finally {
    await page.cleanup()
  }
})

test('choosing a brand and searching send those filters and nothing else', async () => {
  const page = await mountAdd()
  try {
    await page.wrapper.findAll('.brand-chip')[1].trigger('tap')
    await page.flushPromises()
    const lists = () => page.calls.products.filter((params) => !params.house)
    assert.deepEqual(lists().at(-1), { category: 'cue', brand_id: 10, page: 1, page_size: 20 })
    await page.wrapper.find('.search-input').trigger('input', inputEvent(' A-100 '))
    await page.wrapper.find('.search-input').trigger('confirm')
    await page.flushPromises()
    assert.deepEqual(lists().at(-1), { category: 'cue', brand_id: 10, keyword: 'A-100', page: 1, page_size: 20 })
    await page.wrapper.findAll('.brand-chip')[0].trigger('tap')
    await page.flushPromises()
    assert.deepEqual(lists().at(-1), { category: 'cue', keyword: 'A-100', page: 1, page_size: 20 })
  } finally {
    await page.cleanup()
  }
})

test('tapping a catalog product asks for confirmation, creates it and opens the new item', async () => {
  const page = await mountAdd()
  try {
    await page.wrapper.findAll('.product-row')[0].trigger('tap')
    await page.flushPromises()
    assert.match(page.modals[0].content, /Alpha A-100/)
    assert.deepEqual(page.calls.created, [{ category: 'cue', product_id: 11 }])
    assert.equal(page.recorder.lastOf('redirectTo').options.url, '/subPages/gear/item?id=7&added=1')
  } finally {
    await page.cleanup()
  }
})

test('the house cue quick row creates a house cue the same way', async () => {
  const page = await mountAdd()
  try {
    await page.wrapper.find('.house-row').trigger('tap')
    await page.flushPromises()
    assert.deepEqual(page.calls.created, [{ category: 'cue', product_id: 2 }])
  } finally {
    await page.cleanup()
  }
})

test('a rejected creation keeps the user on the page and shows the reason', async () => {
  const page = await mountAdd({ api: { createGearItem: async () => ({ success: false, message: '装备最多 30 件，请先删除不用的装备' }) } })
  try {
    await page.wrapper.findAll('.product-row')[0].trigger('tap')
    await page.flushPromises()
    assert.equal(page.recorder.callsOf('redirectTo').length, 0)
    assert.equal(page.recorder.lastOf('showToast').options.title, '装备最多 30 件，请先删除不用的装备')
  } finally {
    await page.cleanup()
  }
})

test('declining the confirmation creates nothing', async () => {
  const page = await mountAdd({ uniOverrides: { showModal: (modal) => { modal.success?.({ confirm: false }) } } })
  try {
    await page.wrapper.findAll('.product-row')[0].trigger('tap')
    await page.flushPromises()
    assert.deepEqual(page.calls.created, [])
  } finally {
    await page.cleanup()
  }
})

test('pick mode for tips never offers unlisted registration and hands the chosen product back by token', async () => {
  const page = await mountAdd({
    options: { mode: 'pick', category: 'tip', token: 'tok-1' },
    api: {
      getGearProducts: async () => ({ success: true, total: 1, list: [product(21, 'A-Tip', { category: 'tip', cue_style: '' })] })
    }
  })
  try {
    assert.equal(page.calls.products.some((params) => params.house), false, 'tips have no house entry')
    assert.deepEqual(page.calls.products[0].category, 'tip')
    assert.doesNotMatch(text(page.wrapper), /找不到我的型号/)
    assert.match(text(page.wrapper), /选择皮头/)

    await page.wrapper.findAll('.product-row')[0].trigger('tap')
    assert.equal(page.modals.length, 0, 'picking never asks for creation confirmation')
    const picked = page.emitted.find((entry) => entry.name === 'gear:pick:tok-1')
    assert.equal(picked.payload.product.id, 21)
    assert.equal(page.recorder.callsOf('navigateBack').length, 1)
    assert.deepEqual(page.calls.created, [])
  } finally {
    await page.cleanup()
  }
})

test('pick mode can also clear the choice', async () => {
  const page = await mountAdd({ options: { mode: 'pick', category: 'chalk', token: 'tok-2' } })
  try {
    const clear = page.wrapper.findAll('text').find((node) => node.text() === '不使用')
    await clear.trigger('tap')
    const picked = page.emitted.find((entry) => entry.name === 'gear:pick:tok-2')
    assert.equal(picked.payload.product, null)
    assert.equal(page.recorder.callsOf('navigateBack').length, 1)
    // 巧克有球房巧克这一行
    assert.ok(page.calls.products.some((params) => params.house && params.category === 'chalk'))
  } finally {
    await page.cleanup()
  }
})

test('the unlisted form validates before sending and builds the right request', async () => {
  const page = await mountAdd()
  try {
    await page.wrapper.find('.unlisted-entry').trigger('tap')
    assert.equal(page.wrapper.findAll('.unlisted-form').length, 1)

    const submit = page.wrapper.find('.submit-btn')
    await submit.trigger('tap')
    assert.equal(page.recorder.lastOf('showToast').options.title, '请填写型号名称')
    assert.deepEqual(page.calls.created, [])

    const inputs = page.wrapper.findAll('.form-input')
    // 默认没选目录品牌，先出现自填品牌输入框，再是型号输入框
    await inputs[0].trigger('input', inputEvent('自家小厂'))
    await inputs[1].trigger('input', inputEvent('手工杆 1'))
    await submit.trigger('tap')
    assert.equal(page.recorder.lastOf('showToast').options.title, '请选择杆型')

    await page.wrapper.findAll('.chip').find((chip) => chip.text() === '美式杆').trigger('tap')
    await submit.trigger('tap')
    await page.flushPromises()
    assert.deepEqual(page.calls.created, [{ category: 'cue', model_name: '手工杆 1', brand_name: '自家小厂', cue_style: 'pool' }])
    assert.equal(page.recorder.lastOf('redirectTo').options.url, '/subPages/gear/item?id=7&added=1')
  } finally {
    await page.cleanup()
  }
})

test('choosing a catalog brand in the unlisted form sends brand_id and hides the free text brand', async () => {
  const page = await mountAdd()
  try {
    await page.wrapper.find('.unlisted-entry').trigger('tap')
    await page.wrapper.findAll('.unlisted-form .chip').find((chip) => chip.text() === 'Alpha').trigger('tap')
    assert.equal(page.wrapper.findAll('.unlisted-form .form-input').length, 1, 'only the model name input remains')
    await page.wrapper.find('.unlisted-form .form-input').trigger('input', inputEvent('新型号'))
    await page.wrapper.findAll('.unlisted-form .chip').find((chip) => chip.text() === '斯诺克 / 中式杆').trigger('tap')
    await page.wrapper.find('.submit-btn').trigger('tap')
    await page.flushPromises()
    assert.deepEqual(page.calls.created, [{ category: 'cue', model_name: '新型号', brand_id: 10, cue_style: 'snooker' }])
  } finally {
    await page.cleanup()
  }
})

test('switching to break and jump cues reloads the catalog and swaps the form to roles', async () => {
  const page = await mountAdd()
  try {
    await page.wrapper.findAll('.category-tab')[1].trigger('tap')
    await page.flushPromises()
    assert.ok(page.calls.brands.some((params) => params.category === 'break_jump'))
    assert.ok(page.calls.products.some((params) => params.category === 'break_jump' && !params.house))
    assert.equal(page.wrapper.findAll('.house-row').length, 0, 'there is no house break cue')

    await page.wrapper.find('.unlisted-entry').trigger('tap')
    const labels = page.wrapper.findAll('.unlisted-form .chip').map((chip) => chip.text())
    assert.ok(labels.includes('冲杆') && labels.includes('跳杆'))
    assert.ok(!labels.includes('美式杆'))
  } finally {
    await page.cleanup()
  }
})

test('a slow response for an old brand filter never overwrites the newer list', async () => {
  const slow = defer()
  let listCalls = 0
  const page = await mountAdd({
    api: {
      getGearProducts: (params) => {
        if (params.house) return Promise.resolve({ success: true, total: 0, list: [] })
        listCalls += 1
        if (listCalls === 1) return Promise.resolve({ success: true, total: 1, list: [product(11, '第一页的旧列表')] })
        if (listCalls === 2) return slow.promise
        return Promise.resolve({ success: true, total: 1, list: [product(13, '最新品牌的列表')] })
      }
    }
  })
  try {
    await page.wrapper.findAll('.brand-chip')[1].trigger('tap') // 选 Alpha：第 2 次请求，慢
    await page.wrapper.findAll('.brand-chip')[0].trigger('tap') // 回到全部品牌：第 3 次请求，快
    await page.flushPromises()
    assert.match(text(page.wrapper), /最新品牌的列表/)
    slow.resolve({ success: true, total: 1, list: [product(12, '迟到的旧列表')] })
    await page.flushPromises()
    assert.doesNotMatch(text(page.wrapper), /迟到的旧列表/)
    assert.match(text(page.wrapper), /最新品牌的列表/)
  } finally {
    await page.cleanup()
  }
})

test('a failed first page shows a retry that reloads the list', async () => {
  let attempt = 0
  const page = await mountAdd({
    api: {
      getGearProducts: async (params) => {
        if (params.house) return { success: true, total: 0, list: [] }
        attempt += 1
        if (attempt === 1) throw new Error('网络错误')
        return { success: true, total: 1, list: [product(11, 'A-100')] }
      }
    }
  })
  try {
    assert.match(text(page.wrapper), /型号加载失败/)
    await page.wrapper.findAll('button').find((button) => button.text() === '重试').trigger('tap')
    await page.flushPromises()
    assert.match(text(page.wrapper), /Alpha A-100/)
  } finally {
    await page.cleanup()
  }
})

test('the next page is requested when the page reaches the bottom and more rows remain', async () => {
  const page = await mountAdd({
    api: {
      getGearProducts: async (params) => {
        if (params.house) return { success: true, total: 0, list: [] }
        const rows = Array.from({ length: 20 }, (_, index) => product(100 + (params.page - 1) * 20 + index, `型号-${params.page}-${index}`))
        return { success: true, total: 25, list: params.page === 1 ? rows : rows.slice(0, 5) }
      }
    }
  })
  try {
    assert.equal(page.wrapper.findAll('.product-row').length, 20)
    page.context.hooks.reachBottom.forEach((callback) => callback())
    await page.flushPromises()
    assert.equal(page.wrapper.findAll('.product-row').length, 25)
    assert.equal(page.calls.products.filter((params) => !params.house).at(-1).page, 2)
    // 已经到底：再次触底不再请求
    const before = page.calls.products.length
    page.context.hooks.reachBottom.forEach((callback) => callback())
    await page.flushPromises()
    assert.equal(page.calls.products.length, before)
    assert.match(text(page.wrapper), /已经到底了/)
  } finally {
    await page.cleanup()
  }
})
