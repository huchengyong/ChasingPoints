import test from 'node:test'
import assert from 'node:assert/strict'

import {
  resolvePkEntryActions,
  resolveQrCodeModalCopy
} from '../utils/pk-entry-actions.js'

test('resolvePkEntryActions exposes start and show-code labels for logged-in users', () => {
  assert.deepEqual(resolvePkEntryActions({ isLoggedIn: true }), {
    primaryText: '发起PK',
    secondaryText: '出示PK码'
  })
})

test('resolvePkEntryActions keeps the same pair for guests because copy is role-based', () => {
  assert.deepEqual(resolvePkEntryActions({ isLoggedIn: false }), {
    primaryText: '发起PK',
    secondaryText: '出示PK码'
  })
})

test('resolveQrCodeModalCopy explains how the opponent should use the code', () => {
  assert.deepEqual(resolveQrCodeModalCopy(), {
    title: '出示PK码',
    hint: '让对手打开“发起PK”后扫描此码，即可开始匹配',
    helper: '如果对手找不到入口，请让他前往“我的”页点击“发起PK”'
  })
})
