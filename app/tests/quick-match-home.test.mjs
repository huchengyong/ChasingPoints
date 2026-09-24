import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const appRoot = new URL('..', import.meta.url)
const readAppFile = (path) => readFileSync(new URL(path, appRoot), 'utf8')

test('对局首页恢复「再来一局」扫码上下文消费', () => {
  const source = readAppFile('pages/match/index.vue')
  assert.match(source, /consumePendingMatchContext\(\)/)
  assert.match(source, /normalizePendingMatchContext\('pending_match_rematch', raw\)/)
  assert.match(source, /uni\.removeStorageSync\('pending_match_rematch'\)/)
  assert.match(readAppFile('subPages/match/matchResult.vue'), /pending_match_rematch/)
})

test('对局首页回源必须经过活动 Store 的 dirty 与 TTL 判断', () => {
  const source = readAppFile('pages/match/index.vue')
  assert.match(source, /activityStore\.shouldFetch\(\)/)
  assert.doesNotMatch(source, /activityStore\.dirty\)/)
})

test('约球区块读取失败时两个主 Tab 提供重试入口', () => {
  assert.match(readAppFile('pages/match/index.vue'), /challenge-unavailable[\s\S]*?retryChallengeActivity/)
  assert.match(readAppFile('pages/user/index.vue'), /challenge-status-unavailable[\s\S]*?retryChallengeActivity/)
  assert.match(readAppFile('store/activity.js'), /challengeAvailable/)
})

test('实际预约日是唯一日期状态，改选日期立即生效', () => {
  const compose = readAppFile('subPages/match/challengeCompose.vue')
  assert.match(compose, /from=challenge/)
  assert.match(compose, /`match_mode=\$\{matchMode\.value\}`/)
  assert.match(compose, /`scheduled_date=\$\{scheduledDate\.value\}`/)
  assert.match(compose, /const scheduledDate = ref\(''\)/)
  assert.match(compose, /const activeDayOffset = computed\(\(\) => dayOffsetOfScheduledDate\(serverTime\.value, scheduledDate\.value\)\)/)
  assert.match(compose, /active: activeDayOffset === item\.value/)
  assert.match(compose, /chooseDay = \(value\) => \{[\s\S]*?computeScheduledDate\(serverTime\.value, value\)[\s\S]*?scheduledDate\.value = date/)
  assert.match(compose, /dayOffsetOfScheduledDate\(serverTime\.value, scheduledDate\.value\)/)
  assert.doesNotMatch(compose, /preservedDate/)
  const selector = readAppFile('subPages/match/opponentSelector.vue')
  assert.match(selector, /formPrefill\.value/)
  assert.match(selector, /`match_mode=\$\{query\.match_mode \|\| 'ranked'\}`/)
  assert.match(selector, /scheduled_date=\$\{query\.scheduled_date\}/)
})

test('「查看并取消」直达约球详情，主 Tab 入口明确指定 Tab', () => {
  const compose = readAppFile('subPages/match/challengeCompose.vue')
  assert.match(compose, /challengeWaiting\?challenge_id=\$\{res\.pending_challenge_id\}/)
  const home = readAppFile('pages/match/index.vue')
  assert.match(home, /goChallenges\('history'\)/)
  assert.match(home, /goChallenges\('received'\)/)
})

test('比赛终态在本机立即失效约球活动卡', () => {
  const playing = readAppFile('subPages/match/playing.vue')
  assert.match(playing, /import \{ useActivityStore \} from '@\/store\/activity\.js'/)
  assert.match(playing, /activityStore\.markDirty\(\)/)
})

test('历史状态按账号隔离，活动失败不遮蔽历史', () => {
  const source = readAppFile('subPages/social/challenges.vue')
  assert.match(source, /tab\.value === 'history' \? historyPageError\.value : challengePageError\.value/)
  assert.match(source, /historyState\.value\.authGeneration !== userStore\.authGeneration/)
  assert.match(source, /activeFailureBanner/)
  assert.match(source, /tab\.value === 'history'\s*\n\s*\? historyState\.value\.status === ASYNC_PAGE_STATUS\.LOADING/)
})

test('发起页回前台刷新服务端时间但不重置用户选择', () => {
  const source = readAppFile('subPages/match/challengeCompose.vue')
  assert.match(source, /onShow\(\(\) => \{[\s\S]{0,160}?loadServerTime\(\)/)
  assert.match(source, /if \(current && !slotInitialized\) \{/)
})

test('等待页确认弹窗分支同样受页面生命周期保护', () => {
  const source = readAppFile('subPages/match/challengeWaiting.vue')
  assert.match(source, /need_confirm_close_own[\s\S]{0,160}if \(!pageActive\) return/)
  assert.match(source, /type_conflict\)[\s\S]{0,120}if \(!pageActive\) return/)
})

test('发送结果、时间基准与弹窗回调按身份和生命周期隔离', () => {
  const compose = readAppFile('subPages/match/challengeCompose.vue')
  assert.match(compose, /import \{ onLoad, onShow, onHide, onUnload \} from '@dcloudio\/uni-app'/)
  assert.match(compose, /const isSameIdentity = \(identity\) =>/)
  assert.match(compose, /发送结果按发起时的身份与页面世代隔离：切号或离开后旧响应不弹窗、不标脏、不导航。/)
  assert.match(compose, /let pageGeneration = 0/)
  assert.match(compose, /pageGeneration \+= 1/)
  assert.match(compose, /const generation = pageGeneration/)
  assert.match(compose, /const refreshed = await refreshServerTimeForSubmit\(\)/)
  assert.match(compose, /if \(!refreshed \|\| !isSameIdentity\(identity\) \|\| generation !== pageGeneration\) \{/)
  assert.match(compose, /submitting\.value = true\s*\n\tconst identity = readIdentity\(\)/)
  assert.match(compose, /if \(!isSameIdentity\(identity\) \|\| generation !== pageGeneration\) return/)
  assert.match(compose, /if \(!pageActive \|\| !isSameIdentity\(identity\)\) return/)
  assert.match(compose, /const requestId = \+\+serverTimeRequestSeq/)
  assert.match(compose, /if \(requestId !== serverTimeRequestSeq\) return false/)
  assert.match(compose, /if \(current && !slotInitialized\) \{/)
  assert.match(compose, /confirm && pageActive && isSameIdentity\(identity\)/)
  const waiting = readAppFile('subPages/match/challengeWaiting.vue')
  assert.match(waiting, /confirm && pageActive && isSameIdentity\(identity\)/)
  assert.match(waiting, /!confirm \|\| !pageActive \|\| acting\.value \|\| !isSameIdentity\(identity\)/)
  assert.match(waiting, /提前确认后的失败（如约球已失效）必须给出原因并立即回源。/)
  assert.match(waiting, /if \(!res\?\.success\) \{[\s\S]{0,200}loadDetail\(\)/)
})

test('切号与写入都会让历史缓存失效回源', () => {
  const source = readAppFile('subPages/social/challenges.vue')
  assert.match(source, /const historyDirty = ref\(true\)/)
  assert.match(source, /const historyDirtySeq = ref\(0\)/)
  assert.match(source, /historyState\.value\.authGeneration !== userStore\.authGeneration \|\| historyDirty\.value/)
  assert.match(source, /const markActivityDirty = \(\) => \{\s*\n\s*activityStore\.markDirty\(\)\s*\n\s*markHistoryDirty\(\)/)
  assert.match(source, /historyDirty\.value = historyDirtySeq\.value > startDirtySeq/)
  assert.match(source, /challengeState\.value\.authGeneration !== userStore\.authGeneration \|\|\s*\n\t\thistoryState\.value\.authGeneration !== userStore\.authGeneration/)
})

test('列表页写响应与前台恢复都受页面生命周期约束', () => {
  const source = readAppFile('subPages/social/challenges.vue')
  assert.match(source, /import \{ onShow, onHide, onUnload \} from '@dcloudio\/uni-app'/)
  assert.match(source, /let pageActive = true/)
  assert.match(source, /onHide\(\(\) => \{\s*\n\s*pageActive = false/)
  assert.match(source, /markActivityDirty\(\)\s*\n\s*if \(!pageActive\) return/)
  assert.match(source, /onShow\(\(\) => \{\s*\n\s*pageActive = true/)
  assert.match(source, /onShow\(\(\) => \{[\s\S]{0,420}markHistoryDirty\(\)/)
  assert.match(source, /if \(confirm && pageActive\) action\(\)/)
})
