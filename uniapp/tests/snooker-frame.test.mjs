import test from 'node:test'
import assert from 'node:assert/strict'

import {
  resolveSnookerFinishMatchAction,
  resolveSnookerNextFrameAction
} from '../utils/snooker-frame.js'

test('resolveSnookerNextFrameAction starts next frame directly when current frame is already closed', () => {
  assert.deepEqual(resolveSnookerNextFrameAction({
    currentFrameStarted: false,
    myFrameScore: 0,
    opponentFrameScore: 0
  }), {
    action: 'start_next_round'
  })
})

test('resolveSnookerNextFrameAction auto awards current frame to the higher score before starting next round', () => {
  assert.deepEqual(resolveSnookerNextFrameAction({
    currentFrameStarted: true,
    myFrameScore: 67,
    opponentFrameScore: 42
  }), {
    action: 'settle_and_start_next_round',
    winner: 1
  })

  assert.deepEqual(resolveSnookerNextFrameAction({
    currentFrameStarted: true,
    myFrameScore: 18,
    opponentFrameScore: 39
  }), {
    action: 'settle_and_start_next_round',
    winner: 2
  })
})

test('resolveSnookerNextFrameAction blocks auto settlement when frame scores are tied', () => {
  assert.deepEqual(resolveSnookerNextFrameAction({
    currentFrameStarted: true,
    myFrameScore: 35,
    opponentFrameScore: 35
  }), {
    action: 'blocked',
    message: '当前局比分相同，无法自动判定胜负，请继续记分或结束本场对局'
  })
})

test('resolveSnookerFinishMatchAction auto settles the current frame before finishing match', () => {
  assert.deepEqual(resolveSnookerFinishMatchAction({
    currentFrameStarted: true,
    myFrameScore: 52,
    opponentFrameScore: 37
  }), {
    action: 'settle_and_finish_match',
    winner: 1
  })
})

test('resolveSnookerFinishMatchAction allows finishing directly when current frame is empty or already closed', () => {
  assert.deepEqual(resolveSnookerFinishMatchAction({
    currentFrameStarted: false,
    myFrameScore: 0,
    opponentFrameScore: 0
  }), {
    action: 'finish_match'
  })

  assert.deepEqual(resolveSnookerFinishMatchAction({
    currentFrameStarted: true,
    myFrameScore: 0,
    opponentFrameScore: 0
  }), {
    action: 'finish_match'
  })
})

test('resolveSnookerFinishMatchAction blocks finishing when non-zero frame scores are tied', () => {
  assert.deepEqual(resolveSnookerFinishMatchAction({
    currentFrameStarted: true,
    myFrameScore: 41,
    opponentFrameScore: 41
  }), {
    action: 'blocked',
    message: '当前局比分相同，无法自动判定胜负，请继续记分后再结束本场对局'
  })
})
