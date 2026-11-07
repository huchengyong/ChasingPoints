import test from 'node:test'
import assert from 'node:assert/strict'

import { GEAR_BRAND_NAME_MAX, GEAR_MODEL_NAME_MAX, buildUnlistedGearPayload, toggleGearRole } from '../utils/gear-form.js'

test('a cue with a typed brand builds the unlisted payload and trims the names', () => {
  const result = buildUnlistedGearPayload({ category: 'cue', brandId: 0, brandName: '  自家小厂 ', modelName: ' 手工杆 1 ', cueStyle: 'snooker' })
  assert.deepEqual(result, {
    ok: true,
    payload: { category: 'cue', model_name: '手工杆 1', brand_name: '自家小厂', cue_style: 'snooker' }
  })
})

test('a catalog brand wins over a typed brand name and is sent as brand_id only', () => {
  const result = buildUnlistedGearPayload({ category: 'cue', brandId: 12, brandName: '残留的自填内容', modelName: 'Pro 1', cueStyle: 'pool' })
  assert.equal(result.ok, true)
  assert.equal(result.payload.brand_id, 12)
  assert.equal('brand_name' in result.payload, false)
})

test('break and jump cues need at least one role and send roles in a stable order', () => {
  const none = buildUnlistedGearPayload({ category: 'break_jump', brandId: 3, modelName: 'X', roles: [] })
  assert.equal(none.ok, false)
  assert.match(none.message, /冲杆或跳杆/)

  const both = buildUnlistedGearPayload({ category: 'break_jump', brandId: 3, modelName: 'X', roles: ['jump', 'break'] })
  assert.deepEqual(both.payload.roles, ['break', 'jump'])
  assert.equal('cue_style' in both.payload, false)
})

test('every required field is validated with a specific message', () => {
  const base = { category: 'cue', brandId: 0, brandName: '品牌', modelName: '型号', cueStyle: 'pool' }
  assert.match(buildUnlistedGearPayload({ ...base, modelName: '   ' }).message, /型号名称/)
  assert.match(buildUnlistedGearPayload({ ...base, brandName: ' ' }).message, /品牌/)
  assert.match(buildUnlistedGearPayload({ ...base, cueStyle: '' }).message, /杆型/)
  assert.match(buildUnlistedGearPayload({ ...base, cueStyle: 'carom' }).message, /杆型/)
  assert.match(buildUnlistedGearPayload({ ...base, category: 'tip' }).message, /类别/)
  assert.equal(buildUnlistedGearPayload().ok, false)
})

test('length limits count characters, not UTF-16 units', () => {
  const base = { category: 'cue', brandId: 0, brandName: '品牌', modelName: '型号', cueStyle: 'pool' }
  assert.equal(buildUnlistedGearPayload({ ...base, modelName: '型'.repeat(GEAR_MODEL_NAME_MAX) }).ok, true)
  assert.equal(buildUnlistedGearPayload({ ...base, modelName: '型'.repeat(GEAR_MODEL_NAME_MAX + 1) }).ok, false)
  assert.equal(buildUnlistedGearPayload({ ...base, brandName: '牌'.repeat(GEAR_BRAND_NAME_MAX) }).ok, true)
  assert.equal(buildUnlistedGearPayload({ ...base, brandName: '牌'.repeat(GEAR_BRAND_NAME_MAX + 1) }).ok, false)
  // 每个 emoji 在 JS 里占两个 UTF-16 单元，但只算一个字
  assert.equal(buildUnlistedGearPayload({ ...base, modelName: '🎱'.repeat(GEAR_MODEL_NAME_MAX) }).ok, true)
})

test('toggling a role keeps break before jump', () => {
  assert.deepEqual(toggleGearRole([], 'jump'), ['jump'])
  assert.deepEqual(toggleGearRole(['jump'], 'break'), ['break', 'jump'])
  assert.deepEqual(toggleGearRole(['break', 'jump'], 'break'), ['jump'])
})
