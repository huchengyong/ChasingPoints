import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc'

const filename = new URL('../subPages/match/challengeWaiting.vue', import.meta.url)
const source = readFileSync(filename, 'utf8')

test('约球等待页能直接回应邀请，并在用户切换后拒绝旧详情写回', () => {
  const { descriptor, errors } = parse(source, { filename: filename.pathname })
  assert.equal(errors.length, 0)
  assert.doesNotThrow(() => compileScript(descriptor, { id: 'challenge-waiting' }))
  assert.equal(compileTemplate({ source: descriptor.template.content, filename: filename.pathname, id: 'challenge-waiting' }).errors.length, 0)
  assert.match(source, /challenge\.status === 0 && !expiredNow[\s\S]*?@tap="acceptInvite\(\)"[\s\S]*?@tap="rejectInvite"/)
  assert.match(source, /@tap="goMatch\(\)"/)
  assert.match(source, /confirm_close_own_pending: confirmed === true/)
  assert.match(source, /isSameIdentity\(identity\)/)
  assert.match(source, /userStore\.userId === userId && userStore\.authGeneration === generation/)
  assert.match(source, /getChallengeDetail\([\s\S]*?getChallengeSummary\(/)
  assert.match(source, /activityStore\.markDirty\(\)/)
  assert.match(source, /const res = await cancelChallenge\(\{ challenge_id: challengeId\.value \}\)\s*\n\s*if \(!isSameIdentity\(identity\)\) return/)
  assert.match(source, /const res = await abandonChallenge\(\{ challenge_id: challengeId\.value \}\)\s*\n\s*if \(!isSameIdentity\(identity\)\) return/)
  assert.match(source, /let pageActive = false/)
  assert.match(source, /isCurrent = \(\) => detailRequestId === requestId && userStore\.userId === userId && userStore\.authGeneration === generation && pageActive/)
  assert.match(source, /onHide\(pausePage\)/)
  assert.match(source, /onUnload\(pausePage\)/)
})
