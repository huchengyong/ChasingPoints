import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import { resolveMatchRankingRightsSummary } from '../utils/member-ranking-rights.js'
import {
  buildMatchRewardHonorWallUrl,
  MATCH_REWARD_MAX_ATTEMPTS,
  MATCH_REWARD_RETRY_DELAY_MS,
  resolveMatchRewardSummary
} from '../utils/match-result-page.js'

const matchApiSource = readFileSync(new URL('../api/match.js', import.meta.url), 'utf8')
const matchResultSource = readFileSync(new URL('../subPages/match/matchResult.vue', import.meta.url), 'utf8')

test('resolveMatchRankingRightsSummary marks highlights as recorded only when no rank detail was counted', () => {
  const summary = resolveMatchRankingRightsSummary({
    achievements: {
      break_clear: 1,
      break_100: 1
    },
    my_rank_details: [
      { label: '基础分', value: 20 }
    ]
  })

  assert.equal(summary.countedInRanking, false)
  assert.equal(summary.badgeText, '高光仅记录')
  assert.match(summary.description, /普通用户高光会保留记录，会员用户才会按等级倍率计入/)
})

test('resolveMatchRankingRightsSummary explains counted member highlights and daily cap clipping', () => {
  const summary = resolveMatchRankingRightsSummary({
    achievements: {
      break_147: 1
    },
    my_rank_details: [
      { label: '基础分', value: 20 },
      { label: '会员特殊战绩分', value: 30 },
      { label: '每日封顶', value: -6 }
    ]
  })

  assert.equal(summary.countedInRanking, true)
  assert.equal(summary.capped, true)
  assert.equal(summary.badgeText, '已计入并触发封顶')
  assert.match(summary.description, /30 分特殊战绩已计入排位/)
  assert.match(summary.description, /6 分受到会员特殊战绩单日上限影响/)
})

test('resolveMatchRankingRightsSummary falls back to base-rank explanation without achievements', () => {
  const summary = resolveMatchRankingRightsSummary({
    achievements: {},
    my_rank_details: [{ label: '基础分', value: 20 }]
  })

  assert.equal(summary.countedInRanking, false)
  assert.equal(summary.recordedAchievementsText, '')
  assert.equal(summary.title, '本场没有可计入排位的特殊战绩')
})

test('resolveMatchRewardSummary maps ready rewards into achievement and title view models', () => {
  const summary = resolveMatchRewardSummary({
    success: true,
    status: 'ready',
    list: [{
      achievement_id: 12,
      achievement_key: 'first_win',
      achievement_name: '首胜',
      description: '赢得第一场有效排位',
      icon: '/static/achievement/first-win.png',
      reward_title_id: 7,
      reward_title_name: '初露锋芒',
      unlocked_at: '2026-07-23 20:00:00'
    }]
  })

  assert.equal(summary.state, 'ready')
  assert.equal(summary.visible, true)
  assert.equal(summary.rewards.length, 1)
  assert.equal(summary.rewards[0].achievementName, '首胜')
  assert.equal(summary.rewards[0].rewardTitleName, '初露锋芒')
  assert.equal(summary.shouldRetry, false)
})

test('resolveMatchRewardSummary hides a ready empty summary', () => {
  const summary = resolveMatchRewardSummary({ success: true, status: 'ready', list: [] })

  assert.equal(summary.state, 'empty')
  assert.equal(summary.visible, false)
  assert.deepEqual(summary.rewards, [])
})

test('resolveMatchRewardSummary keeps pending distinct from empty and limits retries', () => {
  const retrying = resolveMatchRewardSummary(
    { success: true, status: 'pending', message: '正在结算', list: [] },
    { attempt: 1, maxAttempts: 3 }
  )
  const exhausted = resolveMatchRewardSummary(
    { success: true, status: 'pending', list: [] },
    { attempt: 3, maxAttempts: 3 }
  )

  assert.equal(retrying.state, 'pending')
  assert.equal(retrying.visible, true)
  assert.equal(retrying.message, '正在结算')
  assert.equal(retrying.shouldRetry, true)
  assert.equal(exhausted.state, 'pending')
  assert.equal(exhausted.shouldRetry, false)
})

test('resolveMatchRewardSummary hides the module when the API fails', () => {
  const summary = resolveMatchRewardSummary({
    success: false,
    message: '奖励数据暂不可用'
  })

  assert.equal(summary.state, 'error')
  assert.equal(summary.visible, false)
  assert.equal(summary.message, '奖励数据暂不可用')
  assert.equal(summary.shouldRetry, false)
})

test('match reward honor wall url carries the match game type', () => {
  assert.equal(
    buildMatchRewardHonorWallUrl(4),
    '/subPages/achievement/index?game_type=4&tab=career'
  )
  assert.equal(
    buildMatchRewardHonorWallUrl('unsupported'),
    '/subPages/achievement/index?game_type=3&tab=career'
  )
})

test('match result page queries the current user summary by match id and renders rewards before actions', () => {
  assert.match(matchApiSource, /getMatchRewardSummary/)
  assert.match(matchApiSource, /\/api\/match\/reward-summary/)
  assert.match(matchResultSource, /getMatchRewardSummary\(\{ match_id: matchId\.value \}\)/)
  assert.doesNotMatch(matchResultSource, /getMatchRewardSummary\(\{[^}]*user_id/)
  assert.match(matchResultSource, /v-if="rewardSummary\.visible"/)
  assert.match(matchResultSource, /rewardSummary\.state === 'pending'/)
  assert.match(matchResultSource, /MATCH_REWARD_MAX_ATTEMPTS/)
  assert.match(matchResultSource, /MATCH_REWARD_RETRY_DELAY_MS/)
  assert.match(matchResultSource, /buildMatchRewardHonorWallUrl\(matchData\.value\.game_type\)/)
  assert.ok(matchResultSource.indexOf('reward-section') < matchResultSource.indexOf('class="footer"'))
  assert.equal(MATCH_REWARD_MAX_ATTEMPTS, 3)
  assert.equal(MATCH_REWARD_RETRY_DELAY_MS, 800)
})
