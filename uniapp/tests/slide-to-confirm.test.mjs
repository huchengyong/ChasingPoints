import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  SLIDE_COMPLETE_RATIO,
  slideProgress,
  isSlideComplete,
  shouldCommitOnRelease,
  isSlideFinishKey,
  isSlideResetKey
} from '../utils/slide-to-confirm.js'

test('slideProgress clamps to [0, 1]', () => {
  assert.equal(slideProgress(-5, 200), 0)
  assert.equal(slideProgress(0, 200), 0)
  assert.equal(slideProgress(100, 200), 0.5)
  assert.equal(slideProgress(300, 200), 1)
  assert.equal(slideProgress(100, 0), 0)
})

test('only release at the track end commits', () => {
  assert.equal(isSlideComplete(100, 200), false)
  assert.equal(isSlideComplete(200, 200), true)
  assert.equal(shouldCommitOnRelease(198, 200), true)
  assert.equal(shouldCommitOnRelease(195, 200), false)
  assert.equal(SLIDE_COMPLETE_RATIO <= 1, true)
})

test('keyboard finish and reset keys', () => {
  assert.equal(isSlideFinishKey('End'), true)
  assert.equal(isSlideFinishKey('ArrowRight'), false)
  assert.equal(isSlideResetKey('Escape'), true)
  assert.equal(isSlideResetKey('Home'), true)
  assert.equal(isSlideResetKey('Enter'), false)
})
