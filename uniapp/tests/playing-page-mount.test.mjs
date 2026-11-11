import test from 'node:test'
import assert from 'node:assert/strict'

import { createPageHarness, createUniRecorder } from './helpers/page-harness.mjs'

const GLOBAL_KEY = '__PLAYING_PAGE__'

const stubSources = {
	'@dcloudio/uni-app': `
const register = (name) => (callback) => globalThis.${GLOBAL_KEY}.hooks[name].push(callback)
export const onLoad = register('onLoad')
export const onShow = register('onShow')
export const onHide = register('onHide')
`,
	'@/store/user.js': `export const useUserStore = () => globalThis.${GLOBAL_KEY}.user\n`,
	'@/store/rank.js': `export const useRankStore = () => ({ invalidate() {} })\n`,
	'@/store/activity.js': `export const useActivityStore = () => ({ markDirty() {} })\n`,
	'@/utils/websocket.js': `
export const WS_MESSAGE_TYPES = {
	SCORE_UPDATE: 'score_update',
	ROUND_END: 'round_end',
	ROUND_START: 'round_start',
	MATCH_END: 'match_end',
	MATCH_ROLE_CHANGED: 'match_role_changed',
	MATCH_FINISH_REQUEST: 'match_finish_request',
	MATCH_FINISH_CONFIRM: 'match_finish_confirm',
	MATCH_FINISH_DISPUTE: 'match_finish_dispute',
	MATCH_FINISH_WITHDRAW: 'match_finish_withdraw',
	MATCH_FINISH_EXPIRED: 'match_finish_expired',
	MATCH_FINISH_TIMEOUT: 'match_finish_timeout',
	SYNC: 'sync'
}
export const matchWS = globalThis.${GLOBAL_KEY}.ws
`,
	'@/api/match.js': `
const noop = async () => ({})
export const matchScore = noop
export const endRound = noop
export const startNextRound = noop
export const matchFoul = noop
export const matchUndo = noop
export const finishMatch = noop
export const requestFinishMatch = noop
export const confirmFinishMatch = noop
export const disputeFinishMatch = noop
export const withdrawFinishMatch = noop
export const getMatchRefereeQRCode = noop
export const snookerFrameAction = noop
export const updateSnookerFormat = noop
export const updateMatchFormat = noop
export const getMatchDetail = () => globalThis.${GLOBAL_KEY}.detail()
export const getCurrentMatch = () => Promise.resolve(globalThis.${GLOBAL_KEY}.currentMatch)
export const snookerStroke = (payload) => {
	globalThis.${GLOBAL_KEY}.strokes.push(payload)
	return Promise.resolve({ success: true })
}
`,
	'@/utils/local-qrcode.js': 'export const renderLocalQRCode = () => {}\n',
	'@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`,
	'@/components/memberAvatar.vue': `
import { h } from 'vue'
export default {
	props: {
		avatar: { type: String, default: '' },
		userId: { type: [Number, String], default: 0 },
		vipLevel: { type: [Number, String], default: 0 },
		rankLevel: { type: [Number, String], default: 0 },
		size: { type: Number, default: 72 }
	},
	render() {
		return h('span', { class: 'member-avatar-stub' })
	}
}
`
}

const realModules = [
	'@/utils/slide-to-confirm.js',
	'@/utils/finish-countdown.js',
	'@/utils/match-navigation.js',
	'@/utils/match-action.js',
	'@/utils/match-role-view.js',
	'@/utils/match-scoring-target.js',
	'@/utils/snooker-frame.js',
	'@/utils/snooker-v2.js',
	'@/utils/user-profile.js',
	'@/utils/rank-cache.js',
	'@/utils/snooker-match-format.js',
	'@/utils/pool-match-format.js'
]

const harness = createPageHarness({
	pageFile: 'subPages/match/playing.vue',
	globalKey: GLOBAL_KEY,
	stubSources,
	realModules,
	customElements: ['view', 'text', 'image', 'canvas', 'picker-view', 'picker-view-column', 'button', 'uni-icons']
})

const buildMatch = (overrides = {}) => ({
	id: 7,
	game_type: 1,
	status: 1,
	is_player1: true,
	viewer_role: 'player1',
	my_score: 10,
	opponent_score: 4,
	current_frame_my_score: 10,
	current_frame_opponent_score: 4,
	current_frame_started: true,
	my_max_score: 30,
	opponent_max_score: 20,
	my_name: '我',
	opponent_name: '对手',
	server_revision: 5,
	snooker_rules_version: 2,
	snooker_phase: 'reds',
	snooker_ball_on: 'red',
	snooker_striker: 1,
	snooker_reds_remaining: 5,
	...overrides
})

const mountPlaying = async (match, uniOverrides = {}) => {
	const recorder = createUniRecorder({
		getSystemInfoSync: () => ({ statusBarHeight: 20 }),
		createSelectorQuery: () => ({ in: () => ({ select: () => ({ boundingClientRect: () => ({ exec: () => {} }) }) }) }),
		...uniOverrides
	})
	const context = {
		hooks: { onLoad: [], onShow: [], onHide: [] },
		user: { userId: 1, authGeneration: 1, isLoggedIn: true, userInfo: { id: 1, nickname: '我' } },
		detail: async () => ({ success: true, match }),
		currentMatch: { success: true, match: { id: 7 } },
		strokes: [],
		ws: {
			on() {},
			off() {},
			connect: async () => {},
			disconnect() {},
			isConnected: () => true,
			requestSync() {},
			setTargetActive() {}
		}
	}
	const mounted = await harness.mountPage({ context, uni: recorder.uni })
	context.hooks.onLoad.forEach((callback) => callback({ match_id: '7' }))
	context.hooks.onShow.forEach((callback) => callback())
	await mounted.flushPromises()
	await mounted.flushPromises()
	return { ...mounted, context, recorder }
}

test('snooker shows the single-break stat and a one-tap red ball records exactly one red', async () => {
	const page = await mountPlaying(buildMatch({ snooker_reds_remaining: 8 }))
	try {
		const content = page.wrapper.text()
		assert.match(content, /单杆 30/)
		assert.match(content, /单杆 20/)
		assert.match(content, /一杆多颗红球/)

		const redButton = page.wrapper.findAll('button').find((button) => button.text().includes('红球') && button.text().includes('1 分'))
		assert.ok(redButton, 'the red ball pot button is rendered')
		await redButton.trigger('click')
		await page.flushPromises()

		assert.equal(page.context.strokes.length, 1)
		assert.equal(page.context.strokes[0].potted_reds, 1)
		assert.equal(page.recorder.callsOf('showActionSheet').length, 0, 'tapping red does not open the count sheet')
	} finally {
		await page.cleanup()
	}
})

test('the multi-red entry is hidden when fewer than two reds remain', async () => {
	const page = await mountPlaying(buildMatch({ snooker_reds_remaining: 1 }))
	try {
		assert.doesNotMatch(page.wrapper.text(), /一杆多颗红球/)
		assert.match(page.wrapper.text(), /单杆 30/)
	} finally {
		await page.cleanup()
	}
})

test('non-snooker matches hide the single-break stat and the snooker panel', async () => {
	const page = await mountPlaying(buildMatch({ game_type: 3, snooker_rules_version: 0, my_max_score: 0, opponent_max_score: 0 }))
	try {
		const content = page.wrapper.text()
		assert.doesNotMatch(content, /单杆/)
		assert.doesNotMatch(content, /一杆多颗红球/)
	} finally {
		await page.cleanup()
	}
})
