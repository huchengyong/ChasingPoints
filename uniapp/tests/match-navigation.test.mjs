import test from 'node:test'
import assert from 'node:assert/strict'

import {
  shouldLeavePlayingPage,
  consumeResultNavigationGuard,
  getMatchHistoryPageUrl,
  getMatchHistoryTabUrl
} from '../utils/match-navigation.js'

test('shouldLeavePlayingPage returns true when match detail status is completed', () => {
  assert.equal(shouldLeavePlayingPage({
    pageMatchId: 42,
    detailStatus: 2
  }), true)
})

test('shouldLeavePlayingPage returns true when current active match no longer matches page match', () => {
  assert.equal(shouldLeavePlayingPage({
    pageMatchId: 42,
    detailStatus: 1,
    currentMatchId: 99
  }), true)
})

test('shouldLeavePlayingPage returns true when there is no current active match for this page match', () => {
  assert.equal(shouldLeavePlayingPage({
    pageMatchId: 42,
    detailStatus: 1,
    currentMatchId: 0
  }), true)
})

test('shouldLeavePlayingPage returns false when current active match is still this match', () => {
  assert.equal(shouldLeavePlayingPage({
    pageMatchId: 42,
    detailStatus: 1,
    currentMatchId: 42
  }), false)
})

test('shouldLeavePlayingPage returns false when current active match is temporarily unknown', () => {
  assert.equal(shouldLeavePlayingPage({
    pageMatchId: 42,
    detailStatus: 1
  }), false)
})

test('match history navigation routes point to user center and match history page', () => {
  assert.equal(getMatchHistoryTabUrl(), '/pages/user/index')
  assert.equal(getMatchHistoryPageUrl(), '/subPages/user/matchHistory')
})

test('consumeResultNavigationGuard only allows the first result-page navigation', () => {
  const state = {}

  assert.equal(consumeResultNavigationGuard(state), true)
  assert.equal(consumeResultNavigationGuard(state), false)
})
