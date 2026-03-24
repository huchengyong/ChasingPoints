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

export function resolveWelcomeActions({ isHarmony }) {
  return {
    primaryText: '手机号登录 / 注册',
    secondaryText: '华为账号登录',
    tertiaryText: '先逛逛',
    showHuaweiLogin: Boolean(isHarmony)
  }
}
