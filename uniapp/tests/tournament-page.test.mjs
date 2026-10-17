import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const pageSource = readFileSync(
  new URL('../pages/tournament/index.vue', import.meta.url),
  'utf8'
)

const styleSource = readFileSync(
  new URL('../pages/tournament/index.scss', import.meta.url),
  'utf8'
)

test('tournament page reapplies theme on show for the native navigation bar', () => {
  assert.match(pageSource, /usePageTheme/)
  assert.match(pageSource, /const \{ isDarkMode \} = usePageTheme\(\)/)
  assert.doesNotMatch(pageSource, /useThemeStore|THEME_CHANGE_EVENT|applyNavigationBarTheme/)
})

test('tournament list and detail share the warm theme tokens with a dark-mode override', () => {
  const themeSource = readFileSync(new URL('../pages/tournament/saixun-theme.scss', import.meta.url), 'utf8')
  const detailStyleSource = readFileSync(new URL('../subPages/tournament/detail.scss', import.meta.url), 'utf8')

  assert.match(themeSource, /@mixin saixun-theme-vars\s*\{[\s\S]*--sx-page:\s*#F7F4EC;/)
  assert.match(themeSource, /@mixin saixun-dark-theme-vars\s*\{[\s\S]*--sx-page:\s*#141109;[\s\S]*--sx-text:\s*#FFF7E1;/)
  assert.match(styleSource, /@import '\.\/saixun-theme\.scss';[\s\S]*\.tournament-page\s*\{\s*@include saixun-theme-vars;[\s\S]*&\.dark-mode\s*\{\s*@include saixun-dark-theme-vars;/)
  assert.match(detailStyleSource, /@import '\.\.\/\.\.\/pages\/tournament\/saixun-theme\.scss';[\s\S]*\.detail-page\s*\{\s*@include saixun-theme-vars;[\s\S]*&\.dark-mode\s*\{\s*@include saixun-dark-theme-vars;/)
  // 微信小程序 WXSS 不支持通配选择器
  assert.doesNotMatch(`${themeSource}${styleSource}${detailStyleSource}`, /(^|[\s,{])\*(\s|,|\{|::)/m)
})

test('tournament list filters by status tabs and action sheets without a custom bottom sheet', () => {
  assert.match(pageSource, /v-for="tab in STATUS_TABS"[\s\S]*@tap="selectStatus\(tab\.value\)"/)
  assert.match(pageSource, /const openYearSheet[\s\S]*uni\.showActionSheet/)
  assert.match(pageSource, /const openGameTypeSheet[\s\S]*uni\.showActionSheet/)
  assert.doesNotMatch(pageSource, /hideTabBar|showTabBar|SAIXUN_DATE_PRESETS|custom-date/)
  assert.match(pageSource, /buildSaiXunListSections\(list\.value\)/)
})

test('tournament list is the only tabbar registration and retired paths are gone', () => {
  const pagesConfig = JSON.parse(readFileSync(new URL('../pages.json', import.meta.url), 'utf8'))
  const pagePaths = pagesConfig.pages.map((page) => page.path)
  assert.ok(pagePaths.includes('pages/tournament/index'))
  assert.ok(pagePaths.includes('pages/social/index'))

  const tabPaths = pagesConfig.tabBar.list.map((item) => item.pagePath)
  assert.ok(tabPaths.includes('pages/tournament/index'))
  assert.ok(!tabPaths.includes('pages/social/index'))

  const tournamentPkg = pagesConfig.subPackages.find((pkg) => pkg.root === 'subPages/tournament')
  assert.deepEqual(tournamentPkg.pages.map((page) => page.path), ['create', 'detail', 'bracket'])
})

test('old social compat page is a lightweight redirect to the tournament tab', () => {
  const compatSource = readFileSync(new URL('../pages/social/index.vue', import.meta.url), 'utf8')
  assert.match(compatSource, /switchTab.*\/pages\/tournament\/index/)
  assert.doesNotMatch(compatSource, /loadEventNews|fetchData|saixun/)
})

test('home and user entries open the tournament tab via switchTab', () => {
  const homeSource = readFileSync(new URL('../pages/index/index.vue', import.meta.url), 'utf8')
  assert.match(homeSource, /goTo\('\/pages\/tournament\/index', true\)/)
  assert.doesNotMatch(homeSource, /subPages\/tournament\/index/)

  const userSource = readFileSync(new URL('../pages/user/index.vue', import.meta.url), 'utf8')
  assert.match(userSource, /openRoute\('\/pages\/tournament\/index', true\)/)
  assert.doesNotMatch(userSource, /pages\/social\/index|subPages\/tournament\/index/)
})

test('filter merge preserves cross-filter state when year or date changes', () => {
  assert.match(pageSource, /\.\.\.filter\.value,[\s\S]*?\.\.\.nextFilter/)
})

test('filter change resets pagination to page 1', () => {
  assert.match(pageSource, /page\.value\s*=\s*1/)
  assert.match(pageSource, /list\.value\s*=\s*\[\]/)
  assert.match(pageSource, /total\.value\s*=\s*0/)
  assert.match(pageSource, /hasMore\.value\s*=\s*false/)
})

test('detail page offers round tabs and forced pull-down refresh', () => {
  const detailSource = readFileSync(new URL('../subPages/tournament/detail.vue', import.meta.url), 'utf8')
  const pagesConfig = JSON.parse(readFileSync(new URL('../pages.json', import.meta.url), 'utf8'))
  const tournamentPkg = pagesConfig.subPackages.find((pkg) => pkg.root === 'subPages/tournament')
  const detailConfig = tournamentPkg.pages.find((page) => page.path === 'detail')

  assert.equal(detailConfig.style.enablePullDownRefresh, true)
  assert.match(detailSource, /onPullDownRefresh\(async \(\) => \{\s*await fetchDetail\(\{ force: true \}\)\s*uni\.stopPullDownRefresh\(\)/)
  assert.match(detailSource, /publicReadStore\.loadEventNewsView\(\{ event_id: eventNewsId\.value \}, \{ force \}\)/)
  assert.match(detailSource, /v-for="\(round, index\) in eventView\.rounds"[\s\S]*@tap="selectRound\(round\.key\)"/)
  assert.match(detailSource, /v-for="day in activeRound\.days"/)
})

test('detail back preserves filter and scroll state via onShow guard', () => {
  assert.match(pageSource, /hasLoadedOnce\.value/)
  assert.match(pageSource, /uni\.navigateTo\(\{\s*url:\s*[^}]*subPages\/tournament\/detail/)
  assert.doesNotMatch(pageSource, /redirectTo[\s\S]*subPages\/tournament\/detail|reLaunch[\s\S]*subPages\/tournament\/detail/)
})
