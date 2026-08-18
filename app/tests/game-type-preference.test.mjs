import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  DEFAULT_GAME_TYPE_OPTIONS,
  getDefaultGameTypeStorageKey,
  normalizeDefaultGameType,
  readDefaultGameType,
  saveDefaultGameType
} from '../utils/game-type-preference.js'

const createStorage = () => {
  const values = new Map()
  return {
    values,
    getStorageSync: (key) => values.get(key),
    setStorageSync: (key, value) => values.set(key, value),
    removeStorageSync: (key) => values.delete(key)
  }
}

test('default game type accepts only supported values', () => {
  assert.equal(normalizeDefaultGameType(1), 1)
  assert.equal(normalizeDefaultGameType('3'), 3)
  assert.equal(normalizeDefaultGameType(0), 0)
  assert.equal(normalizeDefaultGameType(99), 0)
  assert.deepEqual(DEFAULT_GAME_TYPE_OPTIONS.map((item) => item.value), [0, 3, 2, 1, 4])
})

test('default game type storage is isolated by user id', () => {
  const storage = createStorage()
  saveDefaultGameType(storage, 101, 3)
  saveDefaultGameType(storage, 202, 2)

  assert.equal(readDefaultGameType(storage, 101), 3)
  assert.equal(readDefaultGameType(storage, 202), 2)
  assert.notEqual(getDefaultGameTypeStorageKey(101), getDefaultGameTypeStorageKey(202))
})

test('every-time choice clears the stored preference', () => {
  const storage = createStorage()
  saveDefaultGameType(storage, 101, 1)
  assert.equal(readDefaultGameType(storage, 101), 1)

  saveDefaultGameType(storage, 101, 0)
  assert.equal(readDefaultGameType(storage, 101), 0)
  assert.equal(storage.values.has(getDefaultGameTypeStorageKey(101)), false)
})

test('invalid stored values safely fall back to every-time choice', () => {
	const storage = createStorage()
	assert.equal(readDefaultGameType(storage, 101), 0)
	storage.setStorageSync(getDefaultGameTypeStorageKey(101), 88)
	assert.equal(readDefaultGameType(storage, 101), 0)
	assert.equal(readDefaultGameType(storage, 0), 0)
})

test('a saved preference remains after the following scan flow is cancelled', () => {
	const storage = createStorage()
	saveDefaultGameType(storage, 101, 3)

	// 扫码取消不写偏好，已明确保存的默认值应继续存在。
	assert.equal(readDefaultGameType(storage, 101), 3)
})

test('settings and all direct PK entries use the shared preference helper', () => {
  const settings = readFileSync(new URL('../subPages/user/settings.vue', import.meta.url), 'utf8')
  assert.match(settings, /DEFAULT_GAME_TYPE_OPTIONS/)
  assert.match(settings, /每次询问/)
  assert.match(settings, /saveDefaultGameType/)

	for (const path of ['../pages/index/index.vue', '../pages/match/index.vue', '../pages/ranking/index.vue', '../pages/user/index.vue']) {
		const page = readFileSync(new URL(path, import.meta.url), 'utf8')
		assert.match(page, /:default-type="defaultGameType"/)
		assert.match(page, /readDefaultGameType/)
		assert.match(page, /saveDefaultGameType/)
		const handlerStart = page.indexOf('const handleGameTypeConfirm')
		const saveIndex = page.indexOf('defaultGameType.value = saveDefaultGameType', handlerStart)
		const selectionIndex = page.indexOf('selectedGameType.value = gameType', handlerStart)
		assert.ok(handlerStart >= 0 && saveIndex > handlerStart && saveIndex < selectionIndex)
	}
})
