import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'

const appDir = dirname(dirname(fileURLToPath(import.meta.url)))
const pagePath = join(appDir, 'subPages/match/matchResult.vue')
const real = (relative) => pathToFileURL(join(appDir, relative)).href

const moduleStubs = {
	'@dcloudio/uni-app': `
const register = (name) => (callback) => globalThis.__RESULT_PAGE__.hooks[name].push(callback)
export const onLoad = register('onLoad')
export const onUnload = register('onUnload')
export const onShareAppMessage = register('onShareAppMessage')
`,
	'@/api/match.js': `
export const getMatchDetail = () => globalThis.__RESULT_PAGE__.request()
export const getMatchRewardSummary = () => {
	globalThis.__RESULT_PAGE__.rewardRequests += 1
	return Promise.resolve({ success: true, status: 'ready', list: [] })
}
`,
	'@/components/memberAvatar.vue': `
import { h } from 'vue'
export default {
	props: {
		avatar: { type: String, default: '' },
		userId: { type: [Number, String], default: 0 },
		vipLevel: { type: [Number, String], default: 0 },
		rankLevel: { type: [Number, String], default: 0 },
		size: { type: Number, default: 96 }
	},
	render() {
		return h('span', {
			class: 'member-avatar-stub',
			'data-user-id': String(this.userId),
			'data-avatar': this.avatar || ''
		})
	}
}
`,
	'@/components/matchGearCard.vue': `
import { h } from 'vue'
export default {
	props: {
		matchId: { type: [Number, String], default: null },
		enabled: { type: Boolean, default: false }
	},
	render() {
		return this.enabled ? h('text', '本场用杆') : null
	}
}
`,
	'@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`,
	'@/utils/quick-scan-entry.js': 'export const startQuickScan = () => {}\n',
	'@/utils/format.js': `export * from '${real('utils/format.js')}'`,
	'@/utils/gear-match.js': `export * from '${real('utils/gear-match.js')}'`,
	'@/utils/member-ranking-rights.js': `export * from '${real('utils/member-ranking-rights.js')}'`,
	'@/utils/user-profile.js': `export * from '${real('utils/user-profile.js')}'`,
	'@/utils/share-card.js': `export * from '${real('utils/share-card.js')}'`,
	'@/utils/match-core-flow.js': `export * from '${real('utils/match-core-flow.js')}'`,
	'@/utils/match-replay-entry.js': `export * from '${real('utils/match-replay-entry.js')}'`,
	'@/utils/finish-countdown.js': `export * from '${real('utils/finish-countdown.js')}'`,
	'@/utils/match-referee-view.js': `export * from '${real('utils/match-referee-view.js')}'`,
	'@/utils/match-result-page.js': `export * from '${real('utils/match-result-page.js')}'`,
	'@/utils/chasing-score.js': `export * from '${real('utils/chasing-score.js')}'`,
	'@/components/matchChasingScoreCard.vue': `export { default } from './MatchChasingScoreCard.mjs'\n`
}

const chasingBreakdown = {
	available: true,
	player1: { score: 16, normal_wins: 2, small_golds: 1, big_golds: 0, foul_count: 3, foul_points_given: 3, other_points: 0 },
	player2: { score: 13, normal_wins: 0, small_golds: 0, big_golds: 1, foul_count: 1, foul_points_given: 1, other_points: 0 },
	events: [],
	events_truncated: false
}

async function mountResultPage(match) {
	const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
	const previousGlobals = installDOMGlobals(dom.window)
	const tempDir = await mkdtemp(join(appDir, '.match-result-page-mount-'))
	let wrapper
	try {
		const componentPath = join(appDir, 'components/matchChasingScoreCard.vue')
		const componentSource = await readFile(componentPath, 'utf8')
		const { descriptor: componentDescriptor } = parse(componentSource, { filename: componentPath })
		const compiledComponent = compileScript(componentDescriptor, {
			id: 'match-chasing-score-card',
			inlineTemplate: true,
			templateOptions: { compilerOptions: { isCustomElement: (tag) => ['view', 'text'].includes(tag) } }
		})
		await writeFile(join(tempDir, 'MatchChasingScoreCard.mjs'), compiledComponent.content)

		const source = await readFile(pagePath, 'utf8')
		const { descriptor, errors } = parse(source, { filename: pagePath })
		assert.deepEqual(errors, [])
		const compiled = compileScript(descriptor, {
			id: 'match-result-page-mount',
			inlineTemplate: true,
			templateOptions: { compilerOptions: { isCustomElement: (tag) => ['view', 'text', 'image', 'button', 'scroll-view', 'uni-icons'].includes(tag) } }
		})
		let moduleSource = compiled.content
		let stubIndex = 0
		for (const [specifier, content] of Object.entries(moduleStubs)) {
			const stubName = `stub-${stubIndex++}.mjs`
			await writeFile(join(tempDir, stubName), content)
			moduleSource = moduleSource
				.replaceAll(`from '${specifier}'`, `from './${stubName}'`)
				.replaceAll(`from "${specifier}"`, `from "./${stubName}"`)
		}
		await writeFile(join(tempDir, 'MatchResultPage.mjs'), moduleSource)

		const requests = []
		globalThis.__RESULT_PAGE__ = {
			hooks: { onLoad: [], onUnload: [], onShareAppMessage: [] },
			rewardRequests: 0,
			request: () => {
				let resolve
				const promise = new Promise((res) => { resolve = res })
				requests.push({ resolve })
				return promise
			}
		}
		globalThis.uni = {
			getSystemInfoSync: () => ({ statusBarHeight: 20 }),
			navigateTo() {},
			showToast() {},
			setClipboardData() {}
		}

		const [{ mount, flushPromises }, { nextTick }] = await Promise.all([import('@vue/test-utils'), import('vue')])
		const componentModule = await import(`${pathToFileURL(join(tempDir, 'MatchResultPage.mjs')).href}?v=${Date.now()}`)
		wrapper = mount(componentModule.default, { attachTo: document.body, global: { stubs: { 'uni-icons': true, 'scroll-view': { template: '<div><slot /></div>' } } } })
		globalThis.__RESULT_PAGE__.hooks.onLoad.forEach((hook) => hook({ match_id: '42' }))
		await nextTick()

		requests[0].resolve({ success: true, match })
		await flushPromises()
		await nextTick()
		return { wrapper, text: wrapper.text(), rewardRequests: globalThis.__RESULT_PAGE__.rewardRequests }
	} finally {
		wrapper?.unmount()
		delete globalThis.__RESULT_PAGE__
		delete globalThis.uni
		await rm(tempDir, { recursive: true, force: true })
		restoreDOMGlobals(previousGlobals)
		dom.window.close()
	}
}

test('match result page shows the nine ball score breakdown before the stats card', async () => {
	const { text } = await mountResultPage({
		game_type: 2,
		is_player1: false,
		my_score: 13,
		opponent_score: 16,
		opponent_name: '对手',
		score_breakdown: chasingBreakdown,
		summary_stats: [{ label: '最高连续得分', value: '12', desc: '本场最长连续得分' }]
	})

	assert.match(text, /得分构成/)
	assert.match(text, /总分/)
	assert.match(text, /13/)
	assert.match(text, /16/)
	assert.match(text, /大金/)
	assert.match(text, /犯规/)
	assert.match(text, /本场数据/)
	assert.doesNotMatch(text, /得分明细/)
	assert.ok(text.indexOf('得分构成') < text.indexOf('本场数据'))
})

test('match result page keeps other game types free of the score breakdown card', async () => {
	const { text } = await mountResultPage({
		game_type: 3,
		is_player1: true,
		my_score: 3,
		opponent_score: 1,
		summary_stats: [{ label: '炸清', value: '1', desc: '本场炸清次数' }]
	})

	assert.doesNotMatch(text, /得分构成/)
	assert.match(text, /本场数据/)
	assert.match(text, /炸清/)
})

test('referee result page hides player-only sections and skips the reward request', async () => {
	const { text, rewardRequests } = await mountResultPage({
		game_type: 2,
		status: 2,
		viewer_role: 'referee',
		is_player1: true,
		my_score: 16,
		opponent_score: 13,
		my_name: '甲',
		opponent_name: '乙',
		referee_bound: true,
		referee_user_id: 3,
		referee_name: '裁判丙',
		referee_duration_seconds: 720,
		completion_source: 'referee',
		score_breakdown: chasingBreakdown,
		summary_highlights: [{ label: '连续得分', value: '6', desc: '连续得分' }],
		summary_stats: [{ label: '最高连续得分', value: '12', desc: '本场最长连续得分' }]
	})

	assert.match(text, /本场裁判/)
	assert.match(text, /执裁时长 12分钟/)
	assert.match(text, /完成方式：裁判结算/)
	assert.match(text, /得分构成/)
	assert.match(text, /选手1/)
	assert.match(text, /选手2/)
	assert.doesNotMatch(text, /本场用杆/)
	assert.doesNotMatch(text, /本场亮点/)
	assert.doesNotMatch(text, /本场数据/)
	assert.doesNotMatch(text, /高光仅记录/)
	assert.equal(rewardRequests, 0)
})

test('player result page keeps personal sections and requests rewards once', async () => {
	const { text, rewardRequests } = await mountResultPage({
		game_type: 3,
		status: 2,
		viewer_role: 'player1',
		is_player1: true,
		my_score: 3,
		opponent_score: 1,
		summary_stats: [{ label: '炸清', value: '1', desc: '本场炸清次数' }]
	})

	assert.match(text, /本场用杆/)
	assert.match(text, /本场亮点/)
	assert.match(text, /本场数据/)
	assert.equal(rewardRequests, 1)
})

test('player 2 result page maps hero avatars to player2 then player1', async () => {
	const { wrapper } = await mountResultPage({
		game_type: 3,
		status: 2,
		viewer_role: 'player2',
		is_player1: false,
		player1_id: 101,
		player2_id: 202,
		opponent_id: 101,
		my_score: 1,
		opponent_score: 3
	})

	assert.deepEqual(heroAvatarUserIds(wrapper), ['202', '101'])
})

test('referee result page maps hero avatars to player1 then player2', async () => {
	const { wrapper } = await mountResultPage({
		game_type: 3,
		status: 2,
		viewer_role: 'referee',
		is_player1: true,
		player1_id: 101,
		player2_id: 202,
		opponent_id: 202,
		my_score: 1,
		opponent_score: 3
	})

	assert.deepEqual(heroAvatarUserIds(wrapper), ['101', '202'])
})

test('completed match with duration shows the end time derived from start plus duration', async () => {
	const { text } = await mountResultPage({
		game_type: 3,
		status: 2,
		viewer_role: 'player1',
		is_player1: true,
		created_at: '2026-11-10 23:30:00',
		duration_seconds: 3600,
		my_score: 3,
		opponent_score: 1
	})

	assert.match(text, /结束时间/)
	assert.match(text, /2026年11月11日 00:30/)
	assert.doesNotMatch(text, /2026年11月10日 23:30/)
})

test('completed match without duration labels the start time instead of the end time', async () => {
	const { text } = await mountResultPage({
		game_type: 3,
		status: 2,
		viewer_role: 'player1',
		is_player1: true,
		created_at: '2026-11-10 23:30:00',
		duration_seconds: 0,
		my_score: 3,
		opponent_score: 1
	})

	assert.match(text, /开始时间/)
	assert.doesNotMatch(text, /结束时间/)
	assert.match(text, /2026年11月10日 23:30/)
})

test('settled draw shows a neutral banner without a winning side', async () => {
	const { wrapper, text } = await mountResultPage({
		game_type: 3,
		status: 2,
		viewer_role: 'player1',
		is_player1: true,
		my_score: 2,
		opponent_score: 1,
		result: 3
	})

	const banner = wrapper.element.querySelector('.hero-banner')
	assert.ok(banner.classList.contains('is-neutral'))
	assert.equal(banner.classList.contains('is-win'), false)
	assert.equal(banner.classList.contains('is-lose'), false)
	assert.match(text, /平局战报/)
	assert.match(text, /握手言和/)
	assert.match(text, /双方打成平手，下一场再分高下。/)
	assert.match(text, /双方局数持平/)
	assert.equal(wrapper.element.querySelectorAll('.avatar-shell.is-winner').length, 0)
	assert.equal(wrapper.element.querySelectorAll('.avatar-shell.is-neutral').length, 2)
})

test('equal scores without a settled result are treated as a draw', async () => {
	const { text } = await mountResultPage({
		game_type: 2,
		status: 2,
		viewer_role: 'player1',
		is_player1: true,
		my_score: 13,
		opponent_score: 13
	})

	assert.match(text, /平局战报/)
	assert.match(text, /双方比分持平/)
})

test('practice match hides the rank change section', async () => {
	const { text } = await mountResultPage({
		game_type: 3,
		status: 2,
		viewer_role: 'player1',
		is_player1: true,
		match_mode: 'practice',
		my_score: 3,
		opponent_score: 1,
		my_rank_change: 20,
		my_rank_details: [{ label: '基础分', value: 20 }],
		achievements: { break_clear: 1 }
	})

	assert.doesNotMatch(text, /排位变化/)
	assert.doesNotMatch(text, /我的排位分/)
	assert.match(text, /练习赛不计入排位/)
	assert.match(text, /本场记录了 炸清x1，高光会保留在战报和成就里。/)
	assert.doesNotMatch(text, /本场排位只按基础分结算/)
	assert.doesNotMatch(text, /VIP用户才会/)
})

test('a response without match_mode still shows the rank change section', async () => {
	const { text } = await mountResultPage({
		game_type: 3,
		status: 2,
		viewer_role: 'player1',
		is_player1: true,
		my_score: 3,
		opponent_score: 1,
		my_rank_change: 20,
		my_rank_details: [{ label: '基础分', value: 20 }]
	})

	assert.match(text, /排位变化/)
})

test('ranked match keeps the rank change section', async () => {
	const { text } = await mountResultPage({
		game_type: 3,
		status: 2,
		viewer_role: 'player1',
		is_player1: true,
		match_mode: 'ranked',
		my_score: 3,
		opponent_score: 1,
		my_rank_change: 20,
		my_rank_details: [{ label: '基础分', value: 20 }]
	})

	assert.match(text, /排位变化/)
	assert.match(text, /我的排位分/)
})

test('match result style gives the referee hero a neutral background and a fixed avatar size', async () => {
	const style = await readFile(join(appDir, 'subPages/match/matchResult.scss'), 'utf8')
	assert.match(style, /&\.is-neutral\s*\{[^}]*background:\s*linear-gradient/)
	assert.match(style, /\.referee-card-avatar\s*\{[^}]*width:\s*88rpx;[^}]*height:\s*88rpx;/s)
})

test('match result dark cards use the warm surface tokens instead of cool slate', async () => {
	const style = await readFile(join(appDir, 'subPages/match/matchResult.scss'), 'utf8')
	assert.match(style, /\$card-dark:\s*#1E180D;/)
	assert.match(style, /\$border-dark:\s*#3A2E16;/)
	assert.doesNotMatch(style, /rgba\(30,\s*41,\s*59/)
})

function heroAvatarUserIds(wrapper) {
	return [...wrapper.element.querySelectorAll('.member-avatar-stub')]
		.map((node) => node.getAttribute('data-user-id'))
}

function installDOMGlobals(window) {
	const keys = ['window', 'document', 'navigator', 'Node', 'Element', 'HTMLElement', 'SVGElement', 'Event']
	const previous = new Map()
	for (const key of keys) {
		previous.set(key, Object.getOwnPropertyDescriptor(globalThis, key))
		Object.defineProperty(globalThis, key, { configurable: true, writable: true, value: window[key] })
	}
	return previous
}

function restoreDOMGlobals(previous) {
	for (const [key, descriptor] of previous) {
		if (descriptor) Object.defineProperty(globalThis, key, descriptor)
		else delete globalThis[key]
	}
}
