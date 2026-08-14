import test from 'node:test'
import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'

const manifestSource = readFileSync(new URL('../manifest.json', import.meta.url), 'utf8')
const appSource = readFileSync(new URL('../App.vue', import.meta.url), 'utf8')
const storeSource = readFileSync(new URL('../store/theme.js', import.meta.url), 'utf8')
const pagesConfig = JSON.parse(readFileSync(new URL('../pages.json', import.meta.url), 'utf8'))
const themeConfig = JSON.parse(readFileSync(new URL('../theme.json', import.meta.url), 'utf8'))

test('App, WeChat mini program and Harmony builds enable native dark mode themes', () => {
  for (const platform of ['app-plus', 'mp-weixin', 'app-harmony']) {
    const start = manifestSource.indexOf(`"${platform}"`)
    assert.notEqual(start, -1, platform)
    const platformSource = manifestSource.slice(start, start + 1200)
    assert.match(platformSource, /"darkmode"\s*:\s*true/, platform)
    assert.match(platformSource, /"themeLocation"\s*:\s*"theme\.json"/, platform)
  }
})

test('system theme listeners use cross-platform fields and tolerate unsupported APIs', () => {
  assert.match(appSource, /resolveSystemDarkMode/)
  assert.match(appSource, /typeof uni\.onThemeChange === 'function'/)
  assert.match(appSource, /typeof uni\.offThemeChange === 'function'/)
})

test('manual theme preference is stored separately from the current system theme', () => {
  assert.match(storeSource, /themeMode:/)
  assert.match(storeSource, /systemIsDark:/)
  assert.match(storeSource, /setThemeMode\(mode\)/)
  assert.match(storeSource, /setThemeFromSystem\(isDark\)/)
  assert.doesNotMatch(storeSource, /this\.themeMode\s*=\s*isDark/)
})

test('global theme variables are class-driven instead of media-query-driven', () => {
  assert.match(appSource, /\.dark-mode\s*\{[\s\S]*--ui-surface-page:\s*#141109;/)
  assert.doesNotMatch(appSource, /prefers-color-scheme/)
})

test('every pages.json theme token exists in both themes and referenced tab icons exist', () => {
  const tokens = []
  const collectTokens = (value) => {
    if (typeof value === 'string' && /^@[A-Za-z0-9_]+$/.test(value)) {
      tokens.push(value.slice(1))
      return
    }
    if (Array.isArray(value)) {
      value.forEach(collectTokens)
      return
    }
    if (value && typeof value === 'object') {
      Object.values(value).forEach(collectTokens)
    }
  }
  collectTokens(pagesConfig)

  for (const token of new Set(tokens)) {
    assert.ok(Object.hasOwn(themeConfig.light, token), `light.${token}`)
    assert.ok(Object.hasOwn(themeConfig.dark, token), `dark.${token}`)

    if (token.endsWith('IconPath')) {
      for (const theme of ['light', 'dark']) {
        const relativePath = themeConfig[theme][token].replace(/^\//, '')
        assert.equal(existsSync(new URL(`../${relativePath}`, import.meta.url)), true, `${theme}.${token}`)
      }
    }
  }
})

test('native, CSS and runtime theme entry points share the warm-neutral brand palette', () => {
  const runtimeThemeSource = readFileSync(new URL('../utils/theme-application.js', import.meta.url), 'utf8')

  assert.equal(themeConfig.light.primary, '#E0AE12')
  assert.equal(themeConfig.light.background, '#F7F4EC')
  assert.equal(themeConfig.light.textPrimary, '#231C0B')
  assert.equal(themeConfig.dark.background, '#141109')
  assert.equal(themeConfig.dark.textPrimary, '#fff7e1')

  assert.match(appSource, /--ui-brand-primary:\s*#E0AE12;/)
  assert.match(appSource, /--ui-surface-page:\s*#F7F4EC;/)
  assert.match(appSource, /--ui-text-primary:\s*#231C0B;/)

  assert.match(runtimeThemeSource, /backgroundColor:\s*'#F7F4EC'/)
  assert.match(runtimeThemeSource, /backgroundColor:\s*'#141109'/)
  assert.match(runtimeThemeSource, /selectedColor:\s*'#E0AE12'/)
})
