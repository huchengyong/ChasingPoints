export const POST_LOGIN_INTENT_KEY = 'post_login_intent'

export const POST_LOGIN_ACTIONS = {
  START_PK: 'start_pk',
  SHOW_PK_CODE: 'show_pk_code'
}

const getUniStorage = () => globalThis.uni || null

export const normalizePostLoginIntent = (action) => {
  if (action === POST_LOGIN_ACTIONS.START_PK) return POST_LOGIN_ACTIONS.START_PK
  if (action === POST_LOGIN_ACTIONS.SHOW_PK_CODE) return POST_LOGIN_ACTIONS.SHOW_PK_CODE
  return ''
}

export const clearPostLoginIntent = () => {
  const storage = getUniStorage()
  if (!storage?.removeStorageSync) return
  storage.removeStorageSync(POST_LOGIN_INTENT_KEY)
}

export const setPostLoginIntent = (action) => {
  const storage = getUniStorage()
  const normalizedAction = normalizePostLoginIntent(action)

  if (!storage?.setStorageSync) {
    return normalizedAction
  }

  if (!normalizedAction) {
    clearPostLoginIntent()
    return ''
  }

  storage.setStorageSync(POST_LOGIN_INTENT_KEY, normalizedAction)
  return normalizedAction
}

export const getPostLoginIntent = () => {
  const storage = getUniStorage()
  if (!storage?.getStorageSync) return ''

  const normalizedAction = normalizePostLoginIntent(storage.getStorageSync(POST_LOGIN_INTENT_KEY))
  if (normalizedAction) {
    return normalizedAction
  }

  clearPostLoginIntent()
  return ''
}

export const consumePostLoginIntent = () => {
  const intent = getPostLoginIntent()
  if (intent) {
    clearPostLoginIntent()
  }
  return intent
}
