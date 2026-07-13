import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildH2HCalendarViewModel,
  buildH2HHeroViewModel,
  buildH2HMatchCards,
  resolveH2HPageStatus,
  shouldShowH2HSummaryCard
} from '../utils/h2h-view-model.js'

test('buildH2HHeroViewModel exposes title, subtitle and badges from stats/history', () => {
  const viewModel = buildH2HHeroViewModel({
    stats: {
      total_matches: 8,
      my_wins: 5,
      opponent_wins: 3,
      avg_score_diff: 1.8,
      max_win_streak: 3
    },
    opponent: {
      name: '阿杰',
      avatar: 'https://img.example/a.png'
    },
    history: [
      { id: 1, result: 1 },
      { id: 2, result: 1 },
      { id: 3, result: 2 },
      { id: 4, result: 1 },
      { id: 5, result: 1 }
    ],
    subjectName: '你',
    subjectAvatar: 'https://img.example/me.png'
  })

  assert.equal(viewModel.title, '你对阿杰略占上风')
  assert.equal(viewModel.subtitle, '总交锋 5 胜 3 负，近 5 场 4 胜 1 负')
  assert.equal(viewModel.scoreText, '5 : 3')
  assert.equal(viewModel.badges.length, 3)
})

test('buildH2HCalendarViewModel aggregates same-day wins and losses', () => {
  const calendar = buildH2HCalendarViewModel([
    { id: 1, result: 1, match_time: '2026-04-15T10:00:00+08:00' },
    { id: 2, result: 1, match_time: '2026-04-15T12:00:00+08:00' },
    { id: 3, result: 2, match_time: '2026-04-15T14:00:00+08:00' },
    { id: 4, result: 2, match_time: '2026-04-17T18:00:00+08:00' }
  ], { monthKey: '2026-04' })

  assert.equal(calendar.title, '2026年4月')
  assert.equal(calendar.cells.length, 42)

  const day15 = calendar.cells.find((cell) => cell.dateKey === '2026-04-15')
  assert.equal(day15.day, 15)
  assert.equal(day15.isCurrentMonth, true)
  assert.equal(day15.winCount, 2)
  assert.equal(day15.lossCount, 1)
  assert.equal(day15.matchCount, 3)
  assert.deepEqual(day15.matchIds, [1, 2, 3])

  const leadingCell = calendar.cells[0]
  assert.equal(leadingCell.dateKey, '2026-03-29')
  assert.equal(leadingCell.isCurrentMonth, false)
})

test('buildH2HMatchCards produces cards with diff and default tags', () => {
  const cards = buildH2HMatchCards([
    {
      id: 10,
      game_type_name: '中八',
      my_score: 7,
      opponent_score: 5,
      result: 1,
      match_time: '2026-03-28T10:00:00Z'
    }
  ])

  assert.equal(cards[0].scoreText, '7 : 5')
  assert.equal(cards[0].diffText, '+2')
  assert.deepEqual(cards[0].tags, ['险胜'])
})

test('resolveH2HPageStatus returns partial when only stats are available', () => {
  assert.equal(
    resolveH2HPageStatus({
      statsLoaded: true,
      historyLoaded: false,
      totalMatches: 6,
      historyLength: 0,
      hasError: true
    }),
    'partial'
  )
})

test('resolveH2HPageStatus returns empty when there are no matches', () => {
  assert.equal(
    resolveH2HPageStatus({
      statsLoaded: true,
      historyLoaded: true,
      totalMatches: 0,
      historyLength: 0,
      hasError: false
    }),
    'empty'
  )
})

test('resolveH2HPageStatus prefers partial over ready when stale loaded flags meet a fresh error', () => {
  assert.equal(
    resolveH2HPageStatus({
      statsLoaded: true,
      historyLoaded: true,
      totalMatches: 6,
      historyLength: 6,
      hasError: true
    }),
    'partial'
  )
})

test('shouldShowH2HSummaryCard hides hero when stats are unavailable in partial states', () => {
  assert.equal(
    shouldShowH2HSummaryCard({
      pageStatus: 'partial',
      statsLoaded: false
    }),
    false
  )
  assert.equal(
    shouldShowH2HSummaryCard({
      pageStatus: 'partial',
      statsLoaded: true
    }),
    true
  )
})
