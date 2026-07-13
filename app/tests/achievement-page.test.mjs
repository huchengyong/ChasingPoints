import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  ACHIEVEMENT_CATEGORY_GROUPS,
  getAchievementCategoryEmoji,
  getAchievementCategoryLabel,
  groupAchievementsByCategory,
  getTitleSourceClass,
  getTitleSourceLabel
} from '../utils/achievement-page.js'

const achievementIndexSource = readFileSync(
  new URL('../subPages/achievement/index.vue', import.meta.url),
  'utf8'
)
const achievementDetailSource = readFileSync(
  new URL('../subPages/achievement/detail.vue', import.meta.url),
  'utf8'
)
const titleSource = readFileSync(
  new URL('../subPages/achievement/titles.vue', import.meta.url),
  'utf8'
)
const matchResultSource = readFileSync(
  new URL('../subPages/match/matchResult.vue', import.meta.url),
  'utf8'
)

test('achievement category groups use backend codes and Chinese display labels', () => {
  assert.deepEqual(
    ACHIEVEMENT_CATEGORY_GROUPS.map(group => group.key),
    ['wins', 'streak', 'special', 'match', 'tournament']
  )
  assert.equal(getAchievementCategoryLabel('wins'), '胜场')
  assert.equal(getAchievementCategoryLabel('streak'), '连胜')
  assert.equal(getAchievementCategoryLabel('special'), '特殊')
  assert.equal(getAchievementCategoryLabel('match'), '对局')
  assert.equal(getAchievementCategoryLabel('tournament'), '赛事')
  assert.equal(getAchievementCategoryEmoji('wins'), '🏅')
})

test('achievement list groups by backend category code and hides empty groups', () => {
  const list = [
    { id: 1, category: 'wins' },
    { id: 2, category: 'streak' },
    { id: 3, category: 'wins' },
    { id: 4, category: 'tournament' }
  ]
  const groups = groupAchievementsByCategory(list)

  assert.deepEqual(
    groups.map(group => ({ key: group.key, count: group.list.length })),
    [
      { key: 'wins', count: 2 },
      { key: 'streak', count: 1 },
      { key: 'tournament', count: 1 }
    ]
  )
  assert.deepEqual(groups[0].list.map(item => item.id), [1, 3])
})

test('title sources display Chinese labels while keeping stable style classes', () => {
  assert.equal(getTitleSourceLabel('achievement'), '成就')
  assert.equal(getTitleSourceLabel('season'), '赛季')
  assert.equal(getTitleSourceLabel('tournament'), '赛事')
  assert.equal(getTitleSourceClass('achievement'), 'achievement')
  assert.equal(getTitleSourceClass('赛季'), 'season')
})

test('achievement index uses grouped full-row cards instead of category tabs', () => {
  assert.match(achievementIndexSource, /achievementGroups/)
  assert.match(achievementIndexSource, /achievement-section/)
  assert.match(achievementIndexSource, /achievement-row/)
  assert.doesNotMatch(achievementIndexSource, /category-tabs/)
  assert.doesNotMatch(achievementIndexSource, /switchCategory/)
})

test('achievement pages use shared mapping and refresh equipped title on show', () => {
  assert.match(achievementIndexSource, /groupAchievementsByCategory/)
  assert.match(achievementIndexSource, /onShow/)
  assert.match(achievementDetailSource, /getAchievementCategoryLabel/)
})

test('titles page normalizes title source labels', () => {
  assert.match(titleSource, /getTitleSourceLabel\(item\.source\)/)
  assert.match(titleSource, /getTitleSourceClass\(item\.source\)/)
})

test('match result empty copy uses special-record wording', () => {
  assert.match(matchResultSource, /暂无特殊战绩/)
  assert.doesNotMatch(matchResultSource, /暂无特殊成就/)
})
