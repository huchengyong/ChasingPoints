import test from 'node:test'
import assert from 'node:assert/strict'

import {
  resolveCoreMetrics,
  resolveMemberEntry,
  resolveMemberRankAvatarFrame,
  resolveRankAvatarFramePath,
  resolveRankCardView,
  resolveReputationChip
} from '../utils/user-homepage.js'

const now = new Date('2026-07-20T12:00:00+08:00')
const activeMember = {
  is_active: true,
  member_expires_at: '2026-08-01 12:00:00'
}

test('resolveRankAvatarFramePath maps all six rank levels to unique local assets', () => {
  assert.deepEqual(
    [1, 2, 3, 4, 5, 6].map(resolveRankAvatarFramePath),
    [
      '/static/images/avatar-frames/frame_bronze.png',
      '/static/images/avatar-frames/frame_silver.png',
      '/static/images/avatar-frames/frame_gold.png',
      '/static/images/avatar-frames/frame_platinum.png',
      '/static/images/avatar-frames/frame_diamond.png',
      '/static/images/avatar-frames/frame_king.png'
    ]
  )
  assert.equal(resolveRankAvatarFramePath(0), '')
  assert.equal(resolveRankAvatarFramePath(7), '')
})

test('resolveMemberRankAvatarFrame shows the current rank frame for an active member', () => {
  assert.equal(resolveMemberRankAvatarFrame({
    isLoggedIn: true,
    memberStatus: activeMember,
    rankInfo: { level: 5 },
    now
  }), '/static/images/avatar-frames/frame_diamond.png')
})

test('resolveMemberRankAvatarFrame hides frames for ordinary, expired, and failed member states', () => {
  const states = [
    null,
    { is_active: false, member_expires_at: '2026-08-01 12:00:00' },
    { is_active: true, member_expires_at: '2026-07-20 12:00:00' },
    { is_active: true, member_expires_at: 'invalid' }
  ]

  for (const memberStatus of states) {
    assert.equal(resolveMemberRankAvatarFrame({
      isLoggedIn: true,
      memberStatus,
      rankInfo: { level: 6 },
      now
    }), '')
  }
})

test('resolveMemberRankAvatarFrame waits for the real rank and hides stale switch data', () => {
  assert.equal(resolveMemberRankAvatarFrame({
    isLoggedIn: true,
    memberStatus: activeMember,
    rankInfo: null,
    rankLoading: true,
    now
  }), '')

  assert.equal(resolveMemberRankAvatarFrame({
    isLoggedIn: true,
    memberStatus: activeMember,
    rankInfo: { level: 5 },
    rankLoading: true,
    now
  }), '')

  assert.equal(resolveMemberRankAvatarFrame({
    isLoggedIn: true,
    memberStatus: activeMember,
    rankInfo: null,
    rankLoading: false,
    now
  }), '')
})

test('career metrics cover all game types and leave the per-game rank score to the rank card', () => {
  const metrics = resolveCoreMetrics({ stats: { total_matches: 86, wins: 51, win_rate: 59.3, max_win_streak: 7 } })
  assert.deepEqual(metrics.map(item => [item.label, item.value]), [
    ['场次', 86],
    ['胜场', 51],
    ['胜率', '59.3%'],
    ['最高连胜', 7]
  ])
  assert.deepEqual(resolveCoreMetrics({ stats: null, isLoading: true }).map(item => item.value), ['—', '—', '—', '—'])
})

test('rank card shows the score gap to the next rank, and only level six is the max rank', () => {
  assert.deepEqual(resolveRankCardView({
    level: 3, name: '黄金', icon: 'gold.png', rank_score: 1320, next_name: '铂金', next_score: 1500, progress: 64
  }), { ranked: true, name: '黄金', icon: 'gold.png', score: 1320, progress: 64, hint: '距铂金还差 180 分' })

  const diamond = resolveRankCardView({ level: 5, name: '钻石', rank_score: 2400, next_name: '王者', next_score: 3000, progress: 40 })
  assert.equal(diamond.hint, '距王者还差 600 分')
  assert.equal(diamond.progress, 40)

  const king = resolveRankCardView({ level: 6, name: '王者', rank_score: 3200, next_score: 3000, progress: 100 })
  assert.equal(king.hint, '已达到最高段位')
  assert.equal(king.progress, 100)

  const unranked = resolveRankCardView(null)
  assert.equal(unranked.name, '未定级')
  assert.equal(unranked.ranked, false)
  assert.equal(unranked.hint, '完成首场排位赛后开始定级')
})

test('member entry only labels active or expired memberships', () => {
  assert.deepEqual(resolveMemberEntry(activeMember, now), { state: 'active', text: '2026-08-01 到期' })
  assert.deepEqual(resolveMemberEntry({ is_active: true, member_expires_at: '2026-06-20 12:00:00' }, now), { state: 'expired', text: '已到期' })
  assert.deepEqual(resolveMemberEntry({}, now), { state: 'none', text: '' })
  assert.deepEqual(resolveMemberEntry(null, now), { state: 'none', text: '' })
})

test('reputation chip names the dimension and flags a ban', () => {
  assert.deepEqual(resolveReputationChip({ status: 'good', status_text: '良好' }), { text: '信誉良好', restricted: false })
  assert.deepEqual(resolveReputationChip({ status: 'restricted', status_text: '禁赛' }), { text: '禁赛', restricted: true })
  assert.equal(resolveReputationChip(null, true).text, '信誉同步中')
  assert.equal(resolveReputationChip(null, false).text, '信誉暂不可用')
})
