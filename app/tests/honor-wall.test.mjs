import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildCareerSummaryItems,
  buildChallengeViewModel,
  buildHistorySeasonOptions,
  buildHonorWallTabs,
  buildHonorWallUrl,
  buildRecentHonorViewModel,
  buildSeasonRolloverModal,
  buildUpcomingAchievementSection,
  createLatestRequestGuard,
  findLatestUnreadSeasonRollover,
  normalizeHonorWallOptions,
  presentLatestSeasonRollover,
  resolveCurrentSeasonState
} from '../utils/honor-wall.js'

const honorWallSource = readFileSync(new URL('../subPages/achievement/index.vue', import.meta.url), 'utf8')
const honorWallStyle = readFileSync(new URL('../subPages/achievement/index.scss', import.meta.url), 'utf8')
const userTabSource = readFileSync(new URL('../pages/user/index.vue', import.meta.url), 'utf8')
const friendHomepageSource = readFileSync(new URL('../subPages/social/friendHomepage.vue', import.meta.url), 'utf8')
const notificationSource = readFileSync(new URL('../subPages/notification/index.vue', import.meta.url), 'utf8')
const achievementApiSource = readFileSync(new URL('../api/achievement.js', import.meta.url), 'utf8')
const pagesJsonSource = readFileSync(new URL('../pages.json', import.meta.url), 'utf8')

test('honor wall options and tabs keep self progress private from friend views', () => {
  assert.deepEqual(normalizeHonorWallOptions({ user_id: '18', game_type: '4', tab: 'season' }), {
    userId: 18,
    gameType: 4,
    activeTab: 'season',
    historySeasonId: 0
  })
  assert.deepEqual(buildHonorWallTabs('self').map(item => item.key), ['career', 'season', 'history'])
  assert.deepEqual(buildHonorWallTabs('friend').map(item => item.key), ['career', 'history'])
})

test('career summary separates universal and selected specialty progress', () => {
  assert.deepEqual(buildCareerSummaryItems({
    universal_unlocked: 8,
    universal_total: 15,
    specialty_game_type: 3,
    specialty_unlocked: 2,
    specialty_total: 3,
    season_honors: 4,
    tournament_honors: 5
  }, 3, '中式八球'), [
    { key: 'universal', label: '通用成就', value: '8/15' },
    { key: 'specialty', label: '中式八球专精', value: '2/3' },
    { key: 'season', label: '赛季荣誉', value: 4 },
    { key: 'tournament', label: '赛事荣誉', value: 5 }
  ])
})

test('upcoming achievements use universal plus selected game type with stable ranking', () => {
  const achievements = [
    { id: 1, game_type: 0, progress: 8, threshold: 10, unlocked: false },
    { id: 2, game_type: 3, progress: 4, threshold: 5, unlocked: false },
    { id: 3, game_type: 3, progress: 0, threshold: 1, unlocked: false },
    { id: 4, game_type: 1, progress: 99, threshold: 100, unlocked: false },
    { id: 5, game_type: 0, progress: 1, threshold: 1, unlocked: true },
    { id: 6, game_type: 0, progress: 2, threshold: 10, unlocked: false }
  ]
  const section = buildUpcomingAchievementSection(achievements, 3, 'self')
  assert.equal(section.hidden, false)
  assert.equal(section.completed, false)
  assert.deepEqual(section.items.map(item => item.id), [1, 2, 6])
  assert.equal(section.items[0].progressPercent, 80)
  assert.equal(section.items[2].remainingText, '还差 8 达成')

  const zeroFallback = buildUpcomingAchievementSection([
    { id: 10, game_type: 3, progress: 0, threshold: 1, unlocked: false },
    { id: 11, game_type: 0, progress: 0, threshold: 10, unlocked: false }
  ], 3, 'self')
  assert.deepEqual(zeroFallback.items.map(item => item.id), [10, 11])

  const complete = buildUpcomingAchievementSection([
    { id: 20, game_type: 0, progress: 1, threshold: 1, unlocked: true },
    { id: 21, game_type: 3, progress: 1, threshold: 1, unlocked: true },
    { id: 22, game_type: 4, progress: 0, threshold: 1, unlocked: false }
  ], 3, 'self')
  assert.equal(complete.completed, true)
  assert.deepEqual(complete.items, [])

  const invalidData = buildUpcomingAchievementSection([
    null,
    { id: 30, game_type: 3, progress: -5, threshold: 1, unlocked: false },
    { id: 31, game_type: 3, progress: 1, threshold: 0, unlocked: false },
    { id: 32, game_type: 4, progress: 1, threshold: 1, unlocked: false }
  ], 99, 'self')
  assert.deepEqual(invalidData.items.map(item => item.id), [30])
  assert.equal(invalidData.items[0].progress, 0)

  const friend = buildUpcomingAchievementSection(achievements, 3, 'friend')
  assert.equal(friend.hidden, true)
  assert.deepEqual(friend.items, [])
})

test('latest honor wall request wins when an older response arrives later', async () => {
  const guard = createLatestRequestGuard()
  const applied = []
  let resolveFirst
  let resolveSecond
  const firstResponse = new Promise(resolve => { resolveFirst = resolve })
  const secondResponse = new Promise(resolve => { resolveSecond = resolve })
  const applyResponse = async (label, response) => {
    const requestId = guard.next()
    await response
    if (guard.isLatest(requestId)) applied.push(label)
  }

  const firstTask = applyResponse('斯诺克', firstResponse)
  const secondTask = applyResponse('美式九球', secondResponse)
  resolveSecond()
  await secondTask
  resolveFirst()
  await firstTask

  assert.deepEqual(applied, ['美式九球'])
})

test('challenge view model exposes readable progress and completion copy', () => {
  assert.deepEqual(buildChallengeViewModel({ progress: 12, threshold: 20, completed: false }), {
    progress: 12,
    threshold: 20,
    completed: false,
    progressPercent: 60,
    progressText: '12/20',
    remainingText: '还差 8 完成'
  })
  assert.equal(buildChallengeViewModel({ progress: 23, threshold: 20 }).progressPercent, 100)
  assert.equal(buildChallengeViewModel({ progress: 20, threshold: 20 }).progressText, '已完成')
})

test('recent honors normalize achievement title pairing without duplicate display rows', () => {
  assert.deepEqual(buildRecentHonorViewModel({
    id: 1,
    source_type: 'achievement',
    name: '十胜起步',
    reward_title_name: '胜场新星'
  }), {
    id: 1,
    source_type: 'achievement',
    name: '十胜起步',
    reward_title_name: '胜场新星',
    sourceLabel: '生涯成就',
    detailText: '同时获得称号「胜场新星」'
  })
})

test('current season state distinguishes active season and intermission', () => {
  assert.equal(resolveCurrentSeasonState(null).mode, 'intermission')
  assert.equal(resolveCurrentSeasonState({ season_id: 3, season_name: 'S3' }).title, 'S3')
})

test('history season options dedupe permanent season honors', () => {
  assert.deepEqual(buildHistorySeasonOptions([
    { source_type: 'season', source_ref_id: 3, source_ref_name: 'S3' },
    { source_type: 'season', source_ref_id: 3, source_ref_name: 'S3' },
    { source_type: 'tournament', source_ref_id: 9, source_ref_name: '城市杯' }
  ]), [{ id: 3, label: 'S3' }])
})

test('rollover modal clearly explains automatic S3 to S4 transition', () => {
  const modal = buildSeasonRolloverModal({
    data: JSON.stringify({
      from_season_id: 3,
      from_season_name: 'S3',
      to_season_id: 4,
      to_season_name: 'S4',
      challenge_archived_count: 3,
      challenge_completed_count: 2,
      season_title_count: 1,
      intermission: false
    })
  })
  assert.equal(modal.title, 'S4 已开启')
  assert.match(modal.content, /S3 已归档/)
  assert.match(modal.content, /S4 已自动开启/)
  assert.match(modal.content, /生涯成就保持不变/)

  const intermission = buildSeasonRolloverModal({
    data: { from_season_name: 'S3', to_season_id: 0, intermission: true }
  })
  assert.equal(intermission.title, 'S3 已结算')
  assert.match(intermission.content, /赛季间歇期/)
  assert.doesNotMatch(intermission.content, /S4/)
})

test('latest unread rollover selection ignores read and unrelated notifications', () => {
  const latest = findLatestUnreadSeasonRollover([
    { id: 1, type: 'season_rollover', is_read: true, created_at: '2026-07-01' },
    { id: 2, type: 'match_result', is_read: false, created_at: '2026-08-01' },
    { id: 3, type: 'season_rollover', is_read: false, created_at: '2026-07-02' },
    { id: 4, type: 'season_rollover', is_read: false, created_at: '2026-07-03' }
  ])
  assert.equal(latest.id, 4)
})

test('presentLatestSeasonRollover shows once then marks the notification read', async () => {
  const calls = []
  const notification = await presentLatestSeasonRollover({
    getNotificationList: async (params) => {
      calls.push(['list', params])
      return { list: [{ id: 9, type: 'season_rollover', is_read: false, data: { from_season_name: 'S3', to_season_id: 0 } }] }
    },
    showModal: (options) => {
      calls.push(['modal', options.title])
      options.complete()
    },
    markAsRead: async (payload) => calls.push(['read', payload]),
    onRead: async () => calls.push(['onRead'])
  })
  assert.equal(notification.id, 9)
  assert.deepEqual(calls, [
    ['list', { page: 1, page_size: 10, type: 'season_rollover' }],
    ['modal', 'S3 已结算'],
    ['read', { notification_id: 9 }],
    ['onRead']
  ])
})

test('honor wall urls preserve friend and game type context', () => {
  assert.equal(
    buildHonorWallUrl({ userId: 18, gameType: 4, tab: 'history', historySeasonId: 3 }),
    '/subPages/achievement/index?game_type=4&tab=history&user_id=18&history_season_id=3'
  )
})

test('honor wall page keeps private season progress out of friend tabs and opens title selection inline for self', () => {
  assert.match(honorWallSource, /buildCareerSummaryItems/)
  assert.match(honorWallSource, /buildUpcomingAchievementSection/)
  assert.match(honorWallSource, /createLatestRequestGuard/)
  assert.match(honorWallSource, /if \(!wallRequestGuard\.isLatest\(requestId\)\) return/)
  assert.match(honorWallSource, /通用里程碑/)
  assert.match(honorWallSource, /球种绝技/)
  assert.match(honorWallSource, /buildHonorWallTabs\(wall\.value\.viewer_scope\)/)
  assert.match(honorWallSource, /if \(!isSelf\.value \|\| !id\) return/)
  assert.match(honorWallSource, /\/subPages\/achievement\/detail\?id=/)
  assert.match(honorWallSource, /@tap="openTitleSelector"/)
  assert.match(honorWallSource, /const openTitleSelector = async \(\) => \{\s*if \(!isSelf\.value\) return/)
  assert.match(honorWallSource, /v-if="isSelf && showTitleSelector"/)
  assert.match(honorWallSource, /await getUserTitles\(\)/)
  assert.match(honorWallSource, /presentLatestSeasonRollover/)
  assert.doesNotMatch(honorWallSource, /\/subPages\/achievement\/titles/)
  assert.doesNotMatch(honorWallSource, /分享荣誉墙|精选展示|手动精选/)
})

test('inline title selector covers loading, retry, empty, selection and system-back states without phase-one extras', () => {
  assert.match(honorWallSource, /title-selector-loading/)
  assert.match(honorWallSource, /重新加载称号/)
  assert.match(honorWallSource, /暂无可佩戴称号/)
  assert.match(honorWallSource, /不佩戴称号/)
  assert.match(honorWallSource, /@tap="selectTitle\(item\)"/)
  assert.match(honorWallSource, /@tap="selectNoTitle"/)
  assert.match(honorWallSource, /if \(titleListLoaded\.value\) \{/)
  assert.match(honorWallSource, /titleListFailed\.value = true/)
  assert.match(honorWallSource, /if \(titleSubmitting\.value\) return/)
  assert.match(honorWallSource, /equip:\s*selection\.action === 'equip'/)
  assert.match(honorWallSource, /onBackPress/)
  assert.doesNotMatch(honorWallSource, /搜索称号|来源筛选|确认佩戴|确认卸下|批量编辑|称号统计/)
})

test('honor wall layout protects small screens and long copy from horizontal overflow', () => {
  assert.match(honorWallStyle, /min-width:\s*0/)
  assert.match(honorWallStyle, /word-break:\s*break-word/)
  assert.match(honorWallStyle, /flex-wrap:\s*wrap/)
  assert.match(honorWallStyle, /max-width:\s*360px/)
  assert.match(honorWallStyle, /title-selector-list[\s\S]*max-height:/)
  assert.match(honorWallStyle, /env\(safe-area-inset-bottom\)/)
  assert.doesNotMatch(honorWallStyle, /white-space:\s*nowrap/)
})

test('user and friend entries carry honor wall ownership and game type context', () => {
  assert.match(userTabSource, /buildHonorWallUrl\(\{ gameType: currentRankGameType\.value \}\)/)
  assert.match(friendHomepageSource, /buildHonorWallUrl\(\{ userId: friendProfile\.id, gameType: 3 \}\)/)
  assert.match(friendHomepageSource, /查看荣誉墙/)
})

test('notification mapping recognizes season rollover and honors its url payload', () => {
  assert.match(notificationSource, /season_rollover:\s*'🔄'/)
  assert.match(notificationSource, /data\.target \|\| data\.url/)
  assert.match(userTabSource, /presentLatestSeasonRollover/)
})

test('new client removes the title-management route while keeping existing title API facades', () => {
  assert.doesNotMatch(pagesJsonSource, /"path"\s*:\s*"titles"/)
  assert.match(achievementApiSource, /getHonorWall/)
  assert.match(achievementApiSource, /\/api\/achievement\/honor-wall/)
  assert.match(achievementApiSource, /export const getUserTitles/)
  assert.match(achievementApiSource, /\/api\/achievement\/titles/)
  assert.match(achievementApiSource, /export const equipTitle/)
  assert.match(achievementApiSource, /\/api\/achievement\/title\/equip/)
})
