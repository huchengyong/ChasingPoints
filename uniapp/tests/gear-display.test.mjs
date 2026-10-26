import test from 'node:test'
import assert from 'node:assert/strict'
import { existsSync } from 'node:fs'

import {
  GEAR_DEFAULT_IMAGE_PATHS,
  buildFriendGearUrl,
  buildGearDefaultsView,
  canBeMainBreakCue,
  canBeMainCue,
  formatGearCueStyle,
  formatGearRoles,
  formatGearSpecs,
  gearItemSubtitle,
  gearItemTitle,
  gearProductTitle,
  groupGearItems,
  isGearImagePath,
  listMainCandidates,
  listMainGameLabels,
  resolveGearDefaultImagePath,
  resolveGearImage
} from '../utils/gear-display.js'

const CDN = 'https://cdn.zhuifen.cn'

test('every default image path points at a real file in cdn-assets', () => {
  for (const [name, path] of Object.entries(GEAR_DEFAULT_IMAGE_PATHS)) {
    assert.ok(existsSync(new URL(`../cdn-assets${path}`, import.meta.url)), `${name} -> ${path} 缺少对应的 cdn-assets 文件`)
  }
})

test('default image follows the category: house cue, cue style, break/jump role, tip and chalk', () => {
  assert.equal(resolveGearDefaultImagePath({ category: 'cue', house: true }), GEAR_DEFAULT_IMAGE_PATHS.houseCue)
  assert.equal(resolveGearDefaultImagePath({ category: 'cue', cueStyle: 'pool' }), GEAR_DEFAULT_IMAGE_PATHS.poolCue)
  assert.equal(resolveGearDefaultImagePath({ category: 'cue', cueStyle: 'snooker' }), GEAR_DEFAULT_IMAGE_PATHS.snookerCue)
  assert.equal(resolveGearDefaultImagePath({ category: 'cue' }), GEAR_DEFAULT_IMAGE_PATHS.snookerCue)
  assert.equal(resolveGearDefaultImagePath({ category: 'break_jump', roles: ['break'] }), GEAR_DEFAULT_IMAGE_PATHS.breakCue)
  assert.equal(resolveGearDefaultImagePath({ category: 'break_jump', roles: ['break', 'jump'] }), GEAR_DEFAULT_IMAGE_PATHS.breakCue, '开跳杆用冲杆图')
  assert.equal(resolveGearDefaultImagePath({ category: 'break_jump', roles: ['jump'] }), GEAR_DEFAULT_IMAGE_PATHS.jumpCue)
  assert.equal(resolveGearDefaultImagePath({ category: 'tip' }), GEAR_DEFAULT_IMAGE_PATHS.tip)
  assert.equal(resolveGearDefaultImagePath({ category: 'chalk', house: true }), GEAR_DEFAULT_IMAGE_PATHS.chalk)
  assert.equal(resolveGearDefaultImagePath(), GEAR_DEFAULT_IMAGE_PATHS.snookerCue)
})

test('catalog image paths are only accepted from the gear directory with a png or webp file name', () => {
  assert.equal(isGearImagePath('/static/images/gear/a-100.png'), true)
  assert.equal(isGearImagePath('/static/images/gear/A_1.v2.webp'), true)
  for (const bad of [
    '', null, undefined, 42,
    'https://evil.example/a.png',
    '/static/images/gear/../ranks/a.png',
    '/static/images/gear/sub/a.png',
    '/static/images/ranks/a.png',
    '/static/images/gear/a.gif',
    '/static/images/gear/.png',
    '/static/images/gear/a.png?x=1'
  ]) {
    assert.equal(isGearImagePath(bad), false, `${bad} 不应被接受`)
  }
})

test('resolveGearImage uses the catalog image when valid and falls back to the category default otherwise', () => {
  const poolItem = { category: 'cue', cue_style: 'pool', house: false, product: { image_path: '/static/images/gear/a-100.png' } }
  assert.deepEqual(resolveGearImage(poolItem), {
    src: `${CDN}/static/images/gear/a-100.png`,
    fallback: `${CDN}${GEAR_DEFAULT_IMAGE_PATHS.poolCue}`
  })

  const noImage = { category: 'cue', cue_style: 'pool', product: { image_path: '' } }
  assert.equal(resolveGearImage(noImage).src, `${CDN}${GEAR_DEFAULT_IMAGE_PATHS.poolCue}`)

  const hostile = { category: 'tip', product: { image_path: 'https://evil.example/a.png' } }
  assert.equal(resolveGearImage(hostile).src, `${CDN}${GEAR_DEFAULT_IMAGE_PATHS.tip}`)

  // 目录型号本身（不是装备）同样适用
  const product = { category: 'break_jump', roles: ['jump'], image_path: '/static/images/gear/jump-1.webp' }
  assert.equal(resolveGearImage(product).src, `${CDN}/static/images/gear/jump-1.webp`)
  assert.equal(resolveGearImage(product).fallback, `${CDN}${GEAR_DEFAULT_IMAGE_PATHS.jumpCue}`)

  // 未收录装备没有 product：按杆型回退
  const unlisted = { category: 'cue', cue_style: 'snooker', unlisted: true }
  assert.equal(resolveGearImage(unlisted).src, `${CDN}${GEAR_DEFAULT_IMAGE_PATHS.snookerCue}`)
  assert.equal(resolveGearImage(null).src, `${CDN}${GEAR_DEFAULT_IMAGE_PATHS.snookerCue}`)
})

test('specs hide what the brand did not provide and show both grams and ounces', () => {
  assert.deepEqual(formatGearSpecs({ length_mm: 1470, weight_g: 540, tip_diameter_mm: 12.9 }), [
    { key: 'length', label: '长度', value: '1470 mm' },
    { key: 'weight', label: '重量', value: '540 g（19.0 oz）' },
    { key: 'tipDiameter', label: '皮头直径', value: '12.9 mm' }
  ])
  assert.deepEqual(formatGearSpecs({ length_mm: 0, weight_g: 0, tip_diameter_mm: 0 }), [], '0 表示未提供，不能显示成 0')
  assert.deepEqual(formatGearSpecs({ weight_g: 510 }).map((spec) => spec.key), ['weight'])
  assert.equal(formatGearSpecs({ tip_diameter_mm: 13 })[0].value, '13 mm')
  assert.equal(formatGearSpecs({ tip_diameter_mm: 11.75 })[0].value, '11.75 mm')
  assert.deepEqual(formatGearSpecs(null), [])
  assert.deepEqual(formatGearSpecs(undefined), [])
  assert.deepEqual(formatGearSpecs({ weight_g: -3, length_mm: 'abc' }), [])
})

test('titles combine brand and model, and house or unlisted entries read naturally', () => {
  assert.equal(gearItemTitle({ brand_name: 'Alpha', model_name: 'A-100' }), 'Alpha A-100')
  assert.equal(gearItemTitle({ brand_name: '', model_name: '球房公杆', house: true }), '球房公杆')
  assert.equal(gearItemTitle({ brand_name: ' 其他品牌 ', model_name: '未收录型号' }), '其他品牌 未收录型号')
  assert.equal(gearItemTitle({}), '未命名装备')
  assert.equal(gearItemTitle(null), '')
  assert.equal(gearProductTitle({ brand_name: 'Alpha', name: 'A-Tip' }), 'Alpha A-Tip')
  assert.equal(gearProductTitle({ brand_name: '', name: '球房巧克' }), '球房巧克')
})

test('subtitle shows the cue style or the roles and flags unlisted models', () => {
  assert.equal(gearItemSubtitle({ category: 'cue', cue_style: 'pool' }), '美式杆')
  assert.equal(gearItemSubtitle({ category: 'cue', cue_style: 'snooker', unlisted: true }), '斯诺克 / 中式杆 · 未收录')
  assert.equal(gearItemSubtitle({ category: 'cue', house: true }), '球房公用')
  assert.equal(gearItemSubtitle({ category: 'break_jump', roles: ['break', 'jump'] }), '冲杆 · 跳杆')
  assert.equal(formatGearRoles(['jump']), '跳杆')
  assert.equal(formatGearRoles([]), '')
  assert.equal(formatGearCueStyle('carom'), '')
})

const sample = () => ([
  { id: 1, category: 'cue', status: 1, roles: [] },
  { id: 2, category: 'cue', status: 2, roles: [] },
  { id: 3, category: 'break_jump', status: 1, roles: ['break'] },
  { id: 4, category: 'break_jump', status: 1, roles: ['jump'] },
  { id: 5, category: 'break_jump', status: 2, roles: ['break', 'jump'] },
  { id: 6, category: 'cue', status: 3, roles: [] },
  { id: 7, category: 'break_jump', status: 1, roles: ['jump', 'break'] }
])

test('items are grouped into cues, break/jump cues and retired, and deleted ones disappear', () => {
  const groups = groupGearItems(sample())
  assert.deepEqual(groups.cues.map((item) => item.id), [1])
  assert.deepEqual(groups.breakJump.map((item) => item.id), [3, 4, 7])
  assert.deepEqual(groups.retired.map((item) => item.id), [2, 5])
  assert.deepEqual(groupGearItems(), { cues: [], breakJump: [], retired: [] })
  assert.deepEqual(groupGearItems([null, undefined]), { cues: [], breakJump: [], retired: [] })
})

test('main candidates only include active cues for the cue slot and active break cues for the break slot', () => {
  assert.deepEqual(listMainCandidates(sample(), 'cue').map((item) => item.id), [1])
  assert.deepEqual(listMainCandidates(sample(), 'break').map((item) => item.id), [3, 7])
  assert.equal(canBeMainCue({ category: 'cue', status: 2 }), false)
  assert.equal(canBeMainBreakCue({ category: 'break_jump', status: 1, roles: ['jump'] }), false)
  assert.equal(canBeMainBreakCue(null), false)
})

test('defaults view lists game types in the fixed order and snooker has no break slot', () => {
  const items = sample()
  const view = buildGearDefaultsView([
    { game_type: 3, slot: 'cue', item_id: 1 },
    { game_type: 3, slot: 'break', item_id: 3 },
    { game_type: 1, slot: 'cue', item_id: 1 },
    { game_type: 1, slot: 'break', item_id: 3 },
    { game_type: 4, slot: 'cue', item_id: 999 }
  ], items)
  assert.deepEqual(view.map((row) => row.gameType), [3, 2, 1, 4])
  const eight = view[0]
  assert.equal(eight.label, '中式八球')
  assert.equal(eight.cue.id, 1)
  assert.equal(eight.breakCue.id, 3)
  const snooker = view[2]
  assert.equal(snooker.supportsBreak, false)
  assert.equal(snooker.breakCue, null, '斯诺克没有冲杆槽位，即使数据里有也不展示')
  assert.equal(view[1].cue, null)
  assert.equal(view[3].cue, null, '指向不存在装备的设置按未设置展示')
  assert.equal(buildGearDefaultsView().length, 4)
})

test('main-game labels read as "game + main" for cues and "game + break main" for break cues, in the fixed game order', () => {
  const defaults = [
    { game_type: 4, slot: 'cue', item_id: 1 },
    { game_type: 3, slot: 'cue', item_id: 1 },
    { game_type: 2, slot: 'break', item_id: 1 },
    { game_type: 3, slot: 'cue', item_id: 2 }
  ]
  assert.deepEqual(listMainGameLabels(defaults, 1), ['中式八球 主力', '九球追分 冲杆主力', '美式九球 主力'])
  assert.deepEqual(listMainGameLabels(defaults, 2), ['中式八球 主力'])
  assert.deepEqual(listMainGameLabels(defaults, 99), [])
  assert.deepEqual(listMainGameLabels(), [])
})

test('the friend gear address needs a numeric user id and encodes the nickname', () => {
  assert.equal(buildFriendGearUrl({ userId: 7, name: '老王' }), `/subPages/gear/friend?user_id=7&name=${encodeURIComponent('老王')}`)
  assert.equal(buildFriendGearUrl({ userId: '7', name: '  ' }), '/subPages/gear/friend?user_id=7')
  assert.equal(buildFriendGearUrl({ userId: 'abc' }), '/subPages/gear/friend?user_id=0')
  assert.equal(buildFriendGearUrl(), '/subPages/gear/friend?user_id=0')
  assert.equal(buildFriendGearUrl({ userId: 7, name: 'a&b=c' }), '/subPages/gear/friend?user_id=7&name=a%26b%3Dc')
})
