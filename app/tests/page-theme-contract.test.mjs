import test from 'node:test'
import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

const appRoot = new URL('..', import.meta.url)
const pagesJson = JSON.parse(readFileSync(new URL('../pages.json', import.meta.url), 'utf8'))

const registeredRoutes = [
  ...pagesJson.pages.map((page) => page.path),
  ...pagesJson.subPackages.flatMap((pack) => pack.pages.map((page) => `${pack.root}/${page.path}`))
]

const getRouteSource = (route) => readFileSync(new URL(`../${route}.vue`, import.meta.url), 'utf8')

test('all registered pages have a matching Vue file', () => {
  for (const route of registeredRoutes) {
    assert.equal(existsSync(join(appRoot.pathname, `${route}.vue`)), true, route)
  }
})

test('registered pages use the shared page theme hook', () => {
  for (const route of registeredRoutes) {
    const source = getRouteSource(route)
    assert.match(source, /usePageTheme/, `${route} should use usePageTheme`)
    assert.match(source, /isDarkMode/, `${route} should expose isDarkMode`)
    assert.match(source, /dark-mode/, `${route} should bind a dark-mode class`)
  }
})

test('registered pages do not bypass the shared page theme hook', () => {
  for (const route of registeredRoutes) {
    const source = getRouteSource(route)
    assert.doesNotMatch(source, /useThemeStore|THEME_CHANGE_EVENT|applyNavigationBarTheme/, route)
  }
})

test('page-specific light navigation backgrounds do not bypass runtime theme', () => {
  const routesWithFixedLightNav = []

  for (const page of pagesJson.subPackages.flatMap((pack) => pack.pages.map((item) => ({
    path: `${pack.root}/${item.path}`,
    style: item.style || {}
  }))).concat(pagesJson.pages.map((item) => ({ path: item.path, style: item.style || {} })))) {
    const color = String(page.style.navigationBarBackgroundColor || '')
    if (color && !color.startsWith('@') && color.toLowerCase() !== '#141109') {
      routesWithFixedLightNav.push(page.path)
    }
  }

  assert.deepEqual(routesWithFixedLightNav, [])
})

test('custom fixed-visual pages expose light defaults and dark overrides', () => {
  const shareResultSource = getRouteSource('subPages/match/shareResult')
  assert.match(shareResultSource, /class="share-page" :class="\{ 'dark-mode': isDarkMode \}"/)
  assert.match(shareResultSource, /\.share-page\s*\{[\s\S]*background:\s*#f8fafc;/)
  assert.match(shareResultSource, /&\.dark-mode\s*\{[\s\S]*background:\s*#0f172a;/)

  const seasonReportSource = getRouteSource('subPages/season/report')
  assert.match(seasonReportSource, /class="report-page" :class="\{ 'dark-mode': isDarkMode \}"/)
  assert.match(seasonReportSource, /\.report-page\s*\{[\s\S]*background:\s*linear-gradient\(180deg,\s*#f8fafc 0%,\s*#e2e8f0 100%\);/)
  assert.match(seasonReportSource, /&\.dark-mode\s*\{[\s\S]*background:\s*linear-gradient\(180deg,\s*#1e293b 0%,\s*#0f172a 100%\);/)
})
