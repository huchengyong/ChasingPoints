import test from 'node:test'
import assert from 'node:assert/strict'

import {
  canAttemptLogin,
  canRequestSms,
  canSubmitLogin,
  getCodeError,
  getPhoneError,
  isCodeValid,
  isPhoneValid,
  maskPhone,
  resolvePostLoginNavigation,
  resolveSmsFeedback,
  resolveWelcomeActions,
  shouldClearPendingPostLoginIntent
} from '../utils/entry-funnel.js'

test('resolveWelcomeActions hides Huawei login outside HarmonyOS', () => {
  const result = resolveWelcomeActions({ isHarmony: false, isAgreed: false })

  assert.deepEqual(result, {
    primaryText: '手机号登录 / 注册',
    secondaryText: '华为账号登录',
    tertiaryText: '先逛逛',
    showHuaweiLogin: false,
    primaryDisabled: false,
    secondaryDisabled: false
  })
})

test('resolveWelcomeActions shows Huawei login on HarmonyOS', () => {
  const result = resolveWelcomeActions({ isHarmony: true, isAgreed: true })

  assert.equal(result.showHuaweiLogin, true)
  assert.equal(result.secondaryText, '华为账号登录')
  assert.equal(result.primaryDisabled, false)
  assert.equal(result.secondaryDisabled, false)
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

test('canAttemptLogin ignores agreement state and only depends on form readiness', () => {
  assert.equal(canAttemptLogin({
    phone: '13800138000',
    code: '123456',
    isLogging: false
  }), true)

  assert.equal(canAttemptLogin({
    phone: '13800138000',
    code: '12345',
    isLogging: false
  }), false)
})

test('getPhoneError only surfaces a message after the user starts typing an invalid number', () => {
  assert.equal(getPhoneError(''), '')
  assert.equal(getPhoneError('13800138000'), '')
  assert.equal(getPhoneError('1380013'), '请输入正确的手机号')
})

test('getCodeError only surfaces a message after the user starts typing an invalid code', () => {
  assert.equal(getCodeError(''), '')
  assert.equal(getCodeError('123456'), '')
  assert.equal(getCodeError('1234'), '请输入6位验证码')
})

test('maskPhone returns a masked number for valid phones only', () => {
  assert.equal(maskPhone('13800138000'), '138****8000')
  assert.equal(maskPhone('123'), '')
})

test('resolveSmsFeedback prefers a masked destination when the phone is valid', () => {
  assert.equal(resolveSmsFeedback('13800138000'), '验证码已发送至 138****8000')
  assert.equal(resolveSmsFeedback('bad phone'), '验证码已发送，请注意查收')
})

test('resolvePostLoginNavigation returns back when the user came from another page', () => {
  assert.equal(resolvePostLoginNavigation({ pageCount: 2, pendingAction: '' }), 'back')
  assert.equal(resolvePostLoginNavigation({ pageCount: 1, pendingAction: '' }), 'home')
})

test('resolvePostLoginNavigation prefers pending post-login actions over default back behavior', () => {
  assert.equal(resolvePostLoginNavigation({
    pageCount: 3,
    pendingAction: 'start_pk'
  }), 'intent')
})

test('shouldClearPendingPostLoginIntent only clears abandoned pending actions on login exit', () => {
  assert.equal(shouldClearPendingPostLoginIntent({
    hasPendingIntent: true,
    completedLoginFlow: false
  }), true)

  assert.equal(shouldClearPendingPostLoginIntent({
    hasPendingIntent: true,
    completedLoginFlow: true
  }), false)

  assert.equal(shouldClearPendingPostLoginIntent({
    hasPendingIntent: false,
    completedLoginFlow: false
  }), false)
})
