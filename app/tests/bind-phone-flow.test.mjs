import test from 'node:test'
import assert from 'node:assert/strict'

import {
  canCloseBindPhone,
  canRequestBindPhoneSms,
  resolveBindPhoneSuccess,
  shouldResetBindPhoneVerification
} from '../utils/bind-phone-flow.js'

test('canCloseBindPhone blocks closing only while a binding request is active', () => {
  assert.equal(canCloseBindPhone({ isBinding: false }), true)
  assert.equal(canCloseBindPhone({ isBinding: true }), false)
})

test('resolveBindPhoneSuccess returns relogin outcome for merged account responses', () => {
  const result = resolveBindPhoneSuccess({
    response: {
      success: true,
      message: '账号已合并，请使用手机号登录',
      merged_account: true
    },
    phone: '13800138000'
  })

  assert.deepEqual(result, {
    action: 'relogin',
    message: '账号已合并，请使用手机号登录',
    maskedPhone: '138****8000',
    phone: '13800138000'
  })
})

test('resolveBindPhoneSuccess replaces session when merge returns target user tokens', () => {
  const result = resolveBindPhoneSuccess({
    response: {
      success: true,
      message: '账号已合并',
      merged_account: true,
      access_token: 'new-access',
      refresh_token: 'new-refresh',
      user_info: { id: 2002, phone: '138****8000' }
    },
    phone: ''
  })

  assert.equal(result.sessionReplaced, true)
  assert.equal(result.action, 'complete')
  assert.equal(result.message, '账号已合并')
  assert.doesNotMatch(result.message, /登录/)
  assert.equal(result.maskedPhone, '138****8000')
})

test('resolveBindPhoneSuccess falls back to relogin when merged account lacks tokens', () => {
  const result = resolveBindPhoneSuccess({
    response: {
      success: true,
      message: '账号已合并，请使用手机号登录',
      merged_account: true,
      access_token: '',
      refresh_token: ''
    },
    phone: '13800138000'
  })

  assert.deepEqual(result, {
    action: 'relogin',
    message: '账号已合并，请使用手机号登录',
    maskedPhone: '138****8000',
    phone: '13800138000'
  })
})

test('resolveBindPhoneSuccess returns complete outcome for normal bindings', () => {
  const result = resolveBindPhoneSuccess({
    response: {
      success: true,
      message: '绑定成功',
      merged_account: false
    },
    phone: '13912345678'
  })

  assert.deepEqual(result, {
    action: 'complete',
    message: '绑定成功',
    maskedPhone: '139****5678',
    phone: '13912345678'
  })
})

test('canRequestBindPhoneSms blocks repeated send while request is in flight', () => {
  assert.equal(canRequestBindPhoneSms({
    phone: '13800138000',
    countdown: 0,
    isSending: true
  }), false)
})

test('shouldResetBindPhoneVerification resets verification state when phone changes after code was requested', () => {
  assert.equal(shouldResetBindPhoneVerification('13800138000', '13900139000'), true)
})
