import test from 'node:test'
import assert from 'node:assert/strict'

import {
  resolveCoreMetrics,
  resolveMemberEntry,
  resolveSelfMemberIdentity,
  resolveRankAvatarFramePath,
  resolveRankCardView,
  resolveRankGameTypeView,
  resolveReputationChip
} from '../utils/user-homepage.js'

const now = new Date('2026-07-20T12:00:00+08:00')
const activeMember = {
  is_active: true,
  member_expires_at: '2026-08-01 12:00:00'
}

test('resolveRankAvatarFramePath maps all six rank levels to unique CDN assets', () => {
  assert.deepEqual(
    [1, 2, 3, 4, 5, 6].map(resolveRankAvatarFramePath),
    [
      'https://cdn.zhuifen.cn/static/images/avatar-frames/frame_bronze.png',
      'https://cdn.zhuifen.cn/static/images/avatar-frames/frame_silver.png',
      'https://cdn.zhuifen.cn/static/images/avatar-frames/frame_gold.png',
      'https://cdn.zhuifen.cn/static/images/avatar-frames/frame_platinum.png',
      'https://cdn.zhuifen.cn/static/images/avatar-frames/frame_diamond.png',
      'https://cdn.zhuifen.cn/static/images/avatar-frames/frame_king.png'
    ]
  )
  assert.equal(resolveRankAvatarFramePath(0), '')
  assert.equal(resolveRankAvatarFramePath(7), '')
})

test('resolveSelfMemberIdentity shows the current rank frame and the growth tier badge for an active member', () => {
  const identity = resolveSelfMemberIdentity({
    isLoggedIn: true,
    memberStatus: { ...activeMember, growth_level: 3 },
    rankInfo: { level: 5 },
    now
  })
  assert.equal(identity.frameUrl, 'https://cdn.zhuifen.cn/static/images/avatar-frames/frame_diamond.png')
  assert.equal(identity.badgeText, 'VIP3')
  assert.equal(identity.tierLabel, '进阶')

  // 没有显式等级时按成长值换算，和会员中心一致。
  assert.equal(resolveSelfMemberIdentity({
    isLoggedIn: true,
    memberStatus: { ...activeMember, growth_points: 300 },
    rankInfo: { level: 1 },
    now
  }).badgeText, 'VIP5')
})

test('resolveSelfMemberIdentity hides the identity for guests, ordinary, expired, and failed member states', () => {
  const states = [
    null,
    { is_active: false, member_expires_at: '2026-08-01 12:00:00', growth_level: 4 },
    { is_active: true, member_expires_at: '2026-07-20 12:00:00', growth_level: 4 },
    { is_active: true, member_expires_at: 'invalid', growth_level: 4 }
  ]

  for (const memberStatus of states) {
    const identity = resolveSelfMemberIdentity({ isLoggedIn: true, memberStatus, rankInfo: { level: 6 }, now })
    assert.equal(identity.showBadge, false)
    assert.equal(identity.frameUrl, '')
  }
  const guest = resolveSelfMemberIdentity({ isLoggedIn: false, memberStatus: { ...activeMember, growth_level: 4 }, rankInfo: { level: 6 }, now })
  assert.equal(guest.showBadge, false)
  assert.equal(guest.frameUrl, '')
})

test('resolveSelfMemberIdentity waits for the real rank and hides stale switch data', () => {
  const member = { ...activeMember, growth_level: 2 }
  const stale = [
    { rankInfo: null, rankLoading: true },
    { rankInfo: { level: 5 }, rankLoading: true },
    { rankInfo: null, rankLoading: false }
  ]
  for (const state of stale) {
    const identity = resolveSelfMemberIdentity({ isLoggedIn: true, memberStatus: member, ...state, now })
    assert.equal(identity.frameUrl, '')
    // 段位没读到不影响会员标识本身。
    assert.equal(identity.badgeText, 'VIP2')
  }
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

const rankInfo = (wins, losses, score = 1000) => ({ total_wins: wins, total_losses: losses, rank_score: score })

test('resolveRankGameTypeView prefers the saved game type and keeps played ones switchable', () => {
  const view = resolveRankGameTypeView({
    preferredGameType: 1,
    rankInfoMap: { 1: rankInfo(0, 0), 2: rankInfo(30, 10), 3: rankInfo(0, 0), 4: rankInfo(1, 0) }
  })

  assert.equal(view.primaryGameType, 1)
  assert.equal(view.primaryLabel, '斯诺克')
  assert.deepEqual(view.tabs.map((item) => item.value), [2, 1, 4])
  assert.equal(view.needsPick, false)
})

test('resolveRankGameTypeView falls back to the most played game type, then rank score, then fixed order', () => {
  assert.equal(resolveRankGameTypeView({
    rankInfoMap: { 1: rankInfo(3, 2, 1800), 2: rankInfo(20, 25), 3: rankInfo(10, 5, 2400), 4: rankInfo(0, 0) }
  }).primaryGameType, 2)
  assert.equal(resolveRankGameTypeView({
    rankInfoMap: { 1: rankInfo(5, 5, 1300), 2: rankInfo(4, 6, 1200), 3: rankInfo(0, 0), 4: rankInfo(0, 0) }
  }).primaryGameType, 1)
  assert.equal(resolveRankGameTypeView({
    rankInfoMap: { 1: rankInfo(5, 5, 1200), 2: rankInfo(4, 6, 1200), 3: rankInfo(0, 0), 4: rankInfo(0, 0) }
  }).primaryGameType, 2)
})

test('resolveRankGameTypeView hides the switcher for single game type players', () => {
  const view = resolveRankGameTypeView({
    preferredGameType: 0,
    rankInfoMap: { 1: rankInfo(0, 0), 2: rankInfo(12, 3), 3: rankInfo(0, 0), 4: rankInfo(0, 0) }
  })

  assert.equal(view.primaryGameType, 2)
  assert.deepEqual(view.tabs, [])
  assert.equal(view.needsPick, false)
})

test('resolveRankGameTypeView asks new players to pick only when nothing is saved or played', () => {
  const fresh = resolveRankGameTypeView({ rankInfoMap: { 1: rankInfo(0, 0), 2: rankInfo(0, 0), 3: rankInfo(0, 0), 4: rankInfo(0, 0) } })
  assert.equal(fresh.primaryGameType, 3)
  assert.deepEqual(fresh.tabs, [])
  assert.equal(fresh.needsPick, true)

  assert.equal(resolveRankGameTypeView({ preferredGameType: 4, rankInfoMap: {} }).needsPick, false)
  assert.equal(resolveRankGameTypeView({ preferredGameType: 99, rankInfoMap: null }).primaryGameType, 3)
})

test('rank card icon is resolved to the CDN, including the unranked default', () => {
  assert.equal(
    resolveRankCardView({ level: 3, name: '黄金', icon: '/static/images/ranks/rank_gold.png', rank_score: 1320 }).icon,
    'https://cdn.zhuifen.cn/static/images/ranks/rank_gold.png'
  )
  assert.equal(resolveRankCardView(null).icon, 'https://cdn.zhuifen.cn/static/images/ranks/rank_bronze.png')
})
