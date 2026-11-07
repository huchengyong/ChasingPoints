import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import { compileScript, parse } from '@vue/compiler-sfc'

import {
	FINISH_CONFIRM_SECONDS,
	buildFinishSlideNote,
	captureFinishSession,
	createFinishCountdownAnchor,
	finishCountdownSecondsLeft,
	isSameFinishSession,
	resolveFinishConfirmView,
	resolveFinishMethodText
} from '../utils/finish-countdown.js'

const pending = (overrides = {}) => ({
	finishState: 'pending_confirmation',
	secondsLeft: 25,
	isRequester: false,
	opponentName: '小李',
	myScore: 3,
	opponentScore: 5,
	canWithdrawFinish: false,
	canConfirmFinish: true,
	canDisputeFinish: true,
	...overrides
})

test('countdown starts from the server remaining seconds and only uses local elapsed time', () => {
	// 手机时钟比服务端快一小时也不影响：只用收到快照后的本地经过时间。
	const receivedAt = Date.parse('2026-08-25T17:00:00+08:00')
	const anchor = createFinishCountdownAnchor(30, receivedAt)
	assert.equal(finishCountdownSecondsLeft(anchor, receivedAt), 30)
	assert.equal(finishCountdownSecondsLeft(anchor, receivedAt + 999), 30)
	assert.equal(finishCountdownSecondsLeft(anchor, receivedAt + 1000), 29)
	assert.equal(finishCountdownSecondsLeft(anchor, receivedAt + 29_999), 1)
	assert.equal(finishCountdownSecondsLeft(anchor, receivedAt + 30_000), 0)
	assert.equal(finishCountdownSecondsLeft(anchor, receivedAt + 90_000), 0)
	assert.equal(finishCountdownSecondsLeft(anchor, receivedAt - 60_000), 30, '本地时钟回拨不增加剩余时间')
	assert.equal(finishCountdownSecondsLeft(createFinishCountdownAnchor(12, receivedAt), receivedAt + 5_000), 7, '后续快照重新校准')
	assert.equal(finishCountdownSecondsLeft(createFinishCountdownAnchor('x', receivedAt), receivedAt), 0)
	assert.equal(finishCountdownSecondsLeft(null, receivedAt), 0)
})

test('the other player gets the confirm dialog with countdown, swipe confirm and dispute', () => {
	const view = resolveFinishConfirmView(pending())
	assert.equal(view.visible, true)
	assert.equal(view.role, 'responder')
	assert.equal(view.finishing, false)
	assert.equal(view.title, '小李申请结束对局')
	assert.equal(view.desc, '比分 3 : 5。25 秒内未处理，将按此比分自动结束，结束后不能再提出异议。')
	assert.equal(view.showConfirm, true)
	assert.equal(view.showDispute, true)
	assert.equal(view.showWithdraw, false)
})

test('the requester sees only the countdown and undo, never a confirm entry', () => {
	const view = resolveFinishConfirmView(pending({
		isRequester: true, opponentName: '小王', myScore: 5, opponentScore: 3,
		canWithdrawFinish: true, canConfirmFinish: true, canDisputeFinish: true
	}))
	assert.equal(view.role, 'requester')
	assert.equal(view.title, '等待小王确认')
	assert.equal(view.secondsLeft, 25)
	assert.equal(view.desc, '30秒内无人处理，将按 5 : 3 自动结束，之后不能再提出异议。')
	assert.equal(view.showWithdraw, true)
	assert.equal(view.showConfirm, false, '发起方没有确认入口')
	assert.equal(view.showDispute, false)
})

test('at zero, or once the server stops offering actions, both sides only see 正在结束', () => {
	for (const input of [
		pending({ secondsLeft: 0 }),
		pending({ secondsLeft: 12, canConfirmFinish: false, canDisputeFinish: false }),
		pending({ isRequester: true, secondsLeft: 0, canWithdrawFinish: true, canConfirmFinish: false, canDisputeFinish: false })
	]) {
		const view = resolveFinishConfirmView(input)
		assert.equal(view.visible, true)
		assert.equal(view.finishing, true, JSON.stringify(input))
		assert.equal(view.title, '正在结束…')
		assert.match(view.desc, /确认期已满，正在按 \d+ : \d+ 结束对局/)
		assert.equal(view.showConfirm || view.showDispute || view.showWithdraw, false)
	}
	assert.equal(resolveFinishConfirmView({ finishState: 'none' }).visible, false)
	assert.equal(resolveFinishConfirmView().visible, false)
})

test('slide note explains the 30 second confirmation for ranked and direct finish for practice', () => {
	assert.equal(
		buildFinishSlideNote({ ranked: true, opponentName: '小王', myScore: 5, opponentScore: 3 }),
		'松手后需小王在30秒内确认，期间你可撤销；无人处理则按 5 : 3 自动结束，之后不能再提出异议。'
	)
	assert.equal(buildFinishSlideNote({ ranked: false, opponentName: '小王', myScore: 5, opponentScore: 3 }), '滑到终点后松手即可结束，无需对方确认。')
	assert.equal(buildFinishSlideNote(), '滑到终点后松手即可结束，无需对方确认。')
	assert.equal(FINISH_CONFIRM_SECONDS, 30)
})

test('ranked result page tells who started the finish and how it ended', () => {
	const base = { match_mode: 'ranked', player1_id: 1001, player2_id: 2002, opponent_name: '小王' }
	// 对方确认：完成人是确认方，发起方是另一名参赛者。
	assert.equal(resolveFinishMethodText({ ...base, is_player1: true, completion_source: 'player_confirmed', completed_by_user_id: 2002 }), '你发起 · 小王确认')
	assert.equal(resolveFinishMethodText({ ...base, is_player1: false, opponent_name: '小李', completion_source: 'player_confirmed', completed_by_user_id: 2002 }), '小李发起 · 你确认')
	// 超时：完成人是发起方。
	assert.equal(resolveFinishMethodText({ ...base, is_player1: true, completion_source: 'finish_timeout', completed_by_user_id: 1001 }), '你发起 · 30秒未处理，自动结束')
	assert.equal(resolveFinishMethodText({ ...base, is_player1: false, opponent_name: '小李', completion_source: 'finish_timeout', completed_by_user_id: 1001 }), '小李发起 · 30秒未处理，自动结束')
	for (const match of [
		{ ...base, match_mode: 'practice', is_player1: true, completion_source: 'player_direct', completed_by_user_id: 1001 },
		{ ...base, is_player1: true, completion_source: 'player_direct', completed_by_user_id: 1001 },
		{ ...base, is_player1: true, completion_source: 'referee', completed_by_user_id: 3003 },
		{ ...base, is_player1: true, completion_source: 'finish_timeout', completed_by_user_id: 3003 },
		{ ...base, is_player1: true, completion_source: 'unknown' },
		null
	]) {
		assert.equal(resolveFinishMethodText(match), '', JSON.stringify(match))
	}
})

test('late responses are dropped after switching account or auth generation', () => {
	const store = { userId: 1001, authGeneration: 3 }
	const session = captureFinishSession(store)
	assert.equal(isSameFinishSession(session, store), true)
	assert.equal(isSameFinishSession(session, { userId: 2002, authGeneration: 3 }), false)
	assert.equal(isSameFinishSession(session, { userId: 1001, authGeneration: 4 }), false)
	assert.equal(isSameFinishSession(null, store), false)
})

const playingSource = readFileSync(new URL('../subPages/match/playing.vue', import.meta.url), 'utf8')
const resultSource = readFileSync(new URL('../subPages/match/matchResult.vue', import.meta.url), 'utf8')
const websocketSource = readFileSync(new URL('../utils/websocket.js', import.meta.url), 'utf8')

const compileSfc = (source, filename) => {
	const { descriptor, errors } = parse(source, { filename })
	assert.deepEqual(errors, [], `${filename} parse errors`)
	return compileScript(descriptor, { id: filename, inlineTemplate: true })
}

test('playing and result pages still compile with the new confirmation UI', () => {
	assert.doesNotThrow(() => compileSfc(playingSource, 'playing.vue'))
	assert.doesNotThrow(() => compileSfc(resultSource, 'matchResult.vue'))
})

test('the confirm dialog reuses the finish slider gestures, so taps and half swipes never confirm', () => {
	const dialog = playingSource.slice(playingSource.indexOf('class="finish-confirm-overlay"'), playingSource.indexOf('class="referee-modal-overlay"'))
	assert.match(dialog, /向右滑动确认结束/)
	for (const handler of ['handleSlideTouchStart', 'handleSlideTouchMove', 'handleSlideTouchEnd', 'handleSlideTouchCancel', 'handleSlideMouseDown', 'handleSlideKeydown']) {
		assert.match(dialog, new RegExp(handler), handler)
	}
	assert.match(dialog, /有异议，继续比赛/)
	assert.doesNotMatch(dialog, /handleWithdrawFinish/, '发起方的撤销不在对方的确认弹窗里')
	// 只有滑到终点松手才提交；确认弹窗中的滑块确认结束，底部滑块对排位发起确认、不再弹窗。
	assert.match(playingSource, /shouldCommitOnRelease\(slideProgressRatio\.value, 1\)/)
	assert.match(playingSource, /showFinishConfirmDialog\.value && finishConfirmView\.value\.showConfirm\) \{\s*await handleConfirmFinish\(\)/)
	const slideRequest = playingSource.slice(playingSource.indexOf('const executeChallengeSlideRequest'), playingSource.indexOf('const executeChallengeSlideFinish'))
	assert.match(slideRequest, /submitFinishAction\(requestFinishMatch/)
	assert.doesNotMatch(slideRequest, /showModal/)
})

test('finish actions guard double submits and only write back for the same account and auth generation', () => {
	const submit = playingSource.slice(playingSource.indexOf('const submitFinishAction'), playingSource.indexOf('const handleRequestFinish'))
	assert.match(submit, /if \(!matchId\.value \|\| finishActionBusy\.value\) return false/)
	assert.match(submit, /const session = captureFinishSession\(userStore\)/)
	assert.match(submit, /if \(!isSameFinishSession\(session, userStore\)\) return false\s*return handleFinishActionResponse/)
	// 被拒时以权威快照刷新；确认期已满时服务端已结束本场，直接进入赛果。
	assert.match(playingSource, /if \(res\?\.snapshot\?\.status === 2\) navigateToResultOnce\(\)/)
	for (const action of ['confirmFinishMatch', 'disputeFinishMatch', 'withdrawFinishMatch']) {
		assert.match(playingSource, new RegExp(`submitFinishAction\\(${action}`), action)
	}
	const dispute = playingSource.slice(playingSource.indexOf('const handleDisputeFinish'), playingSource.indexOf('const handleWithdrawFinish'))
	assert.doesNotMatch(dispute, /showModal/, '有异议只是继续比赛，不需要二次确认')
})

test('countdown follows server snapshots, falls back to a fresh read at zero and is cleaned up on leave', () => {
	assert.match(playingSource, /payload\.finish_request_remaining_seconds !== undefined[\s\S]{0,120}createFinishCountdownAnchor\(payload\.finish_request_remaining_seconds, Date\.now\(\)\)/)
	assert.match(playingSource, /watch\(finishSecondsLeft[\s\S]{0,400}setTimeout\([\s\S]{0,200}loadMatchInfo\(\)[\s\S]{0,40}FINISH_SETTLE_GRACE_MS\)/)
	assert.match(playingSource, /onUnmounted\(\(\) => \{[\s\S]*clearFinishCountdownTimers\(\)[\s\S]*matchWS\.disconnect\(\)/)
	assert.match(websocketSource, /MATCH_FINISH_TIMEOUT: 'match_finish_timeout'/)
	assert.match(playingSource, /matchWS\.on\(WS_MESSAGE_TYPES\.MATCH_FINISH_TIMEOUT, handleFinishStateUpdate\)/)
	assert.match(playingSource, /matchWS\.off\(WS_MESSAGE_TYPES\.MATCH_FINISH_TIMEOUT, handleFinishStateUpdate\)/)
})

test('result page shows the ranked end method and no longer carries dispute-window UI', () => {
	assert.match(resultSource, /resolveFinishMethodText\(matchData\.value\)/)
	assert.match(resultSource, /结束方式：\{\{ finishMethodText \}\}/)
	assert.doesNotMatch(resultSource, /rank_settlement|disputeRankResult|提出异议/)
})
