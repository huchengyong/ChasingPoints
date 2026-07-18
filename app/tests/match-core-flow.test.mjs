import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildRematchContext,
  buildFinishedMatchDetailRoute,
  resolveFinishAction,
  resolveMatchHomePrimaryAction,
  resolveViewerRoleCopy
} from '../utils/match-core-flow.js'

test('home primary action prefers pending confirmation then ongoing match then start', () => {
  assert.equal(resolveMatchHomePrimaryAction({ currentMatch: null }).type, 'start')
  assert.equal(resolveMatchHomePrimaryAction({ currentMatch: { status: 1, finish_state: 'none' } }).type, 'continue')
  assert.equal(resolveMatchHomePrimaryAction({ currentMatch: { status: 1, finish_state: 'pending_confirmation' } }).type, 'confirm')
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

  assert.match(matchPage, /class="match-primary-action"/)
  assert.match(matchPage, /resolveMatchHomePrimaryAction/)
  assert.match(matchPage, /finish_state === 'pending_confirmation'/)
  assert.match(matchPage, /扫描对手二维码.*出示我的二维码.*选择好友或最近对手/)
  assert.match(matchPage, /练习赛（默认私密，不影响竞技权益）.*排位赛（公开展示，需结束确认）/)
  assert.match(matchPage, /challengeId: pendingStartContext\.value\?\.challenge_id \|\| 0/)
  assert.match(matchPage, /\/subPages\/match\/opponentSelector/)
  assert.match(matchPage, /class="match-qr-image"/)
  assert.doesNotMatch(matchPage, /challenges\?mode=select/)
  assert.match(matchPage, /buildFinishedMatchDetailRoute/)

  assert.match(playingPage, /排位赛等待确认/)
  assert.match(playingPage, /确认结束/)
  assert.match(playingPage, /提出异议/)
  assert.match(playingPage, /撤回请求/)
  assert.match(playingPage, /lastAction\?\.description/)
  assert.match(playingPage, /requestFinishMatch.*confirmFinishMatch.*disputeFinishMatch.*withdrawFinishMatch/s)

  assert.match(resultPage, /再来一局/)
  assert.match(resultPage, /分享战绩|生成战绩海报/)
  assert.match(resultPage, /查看交锋记录/)
  assert.doesNotMatch(resultPage, /保存并完成/)
  assert.match(resultPage, /pending_match_rematch/)

  assert.match(detailPage, /查看双方交锋记录/)
  assert.match(detailPage, /handleOpenH2H/)
  assert.match(challengePage, /线下扫码开局/)
  assert.match(challengePage, /已关联对局/)
  assert.match(challengePage, /查看对局/)
  assert.match(challengePage, /pending_match_challenge/)
})
