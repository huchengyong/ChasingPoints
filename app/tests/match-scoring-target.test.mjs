import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildJiuqiuWinOptions,
  buildRoundWinActionOptions,
  resolveFoulFeedback,
  resolveMatchApiActor,
  resolveScoreFeedback,
  resolveScoringParticipantLabel,
  resolveWinFeedback
} from '../utils/match-scoring-target.js'

test('match scoring actor keeps referee fixed and converts both player perspectives', () => {
  assert.equal(resolveMatchApiActor({ viewerRole: 'referee', isPlayer1: true, uiActor: 1 }), 1)
  assert.equal(resolveMatchApiActor({ viewerRole: 'referee', isPlayer1: true, uiActor: 2 }), 2)
  assert.equal(resolveMatchApiActor({ viewerRole: 'player1', isPlayer1: true, uiActor: 1 }), 1)
  assert.equal(resolveMatchApiActor({ viewerRole: 'player1', isPlayer1: true, uiActor: 2 }), 2)
  assert.equal(resolveMatchApiActor({ viewerRole: 'player2', isPlayer1: false, uiActor: 1 }), 2)
  assert.equal(resolveMatchApiActor({ viewerRole: 'player2', isPlayer1: false, uiActor: 2 }), 1)
})

test('referee options use fixed participant labels for both round-win game types', () => {
  const chineseEightBall = buildRoundWinActionOptions({ gameType: 3, viewerRole: 'referee', uiActor: 1 })
  const americanNineBall = buildRoundWinActionOptions({ gameType: 4, viewerRole: 'referee', uiActor: 2 })

  assert.deepEqual(chineseEightBall.map(item => item.title), ['选手1普胜', '选手1炸清', '选手1接清'])
  assert.deepEqual(americanNineBall.map(item => item.title), ['选手2普胜', '选手2小金', '选手2大金'])
})

test('jiuqiu options and score feedback describe the selected participant', () => {
  assert.deepEqual(
    buildJiuqiuWinOptions({ viewerRole: 'referee', uiActor: 1 }).map(item => item.desc),
    ['选手1 +4 分', '选手1 +7 分', '选手1 +10 分']
  )
  assert.equal(resolveScoreFeedback({ viewerRole: 'referee', uiActor: 2, score: 7 }), '已给选手2 +7 分')
  assert.equal(resolveWinFeedback({ viewerRole: 'referee', uiActor: 1 }), '已判给选手1')
})

test('foul feedback distinguishes the fouling participant from the scoring participant', () => {
  assert.equal(resolveFoulFeedback({ viewerRole: 'referee', foulingActor: 1, score: 4 }), '选手1犯规，选手2 +4 分')
  assert.equal(resolveFoulFeedback({ viewerRole: 'referee', foulingActor: 2, score: 7 }), '选手2犯规，选手1 +7 分')
  assert.equal(resolveFoulFeedback({ viewerRole: 'player2', foulingActor: 2, score: 1 }), '对手犯规，我方 +1 分')
  assert.equal(resolveScoringParticipantLabel({ viewerRole: 'player1', uiActor: 1 }), '我方')
})

test('playing page exposes referee controls for all four game types without fixed actor 2 writes', () => {
  const page = readFileSync(new URL('../subPages/match/playing.vue', import.meta.url), 'utf8')

  assert.match(page, /isRoundWinMode && isReferee/)
  assert.match(page, /refereeRoundWinGroups/)
  assert.match(page, /gameType\.value === 3 \|\| gameType\.value === 4/)
  assert.match(page, /gameType === 2 && isReferee/)
  assert.match(page, /refereeJiuqiuGroups/)
  assert.match(page, /data-referee-score-target/)
  assert.match(page, /data-referee-foul-target/)
  assert.match(page, /actor: convertActor\(scoringActor\)/)
  assert.match(page, /actor: convertActor\(foulingActor\)/)
  assert.doesNotMatch(page, /actor: convertActor\(2\)/)
})
