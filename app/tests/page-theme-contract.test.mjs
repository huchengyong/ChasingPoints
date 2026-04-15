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

const customNavigationRoutes = new Set(
  [
    ...pagesJson.pages.map((page) => ({ path: page.path, style: page.style || {} })),
    ...pagesJson.subPackages.flatMap((pack) => pack.pages.map((page) => ({
      path: `${pack.root}/${page.path}`,
      style: page.style || {}
    })))
  ]
    .filter((page) => page.style.navigationStyle === 'custom')
    .map((page) => page.path)
)

const exemptRoutes = new Set([
  'pages/welcome/index',
  'pages/index/index',
  'subPages/match/matchResult',
  'subPages/match/playing',
  'subPages/match/shareResult',
  'subPages/season/report'
])

const getRouteSource = (route) => readFileSync(new URL(`../${route}.vue`, import.meta.url), 'utf8')

test('all registered pages have a matching Vue file', () => {
  for (const route of registeredRoutes) {
    assert.equal(existsSync(join(appRoot.pathname, `${route}.vue`)), true, route)
  }
})

test('system-navigation pages use the shared page theme hook', () => {
  for (const route of registeredRoutes) {
    if (customNavigationRoutes.has(route) || exemptRoutes.has(route)) continue

    const source = getRouteSource(route)
    assert.match(source, /usePageTheme/, `${route} should use usePageTheme`)
    assert.match(source, /isDarkMode/, `${route} should expose isDarkMode`)
    assert.match(source, /dark-mode/, `${route} should bind a dark-mode class`)
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
