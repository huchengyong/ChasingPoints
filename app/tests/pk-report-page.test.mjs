import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

test('pk report page keeps compliance-safe share actions and removes post publishing entry', () => {
  const source = readFileSync(new URL('../subPages/social/pkReport.vue', import.meta.url), 'utf8')

  assert.match(source, /buildPkReportHero/)
  assert.match(source, /getH2HOverview\(\{ \.\.\.buildRequestParams\(\), page_size: 5 \}\)/)
  assert.doesNotMatch(source, /getH2HStats\(|getH2HHistory\(/)
  assert.match(source, /buildPkEvidenceList/)
  assert.match(source, /copyShareSummary/)
  assert.match(source, /buildShareSummary/)
  assert.match(source, /savePoster/)
  assert.match(source, /复制分享文案/)
  assert.doesNotMatch(source, /发到动态/)
  assert.doesNotMatch(source, /v-for="item in recentMatches".*trend-item/s)
})
