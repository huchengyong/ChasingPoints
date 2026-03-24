import test from 'node:test'
import assert from 'node:assert/strict'

import {
  canRequestSms,
  canSubmitLogin,
  isCodeValid,
  isPhoneValid,
  resolveWelcomeActions
} from '../utils/entry-funnel.js'

test('resolveWelcomeActions hides Huawei login outside HarmonyOS', () => {
  const result = resolveWelcomeActions({ isHarmony: false })

  assert.deepEqual(result, {
    primaryText: '手机号登录 / 注册',
    secondaryText: '华为账号登录',
    tertiaryText: '先逛逛',
    showHuaweiLogin: false
  })
})

test('resolveWelcomeActions shows Huawei login on HarmonyOS', () => {
  const result = resolveWelcomeActions({ isHarmony: true })

  assert.equal(result.showHuaweiLogin, true)
  assert.equal(result.secondaryText, '华为账号登录')
})

test('isPhoneValid accepts mainland mobile numbers', () => {
  assert.equal(isPhoneValid('13800138000'), true)
})

test('isPhoneValid rejects invalid mobile numbers', () => {
  assert.equal(isPhoneValid('123'), false)
  assert.equal(isPhoneValid(''), false)
})

test('isCodeValid only accepts six digit sms codes', () => {
  assert.equal(isCodeValid('123456'), true)
  assert.equal(isCodeValid('12345'), false)
  assert.equal(isCodeValid('1234567'), false)
  assert.equal(isCodeValid('12a456'), false)
})

test('canRequestSms requires a valid phone and idle send state', () => {
  assert.equal(canRequestSms({
    phone: '13800138000',
    countdown: 0,
    isSending: false
  }), true)

  assert.equal(canRequestSms({
    phone: '13800138000',
    countdown: 10,
    isSending: false
  }), false)

  assert.equal(canRequestSms({
    phone: '13800138000',
    countdown: 0,
    isSending: true
  }), false)
})

test('canSubmitLogin requires valid phone, valid code, agreement, and idle submit state', () => {
  assert.equal(canSubmitLogin({
    phone: '13800138000',
    code: '123456',
    isAgreed: true,
    isLogging: false
  }), true)

  assert.equal(canSubmitLogin({
    phone: '13800138000',
    code: '123456',
    isAgreed: false,
    isLogging: false
  }), false)

  assert.equal(canSubmitLogin({
    phone: '13800138000',
    code: '12345',
    isAgreed: true,
    isLogging: false
  }), false)
})
