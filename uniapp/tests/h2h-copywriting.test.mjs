import test from 'node:test'
import assert from 'node:assert/strict'

import {
  resolveAdvantageLevel,
  buildH2HTitle,
  buildH2HSubtitle,
  buildPrimaryActionText
} from '../utils/h2h-copywriting.js'

test('resolveAdvantageLevel downgrades copy for tiny samples', () => {
  assert.equal(
    resolveAdvantageLevel({ totalMatches: 1, myWins: 1, opponentWins: 0 }),
    'slight_lead'
  )
})

test('resolveAdvantageLevel distinguishes clear leads for larger samples', () => {
  assert.equal(
    resolveAdvantageLevel({ totalMatches: 12, myWins: 9, opponentWins: 3 }),
    'clear_lead'
  )
})

test('buildH2HTitle uses relationship copy instead of raw percentages', () => {
  assert.equal(
    buildH2HTitle({
      totalMatches: 8,
      myWins: 5,
      opponentWins: 3,
      opponentName: '阿杰'
    }),
    '你对阿杰略占上风'
  )
})

test('buildH2HSubtitle includes total record and recent form', () => {
  assert.equal(
    buildH2HSubtitle({
      myWins: 5,
      opponentWins: 3,
      recentHistory: [
        { result: 1 },
        { result: 1 },
        { result: 2 },
        { result: 1 },
        { result: 1 }
      ]
    }),
    '总交锋 5 胜 3 负，近 5 场 4 胜 1 负'
  )
})

test('buildPrimaryActionText maps balanced records to final match copy', () => {
  assert.equal(buildPrimaryActionText('balanced'), '约一场决胜局')
})

