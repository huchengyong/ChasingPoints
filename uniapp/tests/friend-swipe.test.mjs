import test from 'node:test'
import assert from 'node:assert/strict'

import {
  clampFriendSwipeOffset,
  resolveFriendSwipeEndOffset
} from '../utils/friend-swipe.js'

test('clampFriendSwipeOffset keeps offset within closed and fully open range', () => {
  assert.equal(clampFriendSwipeOffset(24, 160), 0)
  assert.equal(clampFriendSwipeOffset(-64, 160), -64)
  assert.equal(clampFriendSwipeOffset(-260, 160), -160)
})

test('resolveFriendSwipeEndOffset opens the row when drag passes half width', () => {
  assert.equal(resolveFriendSwipeEndOffset(-90, 160), -160)
})

test('resolveFriendSwipeEndOffset closes the row when drag is below half width', () => {
  assert.equal(resolveFriendSwipeEndOffset(-60, 160), 0)
})
