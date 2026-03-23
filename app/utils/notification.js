import { shouldFetchAuthState } from './auth-guards.js'

export const shouldFetchUnreadCount = (isLoggedIn) => shouldFetchAuthState(isLoggedIn)
