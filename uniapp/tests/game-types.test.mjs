import test from 'node:test'
import assert from 'node:assert/strict'

import {
  GAME_TYPE_FILTER_OPTIONS_WITH_ALL,
  GAME_TYPE_KEY_MAP,
  GAME_TYPE_LABEL_MAP,
  GAME_TYPE_OPTIONS,
  GAME_TYPE_STATS_TABS,
  GAME_TYPE_VALUE_MAP,
  getGameTypeLabel,
  getRuleCategoryLabel
} from '../utils/game-types.js'

test('GAME_TYPE_OPTIONS uses the agreed display order', () => {
  assert.deepEqual(GAME_TYPE_OPTIONS, [
    { value: 3, label: '中式八球' },
    { value: 2, label: '九球追分' },
    { value: 1, label: '斯诺克' },
    { value: 4, label: '美式九球' }
  ])
})

test('GAME_TYPE_FILTER_OPTIONS_WITH_ALL keeps all option first and ordered game types after it', () => {
  assert.deepEqual(GAME_TYPE_FILTER_OPTIONS_WITH_ALL, [
    { value: 0, label: '全部球种' },
    { value: 3, label: '中式八球' },
    { value: 2, label: '九球追分' },
    { value: 1, label: '斯诺克' },
    { value: 4, label: '美式九球' }
  ])
})

test('game type label and stats key mappings stay aligned', () => {
  assert.equal(GAME_TYPE_LABEL_MAP[3], '中式八球')
  assert.equal(GAME_TYPE_LABEL_MAP[2], '九球追分')
  assert.equal(GAME_TYPE_LABEL_MAP[1], '斯诺克')
  assert.equal(GAME_TYPE_LABEL_MAP[4], '美式九球')

  assert.equal(GAME_TYPE_KEY_MAP[3], 'chinese_eight')
  assert.equal(GAME_TYPE_KEY_MAP[2], 'nine_ball')
  assert.equal(GAME_TYPE_KEY_MAP[1], 'snooker')
  assert.equal(GAME_TYPE_KEY_MAP[4], 'american_nine')

  assert.equal(GAME_TYPE_VALUE_MAP.chinese_eight, 3)
  assert.equal(GAME_TYPE_VALUE_MAP.nine_ball, 2)
  assert.equal(GAME_TYPE_VALUE_MAP.snooker, 1)
  assert.equal(GAME_TYPE_VALUE_MAP.american_nine, 4)

  assert.deepEqual(GAME_TYPE_STATS_TABS, [
    { key: 'chinese_eight', label: '中式八球' },
    { key: 'nine_ball', label: '九球追分' },
    { key: 'snooker', label: '斯诺克' },
    { key: 'american_nine', label: '美式九球' }
  ])
})

test('helpers return ordered labels for game types and rule categories', () => {
  assert.equal(getGameTypeLabel(3), '中式八球')
  assert.equal(getGameTypeLabel(2), '九球追分')
  assert.equal(getGameTypeLabel(1), '斯诺克')
  assert.equal(getGameTypeLabel(4), '美式九球')
  assert.equal(getGameTypeLabel(99, '未知球种'), '未知球种')

  assert.equal(getRuleCategoryLabel('chinese_eight'), '中式八球')
  assert.equal(getRuleCategoryLabel('nine_ball'), '九球追分')
  assert.equal(getRuleCategoryLabel('snooker'), '斯诺克')
  assert.equal(getRuleCategoryLabel('american_nine'), '美式九球')
})
