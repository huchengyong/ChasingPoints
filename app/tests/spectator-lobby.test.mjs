import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildSpectatorMatchListParams,
  getSpectatorEmptyState,
  getSpectatorMatchStatusText,
  shouldOpenPlayingForSpectatorMatch
} from '../utils/spectator-lobby.js'

test('buildSpectatorMatchListParams keeps hall status and game type filters explicit', () => {
  assert.deepEqual(
    buildSpectatorMatchListParams({
      scope: 'hall',
      status: 2,
      gameType: 2,
      page: 3,
      pageSize: 20
    }),
    {
      scope: 'hall',
      status: 2,
      game_type: 2,
      page: 3,
      page_size: 20
    }
  )
})

test('buildSpectatorMatchListParams omits status for friend scope and keeps game type filter', () => {
  assert.deepEqual(
    buildSpectatorMatchListParams({
      scope: 'friends',
      status: 1,
      gameType: 3,
      page: 1,
      pageSize: 20
    }),
    {
      scope: 'friends',
      game_type: 3,
      page: 1,
      page_size: 20
    }
  )
})

test('getSpectatorEmptyState separates hall, finished, friend and game type copy', () => {
  assert.deepEqual(
    getSpectatorEmptyState({ scope: 'hall', status: 1, gameType: 0 }),
    {
      title: '暂无正在进行的对局',
      subtitle: '可以先发起 PK，或稍后回来观赛'
    }
  )

  assert.deepEqual(
    getSpectatorEmptyState({ scope: 'hall', status: 2, gameType: 0 }),
    {
      title: '暂无已结束对局',
      subtitle: '有新的完赛记录后会出现在这里'
    }
  )

  assert.deepEqual(
    getSpectatorEmptyState({ scope: 'friends', status: 1, gameType: 2 }),
    {
      title: '暂无九球追分好友对局',
      subtitle: '好友参与该球种的对局会集中展示在这里'
    }
  )
})

test('getSpectatorMatchStatusText formats ongoing duration and finished copy', () => {
  assert.equal(getSpectatorMatchStatusText({ status: 1, durationSeconds: 65 }), '进行中 1分')
  assert.equal(getSpectatorMatchStatusText({ status: 2, durationSeconds: 3600 }), '已结束')
})

test('shouldOpenPlayingForSpectatorMatch only treats my ongoing match as editable', () => {
  assert.equal(shouldOpenPlayingForSpectatorMatch({
    match: { status: 1, player1_id: 18, player2_id: 20 },
    userId: 18,
    isLoggedIn: true
  }), true)

  assert.equal(shouldOpenPlayingForSpectatorMatch({
    match: { status: 2, player1_id: 18, player2_id: 20 },
    userId: 18,
    isLoggedIn: true
  }), false)
})

test('match tab exposes spectator lobby controls and tab text', () => {
  const pageSource = readFileSync(new URL('../pages/match/index.vue', import.meta.url), 'utf8')
  const helperSource = readFileSync(new URL('../utils/spectator-lobby.js', import.meta.url), 'utf8')
  const pagesJson = readFileSync(new URL('../pages.json', import.meta.url), 'utf8')

  assert.match(pageSource, /SPECTATOR_SCOPES/)
  assert.match(helperSource, /大厅/)
  assert.match(helperSource, /好友/)
  assert.match(pageSource, /GAME_TYPE_FILTER_OPTIONS_WITH_ALL/)
  assert.match(pagesJson, /"navigationBarTitleText":\s*"观赛"/)
  assert.match(pagesJson, /"text":\s*"观赛"/)
})
