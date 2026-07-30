import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  SNOOKER_BEST_OF_OPTIONS,
  buildSnookerBestOfLabels,
  chooseSnookerStartFormat
} from '../utils/snooker-start-format.js'

test('snooker start format exposes positive odd best-of options', () => {
  assert.deepEqual(SNOOKER_BEST_OF_OPTIONS, [1, 3, 5, 7, 9])
  assert.deepEqual(buildSnookerBestOfLabels(), [
    '1 局 单局决胜',
    '3 局 （抢 2）',
    '5 局 （抢 3）',
    '7 局 （抢 4）',
    '9 局 （抢 5）'
  ])
})

test('chooseSnookerStartFormat returns the selected format and starter', async () => {
  const selections = [2, 1]
  const calls = []
  const format = await chooseSnookerStartFormat({
    showActionSheet({ itemList, success }) {
      calls.push(itemList)
      success({ tapIndex: selections.shift() })
    }
  })
  assert.deepEqual(format, { best_of_frames: 5, starting_actor: 2 })
  assert.equal(calls.length, 2)
})

test('chooseSnookerStartFormat stops when either selection is cancelled', async () => {
  assert.equal(await chooseSnookerStartFormat({
    showActionSheet({ fail }) { fail() }
  }), null)
})

test('all direct match entry pages use the shared snooker format chooser', () => {
  for (const relativePath of ['../pages/index/index.vue', '../pages/ranking/index.vue', '../pages/user/index.vue', '../pages/match/index.vue']) {
    const page = readFileSync(new URL(relativePath, import.meta.url), 'utf8')
    assert.match(page, /chooseSnookerStartFormat/)
    assert.doesNotMatch(page, /best_of_frames:\s*3,\s*starting_actor:\s*1/)
  }
})
