import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

test('pk report page uses report view model helpers and keeps social actions', () => {
  const source = readFileSync(new URL('../subPages/social/pkReport.vue', import.meta.url), 'utf8')

  assert.match(source, /buildPkReportHero/)
  assert.match(source, /buildPkEvidenceList/)
  assert.match(source, /shareToPost/)
  assert.match(source, /savePoster/)
  assert.doesNotMatch(source, /v-for="item in recentMatches".*trend-item/s)
})

