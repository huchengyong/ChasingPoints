import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import {
  ASYNC_PAGE_STATUS,
  beginAsyncPageLoad,
  createAsyncPageState,
  getAsyncPageRequest,
  rejectAsyncPageLoad,
  resolveAsyncPageErrorFeedback,
  resolveAsyncPageLoad
} from '../utils/async-page-state.js'
import { createRequestError } from '../utils/request-errors.js'
import {
  formatSeasonNetChange,
  mergeLeaderboardPage,
  resolveLeaderboardRank,
  resolveSeasonQualification
} from '../utils/season-qualification.js'

const source = readFileSync(new URL('../subPages/season/index.vue', import.meta.url), 'utf8')

const mountSeasonPageSetup = ({ getSeasonOverview }) => {
  const script = source.match(/<script setup>([\s\S]*?)<\/script>/)?.[1]
  assert.ok(script, 'season page must provide script setup')
  const setup = script.replace(/^import[\s\S]*?from\s+['"][^'"]+['"]\s*$/gm, '')
  const mounted = []
  const shown = []
  const userStore = { userId: 1, authGeneration: 1 }
  const ref = (value) => ({ value })
  const computed = (getter) => ({
    get value() {
      return getter()
    }
  })
  const component = new Function(
    'ref', 'computed', 'onMounted', 'onShow', 'getSeasonOverview', 'getSeasonLeaderboard',
    'useUserDataInvalidationStore', 'useUserStore', 'GAME_TYPE_TABS', 'resolveCurrentSeasonState',
    'resolveSeasonTimeline', 'formatSeasonNetChange', 'mergeLeaderboardPage', 'resolveLeaderboardRank',
    'resolveSeasonQualification', 'usePageTheme', 'resolveAvatarUrl',
    'ASYNC_PAGE_STATUS', 'beginAsyncPageLoad', 'createAsyncPageState', 'getAsyncPageRequest',
    'rejectAsyncPageLoad', 'resolveAsyncPageErrorFeedback', 'resolveAsyncPageLoad', 'createRequestError',
    `${setup}\nreturn { loading, hasLoadedOnce, loadSeasonOverview, seasonPageError, season, myRecord, leaderboard, leaderboardRows, hasMore, recordUnavailable, leaderboardUnavailable, myRank, myRankAvailable, qualification, netChangeText }`
  )(
    ref,
    computed,
    (callback) => mounted.push(callback),
    (callback) => shown.push(callback),
    getSeasonOverview,
    async () => ({ success: true, list: [], total: 0 }),
    () => ({ versionOf: () => 0 }),
    () => userStore,
    [],
    () => ({ title: '', description: '' }),
    () => ({ remainDays: 0, progressPercent: 0 }),
    formatSeasonNetChange,
    mergeLeaderboardPage,
    resolveLeaderboardRank,
    resolveSeasonQualification,
    () => ({ isDarkMode: ref(false) }),
    () => '',
    ASYNC_PAGE_STATUS,
    beginAsyncPageLoad,
    createAsyncPageState,
    getAsyncPageRequest,
    rejectAsyncPageLoad,
    resolveAsyncPageErrorFeedback,
    resolveAsyncPageLoad,
    createRequestError
  )
  return { component, mounted, shown }
}

test('season page presents lifecycle state and keeps legacy active responses compatible', () => {
  assert.match(source, /resolveCurrentSeasonState/)
  assert.match(source, /resolveSeasonTimeline/)
  assert.match(source, /const hasActiveSeason = computed\(\(\) => seasonState\.value === 'active' && Boolean\(season\.value\)\)/)
  assert.match(source, /<template v-if="hasActiveSeason">/)
  assert.match(source, /if \(!hasActiveSeason\.value\) return/)
  assert.match(source, /const seasonState = ref\('not_started'\)/)
  assert.match(source, /const nextSeasonState = res\.season_state \|\| \(nextSeason \? 'active' : 'not_started'\)/)
  assert.match(source, /seasonState\.value = nextSeasonState/)
  assert.match(source, /seasonPageState\.value = rejectAsyncPageLoad/)
  assert.match(source, /v-else-if="seasonPageError"/)
  assert.doesNotMatch(source, /seasonState\.value = 'unavailable'/)
  assert.match(source, /seasonEmptyState\.title/)
  assert.match(source, /seasonEmptyState\.description/)
  assert.doesNotMatch(source, /暂无进行中的赛季/)
  assert.doesNotMatch(source, /赛季间歇中/)
  assert.doesNotMatch(source, /new Date\(season\.value\.end_date\)/)
  assert.match(source, /getSeasonOverview\(\{ game_type: currentGameType\.value \}\)/)
  assert.doesNotMatch(source, /getCurrentSeason\(|getMySeasonRecord\(/)
  assert.match(source, /loadedSeasonScopeVersion\.value !== currentSeasonScopeVersion\(\)/)
})

test('season page setup mounts with one overview request on first entry', async () => {
  let overviewCalls = 0
  const { component, mounted, shown } = mountSeasonPageSetup({
    getSeasonOverview: async () => {
      overviewCalls += 1
      return {
        success: true,
        season: { id: 1 },
        season_state: 'active',
        availability: { record: true, leaderboard: true },
        record: {},
        leaderboard: []
      }
    }
  })

  assert.equal(component.loading.value, false)
  assert.equal(mounted.length, 1)
  assert.equal(shown.length, 1)
  await mounted[0]()
  await shown[0]()
  assert.equal(overviewCalls, 1)
  assert.equal(component.loading.value, false)
})

test('season page keeps the active season visible when optional overview blocks are unavailable', async () => {
  const { component, mounted } = mountSeasonPageSetup({
    getSeasonOverview: async () => ({
      success: true,
      season: { id: 2, name: 'S3', status: 1 },
      season_state: 'active',
      availability: { season: true, record: false, leaderboard: false, competitive_revision: false },
      record: { matches_played: 9 },
      leaderboard: [{ user_id: 1 }]
    })
  })

  await mounted[0]()
  assert.equal(component.seasonPageError.value, null)
  assert.equal(component.season.value.name, 'S3')
  assert.equal(component.recordUnavailable.value, true)
  assert.equal(component.leaderboardUnavailable.value, true)
  assert.equal(component.myRecord.value, null, 'unavailable record must not render as real data')
  assert.deepEqual(component.leaderboard.value, [], 'unavailable leaderboard must not render as real data')
  assert.equal(component.hasMore.value, false)
  assert.equal(component.hasLoadedOnce.value, true)
})

test('season page shows live rank from overview and keeps record when rank is unavailable', async () => {
  const responses = [
    {
      success: true,
      season: { id: 2, name: 'S3', status: 1 },
      season_state: 'active',
      availability: { season: true, record: true, leaderboard: true, my_rank: true },
      record: { matches_played: 4, wins: 3, final_rank: 0 },
      my_rank: 7,
      leaderboard: []
    },
    {
      success: true,
      season: { id: 2, name: 'S3', status: 1 },
      season_state: 'active',
      availability: { season: true, record: true, leaderboard: true, my_rank: false },
      record: { matches_played: 4, wins: 3, final_rank: 0 },
      my_rank: 0,
      leaderboard: []
    }
  ]
  const { component, mounted } = mountSeasonPageSetup({
    getSeasonOverview: async () => responses.shift()
  })

  await mounted[0]()
  assert.equal(component.myRankAvailable.value, true)
  assert.equal(component.myRank.value, 7)

  await component.loadSeasonOverview()
  assert.equal(component.myRankAvailable.value, false, 'rank partial must not be shown as rank 0')
  assert.equal(component.myRank.value, 0)
  assert.equal(component.myRecord.value.matches_played, 4, 'rank partial must keep the record')
  assert.equal(component.seasonPageError.value, null)
})

test('season page replaces the unreachable report button with a history entry', () => {
  assert.doesNotMatch(source, /season\.status === 2/)
  assert.doesNotMatch(source, /goReport/)
  assert.doesNotMatch(source, /myRecord\.final_rank/, 'active season must not rely on settlement-only final_rank')
  assert.match(source, /url: `\/subPages\/season\/history\?game_type=\$\{currentGameType\.value\}`/)
  assert.match(source, /<view class="history-entry" @tap="goHistory">/)
})

test('season page exposes V2 qualification progress, net change and server ranks', async () => {
  const { component, mounted } = mountSeasonPageSetup({
    getSeasonOverview: async () => ({
      success: true,
      season: { id: 4, name: 'S4', status: 1, qualify_min_matches: 5, qualify_min_opponents: 3 },
      season_state: 'active',
      availability: { season: true, record: true, leaderboard: true, my_rank: true },
      record: { matches_played: 3, wins: 2, distinct_opponents: 2, qualified: false, start_rank_score: 1250, end_rank_score: 1282 },
      my_rank: 0,
      leaderboard: [
        { user_id: 7, rank: 1, qualified: true, rank_score: 1400, wins: 6 },
        { user_id: 8, rank: 0, qualified: false, rank_score: 1900, wins: 4 }
      ]
    })
  })
  await mounted[0]()
  assert.equal(component.qualification.value.qualified, false)
  assert.equal(component.qualification.value.progressText, '场次 3/5 · 不同对手 2/3')
  assert.equal(component.netChangeText.value, '+32')
  assert.deepEqual(component.leaderboardRows.value.map((item) => item.rankView.text), ['1', '—'])
  assert.equal(component.leaderboardRows.value[1].rankView.unqualified, true)
  assert.equal(component.myRank.value, 0)
})

test('season page renders server ranks, qualification and deduplicated pages', () => {
  assert.match(source, /v-for="item in leaderboardRows"/)
  assert.match(source, /item\.rankView\.text/)
  assert.doesNotMatch(source, /\{\{ idx \+ 1 \}\}/)
  assert.match(source, /暂未达标，不参与赛季排名/)
  assert.match(source, /qualification\.progressText/)
  assert.match(source, /本季排位分 \{\{ myRecord\.start_rank_score \}\} → \{\{ myRecord\.end_rank_score \}\}/)
  assert.match(source, /mergeLeaderboardPage\(leaderboard\.value, newList\)/)
  assert.match(source, /hasMore\.value = requestedPage \* 20 < \(Number\(res\.total\) \|\| 0\)/)
})
