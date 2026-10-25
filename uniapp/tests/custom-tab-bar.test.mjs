import test from 'node:test'
import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import vm from 'node:vm'

import { MAIN_TABS } from '../utils/main-tab-bar.js'
import { readPagesJson } from './helpers/pages-json.mjs'

const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8')
const componentSource = read('custom-tab-bar/index.js')

// 原生组件不经过 uni-app 编译，这里为 Component / wx 打桩后直接执行源码。
const loadComponent = ({ storage = {}, systemInfo = {}, pages = [], switchTab = () => {} } = {}) => {
  let options
  const wx = {
    getStorageSync: (key) => storage[key] ?? '',
    getAppBaseInfo: () => systemInfo,
    switchTab
  }
  vm.runInNewContext(componentSource, {
    Component: (definition) => { options = definition },
    wx,
    getCurrentPages: () => pages
  })
  return options
}

const createInstance = (options) => {
  const instance = {
    data: JSON.parse(JSON.stringify(options.data)),
    setData(patch) { Object.assign(this.data, patch) },
    ...options.methods
  }
  return instance
}

test('component list stays aligned with the main tabs and pages.json', () => {
  // 组件在 vm 上下文里创建，数组原型与当前上下文不同，先过一遍 JSON 再做 deepEqual。
  const list = JSON.parse(JSON.stringify(loadComponent().data.list))
  const tabBar = readPagesJson().tabBar

  assert.deepEqual(list.map((item) => item.pagePath), MAIN_TABS.map((tab) => `/${tab.route}`))
  assert.deepEqual(list.map((item) => item.key), MAIN_TABS.map((tab) => tab.icon))
  assert.deepEqual(list.map((item) => item.text), tabBar.list.map((item) => item.text))
  assert.equal(tabBar.custom, true)
})

test('attached derives the first-frame theme and selection to avoid a flash on first visit', () => {
  const route = 'pages/message/index'
  const attach = (context) => {
    const options = loadComponent(context)
    const instance = createInstance(options)
    options.lifetimes.attached.call(instance)
    return instance.data
  }

  assert.deepEqual(
    { selected: attach({ pages: [{ route }] }).selected, dark: attach({ pages: [{ route }] }).dark },
    { selected: 2, dark: false }
  )
  assert.equal(attach({ pages: [{ route }], storage: { theme_mode: 'dark' } }).dark, true)
  assert.equal(attach({ pages: [{ route }], storage: { theme_mode: 'light' }, systemInfo: { theme: 'dark' } }).dark, false)
  assert.equal(attach({ pages: [{ route }], systemInfo: { theme: 'dark' } }).dark, true)
  assert.equal(attach({ pages: [{ route }], systemInfo: { hostTheme: 'dark', theme: 'light' } }).dark, true)
  assert.equal(attach({ pages: [{ route: 'subPages/match/playing' }] }).selected, -1)
  assert.equal(attach({ pages: [] }).selected, -1)
})

test('tapping a tab switches to its page except the selected one', () => {
  const switched = []
  const options = loadComponent({ switchTab: (payload) => switched.push(payload.url) })
  const instance = createInstance(options)
  instance.data.selected = 0
  const tap = (index) => instance.onTapTab({ currentTarget: { dataset: { index } } })

  tap(2)
  tap('3')
  tap(0)
  tap(9)
  tap(undefined)

  assert.deepEqual(switched, ['/pages/message/index', '/pages/user/index'])
})

test('scan button calls the injected handler and tolerates a missing one', () => {
  const options = loadComponent()
  const instance = createInstance(options)
  let scans = 0

  assert.doesNotThrow(() => instance.onTapScan())

  instance.setScanHandler(() => { scans += 1 })
  instance.onTapScan()
  assert.equal(scans, 1)

  instance.setScanHandler('not a function')
  instance.onTapScan()
  assert.equal(scans, 1)
})

test('template: scan button sits between the second and third tab with an accessible name', () => {
  const wxml = read('custom-tab-bar/index.wxml')

  assert.match(wxml, /<view wx:if="\{\{index === 1\}\}" class="scan"[^>]*bindtap="onTapScan"[^>]*aria-label="扫码"/)
  assert.match(wxml, /aria-role="tab"/)
  assert.match(wxml, /，当前页/)
  assert.match(wxml, /wx:if="\{\{item\.key === 'message' && badgeType === 'text'\}\}"/)
  assert.match(wxml, /wx:if="\{\{item\.key === 'message' && badgeType === 'dot'\}\}"/)
})

test('every icon the template can request exists, and the button image is a square', () => {
  const wxml = read('custom-tab-bar/index.wxml')
  assert.match(wxml, /src="\/static\/tabbar\/\{\{item\.key\}\}\{\{selected === index \? '-selected' : ''\}\}_\{\{dark \? 'dark' : 'light'\}\}\.png"/)

  for (const tab of MAIN_TABS) {
    for (const state of ['', '-selected']) {
      for (const theme of ['light', 'dark']) {
        assert.equal(existsSync(new URL(`../static/tabbar/${tab.icon}${state}_${theme}.png`, import.meta.url)), true, `${tab.icon}${state}_${theme}`)
      }
    }
  }

  const png = readFileSync(new URL('../static/tabbar/scan-button.png', import.meta.url))
  assert.equal(png.readUInt32BE(16), png.readUInt32BE(20), 'scan-button.png must be square so midButton does not stretch it')
})

test('styles: fixed bar with safe area, plain views, dark override and hidden state', () => {
  const wxss = read('custom-tab-bar/index.wxss')

  assert.match(wxss, /\.tab-bar\s*\{[^}]*position:\s*fixed/)
  assert.match(wxss, /height:\s*calc\(50px \+ env\(safe-area-inset-bottom\)\)/)
  assert.match(wxss, /\.tab-bar\.dark\s*\{[^}]*background:\s*#141109/)
  assert.match(wxss, /\.tab-bar\.hidden\s*\{[^}]*transform:\s*translateY\(100%\)/)
  assert.doesNotMatch(read('custom-tab-bar/index.wxml'), /cover-view|cover-image/)
})
