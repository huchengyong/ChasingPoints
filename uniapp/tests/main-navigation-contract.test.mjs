import test from 'node:test'
import assert from 'node:assert/strict'
import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'

import { MAIN_TABS } from '../utils/main-tab-bar.js'
import { readPagesJson } from './helpers/pages-json.mjs'

const appDir = new URL('..', import.meta.url).pathname
const SKIPPED_DIRS = new Set(['node_modules', 'unpackage', 'tests', 'uni_modules', 'cdn-assets', 'static', 'uniCloud-aliyun', 'harmony-configs'])

const collectSources = (dir = appDir) => readdirSync(dir).flatMap((name) => {
  if (name.startsWith('.') || SKIPPED_DIRS.has(name)) return []
  const full = join(dir, name)
  if (statSync(full).isDirectory()) return collectSources(full)
  return /\.(vue|js)$/.test(name) ? [full] : []
})

const sources = collectSources().map((file) => ({ file: relative(appDir, file), text: readFileSync(file, 'utf8') }))
const tabRoutes = new Set(MAIN_TABS.map((tab) => `/${tab.route}`))

const matchesOf = (pattern) => sources.flatMap(({ file, text }) => (
  [...text.matchAll(pattern)].map((match) => ({ file, url: match[1] }))
))

test('every switchTab target is one of the registered tabs', () => {
  const targets = [
    ...matchesOf(/switchTab\(\{\s*url:\s*['"`]([^'"`]+)['"`]/g),
    ...matchesOf(/getMatchHistoryTabUrl\s*=\s*\(\)\s*=>\s*['"`]([^'"`]+)['"`]/g),
    ...matchesOf(/goTo\(['"`]([^'"`]+)['"`],\s*true\)/g)
  ]
  assert.ok(targets.length > 0)
  for (const { file, url } of targets) {
    assert.equal(tabRoutes.has(url), true, `${file} switches to ${url}, which is not a tab`)
  }
})

test('nothing links to removed navigation targets', () => {
  const removed = [
    { pattern: /\/pages\/match\/index/g, allowedIn: ['pages/index/index.vue'], label: '观赛大厅只能从首页进入，且不是 Tab' },
    { pattern: /pending_match_rematch/g, allowedIn: ['utils/session-storage.js'], label: '再来一局不再借本地存储交接（仅保留登录时清理旧残留）' }
  ]
  for (const { pattern, allowedIn, label } of removed) {
    for (const { file, text } of sources) {
      if (allowedIn.includes(file)) continue
      assert.doesNotMatch(text, pattern, `${file}: ${label}`)
    }
  }
})

test('the lobby page is only reachable through a normal navigation from the home page', () => {
  const home = readFileSync(join(appDir, 'pages/index/index.vue'), 'utf8')
  assert.match(home, /<view class="section-more" @tap="goTo\('\/pages\/match\/index'\)">/)
  assert.doesNotMatch(home, /goTo\('\/pages\/match\/index', true\)/)
})

test('tab pages, custom tab bar and pages.json agree on the tab list', () => {
  const pagesConfig = readPagesJson()
  const registered = new Set(pagesConfig.pages.map((page) => page.path))
  for (const tab of MAIN_TABS) {
    assert.equal(registered.has(tab.route), true, `${tab.route} must be a main package page`)
    assert.equal(existsSync(join(appDir, `${tab.route}.vue`)), true)
  }
  assert.deepEqual(pagesConfig.tabBar.list.map((item) => item.pagePath), MAIN_TABS.map((tab) => tab.route))
})

test('every tab page joins the main tab bar and reserves the bottom space', () => {
  for (const tab of MAIN_TABS) {
    const source = readFileSync(join(appDir, `${tab.route}.vue`), 'utf8')
    assert.match(source, new RegExp(`useMainTabBar\\('${tab.key}'\\)`), `${tab.route} must call useMainTabBar`)
    const styleSource = [source, existsSync(join(appDir, `${tab.route.replace(/index$/, 'index')}.scss`))
      ? readFileSync(join(appDir, `${tab.route}.scss`), 'utf8')
      : ''].join('\n')
    assert.match(styleSource, /--ui-tab-bar-space/, `${tab.route} must leave room for the custom tab bar`)
  }
})
