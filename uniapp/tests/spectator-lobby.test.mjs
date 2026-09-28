import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildSpectatorFinishedMatchDetailUrl,
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

test('buildSpectatorFinishedMatchDetailUrl routes finished spectator matches to h2h records', () => {
  assert.equal(
    buildSpectatorFinishedMatchDetailUrl({
      match: {
        status: 2,
        player1_id: 18,
        player1_name: '我',
        player1_avatar: 'https://img.example/me.png',
        player2_id: 52,
        player2_name: '球友 B',
        player2_avatar: 'https://img.example/b.png'
      },
      userId: 18
    }),
    '/subPages/user/h2hRecord?opponent_id=52&opponent_name=%E7%90%83%E5%8F%8B%20B'
  )

  assert.equal(
    buildSpectatorFinishedMatchDetailUrl({
      match: {
        status: 2,
        player1_id: 52,
        player1_name: '球友 B',
        player1_avatar: 'https://img.example/b.png',
        player2_id: 18,
        player2_name: '我',
        player2_avatar: 'https://img.example/me.png'
      },
      userId: 18
    }),
    '/subPages/user/h2hRecord?opponent_id=52&opponent_name=%E7%90%83%E5%8F%8B%20B'
  )

  assert.equal(
    buildSpectatorFinishedMatchDetailUrl({
      match: {
        status: 2,
        player1_id: 52,
        player1_name: '球友 B',
        player1_avatar: 'https://img.example/b.png',
        player2_id: 77,
        player2_name: '阿杰'
      },
      userId: 18
    }),
    '/subPages/user/h2hRecord?target_user_id=52&target_name=%E7%90%83%E5%8F%8B%20B&target_avatar=https%3A%2F%2Fimg.example%2Fb.png&opponent_id=77&opponent_name=%E9%98%BF%E6%9D%B0'
  )

  assert.equal(
    buildSpectatorFinishedMatchDetailUrl({
      match: {
        status: 1,
        player1_id: 52,
        player1_name: '球友 B',
        player2_id: 77,
        player2_name: '阿杰'
      },
      userId: 18
    }),
    ''
  )
})

test('match tab exposes spectator lobby controls and tab text', () => {
  const pageSource = readFileSync(new URL('../pages/match/index.vue', import.meta.url), 'utf8')
  const styleSource = readFileSync(new URL('../pages/match/index.scss', import.meta.url), 'utf8')
  const helperSource = readFileSync(new URL('../utils/spectator-lobby.js', import.meta.url), 'utf8')
  const pagesJson = readFileSync(new URL('../pages.json', import.meta.url), 'utf8')

  assert.match(pageSource, /SPECTATOR_SCOPES/)
  assert.match(helperSource, /大厅/)
  assert.match(helperSource, /好友/)
  assert.match(pageSource, /GAME_TYPE_FILTER_OPTIONS_WITH_ALL/)
  assert.match(pageSource, /class="lobby-toolbar-row"[\s\S]*class="scope-tabs"[\s\S]*class="filter-button"/)
  assert.doesNotMatch(pageSource, /<view v-if="currentScope === 'hall'" class="status-tabs">/)
  assert.doesNotMatch(pageSource, /<scroll-view class="game-type-filter" scroll-x>/)
  assert.match(pageSource, /showFilterPanel/)
  assert.match(pageSource, /confirmSpectatorFilters/)
  assert.match(pageSource, /buildSpectatorFinishedMatchDetailUrl/)
  assert.match(pageSource, /uni\.hideTabBar/)
  assert.match(pageSource, /uni\.showTabBar/)
  assert.match(pageSource, /class="status-segmented"/)
  assert.match(pageSource, /class="game-type-choice-list"/)
  assert.match(pageSource, /class="game-type-choice"/)
  assert.match(pageSource, /type="checkmarkempty"/)
  assert.doesNotMatch(pageSource, /class="filter-option"/)
  assert.match(styleSource, /\.status-segmented\s*\{[\s\S]*display:\s*grid;[\s\S]*grid-template-columns:\s*repeat\(2,\s*minmax\(0,\s*1fr\)\);/)
  assert.match(styleSource, /\.game-type-choice-list\s*\{[\s\S]*display:\s*flex;[\s\S]*flex-direction:\s*column;/)
  assert.match(styleSource, /\.game-type-choice\s*\{[\s\S]*margin:\s*0;/)
  assert.match(styleSource, /\.filter-cancel,[\s\S]*\.filter-confirm\s*\{[\s\S]*margin:\s*0;/)
  assert.match(pagesJson, /"navigationBarTitleText":\s*"观赛"/)
  assert.match(pagesJson, /"text":\s*"观赛"/)
})
