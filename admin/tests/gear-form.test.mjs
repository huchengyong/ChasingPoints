import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildBrandPayload,
  buildProductPayload,
  defaultBrandForm,
  defaultProductForm,
  describeLinkConfirm,
  formatGearRoles,
  gearImageUrl,
  isValidGearImagePath,
  normalizeAliases,
  productTitle,
  productToForm,
  validateBrandForm,
  validateProductForm
} from '../src/utils/gear-form.ts'

const form = (overrides = {}) => ({ ...defaultProductForm('cue'), brand_id: 5, name: 'A-100', cue_style: 'pool', ...overrides })

test('image paths are accepted only from the gear directory with a png or webp file name', () => {
  assert.equal(isValidGearImagePath(''), true)
  assert.equal(isValidGearImagePath('/static/images/gear/a-100.png'), true)
  assert.equal(isValidGearImagePath('/static/images/gear/A_1.v2.webp'), true)
  for (const bad of [
    'https://evil.example/a.png', '/static/images/gear/../ranks/a.png', '/static/images/gear/sub/a.png',
    '/static/images/ranks/a.png', '/static/images/gear/a.gif', '/static/images/gear/.png', '/static/images/gear/a.png?x=1', ' /static/images/gear/a.png'
  ]) {
    assert.equal(isValidGearImagePath(bad), false, bad)
  }
  assert.equal(gearImageUrl('/static/images/gear/a.png'), 'https://cdn.zhuifen.cn/static/images/gear/a.png')
  assert.equal(gearImageUrl('https://evil.example/a.png'), '', 'a foreign address is never turned into a preview url')
  assert.equal(gearImageUrl(''), '')
})

test('aliases are normalized like the backend does', () => {
  assert.equal(normalizeAliases(' 阿尔法 ，, alp ,'), '阿尔法,alp')
  assert.equal(normalizeAliases(''), '')
  assert.equal(normalizeAliases(undefined), '')
})

test('brand form validation and payload', () => {
  assert.equal(validateBrandForm({ ...defaultBrandForm(), name: 'Alpha' }), '')
  assert.match(validateBrandForm({ ...defaultBrandForm(), name: '  ' }), /品牌名称/)
  assert.match(validateBrandForm({ ...defaultBrandForm(), name: '牌'.repeat(65) }), /品牌名称/)
  assert.match(validateBrandForm({ ...defaultBrandForm(), name: 'A', aliases: '别'.repeat(256) }), /别名/)
  assert.match(validateBrandForm({ ...defaultBrandForm(), name: 'A', status: 2 }), /状态/)

  assert.deepEqual(buildBrandPayload({ id: 0, name: ' Alpha ', aliases: '阿尔法，alp', sort: 3, status: 1 }), {
    name: 'Alpha', aliases: '阿尔法,alp', sort: 3, status: 1
  })
  assert.equal(buildBrandPayload({ id: 9, name: 'A', aliases: '', sort: 0, status: 0 }).id, 9)
})

test('a valid cue, break/jump cue, tip, chalk and house entries pass', () => {
  assert.equal(validateProductForm(form()), '')
  assert.equal(validateProductForm(form({ category: 'break_jump', cue_style: '', roles: ['jump', 'break'] })), '')
  assert.equal(validateProductForm(form({ category: 'tip', cue_style: '' })), '')
  assert.equal(validateProductForm(form({ category: 'chalk', cue_style: '', status: 0 })), '')
  assert.equal(validateProductForm(form({ brand_id: 0, cue_style: '', name: '另一种公杆' })), '', 'a house cue needs no brand and no cue style')
  assert.equal(validateProductForm(form({ category: 'chalk', brand_id: 0, cue_style: '', name: '球房巧克' })), '')
})

test('every backend rule is mirrored before submitting', () => {
  const cases = [
    ['category', form({ category: 'racket' }), /类别/],
    ['empty name', form({ name: ' ' }), /型号名称/],
    ['long name', form({ name: '型'.repeat(129) }), /型号名称/],
    ['bad status', form({ status: 5 }), /状态/],
    ['no brand chosen', form({ brand_id: null }), /请选择品牌/],
    ['negative brand', form({ brand_id: -1 }), /品牌/],
    ['brandless tip', form({ category: 'tip', brand_id: 0, cue_style: '' }), /球房公用/],
    ['brandless break cue', form({ category: 'break_jump', brand_id: 0, cue_style: '', roles: ['break'] }), /球房公用/],
    ['cue without style', form({ cue_style: '' }), /杆型/],
    ['cue with roles', form({ roles: ['break'] }), /冲跳身份/],
    ['break/jump without roles', form({ category: 'break_jump', cue_style: '', roles: [] }), /身份/],
    ['break/jump with bad role', form({ category: 'break_jump', cue_style: '', roles: ['fly'] }), /身份/],
    ['break/jump with style', form({ category: 'break_jump', roles: ['break'] }), /不设置杆型/],
    ['tip with style', form({ category: 'tip' }), /皮头和巧克/],
    ['chalk with roles', form({ category: 'chalk', cue_style: '', roles: ['jump'] }), /皮头和巧克/],
    ['negative length', form({ length_mm: -1 }), /长度/],
    ['too long', form({ length_mm: 3001 }), /长度/],
    ['too heavy', form({ weight_g: 2001 }), /重量/],
    ['wide tip', form({ tip_diameter_mm: 31 }), /皮头直径/],
    ['external image', form({ image_path: 'https://evil.example/a.png' }), /目录图/]
  ]
  for (const [name, input, pattern] of cases) {
    assert.match(validateProductForm(input), pattern, name)
  }
})

test('the payload carries only the fields that belong to the category', () => {
  assert.deepEqual(buildProductPayload(form({ length_mm: 1470.4, weight_g: 540, tip_diameter_mm: 12.905, name: ' A-100 ', roles: [] })), {
    category: 'cue', brand_id: 5, name: 'A-100', cue_style: 'pool', length_mm: 1470, weight_g: 540,
    tip_diameter_mm: 12.91, image_path: '', sort: 0, status: 1
  })
  const breakCue = buildProductPayload(form({ category: 'break_jump', cue_style: 'pool', roles: ['jump', 'break'] }))
  assert.deepEqual(breakCue.roles, ['break', 'jump'])
  assert.equal('cue_style' in breakCue, false, 'a break/jump cue never sends a cue style')
  const tip = buildProductPayload(form({ category: 'tip', cue_style: 'pool', roles: ['break'] }))
  assert.equal('cue_style' in tip, false)
  assert.equal('roles' in tip, false)
  assert.equal(buildProductPayload(form({ id: 7 })).id, 7)
  assert.equal('id' in buildProductPayload(form()), false)
})

test('editing restores the form from a list row without sharing the roles array', () => {
  const row = {
    id: 3, category: 'break_jump', brand_id: 5, name: 'A-BJ', cue_style: '', roles: ['break'], length_mm: 0,
    weight_g: 600, tip_diameter_mm: 0, image_path: '/static/images/gear/a.png', sort: 2, status: 0
  }
  const restored = productToForm(row)
  assert.equal(restored.id, 3)
  assert.equal(restored.status, 0)
  assert.deepEqual(restored.roles, ['break'])
  restored.roles.push('jump')
  assert.deepEqual(row.roles, ['break'])
  assert.equal(productToForm({ ...row, cue_style: undefined, roles: undefined }).cue_style, '')
})

test('titles and the link confirmation read naturally', () => {
  assert.equal(productTitle({ brand_name: 'Alpha', name: 'A-100' }), 'Alpha A-100')
  assert.equal(productTitle({ brand_name: '', name: '球房公杆' }), '球房公杆')
  assert.equal(formatGearRoles(['jump', 'break']), '冲杆 · 跳杆')
  const message = describeLinkConfirm({ brand_name: 'Brand X', model_name: 'Pro 1', count: 12 }, 'Alpha A-100')
  assert.match(message, /12 条用户登记/)
  assert.match(message, /Alpha A-100/)
  assert.match(message, /历史对局/)
})

test('a new product form starts without a brand so a house entry is never created by accident', () => {
  const blank = defaultProductForm('cue')
  assert.equal(blank.brand_id, null)
  assert.match(validateProductForm({ ...blank, name: '公杆', cue_style: 'pool' }), /请选择品牌/)
})
