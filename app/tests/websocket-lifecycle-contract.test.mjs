import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const read = (path) => readFileSync(new URL(path, import.meta.url), 'utf8')

const appSource = read('../App.vue')
const userStoreSource = read('../store/user.js')
const matchDetailSource = read('../subPages/match/matchDetail.vue')
const playingSource = read('../subPages/match/playing.vue')

test('app lifecycle pauses both realtime channels and resumes through foreground and network state', () => {
  assert.match(appSource, /import\s*\{\s*matchWS,\s*userWS,\s*WS_MESSAGE_TYPES\s*\}/)
  assert.match(appSource, /this\.bindNetworkStatusListener\(\)/)
  assert.match(appSource, /uni\.onNetworkStatusChange\(this\.networkStatusCallback\)/)
  assert.match(appSource, /userWS\.setNetworkOnline\(online\)/)
  assert.match(appSource, /matchWS\.setNetworkOnline\(online\)/)
  assert.match(appSource, /userWS\.setForeground\(true\)/)
  assert.match(appSource, /matchWS\.setForeground\(true\)/)
  assert.match(appSource, /userWS\.setForeground\(false\)/)
  assert.match(appSource, /matchWS\.setForeground\(false\)/)
  const onHideSource = appSource.match(/onHide:\s*function\(\)\s*\{([\s\S]*?)\n\t\t\},\n\t\tonUnload/)?.[1] || ''
  assert.ok(onHideSource.indexOf('userWS.setForeground(false)') < onHideSource.indexOf('userWS.disconnect()'))
})

test('match pages own their realtime target while visible and release it on hide or unmount', () => {
  for (const source of [matchDetailSource, playingSource]) {
    assert.match(source, /import\s*\{\s*onHide,\s*onLoad,\s*onShow\s*\}\s*from\s*'@dcloudio\/uni-app'/)
    assert.match(source, /onShow\(\(\)\s*=>\s*\{[\s\S]*matchWS\.setTargetActive\(true\)/)
    assert.match(source, /onHide\(\(\)\s*=>\s*\{\s*matchWS\.setTargetActive\(false\)/)
    assert.match(source, /onUnmounted\(\(\)\s*=>\s*\{[\s\S]*matchWS\.disconnect\(\)/)
  }
})

test('logout and account switch clear both realtime channels and bind new user connections to auth generation', () => {
  assert.match(userStoreSource, /import\s*\{\s*matchWS,\s*userWS\s*\}/)
  assert.match(userStoreSource, /matchSocket:\s*matchWS/)
  assert.match(userStoreSource, /userWS\.connect\(\{\s*authGeneration:\s*this\.authGeneration\s*\}\)/)
})
