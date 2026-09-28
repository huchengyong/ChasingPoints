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
    'resolveSeasonTimeline', 'usePageTheme', 'resolveAvatarUrl',
    'ASYNC_PAGE_STATUS', 'beginAsyncPageLoad', 'createAsyncPageState', 'getAsyncPageRequest',
    'rejectAsyncPageLoad', 'resolveAsyncPageErrorFeedback', 'resolveAsyncPageLoad', 'createRequestError',
    `${setup}\nreturn { loading, hasLoadedOnce, loadSeasonOverview }`
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
