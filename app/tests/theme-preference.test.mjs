import test from 'node:test'
import assert from 'node:assert/strict'

import {
  THEME_MODE_DARK,
  THEME_MODE_LIGHT,
  THEME_MODE_STORAGE_KEY,
  THEME_MODE_SYSTEM,
  normalizeThemeMode,
  readStoredThemeMode,
  resolveDarkMode,
  resolveSystemDarkMode
} from '../utils/theme-preference.js'

const createStorage = (values = {}) => ({
  getStorageSync(key) {
    return Object.hasOwn(values, key) ? values[key] : ''
  }
})

test('normalizeThemeMode accepts only system, light and dark', () => {
  assert.equal(normalizeThemeMode(THEME_MODE_SYSTEM), THEME_MODE_SYSTEM)
  assert.equal(normalizeThemeMode(THEME_MODE_LIGHT), THEME_MODE_LIGHT)
  assert.equal(normalizeThemeMode(THEME_MODE_DARK), THEME_MODE_DARK)
  assert.equal(normalizeThemeMode('unknown'), THEME_MODE_SYSTEM)
})

test('readStoredThemeMode prefers the current three-state preference', () => {
  assert.equal(readStoredThemeMode(createStorage({
    [THEME_MODE_STORAGE_KEY]: THEME_MODE_LIGHT,
    user_theme_dark: true
  })), THEME_MODE_LIGHT)
})

test('readStoredThemeMode migrates legacy boolean preferences', () => {
  assert.equal(readStoredThemeMode(createStorage({ user_theme_dark: true })), THEME_MODE_DARK)
  assert.equal(readStoredThemeMode(createStorage({ user_theme_dark: false })), THEME_MODE_LIGHT)
  assert.equal(readStoredThemeMode(createStorage()), THEME_MODE_SYSTEM)
})

test('resolveSystemDarkMode supports App, WeChat and Harmony theme fields', () => {
  assert.equal(resolveSystemDarkMode({ osTheme: 'dark' }), true)
  assert.equal(resolveSystemDarkMode({ theme: 'dark' }), true)
  assert.equal(resolveSystemDarkMode({ hostTheme: 'light' }, true), false)
  assert.equal(resolveSystemDarkMode({ hostTheme: 'light', osTheme: 'dark' }), false)
  assert.equal(resolveSystemDarkMode({}, true), true)
})

test('resolveDarkMode lets manual modes override later system changes', () => {
  assert.equal(resolveDarkMode(THEME_MODE_SYSTEM, true), true)
  assert.equal(resolveDarkMode(THEME_MODE_SYSTEM, false), false)
  assert.equal(resolveDarkMode(THEME_MODE_LIGHT, true), false)
  assert.equal(resolveDarkMode(THEME_MODE_DARK, false), true)
})
