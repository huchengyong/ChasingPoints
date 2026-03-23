const PHONE_REGEXP = /^1[3-9]\d{9}$/

export function isValidBindPhone(phone) {
  return PHONE_REGEXP.test(phone)
}

export function maskBindPhone(phone) {
  if (!isValidBindPhone(phone)) {
    return ''
  }

  return `${phone.slice(0, 3)}****${phone.slice(7)}`
}

export function canRequestBindPhoneSms({ phone, countdown, isSending }) {
  return isValidBindPhone(phone) && countdown <= 0 && !isSending
}

export function shouldResetBindPhoneVerification(previousPhone, nextPhone) {
  return Boolean(previousPhone) && Boolean(nextPhone) && previousPhone !== nextPhone
}

export function resolveBindPhoneSuccess({ response, phone }) {
  const message = response?.message || '绑定成功'
  const maskedPhone = maskBindPhone(phone)

  if (response?.merged_account) {
    return {
      action: 'relogin',
      message,
      maskedPhone,
      phone
    }
  }

  return {
    action: 'complete',
    message,
    maskedPhone,
    phone
  }
}
