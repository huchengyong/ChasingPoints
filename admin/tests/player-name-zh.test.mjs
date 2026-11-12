import test from 'node:test'
import assert from 'node:assert/strict'

import { PLAYER_NAME_ZH_MAX_LENGTH, isPlayerNameZhChanged, normalizePlayerNameZh } from '../src/utils/player-name-zh.ts'

test('player Chinese name drafts compare after trimming and allow clearing', () => {
  assert.equal(PLAYER_NAME_ZH_MAX_LENGTH, 32)
  assert.equal(normalizePlayerNameZh('  马克·塞尔比 '), '马克·塞尔比')
  assert.equal(normalizePlayerNameZh(undefined), '')
  assert.equal(isPlayerNameZhChanged('赵心童', ' 赵心童 '), false)
  assert.equal(isPlayerNameZhChanged('', '   '), false)
  assert.equal(isPlayerNameZhChanged('', '赵心童'), true)
  assert.equal(isPlayerNameZhChanged('赵心童', ''), true)
})
