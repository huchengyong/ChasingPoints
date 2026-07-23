import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildChallengeViewModel,
  buildHistorySeasonOptions,
  buildHonorWallTabs,
  buildHonorWallUrl,
  buildRecentHonorViewModel,
  buildSeasonRolloverModal,
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

test('honor wall page keeps private season progress out of friend tabs and preserves existing detail/title routes', () => {
  assert.match(honorWallSource, /buildHonorWallTabs\(wall\.value\.viewer_scope\)/)
  assert.match(honorWallSource, /if \(!isSelf\.value \|\| !id\) return/)
  assert.match(honorWallSource, /\/subPages\/achievement\/detail\?id=/)
  assert.match(honorWallSource, /\/subPages\/achievement\/titles/)
  assert.match(honorWallSource, /presentLatestSeasonRollover/)
  assert.doesNotMatch(honorWallSource, /分享荣誉墙|精选展示|手动精选/)
})

test('honor wall layout protects small screens and long copy from horizontal overflow', () => {
  assert.match(honorWallStyle, /min-width:\s*0/)
  assert.match(honorWallStyle, /word-break:\s*break-word/)
  assert.match(honorWallStyle, /flex-wrap:\s*wrap/)
  assert.match(honorWallStyle, /max-width:\s*360px/)
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

test('achievement API exposes the honor wall through the request facade', () => {
  assert.match(achievementApiSource, /getHonorWall/)
  assert.match(achievementApiSource, /\/api\/achievement\/honor-wall/)
})
