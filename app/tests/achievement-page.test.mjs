import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  ACHIEVEMENT_CATEGORY_GROUPS,
  getAchievementCategoryEmoji,
  getAchievementCategoryLabel,
  groupAchievementsByCategory,
  getTitleSourceClass,
  getTitleSourceDescription,
  getTitleSourceLabel,
  resolveTitleSelection,
  sortTitleOptions
} from '../utils/achievement-page.js'

const achievementIndexSource = readFileSync(
  new URL('../subPages/achievement/index.vue', import.meta.url),
  'utf8'
)
const achievementDetailSource = readFileSync(
  new URL('../subPages/achievement/detail.vue', import.meta.url),
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

test('title options put the equipped title first without disturbing the remaining API order', () => {
  const input = [
    { id: 1, title_name: '胜场新星', equipped: false },
    { id: 2, title_name: 'S3 冠军', equipped: true },
    { id: 3, title_name: '城市杯四强', equipped: false }
  ]

  assert.deepEqual(sortTitleOptions(input).map(item => item.id), [2, 1, 3])
  assert.deepEqual(input.map(item => item.id), [1, 2, 3])
})

test('title source descriptions use the source type and readable origin name', () => {
  assert.equal(
    getTitleSourceDescription({ source_type: 'achievement', source_ref_name: '五连胜成就' }),
    '生涯成就 · 五连胜成就'
  )
  assert.equal(
    getTitleSourceDescription({ source_type: 'season', source_ref_name: 'S3' }),
    '赛季荣誉 · S3'
  )
  assert.equal(getTitleSourceDescription({ source_type: 'tournament' }), '赛事荣誉')
})

test('title selection resolves no-op, equip and unequip actions without confirmation state', () => {
  assert.deepEqual(resolveTitleSelection({ currentTitleId: 2, selectedTitleId: 2 }), { action: 'close' })
  assert.deepEqual(resolveTitleSelection({ currentTitleId: 2, selectedTitleId: 3 }), { action: 'equip', titleId: 3 })
  assert.deepEqual(resolveTitleSelection({ currentTitleId: 2, selectedTitleId: 0 }), { action: 'unequip', titleId: 2 })
  assert.deepEqual(resolveTitleSelection({ currentTitleId: 0, selectedTitleId: 0 }), { action: 'close' })
})

test('achievement index is upgraded to the three-track honor wall', () => {
  assert.match(achievementIndexSource, /achievementGroups/)
  assert.match(achievementIndexSource, /honor-hero-card/)
  assert.match(achievementIndexSource, /honor-tabs/)
  assert.match(achievementIndexSource, /achievement-row/)
  assert.match(achievementIndexSource, /生涯成就/)
  assert.match(achievementIndexSource, /当前赛季/)
  assert.match(achievementIndexSource, /历届荣誉/)
  assert.doesNotMatch(achievementIndexSource, /精选展示|分享荣誉墙/)
})

test('achievement pages use shared mapping and refresh the honor wall on show', () => {
  assert.match(achievementIndexSource, /groupAchievementsByCategory/)
  assert.match(achievementIndexSource, /getHonorWall/)
  assert.match(achievementIndexSource, /onShow/)
  assert.match(achievementDetailSource, /getAchievementCategoryLabel/)
})

test('honor wall owns title source copy and direct selection actions', () => {
  assert.match(achievementIndexSource, /getTitleSourceDescription\(item\)/)
  assert.match(achievementIndexSource, /resolveTitleSelection/)
  assert.match(achievementIndexSource, /佩戴/)
  assert.match(achievementIndexSource, /不佩戴称号/)
  assert.doesNotMatch(achievementIndexSource, /前往称号管理|装备称号|卸下称号/)
})

test('match result empty copy uses special-record wording', () => {
  assert.match(matchResultSource, /暂无特殊战绩/)
  assert.doesNotMatch(matchResultSource, /暂无特殊成就/)
})
