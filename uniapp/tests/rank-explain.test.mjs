import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildRankGrowthPath,
  buildRankHeroModel,
  LEGACY_RANK_RULES,
  RATING_V2_RANK_RULES,
  resolveRankExplainLoadingMode,
  resolveRankRules,
  shouldApplyRankExplainResponse
} from '../utils/rank-explain.js'

const pageSource = readFileSync(
  new URL('../subPages/user/rankExplain.vue', import.meta.url),
  'utf8'
)
const styleSource = readFileSync(
  new URL('../subPages/user/rankExplain.scss', import.meta.url),
  'utf8'
)

test('resolveRankExplainLoadingMode only uses full-page loading before first successful render', () => {
  assert.equal(resolveRankExplainLoadingMode({
    hasLoadedOnce: false,
    isFetching: true
  }), 'initial')

  assert.equal(resolveRankExplainLoadingMode({
    hasLoadedOnce: true,
    isFetching: true
  }), 'refreshing')

  assert.equal(resolveRankExplainLoadingMode({
    hasLoadedOnce: true,
    isFetching: false
  }), 'idle')
})

test('shouldApplyRankExplainResponse ignores stale tab-switch responses', () => {
  assert.equal(shouldApplyRankExplainResponse({
    requestId: 2,
    latestRequestId: 3
  }), false)

  assert.equal(shouldApplyRankExplainResponse({
    requestId: 3,
    latestRequestId: 3
  }), true)
})

test('rank explain treats level six as the only max rank', () => {
  assert.match(pageSource, /rankInfo\.level < 6/)
  assert.doesNotMatch(pageSource, /rankInfo\.level < 5/)
})

const RANK_LIST_FIXTURE = [
  { level: 1, name: '青铜球手', icon: '/static/images/ranks/rank_bronze.png', min_score: 0, is_current: false },
  { level: 2, name: '白银球手', icon: '/static/images/ranks/rank_silver.png', min_score: 500, is_current: false },
  { level: 3, name: '黄金球手', icon: '/static/images/ranks/rank_gold.png', min_score: 1000, is_current: true },
  { level: 4, name: '铂金大师', icon: '/static/images/ranks/rank_platinum.png', min_score: 1500, is_current: false },
  { level: 5, name: '钻石', icon: '/static/images/ranks/rank_diamond.png', min_score: 2000, is_current: false },
  { level: 6, name: '王者', icon: '/static/images/ranks/rank_king.png', min_score: 2500, is_current: false }
]

test('buildRankGrowthPath returns six ranks from king to bronze with adjacent score ranges', () => {
  const path = buildRankGrowthPath({ rankList: RANK_LIST_FIXTURE, currentLevel: 3, rankScore: 1280 })

  assert.equal(path.length, 6)
  assert.deepEqual(path.map((item) => item.level), [6, 5, 4, 3, 2, 1])
  assert.deepEqual(path.map((item) => item.rangeText), [
    '2500 分以上',
    '2000–2499 分',
    '1500–1999 分',
    '1000–1499 分',
    '500–999 分',
    '0–499 分'
  ])
})

test('buildRankGrowthPath marks current, reached and pending states for gold user', () => {
  const path = buildRankGrowthPath({ rankList: RANK_LIST_FIXTURE, currentLevel: 3, rankScore: 1280 })
  const stateByLevel = Object.fromEntries(path.map((item) => [item.level, item.state]))

  assert.equal(stateByLevel[3], 'current')
  assert.equal(stateByLevel[1], 'reached')
  assert.equal(stateByLevel[2], 'reached')
  assert.equal(stateByLevel[4], 'pending')
  assert.equal(stateByLevel[5], 'pending')
  assert.equal(stateByLevel[6], 'pending')
  assert.equal(path.filter((item) => item.state === 'current').length, 1)
  assert.equal(path.find((item) => item.level === 3).stateText, '当前')
  assert.equal(path.find((item) => item.level === 1).stateText, '已达成')
  assert.equal(path.find((item) => item.level === 6).stateText, '待晋级')
})

test('buildRankHeroModel computes in-level progress and remaining score', () => {
  const hero = buildRankHeroModel({
    rankInfo: {
      level: 3,
      name: '黄金球手',
      icon: '/static/images/ranks/rank_gold.png',
      rank_score: 1280,
      next_level: 4,
      next_name: '铂金大师',
      next_score: 1500,
      progress: 56
    },
    rankList: RANK_LIST_FIXTURE
  })

  assert.equal(hero.isMax, false)
  assert.equal(hero.nextName, '铂金大师')
  assert.equal(hero.earned, 280)
  assert.equal(hero.span, 500)
  assert.equal(hero.remaining, 220)
  assert.equal(hero.progressPercent, 56)
})

test('buildRankHeroModel returns max rank state for king without next rank', () => {
  const hero = buildRankHeroModel({
    rankInfo: {
      level: 6,
      name: '王者',
      icon: '/static/images/ranks/rank_king.png',
      rank_score: 2780,
      next_level: 6,
      next_name: '王者',
      next_score: 2500,
      progress: 100
    },
    rankList: RANK_LIST_FIXTURE
  })

  assert.equal(hero.isMax, true)
  assert.equal(hero.progressPercent, 100)
  assert.equal(hero.remaining, undefined)
  assert.equal(hero.nextName, undefined)
})

test('buildRankHeroModel returns null when rank info is missing', () => {
  assert.equal(buildRankHeroModel({ rankInfo: null, rankList: RANK_LIST_FIXTURE }), null)
})

const ruleText = (rules) => [
  ...rules.summaryStats.map((item) => `${item.value} ${item.label}`),
  rules.note,
  ...rules.detailRows.map((item) => `${item.title} ${item.description}`),
  rules.example.title,
  rules.example.formula
].join(' ')

test('legacy scoring copy matches the legacy settlement without the nonexistent loss relief', () => {
  const text = ruleText(LEGACY_RANK_RULES)
  assert.match(text, /\+8～40/)
  assert.match(text, /\+500/)
  assert.match(text, /败方基础扣分/)
  assert.match(text, /第 4～6 场按 30%/)
  assert.match(text, /第 7 场起不计/)
  assert.match(text, /最低 \+2/)
  assert.match(text, /单日最多 200 分；输球不计/)
  assert.match(LEGACY_RANK_RULES.example.formula, /\+20/)
  assert.match(LEGACY_RANK_RULES.example.formula, /\+16/)
  assert.doesNotMatch(text, /失败减免/)
  assert.doesNotMatch(text, /减免到 -2/)
})

test('rating v2 scoring copy locks the shared formula examples', () => {
  const text = ruleText(RATING_V2_RANK_RULES)
  assert.match(text, /±45/)
  assert.match(text, /\+68/)
  assert.match(text, /只扣 21/)
  assert.match(text, /抢 1 为 ±20，抢 3 为 ±35，抢 5 为 ±45，抢 9 为 ±60/)
  assert.match(text, /5:0 与 5:4 分值相同/)
  assert.match(text, /前 10 场为定级赛，涨跌加倍/)
  assert.match(text, /涨跌减半/)
  assert.match(text, /第 3 场按 80%，第 4～6 场按 30%，双方同比例/)
  assert.match(text, /第 7 场起不计/)
  assert.match(text, /每天最多上涨 500/)
  assert.match(text, /不改变排位分，会员与普通用户规则相同/)
  assert.match(text, /打满 5 场、遇到至少 3 位不同对手/)
  assert.match(text, /不做重置/)
  assert.doesNotMatch(text, /VIP|倍率/)
})

test('resolveRankRules switches by effective time even when cached configs still say legacy', () => {
  const effectiveAt = '2026-11-01T00:00:00+08:00'
  const scheduled = resolveRankRules({ rulesVersion: 1, rulesEffectiveAt: effectiveAt, now: Date.parse('2026-10-31T23:59:59+08:00') })
  assert.equal(scheduled.version, 1)
  assert.match(scheduled.notice, /2026年11月1日起生效/)
  assert.match(scheduled.notice, /重算你的历史排位/)

  const active = resolveRankRules({ rulesVersion: 1, rulesEffectiveAt: effectiveAt, now: Date.parse('2026-11-01T00:00:00+08:00') })
  assert.equal(active.version, 2)
  assert.match(active.notice, /已于2026年11月1日生效/)

  const unconfigured = resolveRankRules({ rulesVersion: 1, rulesEffectiveAt: '' })
  assert.equal(unconfigured.version, 1)
  assert.equal(unconfigured.notice, '')
  assert.equal(resolveRankRules({ rulesVersion: 2 }).version, 2)
})

test('buildRankHeroModel exposes remaining placement matches', () => {
  const hero = buildRankHeroModel({
    rankInfo: { level: 3, name: '黄金球手', rank_score: 1250, next_score: 1500, placement_remaining: 7 },
    rankList: RANK_LIST_FIXTURE
  })
  assert.equal(hero.placementRemaining, 7)
  assert.equal(buildRankHeroModel({ rankInfo: { level: 3, rank_score: 1300 }, rankList: RANK_LIST_FIXTURE }).placementRemaining, 0)
})

test('rank explain page renders rules from configs and shows placement', () => {
  assert.match(pageSource, /resolveRankRules\(rulesMeta\.value\)/)
  assert.match(pageSource, /rulesEffectiveAt: listRes\.rules_effective_at/)
  assert.match(pageSource, /v-if="rankRules\.notice"/)
  assert.match(pageSource, /heroModel\.placementRemaining > 0/)
  assert.match(styleSource, /\.rule-notice/)
  assert.match(styleSource, /\.placement-pill/)
})

test('rank explain uses explicit initial error and retry instead of default bronze data', () => {
  assert.match(pageSource, /const rankInfo = ref\(null\)/)
  assert.match(pageSource, /v-else-if="isInitialError"/)
  assert.match(pageSource, /class="retry-button"/)
  assert.match(pageSource, /@tap="handleRetry"/)
  assert.match(pageSource, /段位数据加载失败/)
  assert.doesNotMatch(pageSource, /name:\s*'青铜球手'/)
})

test('rank explain keeps selected and loaded game type data consistent after refresh failure', () => {
  assert.match(pageSource, /gameTypeTabs\.some\(\(item\) => item\.value === gameType\)/)
  assert.match(pageSource, /const loadedGameType = ref\(3\)/)
  assert.match(pageSource, /const requestedGameType = currentGameType\.value/)
  assert.match(pageSource, /loadedGameType\.value = requestedGameType/)
  assert.match(pageSource, /currentGameType\.value = loadedGameType\.value/)
})

test('rank explain combines rankStore data with static rank configs', () => {
  assert.match(pageSource, /getRankConfigs/)
  assert.match(pageSource, /loadStatic\('rank-configs', getRankConfigs/)
  assert.doesNotMatch(pageSource, /getRankList\(/)
  assert.match(pageSource, /rankStore\.ensureFresh\(getReadIdentity\(\)\)/)
})

test('rank explain page removes old accordion and stale score rule copy', () => {
  assert.doesNotMatch(pageSource, /expandedLevel/)
  assert.doesNotMatch(pageSource, /toggleExpand/)
  assert.doesNotMatch(pageSource, /胜利基础分：\+20/)
  assert.doesNotMatch(pageSource, /失败基础分：-10/)
  assert.doesNotMatch(pageSource, /每天最多上涨 300/)
  assert.match(pageSource, /class="rank-path"/)
  assert.match(pageSource, /class="step-state"/)
  assert.match(pageSource, /查看完整计分规则/)
})

test('rank explain styles use mini-program-safe selectors and reset custom buttons', () => {
  assert.doesNotMatch(styleSource, /(^|[,{]\s*)\*(?=[\s,{.:#])/m)
  assert.doesNotMatch(styleSource, /:disabled/)
  assert.match(styleSource, /\.retry-button\s*\{[\s\S]*?margin:\s*30rpx 0 0;/)
  assert.match(styleSource, /\.rule-toggle\s*\{[\s\S]*?margin:\s*20rpx 0 0;/)
  assert.match(styleSource, /\.retry-button::after,[\s\S]*?\.rule-toggle::after/)
  assert.match(styleSource, /\.rank-step\.is-current[\s\S]*?\.step-state/)
  assert.match(styleSource, /\.dark-mode/)
})
