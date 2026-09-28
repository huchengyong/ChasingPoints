import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildPkReportHero,
  buildPkEvidenceList,
  buildPkPosterPayload,
  resolvePkReportStatus
} from '../utils/pk-report-view-model.js'

test('buildPkReportHero keeps social copy short and screenshot-friendly', () => {
  const hero = buildPkReportHero({
    stats: { total_matches: 12, my_wins: 7, opponent_wins: 5, win_rate: 58.33 },
    opponent: { name: '阿杰' },
    history: [{ result: 1 }, { result: 1 }, { result: 2 }, { result: 1 }, { result: 1 }]
  })

  assert.equal(hero.title, '你对阿杰略占上风')
  assert.match(hero.metaText, /近 5 场 4 胜 1 负/)
})

test('buildPkEvidenceList returns exactly two evidence rows for the MVP', () => {
  const evidence = buildPkEvidenceList({
    stats: {
      total_matches: 12,
      my_wins: 7,
      opponent_wins: 5,
      avg_score_diff: 1.8,
      max_win_streak: 3
    },
    history: [{ result: 1 }, { result: 1 }, { result: 2 }, { result: 1 }, { result: 1 }]
  })

  assert.equal(evidence.length, 2)
})

test('buildPkPosterPayload keeps summary and win rate humanized', () => {
  const payload = buildPkPosterPayload({
    hero: {
      title: '你对阿杰略占上风',
      scoreText: '7 : 5'
    },
    evidenceList: ['近 5 场 4 胜 1 负', '当前最长连胜 3 场'],
    stats: {
      total_matches: 12,
      my_wins: 7,
      opponent_wins: 5,
      win_rate: 58.33,
      avg_score_diff: 1.8,
      max_win_streak: 3
    },
    userName: '我',
    opponentName: '阿杰',
    gameTypeLabel: '中八'
  })

  assert.equal(payload.winRateLabel, '58%')
  assert.match(payload.summaryText, /你对阿杰略占上风/)
})

test('resolvePkReportStatus separates poster failure from data readiness', () => {
  assert.equal(
    resolvePkReportStatus({ statsLoaded: true, historyLoaded: true, totalMatches: 6 }),
    'ready'
  )
})

