const PHONE_REGEXP = /^1[3-9]\d{9}$/
const CODE_REGEXP = /^\d{6}$/
export const AUTH_SLOW_FEEDBACK_DELAY = 1200
const ENTRY_FUNNEL_ROUTES = new Set([
  'pages/welcome/index',
  'pages/login/login'
])

export function resolveLoginMode({ isWechatMini, requestedMethod = '' } = {}) {
  if (!isWechatMini) {
    return 'phone'
  }

  return requestedMethod === 'phone' ? 'phone' : 'wechat'
}

export function resolveAlternateLoginMode(currentMode) {
  return currentMode === 'wechat' ? 'phone' : 'wechat'
}

export function resolveAuthenticationGate({
  isWechatLogging = false,
  isPhoneLogging = false,
  isHuaweiLogging = false,
  isPageActive = true
} = {}) {
  const isAuthenticating = Boolean(isWechatLogging || isPhoneLogging || isHuaweiLogging)
  const canInteract = Boolean(isPageActive) && !isAuthenticating

  return {
    isAuthenticating,
    canStart: canInteract,
    canLeave: canInteract,
    shouldHandleResult: Boolean(isPageActive)
  }
}

export function resolveAuthenticationPhase({
  isAuthenticating = false,
  elapsedMs = 0
} = {}) {
  if (!isAuthenticating) {
    return 'idle'
  }

  return Number(elapsedMs) >= AUTH_SLOW_FEEDBACK_DELAY ? 'slow' : 'pending'
}

export function resolveWechatPostLoginState({
  needBindPhone,
  bindingSkipped = false,
  bindingCompleted = false
} = {}) {
  const remainsUnbound = Boolean(needBindPhone) && !bindingCompleted

  return {
    action: remainsUnbound && !bindingSkipped ? 'bind-phone' : 'navigate',
    needBindPhone: remainsUnbound
  }
}

export function isPhoneValid(phone) {
  return PHONE_REGEXP.test(String(phone || '').trim())
}

export function isCodeValid(code) {
  return CODE_REGEXP.test(String(code || '').trim())
}

export function canRequestSms({ phone, countdown, isSending }) {
  return isPhoneValid(phone) && countdown <= 0 && !isSending
}

export function canAttemptLogin({ phone, code, isLogging }) {
  return isPhoneValid(phone) && isCodeValid(code) && !isLogging
}

export function canAttemptWechatMiniLogin({ isAgreed, isLogging }) {
  return Boolean(isAgreed) && !isLogging
}

export function canSubmitLogin({ phone, code, isAgreed, isLogging }) {
  return canAttemptLogin({ phone, code, isLogging }) && Boolean(isAgreed)
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

export function resolvePostLoginNavigation({ pageCount, pendingAction = '' }) {
  if (pendingAction) {
    return 'intent'
  }

  return Number(pageCount) > 1 ? 'back' : 'home'
}

export function shouldClearPendingPostLoginIntent({ hasPendingIntent, completedLoginFlow }) {
  return Boolean(hasPendingIntent) && !completedLoginFlow
}

export function resolveEntryFunnelAgreementState({ storedAgreement, sessionActive }) {
  if (!sessionActive) {
    return false
  }

  return Boolean(storedAgreement)
}

export function shouldClearEntryFunnelAgreementSession({ currentRoute, visibleRoutes = [] }) {
  return !visibleRoutes.some((route) => route !== currentRoute && ENTRY_FUNNEL_ROUTES.has(route))
}

export function resolveWelcomeActions({
  isHarmony,
  isWechatMini = false,
  isAgreed,
  isLogging = false,
  isSlowLogging = false
}) {
  if (isWechatMini) {
    return {
      primaryText: isLogging
        ? (isSlowLogging ? '网络稍慢，正在继续…' : '正在安全登录…')
        : '微信一键进入',
      secondaryText: '手机号登录',
      tertiaryText: '先逛逛',
      showHuaweiLogin: false,
      showPhoneLogin: true,
      primaryDisabled: Boolean(isLogging),
      secondaryDisabled: false
    }
  }

  return {
    primaryText: '手机号登录 / 注册',
    secondaryText: '华为账号登录',
    tertiaryText: '先逛逛',
    showHuaweiLogin: Boolean(isHarmony),
    primaryDisabled: false,
    secondaryDisabled: false
  }
}
