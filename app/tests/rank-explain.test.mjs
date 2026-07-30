import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildRankGrowthPath,
  buildRankHeroModel,
  RANK_RULE_SUMMARY_STATS,
  RANK_RULE_DETAIL_ROWS,
  RANK_RULE_EXAMPLE,
  resolveRankExplainLoadingMode,
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

test('scoring rule copy locks current settlement key numbers', () => {
  const summaryText = RANK_RULE_SUMMARY_STATS.map((item) => `${item.value} ${item.label}`).join(' ')
  assert.match(summaryText, /\+8～40/)
  assert.match(summaryText, /\+500/)
  assert.match(summaryText, /败方基础扣分/)

  const detailText = RANK_RULE_DETAIL_ROWS.map((item) => `${item.title} ${item.description}`).join(' ')
  assert.match(detailText, /\+8/)
  assert.match(detailText, /\+40/)
  assert.match(detailText, /100%/)
  assert.match(detailText, /80%/)
  assert.match(detailText, /30%/)
  assert.match(detailText, /第 7 场起不计/)
  assert.match(detailText, /-2/)
  assert.match(detailText, /500/)
  assert.match(detailText, /对手剩余排位分/)

  assert.match(RANK_RULE_EXAMPLE.title, /5 局制/)
  assert.match(RANK_RULE_EXAMPLE.formula, /\+20/)
  assert.match(RANK_RULE_EXAMPLE.formula, /80%/)
  assert.match(RANK_RULE_EXAMPLE.formula, /\+16/)
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
