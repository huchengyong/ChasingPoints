import test from 'node:test'
import assert from 'node:assert/strict'

import { buildHomeShareCard, buildMatchShareCard } from '../utils/share-card.js'

const finishedMatch = (overrides = {}) => ({
  id: 501,
  status: 2,
  visibility: 'public',
  game_type: 3,
  match_mode: 'ranked',
  is_player1: true,
  my_name: '阿伟',
  opponent_name: '小林',
  my_score: 7,
  opponent_score: 4,
  ...overrides
})

test('a public finished match shares names, score, game type and mode', () => {
  assert.deepEqual(buildMatchShareCard(finishedMatch()), {
    title: '阿伟 7:4 小林 · 中式八球排位赛',
    path: '/subPages/match/matchDetail?match_id=501&mode=spectate'
  })
})

test('the order is player 1 then player 2 even when the sharer is player 2', () => {
  const card = buildMatchShareCard(finishedMatch({
    is_player1: false,
    my_name: '小林',
    opponent_name: '阿伟',
    my_score: 4,
    opponent_score: 7
  }))

  assert.equal(card.title, '阿伟 7:4 小林 · 中式八球排位赛')
})

test('an ongoing public match shares a live title without the score', () => {
  const card = buildMatchShareCard(finishedMatch({ status: 1, game_type: 1, match_mode: 'practice' }))

  assert.equal(card.title, '阿伟 vs 小林 正在比赛 · 斯诺克')
})

test('private, cancelled and unloaded matches fall back to the generic card', () => {
  const generic = buildHomeShareCard()
  assert.equal(generic.title, '追分 · 记录每一局真实比分')
  assert.equal(generic.path, '/pages/index/index')

  assert.deepEqual(buildMatchShareCard(finishedMatch({ visibility: 'private' })), generic)
  assert.deepEqual(buildMatchShareCard(finishedMatch({ status: 3 })), generic)
  assert.deepEqual(buildMatchShareCard({}), generic)
  assert.deepEqual(buildMatchShareCard(), generic)
})

test('private cards never contain names or scores', () => {
  const card = buildMatchShareCard(finishedMatch({ visibility: 'private' }))

  assert.equal(card.title.includes('阿伟'), false)
  assert.equal(card.title.includes('7'), false)
})

test('long nicknames are truncated to eight characters', () => {
  const card = buildMatchShareCard(finishedMatch({
    my_name: '一二三四五六七八九十',
    opponent_name: '对手名字也很长很长'
  }))

  assert.equal(card.title, '一二三四五六七八… 7:4 对手名字也很长很… · 中式八球排位赛')
})

test('the public detail shape shares player 1 then player 2', () => {
  const card = buildMatchShareCard({
    id: 502,
    status: 2,
    visibility: 'public',
    game_type: 2,
    match_mode: 'ranked',
    player1_name: '发起者',
    player1_score: 9,
    player2_name: '应战者',
    player2_score: 6
  })

  assert.equal(card.title, '发起者 9:6 应战者 · 九球追分排位赛')
  assert.equal(card.path, '/subPages/match/matchDetail?match_id=502&mode=spectate')
})
