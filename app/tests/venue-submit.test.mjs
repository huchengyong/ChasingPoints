import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildVenueRegionSelection,
  buildVenueSubmitPayload,
  resolveVenueSubmitCopy
} from '../utils/venue-submit.js'

test('buildVenueSubmitPayload only keeps basic venue fields', () => {
  const payload = buildVenueSubmitPayload({
    name: '  星轨台球  ',
    city: ' 深圳 ',
    district: ' 南山区 ',
    address: ' 科苑路 8 号 ',
    phone: '13800138000',
    business_hours: '10:00-23:00',
    table_count: '18',
    price_range: '30-60 元/小时',
    description: '社区球房'
  })

  assert.deepEqual(payload, {
    name: '星轨台球',
    city: '深圳',
    district: '南山区',
    address: '科苑路 8 号'
  })
})

test('resolveVenueSubmitCopy describes the basics-only flow', () => {
  assert.deepEqual(resolveVenueSubmitCopy([]), {
    title: '确认常玩球馆后提交',
    tip: '首次有效补充并审核通过，送 1 个月会员。'
  })

  assert.deepEqual(resolveVenueSubmitCopy(['球馆名称', '详细地址']), {
    title: '还有 2 项未填写',
    tip: '请先填写：球馆名称、详细地址'
  })
})

test('buildVenueRegionSelection maps province city district path into display and payload fields', () => {
  const selection = buildVenueRegionSelection([
    { area_id: 19, name: '广东省' },
    { area_id: 321, name: '深圳市' },
    { area_id: 2723, name: '南山区' }
  ])

  assert.deepEqual(selection, {
    areaIds: [19, 321, 2723],
    regionText: '广东省 深圳市 南山区',
    city: '深圳市',
    district: '南山区'
  })
})

test('buildVenueRegionSelection keeps city when district is absent', () => {
  const selection = buildVenueRegionSelection([
    { area_id: 1, name: '北京' },
    { area_id: 36, name: '北京市' }
  ])

  assert.deepEqual(selection, {
    areaIds: [1, 36],
    regionText: '北京 北京市',
    city: '北京市',
    district: ''
  })
})
