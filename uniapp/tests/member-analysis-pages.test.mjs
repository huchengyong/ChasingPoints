import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { readPagesJson } from './helpers/pages-json.mjs'

const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8')
const pagesConfig = readPagesJson()

const subPackagePaths = (root) => (pagesConfig.subPackages.find((item) => item.root === root)?.pages || []).map((page) => page.path)

test('the analysis pages live in sub-packages, never in the main package', () => {
  assert.ok(subPackagePaths('subPages/user').includes('performance'))
  assert.ok(subPackagePaths('subPages/match').includes('matchReplay'))
  const mainPaths = pagesConfig.pages.map((page) => page.path)
  assert.ok(!mainPaths.some((path) => /performance|matchReplay/.test(path)))
  // 每个已注册页面都有对应文件。
  assert.match(read('subPages/user/performance.vue'), /<template>/)
  assert.match(read('subPages/match/matchReplay.vue'), /<template>/)
})

test('the 竞技分析 page links to the performance analysis with the current game type', () => {
  const source = read('subPages/user/statsDetail.vue')
  assert.match(source, /class="section performance-entry" @tap="openPerformance"/)
  assert.match(source, /uni\.navigateTo\(\{ url: `\/subPages\/user\/performance\?game_type=\$\{gameType\}` \}\)/)
})

test('the performance page reads through the isolated store with the rank and member invalidation versions', () => {
  const source = read('subPages/user/performance.vue')
  assert.match(source, /performanceStore\.load\(identity, gameType, currentVersion\(\)\)/)
  assert.match(source, /versionOf\('rank'\)/)
  assert.match(source, /versionOf\('member'\)/)
  assert.match(source, /canShowPaidMembershipEntry\(\)/)
  // 页面不直接调用 uni.request 或请求层。
  assert.doesNotMatch(source, /uni\.request|utils\/request\.js/)
  assert.match(source, /PERFORMANCE_NOTE/)
})

test('the replay page reads through the API facade and keeps the purchase entry behind the compliance switch', () => {
  const source = read('subPages/match/matchReplay.vue')
  assert.match(source, /import \{ getMemberMatchReplay \} from '@\/api\/member\.js'/)
  assert.match(source, /canShowPaidMembershipEntry\(\)/)
  assert.doesNotMatch(source, /uni\.request|utils\/request\.js/)
  assert.match(source, /v-if="view\.showPurchase"/)
})

test('the charts are drawn with views, without a chart library or canvas', () => {
  for (const path of ['subPages/user/performance.vue', 'subPages/match/matchReplay.vue']) {
    const source = read(path)
    assert.doesNotMatch(source, /<canvas|echarts|uCharts|ucharts/i, path)
  }
  assert.match(read('subPages/user/performance.vue'), /class="chart-bar"/)
  assert.match(read('subPages/match/matchReplay.vue'), /class="trend-bar/)
})

test('the member center shows identity, tiers and analysis entries in the designed order and drops the ranking rights card', () => {
  const source = read('subPages/user/memberCenter.vue')
  const order = ['member-hero', 'member-identity-section', 'member-analysis-section', 'member-growth-section']
  const positions = order.map((name) => source.indexOf(`class="${name}`) >= 0 ? source.indexOf(`class="${name}`) : source.indexOf(` ${name}`))
  assert.ok(positions.every((position) => position >= 0), 'every designed section must exist')
  assert.deepEqual([...positions].sort((a, b) => a - b), positions, 'sections must follow the designed order')
  assert.ok(source.indexOf('member-growth-section') < source.indexOf('订阅套餐'), 'subscription and payment come after the growth section')

  assert.match(source, /resolveMemberIdentityPreview\(/)
  assert.match(source, /<memberAvatar[\s\S]*:vip-level="identityPreview\.identity\.vipLevel"[\s\S]*:rank-level="rankLevel"/)
  assert.match(source, /v-for="item in identityPreview\.tiers"/)
  assert.match(source, /成长等级解锁更高阶外观/)
  assert.match(source, /@click="openPerformance"/)
  assert.match(source, /@click="openMatchHistory"/)
  assert.match(source, /VIP不影响排位分/)

  // 排位权益区块、倍率文案与段位配置读取都已移除。
  assert.doesNotMatch(source, /member-ranking|rankingRightsCard|resolveMemberRankingRightsCard|rankRulesMeta|getRankConfigs/)
  assert.doesNotMatch(source, /倍率|200 分/)
  assert.doesNotMatch(read('subPages/user/memberCenter.scss'), /member-ranking/)
})

test('the paid entry of the member center stays behind the compliance switch', () => {
  const source = read('subPages/user/memberCenter.vue')
  assert.match(source, /<view v-if="!isComplianceMode" class="member-section">[\s\S]*订阅套餐/)
  assert.match(source, /<view v-if="!isComplianceMode" class="member-section">[\s\S]*支付方式/)
  assert.match(source, /:disabled="isComplianceMode \|\|/)
})
