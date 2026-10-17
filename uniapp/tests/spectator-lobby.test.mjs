import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildSpectatorFinishedMatchDetailUrl,
  buildSpectatorMatchListParams,
  buildSpectatorSections,
  getSpectatorEmptyState,
  getSpectatorMatchStatusText,
  normalizeSpectatorMatch,
  resolveSpectatorLobbyQuery,
  resolveSpectatorLobbyTabs,
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
      subtitle: '公开对局开打后会出现在这里'
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

test('lobby tabs hide friends for guests and map to scope and status queries', () => {
  assert.deepEqual(resolveSpectatorLobbyTabs(false).map((tab) => tab.label), ['进行中', '已结束'])
  assert.deepEqual(resolveSpectatorLobbyTabs(true).map((tab) => tab.label), ['好友', '进行中', '已结束'])
  assert.deepEqual(resolveSpectatorLobbyQuery('friends'), { scope: 'friends', status: 0 })
  assert.deepEqual(resolveSpectatorLobbyQuery('live'), { scope: 'hall', status: 1 })
  assert.deepEqual(resolveSpectatorLobbyQuery('finished'), { scope: 'hall', status: 2 })
  assert.deepEqual(resolveSpectatorLobbyQuery('unknown'), { scope: 'hall', status: 1 })
})

test('normalizeSpectatorMatch marks the viewer and only finished winners', () => {
  const live = normalizeSpectatorMatch({
    id: 1,
    status: 1,
    game_type: 3,
    game_type_name: '中式八球',
    match_mode: 'ranked',
    player1_id: 18,
    player1_name: '我',
    player1_score: 4,
    player2_id: 20,
    player2_name: '阿杰',
    player2_score: 2,
    duration_seconds: 65
  }, { userId: 18 })

  assert.equal(live.isLive, true)
  assert.equal(live.isMine, true)
  assert.equal(live.metaText, '中式八球 · 排位')
  assert.equal(live.statusText, '进行中 1分')
  assert.deepEqual(live.players.map((player) => [player.name, player.score, player.isMe, player.outcome]), [
    ['我', 4, true, ''],
    ['阿杰', 2, false, '']
  ])

  const finished = normalizeSpectatorMatch({
    id: 2,
    status: 2,
    game_type: 1,
    match_mode: 'practice',
    player1_id: 30,
    player1_score: 3,
    player2_id: 31,
    player2_name: '球友 B',
    player2_score: 5
  }, { userId: 18 })

  assert.equal(finished.isLive, false)
  assert.equal(finished.isMine, false)
  assert.equal(finished.metaText, '斯诺克 · 练习')
  assert.equal(finished.statusText, '已结束')
  assert.deepEqual(finished.players.map((player) => [player.name, player.outcome]), [
    ['选手1', 'loser'],
    ['球友 B', 'winner']
  ])

  const draw = normalizeSpectatorMatch({ id: 3, status: 2, player1_score: 2, player2_score: 2 })
  assert.deepEqual(draw.players.map((player) => player.outcome), ['', ''])
})

test('buildSpectatorSections groups only the mixed friends tab', () => {
  const live = { id: 1, isLive: true }
  const ended = { id: 2, isLive: false }

  assert.deepEqual(buildSpectatorSections([ended, live], 'friends').map((section) => [section.title, section.items.map((item) => item.id)]), [
    ['进行中', [1]],
    ['已结束', [2]]
  ])
  assert.deepEqual(buildSpectatorSections([ended], 'friends').map((section) => section.key), ['finished'])
  assert.deepEqual(buildSpectatorSections([live], 'live'), [{ key: 'live', title: '', items: [live] }])
  assert.deepEqual(buildSpectatorSections([], 'live'), [])
})

test('match tab uses status tabs, an action-sheet game filter and paged loading', () => {
  const pageSource = readFileSync(new URL('../pages/match/index.vue', import.meta.url), 'utf8')
  const pagesJson = readFileSync(new URL('../pages.json', import.meta.url), 'utf8')

  assert.match(pageSource, /const lobbyTabs = computed\(\(\) => resolveSpectatorLobbyTabs\(userStore\.isLoggedIn\)\)/)
  assert.match(pageSource, /v-for="tab in lobbyTabs"[\s\S]*@tap="selectLobbyTab\(tab\.value\)"/)
  assert.match(pageSource, /const openGameTypeSheet[\s\S]*uni\.showActionSheet/)
  assert.match(pageSource, /GAME_TYPE_FILTER_OPTIONS_WITH_ALL/)
  assert.match(pageSource, /onReachBottom\(\(\) => \{\s*loadMore\(\)/)
  assert.match(pageSource, /page: nextPage/)
  // 切换标签先作废在途请求并清空旧列表
  assert.match(pageSource, /const reloadLobby = \(\) => \{[\s\S]*requestId: lobbyState\.value\.requestId \+ 1[\s\S]*spectatorMatches\.value = \[\]/)
  // 「我的对局」只在顶部卡片出现，列表按 id 排除
  assert.match(pageSource, /const lobbyItems = computed[\s\S]*Number\(match\.id\) !== currentId/)
  // 扫码执裁入口常驻，不再只出现在空态
  assert.match(pageSource, /class="challenge-entry-btn" @tap="handleScanAsReferee"/)
  assert.doesNotMatch(pageSource, /showFilterPanel|confirmSpectatorFilters|status-segmented|game-type-choice/)
  assert.match(pagesJson, /"navigationBarTitleText":\s*"观赛"/)
  assert.match(pagesJson, /"text":\s*"观赛"/)
})
