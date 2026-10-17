import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildRematchContext,
  buildFinishedMatchDetailRoute,
  buildOngoingMatchCard,
  resolveFinishAction,
  resolveMatchHomePrimaryAction,
  resolveViewerRoleCopy
} from '../utils/match-core-flow.js'

test('home primary action prefers pending confirmation then ongoing match then start', () => {
  assert.equal(resolveMatchHomePrimaryAction({ currentMatch: null }).type, 'start')
  assert.equal(resolveMatchHomePrimaryAction({ currentMatch: { status: 1, finish_state: 'none' } }).type, 'continue')
  assert.equal(resolveMatchHomePrimaryAction({ currentMatch: { status: 1, finish_state: 'pending_confirmation' } }).type, 'confirm')
})

test('ongoing match card keeps fixed player order and marks the viewer', () => {
  const card = buildOngoingMatchCard({
    id: 31,
    status: 1,
    viewer_role: 'player2',
    game_type_name: '中式八球',
    match_mode: 'ranked',
    player1_id: 11,
    player1_name: '选手甲',
    player2_id: 22,
    player2_name: '选手乙',
    my_score: 3,
    opponent_score: 5,
    duration_seconds: 720
  }, { id: 22, nickname: '选手乙' })

  assert.equal(card.id, 31)
  assert.equal(card.statusText, '进行中')
  assert.equal(card.metaText, '中式八球 · 排位')
  assert.equal(card.durationText, '12分')
  assert.equal(card.actionLabel, '继续对局')
  assert.deepEqual(card.players.map((item) => [item.name, item.score, item.isMe]), [
    ['选手甲', 5, false],
    ['选手乙', 3, true]
  ])
})

test('ongoing match card covers referee and pending confirmation, and skips finished matches', () => {
  const referee = buildOngoingMatchCard({
    id: 7,
    status: 1,
    viewer_role: 'referee',
    player1_name: '选手甲',
    player2_name: '选手乙',
    my_score: 2,
    opponent_score: 1
  }, { id: 99 })
  assert.equal(referee.statusText, '执裁中')
  assert.equal(referee.actionLabel, '进入裁判记分')
  assert.deepEqual(referee.players.map((item) => item.isMe), [false, false])

  const pending = buildOngoingMatchCard({ id: 8, status: 1, finish_state: 'pending_confirmation', match_mode: 'practice' })
  assert.equal(pending.isPending, true)
  assert.equal(pending.statusText, '待确认结束')
  assert.equal(pending.actionLabel, '处理结束确认')
  assert.equal(pending.metaText, '台球 · 练习')

  assert.equal(buildOngoingMatchCard(null), null)
  assert.equal(buildOngoingMatchCard({ id: 9, status: 2 }), null)
})

test('finish action follows server capabilities instead of inferring role locally', () => {
  assert.equal(resolveFinishAction({ can_request_finish: true }).type, 'request')
  assert.equal(resolveFinishAction({ can_confirm_finish: true, can_dispute_finish: true }).type, 'confirm')
  assert.equal(resolveFinishAction({ can_withdraw_finish: true }).type, 'withdraw')
  assert.equal(resolveFinishAction({ can_finish: true }).type, 'finish')
  assert.equal(resolveFinishAction({}).type, 'readonly')
})

test('viewer role copy uses player labels for players and fixed labels for referees', () => {
  assert.deepEqual(resolveViewerRoleCopy('referee'), { left: '选手1', right: '选手2', role: '裁判视角' })
  assert.deepEqual(resolveViewerRoleCopy('player2'), { left: '我方', right: '对手', role: '选手视角' })
})

test('rematch context keeps opponent game type and previous mode', () => {
  assert.deepEqual(buildRematchContext({
    opponent_id: 2001,
    opponent_name: '球友A',
    opponent_avatar: 'a.png',
    game_type: 3,
    match_mode: 'practice'
  }), {
    opponent_id: 2001,
    opponent_name: '球友A',
    opponent_avatar: 'a.png',
    game_type: 3,
    match_mode: 'practice'
  })
})

test('finished match cards route to single match detail before h2h', () => {
  assert.equal(buildFinishedMatchDetailRoute({ id: 88, status: 2 }), '/subPages/match/matchDetail?match_id=88&mode=spectate')
  assert.equal(
    buildFinishedMatchDetailRoute({ id: 88, status: 2, player1_id: 101, player2_id: 202 }, 101),
    '/subPages/match/matchDetail?match_id=88'
  )
  assert.equal(buildFinishedMatchDetailRoute({ id: 88, status: 1 }), '')
})

test('match core pages keep the full launch, confirmation, and post-match exits', () => {
  const matchPage = readFileSync(new URL('../pages/match/index.vue', import.meta.url), 'utf8')
  const playingPage = readFileSync(new URL('../subPages/match/playing.vue', import.meta.url), 'utf8')
  const resultPage = readFileSync(new URL('../subPages/match/matchResult.vue', import.meta.url), 'utf8')
  const detailPage = readFileSync(new URL('../subPages/match/matchDetail.vue', import.meta.url), 'utf8')
  const challengePage = readFileSync(new URL('../subPages/social/challenges.vue', import.meta.url), 'utf8')

  const startMatchHelper = readFileSync(new URL('../utils/start-match.js', import.meta.url), 'utf8')

  assert.match(matchPage, /class="play-card match-primary-action"/)
  assert.match(matchPage, /buildOngoingMatchCard\(currentMatch\.value/)
  assert.match(matchPage, /扫描对手二维码.*出示我的二维码.*选择好友或最近对手/)
  assert.match(matchPage, /challenge-entry-btn/)
  assert.match(matchPage, /challenge-quick-card/)
  assert.match(matchPage, /START_MATCH_MODE_OPTIONS\.map\(\(item\) => item\.label\)/)
  assert.match(startMatchHelper, /练习赛（默认私密，不影响竞技权益）[\s\S]*排位赛（公开展示，需结束确认）/)
  assert.match(matchPage, /\/subPages\/match\/opponentSelector/)
  assert.match(matchPage, /class="match-qr-image"/)
  assert.doesNotMatch(matchPage, /challenges\?mode=select/)
  assert.match(matchPage, /buildFinishedMatchDetailRoute/)

  assert.match(playingPage, /finishConfirmView\.title/)
  assert.match(playingPage, /向右滑动确认结束/)
  assert.match(playingPage, /有异议，继续比赛/)
  assert.match(playingPage, /撤销结束/)
  assert.match(playingPage, /lastAction\?\.description/)
  assert.match(playingPage, /requestFinishMatch.*confirmFinishMatch.*disputeFinishMatch.*withdrawFinishMatch/s)

  assert.match(resultPage, /再来一局/)
  assert.match(resultPage, /分享战绩|生成战绩海报/)
  assert.match(resultPage, /查看交锋记录/)
  assert.doesNotMatch(resultPage, /保存并完成/)
  assert.match(resultPage, /pending_match_rematch/)

  assert.match(detailPage, /查看双方交锋记录/)
  assert.match(detailPage, /handleOpenH2H/)
  assert.match(challengePage, /接受约球/)
  assert.match(challengePage, /查看对局/)
  assert.match(challengePage, /再约一场/)
  assert.match(challengePage, /getChallengeHistory/)
})
