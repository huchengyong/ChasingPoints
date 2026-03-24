const PHONE_REGEXP = /^1[3-9]\d{9}$/
const CODE_REGEXP = /^\d{6}$/

export function isPhoneValid(phone) {
  return PHONE_REGEXP.test(String(phone || '').trim())
}

export function isCodeValid(code) {
  return CODE_REGEXP.test(String(code || '').trim())
}

export function canRequestSms({ phone, countdown, isSending }) {
  return isPhoneValid(phone) && countdown <= 0 && !isSending
}

export function canSubmitLogin({ phone, code, isAgreed, isLogging }) {
  return isPhoneValid(phone) && isCodeValid(code) && Boolean(isAgreed) && !isLogging
}

export function getPhoneError(phone) {
  const value = String(phone || '').trim()
  if (!value) {
    return ''
  }

  return isPhoneValid(value) ? '' : '请输入正确的手机号'
}

export function getCodeError(code) {
  const value = String(code || '').trim()
  if (!value) {
    return ''
  }

  return isCodeValid(value) ? '' : '请输入6位验证码'
}

export function maskPhone(phone) {
  const value = String(phone || '').trim()
  if (!isPhoneValid(value)) {
    return ''
  }

  return `${value.slice(0, 3)}****${value.slice(7)}`
}

export function resolveSmsFeedback(phone) {
  const maskedPhone = maskPhone(phone)
  if (!maskedPhone) {
    return '验证码已发送，请注意查收'
  }

  return `验证码已发送至 ${maskedPhone}`
}

export function resolvePostLoginNavigation({ pageCount }) {
  return Number(pageCount) > 1 ? 'back' : 'home'
}

export function resolveWelcomeActions({ isHarmony, isAgreed }) {
  const disabled = !Boolean(isAgreed)

  return {
    primaryText: '手机号登录 / 注册',
    secondaryText: '华为账号登录',
    tertiaryText: '先逛逛',
    showHuaweiLogin: Boolean(isHarmony),
    primaryDisabled: disabled,
    secondaryDisabled: disabled
  }
}
