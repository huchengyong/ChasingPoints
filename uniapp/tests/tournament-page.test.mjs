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

test('tournament page dark hero keeps light text on a dark card', () => {
  assert.match(
    styleSource,
    /&\.dark-mode\s*\{[\s\S]*\.hero-card\s*\{[\s\S]*background:\s*linear-gradient\(135deg,\s*#241d10 0%,\s*#1a1408 100%\);[\s\S]*border-color:\s*#3a2e16;/
  )
  assert.match(
    styleSource,
    /&\.dark-mode\s*\{[\s\S]*\.hero-desc,[\s\S]*color:\s*#d7c89b;/
  )
  assert.match(
    styleSource,
    /&\.dark-mode\s*\{[\s\S]*\.hero-title,[\s\S]*color:\s*#fff7e1;/
  )
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

test('detail back preserves filter and scroll state via onShow guard', () => {
  assert.match(pageSource, /hasLoadedOnce\.value/)
  assert.match(pageSource, /uni\.navigateTo\(\{\s*url:\s*[^}]*subPages\/tournament\/detail/)
  assert.doesNotMatch(pageSource, /redirectTo[\s\S]*subPages\/tournament\/detail|reLaunch[\s\S]*subPages\/tournament\/detail/)
})
