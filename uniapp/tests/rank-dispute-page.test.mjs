import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc'

const read = (path) => readFileSync(new URL(path, import.meta.url), 'utf8')
const matchApiSource = read('../api/match.js')
const resultFilename = new URL('../subPages/match/matchResult.vue', import.meta.url)
const resultSource = read('../subPages/match/matchResult.vue')
const playingSource = read('../subPages/match/playing.vue')

test('异议接口门面指向后端路由并透传请求体', () => {
  assert.match(matchApiSource, /export const disputeRankResult = \(data\) => \{\s*return post\('\/api\/match\/result-dispute', data\)\s*\}/)
})

test('赛果页能编译，异议框与排位结算只由权威字段驱动', () => {
  const { descriptor, errors } = parse(resultSource, { filename: resultFilename.pathname })
  assert.equal(errors.length, 0)
  assert.doesNotThrow(() => compileScript(descriptor, { id: 'match-result' }))
  assert.equal(compileTemplate({ source: descriptor.template.content, filename: resultFilename.pathname, id: 'match-result' }).errors.length, 0)
  // 异议入口只在服务端允许时出现；待生效/作废不显示 +0。
  assert.match(resultSource, /<view v-if="settlementView\.canDispute" class="dispute-box">/)
  assert.match(resultSource, /v-if="settlementView\.hideRankChange" class="rank-card-value is-zero is-compact">\{\{ settlementView\.rankValueText \}\}/)
  assert.match(resultSource, /const settlementView = computed\(\(\) => resolveRankSettlementView\(matchData\.value\)\)/)
})

test('提出异议必须先确认，请求后以服务端状态回源，提交中防重', () => {
  assert.match(resultSource, /if \(disputeSubmitting\.value \|\| !settlementView\.value\.canDispute \|\| !matchId\.value\) return\s*\n\s*uni\.showModal\(/)
  assert.match(resultSource, /if \(!confirm\) return\s*\n\s*disputeSubmitting\.value = true/)
  assert.match(resultSource, /disputeRankResult\(\{ match_id: requestedMatchId, client_action_id: createMatchActionId\(\) \}\)/)
  assert.match(resultSource, /finally \{\s*disputeSubmitting\.value = false\s*\n\s*if \(isSameIdentity\(\) && matchId\.value === requestedMatchId\) await loadMatchData\(\{ silent: true \}\)/)
  assert.match(resultSource, /:disabled="disputeSubmitting"/)
  // 失败不在本地推断结果：只提示并回源，入口由刷新后的服务端字段决定。
  assert.doesNotMatch(resultSource, /rank_settlement_status\s*=\s*['"]voided['"]/)
})

test('切号或退出后旧异议响应不提示也不回写', () => {
  assert.match(resultSource, /const identity = \{ userId: userStore\.userId, authGeneration: userStore\.authGeneration \}/)
  assert.match(resultSource, /if \(isSameIdentity\(\)\) \{\s*uni\.showToast\(\{ title: res\?\.message/)
  assert.match(resultSource, /if \(isSameIdentity\(\)\) uni\.showToast\(\{ title: '网络异常，请稍后重试'/)
})

test('滑动结束处写明无需对方确认及排位异议期', () => {
  assert.match(playingSource, /<text v-if="isChallengeMatchFinishSlider" class="finish-slider-note">\{\{ finishSlideNote \}\}<\/text>/)
  assert.match(playingSource, /buildFinishSlideNote\(\{\s*challengeId: challengeId\.value,\s*matchMode: matchMode\.value,\s*myScore: myScore\.value,\s*opponentScore: opponentScore\.value\s*\}\)/)
})
