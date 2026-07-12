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

export function getWechatPhoneNumberCode(event) {
  const code = event?.detail?.code
  return typeof code === 'string' ? code.trim() : ''
}

export function resolveBindPhoneSuccess({ response, phone }) {
  const message = response?.message || '绑定成功'
  const returnedPhone = String(response?.user_info?.phone || '').trim()
  const maskedPhone = returnedPhone || maskBindPhone(phone)

  if (response?.merged_account) {
		if (response?.access_token && response?.refresh_token) {
			return {
				action: 'complete',
				message,
				maskedPhone,
				phone,
				sessionReplaced: true
			}
		}

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
