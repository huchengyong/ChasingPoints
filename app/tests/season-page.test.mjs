import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('../subPages/season/index.vue', import.meta.url), 'utf8')

test('season page presents lifecycle state and keeps legacy active responses compatible', () => {
  assert.match(source, /resolveCurrentSeasonState/)
  assert.match(source, /resolveSeasonTimeline/)
  assert.match(source, /const hasActiveSeason = computed\(\(\) => seasonState\.value === 'active' && Boolean\(season\.value\)\)/)
  assert.match(source, /<template v-if="hasActiveSeason">/)
  assert.match(source, /if \(!hasActiveSeason\.value\) return/)
  assert.match(source, /const seasonState = ref\('not_started'\)/)
  assert.match(source, /seasonState\.value = res\.season_state \|\| \(season\.value \? 'active' : 'not_started'\)/)
  assert.match(source, /seasonState\.value = 'unavailable'/)
  assert.match(source, /seasonEmptyState\.title/)
  assert.match(source, /seasonEmptyState\.description/)
  assert.doesNotMatch(source, /暂无进行中的赛季/)
  assert.doesNotMatch(source, /赛季间歇中/)
  assert.doesNotMatch(source, /new Date\(season\.value\.end_date\)/)
})
