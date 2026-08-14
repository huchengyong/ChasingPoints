import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  applyRuntimeTheme,
  isConfiguredTabBarRoute
} from '../utils/theme-application.js'

const readAppSource = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8')

const createUniRecorder = () => {
  const calls = []

  return {
    calls,
    uniApi: {
      setNavigationBarColor(options) {
        calls.push({ method: 'setNavigationBarColor', options })
      },
      setBackgroundColor(options) {
        calls.push({ method: 'setBackgroundColor', options })
      },
      setBackgroundTextStyle(options) {
        calls.push({ method: 'setBackgroundTextStyle', options })
      },
      setTabBarStyle(options) {
        calls.push({ method: 'setTabBarStyle', options })
      },
      setTabBarItem(options) {
        calls.push({ method: 'setTabBarItem', options })
      }
    }
  }
}

test('isConfiguredTabBarRoute identifies configured tabBar routes', () => {
  assert.equal(isConfiguredTabBarRoute('pages/index/index'), true)
  assert.equal(isConfiguredTabBarRoute('/pages/match/index'), true)
  assert.equal(isConfiguredTabBarRoute('pages/social/index'), true)
  assert.equal(isConfiguredTabBarRoute('pages/user/index'), true)
})

test('isConfiguredTabBarRoute rejects non-tabBar routes', () => {
  assert.equal(isConfiguredTabBarRoute('pages/welcome/index'), false)
  assert.equal(isConfiguredTabBarRoute('pages/login/login'), false)
  assert.equal(isConfiguredTabBarRoute('subPages/user/settings'), false)
  assert.equal(isConfiguredTabBarRoute(''), false)
})

test('applyRuntimeTheme does not depend on an implicit uni global', () => {
  const source = readAppSource('utils/theme-application.js')

  assert.doesNotMatch(source, /uniApi\s*=\s*uni/)
})

test('runtime theme call sites inject the UniApp API explicitly', () => {
  const appSource = readAppSource('App.vue')
  const storeSource = readAppSource('store/theme.js')

  assert.match(appSource, /applyRuntimeTheme\(\{\s*uniApi:\s*uni,/)
  assert.match(storeSource, /applyRuntimeTheme\(\{\s*uniApi:\s*uni,/)
})

test('applyRuntimeTheme applies navigation colors but skips tabBar on non-tabBar pages', () => {
  const recorder = createUniRecorder()

  applyRuntimeTheme({
    uniApi: recorder.uniApi,
    route: 'pages/welcome/index',
    isDarkMode: false
  })

  assert.deepEqual(recorder.calls.map((call) => call.method), [
    'setNavigationBarColor',
    'setBackgroundColor',
    'setBackgroundTextStyle'
  ])
  assert.deepEqual(recorder.calls[0].options, {
    frontColor: '#000000',
    backgroundColor: '#F7F4EC',
    animation: {
      duration: 300,
      timingFunc: 'easeIn'
    }
  })
  assert.deepEqual(recorder.calls[1].options, {
    backgroundColor: '#F7F4EC',
    backgroundColorTop: '#F7F4EC',
    backgroundColorBottom: '#F7F4EC'
  })
  assert.deepEqual(recorder.calls[2].options, { textStyle: 'dark' })
})

test('applyRuntimeTheme applies existing light tabBar colors on tabBar pages', () => {
  const recorder = createUniRecorder()

  applyRuntimeTheme({
    uniApi: recorder.uniApi,
    route: 'pages/index/index',
    isDarkMode: false
  })

  assert.deepEqual(recorder.calls.find((call) => call.method === 'setTabBarStyle'), {
    method: 'setTabBarStyle',
    options: {
      backgroundColor: '#F7F4EC',
      borderStyle: 'black',
      color: '#6E6242',
      selectedColor: '#E0AE12'
    }
  })
})

test('applyRuntimeTheme applies existing dark tabBar colors on tabBar pages', () => {
  const recorder = createUniRecorder()

  applyRuntimeTheme({
    uniApi: recorder.uniApi,
    route: 'pages/user/index',
    isDarkMode: true
  })

  assert.deepEqual(recorder.calls.find((call) => call.method === 'setTabBarStyle'), {
    method: 'setTabBarStyle',
    options: {
      backgroundColor: '#141109',
      borderStyle: 'white',
      color: '#c6b78c',
      selectedColor: '#E0AE12'
    }
  })

  assert.deepEqual(
    recorder.calls.filter((call) => call.method === 'setTabBarItem').map((call) => call.options),
    [
      {
        index: 0,
        iconPath: '/static/tabbar/index_dark.png',
        selectedIconPath: '/static/tabbar/index-selected_dark.png'
      },
      {
        index: 1,
        iconPath: '/static/tabbar/match_dark.png',
        selectedIconPath: '/static/tabbar/match-selected_dark.png'
      },
      {
        index: 2,
        iconPath: '/static/tabbar/social_dark.png',
        selectedIconPath: '/static/tabbar/social-selected_dark.png'
      },
      {
        index: 3,
        iconPath: '/static/tabbar/user_dark.png',
        selectedIconPath: '/static/tabbar/user-selected_dark.png'
      }
    ]
  )
})
