import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildPlayingRoute,
  resolveMatchScanAction,
  resolveStartMatchGuardAction,
  shouldPromptOngoingMatch
} from '../utils/ongoing-match-guard.js'

test('resolveStartMatchGuardAction returns navigate outcome for created matches', () => {
  const result = resolveStartMatchGuardAction({
    response: {
      success: true,
      action: 'created',
      match_id: 18
    },
    selectedGameType: 2,
    scannedOpponent: {
      nickname: '球友A',
      avatar: 'avatar-a.png'
    }
  })

  assert.equal(result.type, 'navigate')
  assert.equal(result.match.match_id, 18)
  assert.equal(result.match.game_type, 2)
  assert.equal(result.match.opponent_name, '球友A')
  assert.equal(result.match.opponent_avatar, 'avatar-a.png')
})

test('resolveStartMatchGuardAction returns resume outcome for existing match reuse', () => {
  const result = resolveStartMatchGuardAction({
    response: {
      success: true,
      action: 'resume_existing',
      match_id: 28,
      ongoing_match: {
        id: 28,
        game_type: 3,
        opponent_name: '球友B',
        opponent_avatar: 'avatar-b.png'
      }
    },
    selectedGameType: 2,
    scannedOpponent: {
      nickname: '球友C',
      avatar: 'avatar-c.png'
    }
  })

  assert.equal(result.type, 'resume')
  assert.equal(result.match.match_id, 28)
  assert.equal(result.match.game_type, 3)
  assert.equal(result.match.opponent_name, '球友B')
})

test('resolveStartMatchGuardAction returns self-block outcome with actionable match', () => {
  const result = resolveStartMatchGuardAction({
    response: {
      success: true,
      action: 'blocked',
      block_reason: 'self_ongoing',
      message: '你还有一场中式八球未结束，请先结束之前的对局',
      match_id: 38,
      ongoing_match: {
        id: 38,
        game_type: 3,
        opponent_name: '球友D',
        opponent_avatar: 'avatar-d.png'
      }
    },
    selectedGameType: 2,
    scannedOpponent: {
      nickname: '球友E',
      avatar: 'avatar-e.png'
    }
  })

  assert.equal(result.type, 'prompt_self_ongoing')
  assert.equal(result.message, '你还有一场中式八球未结束，请先结束之前的对局')
  assert.equal(result.match.match_id, 38)
  assert.equal(result.match.opponent_name, '球友D')
})

test('resolveStartMatchGuardAction returns opponent-block outcome without navigation target', () => {
  const result = resolveStartMatchGuardAction({
    response: {
      success: true,
      action: 'blocked',
      block_reason: 'opponent_ongoing',
      message: '对手还有未结束的对局，暂时无法开始新的 PK'
    },
    selectedGameType: 3,
    scannedOpponent: {
      nickname: '球友F',
      avatar: 'avatar-f.png'
    }
  })

  assert.deepEqual(result, {
    type: 'toast_opponent_ongoing',
    message: '对手还有未结束的对局，暂时无法开始新的 PK'
  })
})

test('shouldPromptOngoingMatch skips prompt when already inside playing page', () => {
  assert.equal(shouldPromptOngoingMatch({
    isLoggedIn: true,
    currentRoute: '/subPages/match/playing?match_id=18',
    currentMatch: { id: 18 },
    hasPromptedInForeground: false
  }), false)
})

test('shouldPromptOngoingMatch skips prompt when this foreground cycle already prompted', () => {
  assert.equal(shouldPromptOngoingMatch({
    isLoggedIn: true,
    currentRoute: '/pages/index/index',
    currentMatch: { id: 18 },
    hasPromptedInForeground: true
  }), false)
})

test('shouldPromptOngoingMatch skips prompt when current route is not ready yet', () => {
  assert.equal(shouldPromptOngoingMatch({
    isLoggedIn: true,
    currentRoute: '',
    currentMatch: { id: 18 },
    hasPromptedInForeground: false
  }), false)
})

test('shouldPromptOngoingMatch prompts when logged in user has ongoing match outside playing page', () => {
  assert.equal(shouldPromptOngoingMatch({
    isLoggedIn: true,
    currentRoute: '/pages/index/index',
    currentMatch: { id: 18 },
    hasPromptedInForeground: false
  }), true)
})

test('buildPlayingRoute encodes opponent info for navigation', () => {
  const route = buildPlayingRoute({
    match_id: 52,
    game_type: 3,
    opponent_name: '球友 G',
    opponent_avatar: 'https://img.example/avatar g.png'
  })

  assert.equal(
    route,
    '/subPages/match/playing?match_id=52&game_type=3&opponent_name=%E7%90%83%E5%8F%8B%20G&opponent_avatar=https%3A%2F%2Fimg.example%2Favatar%20g.png'
  )
})

test('resolveMatchScanAction identifies regular opponent payloads', () => {
  assert.deepEqual(
    resolveMatchScanAction(JSON.stringify({
      user_id: 202,
      nickname: '球友H',
      avatar: 'avatar-h.png'
    })),
    {
      type: 'start_match',
      opponent: {
        user_id: 202,
        nickname: '球友H',
        avatar: 'avatar-h.png'
      }
    }
  )
})

test('resolveMatchScanAction identifies referee join payloads', () => {
  assert.deepEqual(
    resolveMatchScanAction(JSON.stringify({
      type: 'match_referee',
      match_id: 82,
      join_token: 'token-82'
    })),
    {
      type: 'join_referee',
      refereeJoin: {
        match_id: 82,
        join_token: 'token-82'
      }
    }
  )
})
