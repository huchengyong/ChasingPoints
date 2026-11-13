import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildVenueRegionSelection,
  buildVenueSubmitPayload,
  normalizePickedLocation,
  resolveVenueSubmitCopy,
  resolveVenueSubmitIntro,
  stripRegionPrefix
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

test('buildVenueSubmitPayload adds picked coordinates when location is valid', () => {
  const payload = buildVenueSubmitPayload({
    name: '星轨台球',
    city: '上海市',
    district: '浦东新区',
    address: '世纪大道 100 号',
    location: { latitude: '31.234583', longitude: 121.507572, name: '星轨台球', address: '上海市浦东新区世纪大道100号' }
  })

  assert.deepEqual(payload, {
    name: '星轨台球',
    city: '上海市',
    district: '浦东新区',
    address: '世纪大道 100 号',
    latitude: 31.234583,
    longitude: 121.507572
  })
})

test('buildVenueSubmitPayload drops invalid picked coordinates', () => {
  const base = {
    name: '星轨台球',
    city: '上海市',
    district: '浦东新区',
    address: '世纪大道 100 号'
  }
  const expected = { ...base }

  assert.deepEqual(buildVenueSubmitPayload({ ...base, location: null }), expected)
  assert.deepEqual(buildVenueSubmitPayload({ ...base, location: { latitude: 0, longitude: 0 } }), expected)
  assert.deepEqual(buildVenueSubmitPayload({ ...base, location: { latitude: 'abc', longitude: 'abc' } }), expected)
})

test('normalizePickedLocation only keeps finite non-zero coordinates', () => {
  assert.deepEqual(
    normalizePickedLocation({ name: ' 星轨台球 ', address: ' 世纪大道100号 ', latitude: '31.234583', longitude: 121.507572 }),
    { latitude: 31.234583, longitude: 121.507572, name: '星轨台球', address: '世纪大道100号' }
  )
  assert.equal(normalizePickedLocation(null), null)
  assert.equal(normalizePickedLocation({ latitude: 0, longitude: 121.5 }), null)
  assert.equal(normalizePickedLocation({ latitude: 31.2, longitude: 0 }), null)
  assert.equal(normalizePickedLocation({ latitude: 'abc', longitude: 121.5 }), null)
})

test('stripRegionPrefix removes the leading region names once each', () => {
  assert.equal(
    stripRegionPrefix('上海市浦东新区世纪大道100号', ['上海市', '上海市', '浦东新区']),
    '世纪大道100号'
  )
  assert.equal(
    stripRegionPrefix('广东省深圳市南山区科苑路8号', ['广东省', '深圳市', '南山区']),
    '科苑路8号'
  )
  assert.equal(stripRegionPrefix('科苑路8号', ['广东省', '深圳市', '南山区']), '科苑路8号')
})

test('stripRegionPrefix prefers longer region names for municipalities', () => {
  const shanghai = ['上海', '上海市', '浦东新区']
  const beijing = ['北京', '北京市', '朝阳区']

  assert.equal(stripRegionPrefix('上海市浦东新区世纪大道100号', shanghai), '世纪大道100号')
  assert.equal(stripRegionPrefix('北京市朝阳区建国路88号', beijing), '建国路88号')
  assert.equal(stripRegionPrefix('浦东新区世纪大道100号', shanghai), '世纪大道100号')
  assert.equal(stripRegionPrefix('上海市浦东新区世纪大道100号', []), '上海市浦东新区世纪大道100号')
})

test('resolveVenueSubmitCopy only promises VIP while the venue reward is claimable', () => {
  assert.deepEqual(resolveVenueSubmitCopy([], { enabled: true, reward_days: 30, status: 'not_started' }), {
    title: '确认常玩球馆后提交',
    tip: '首次有效补充并审核通过，送 1 个月VIP。'
  })

  assert.deepEqual(resolveVenueSubmitCopy([]), {
    title: '确认常玩球馆后提交',
    tip: '审核通过后会展示给附近球友。'
  })

  assert.deepEqual(resolveVenueSubmitCopy([], { enabled: true, reward_days: 30, status: 'reward_granted' }), {
    title: '确认常玩球馆后提交',
    tip: '审核通过后会展示给附近球友。'
  })

  assert.deepEqual(resolveVenueSubmitCopy(['球馆名称', '详细地址'], { enabled: true, reward_days: 30 }), {
    title: '还有 2 项未填写',
    tip: '请先填写：球馆名称、详细地址'
  })
})

test('resolveVenueSubmitIntro uses the configured reward duration only when claimable', () => {
  assert.deepEqual(resolveVenueSubmitIntro({ enabled: true, reward_days: 14, status: 'not_started' }).title, '免费领取 14 天VIP')
  assert.equal(resolveVenueSubmitIntro({ enabled: false, reward_days: 30, status: 'not_started' }).title, '提交常去球馆')
  assert.doesNotMatch(resolveVenueSubmitIntro({ enabled: true, reward_days: 30, status: 'reward_granted' }).sectionTip, /VIP/)
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
