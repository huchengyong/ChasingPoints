import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const appSource = readFileSync(new URL('../App.vue', import.meta.url), 'utf8')
const requestSource = readFileSync(new URL('../utils/request.js', import.meta.url), 'utf8')
const restoreSource = appSource.slice(
  appSource.indexOf('restoreUserSession()'),
  appSource.indexOf('hasValidatedAppSession()')
)
const onShowSource = appSource.slice(
  appSource.indexOf('onShow: function()'),
  appSource.indexOf('onHide: function()')
)
const onHideSource = appSource.slice(
  appSource.indexOf('onHide: function()'),
  appSource.indexOf('methods: {')
)
const pushRegistrationSource = appSource.slice(
  appSource.indexOf('uni.getPushClientId'),
  appSource.indexOf('uni.onPushMessage')
)
const ongoingMatchSource = appSource.slice(
  appSource.indexOf('checkOngoingMatchReminder() {')
)

test('App performs a silent Bootstrap validation', () => {
  assert.match(appSource, /createSessionRecovery/)
  assert.match(appSource, /getUserBootstrap\(\{ silent: true \}\)/)
  assert.doesNotMatch(appSource, /applyUserInfo:/)
})

test('App applies recovery data only after auth-generation and foreground checks', () => {
  assert.match(restoreSource, /const expectedGeneration = userStore\.authGeneration/)
  assert.match(restoreSource, /const expectedLifecycleGeneration = this\.sessionRecoveryLifecycle/)
  assert.match(restoreSource, /sessionRecovery\.validate\(expectedGeneration\)/)
  assert.match(restoreSource, /canApplySessionRecoveryResult/)
  assert.match(restoreSource, /currentGeneration: currentUserStore\.authGeneration/)
  assert.match(restoreSource, /isForeground: this\.appIsForeground/)
  assert.match(restoreSource, /currentUserStore\.updateUserInfo\(result\.userInfo\)/)
  assert.match(restoreSource, /useActivityStore\(\)\.applyBootstrap\(identity, result\.bootstrap\)/)
  assert.match(restoreSource, /this\.applyBootstrapCompetitiveRevision\(identity, result\.bootstrap\)/)

  const guardIndex = restoreSource.indexOf('canApplySessionRecoveryResult')
  const updateIndex = restoreSource.indexOf('updateUserInfo(result.userInfo)')
  const connectIndex = restoreSource.indexOf('this.connectUserWS()')
  assert.ok(guardIndex >= 0 && guardIndex < updateIndex)
  assert.ok(updateIndex < connectIndex)
})

test('App invalidates delayed recovery callbacks when it goes into the background', () => {
  assert.match(onHideSource, /this\.appIsForeground = false/)
  assert.match(onHideSource, /this\.sessionRecoveryLifecycle \+= 1/)
  assert.match(onHideSource, /this\.validatedAuthGeneration = -1/)
  assert.ok(onHideSource.indexOf('this.appIsForeground = false') < onHideSource.indexOf('userWS.disconnect()'))
})

test('login completion asks App to bootstrap the new session', () => {
  const userStoreSource = readFileSync(new URL('../store/user.js', import.meta.url), 'utf8')
  assert.match(userStoreSource, /uni\.\$emit\('user-session-ready'\)/)
  assert.match(appSource, /uni\.\$on\('user-session-ready', this\.userSessionReadyCallback\)/)
})

test('push upload and ongoing-match checks wait for successful validation', () => {
  assert.doesNotMatch(onShowSource, /this\.checkOngoingMatchReminder\(\)/)
  assert.doesNotMatch(pushRegistrationSource, /post\('\/api\/user\/push-token'/)
  assert.match(pushRegistrationSource, /this\.uploadPushTokenIfValidated\(\)/)
  assert.match(restoreSource, /this\.uploadPushTokenIfValidated\(\)/)
  assert.match(restoreSource, /this\.checkOngoingMatchReminder\(\)/)
  assert.match(appSource, /if \(!this\.hasValidatedAppSession\(\)/)
  assert.match(appSource, /const expectedGeneration = userStore\.authGeneration/)
  assert.match(appSource, /currentGeneration: currentUserStore\.authGeneration/)
})

test('ongoing-match results are scoped to auth and App lifecycle generations', () => {
  assert.match(ongoingMatchSource, /const expectedGeneration = userStore\.authGeneration/)
  assert.match(ongoingMatchSource, /const expectedLifecycleGeneration = this\.sessionRecoveryLifecycle/)
  assert.match(ongoingMatchSource, /canApplySessionRecoveryResult/)
  assert.match(ongoingMatchSource, /currentLifecycleGeneration: this\.sessionRecoveryLifecycle/)
  assert.match(ongoingMatchSource, /ongoingMatchReminderPendingKey/)
})

test('App delegates session invalid navigation to the token-scoped request layer', () => {
  assert.doesNotMatch(appSource, /reLaunch\(\{\s*url: '\/pages\/login\/login'/)
  assert.match(requestSource, /createSessionInvalidHandler/)
  assert.match(requestSource, /sessionKey:\s*requestToken/)
  assert.match(requestSource, /uni\.reLaunch\(\{\s*url\s*\}\)/)
})
