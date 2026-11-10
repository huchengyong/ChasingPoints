import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'

const appDir = dirname(dirname(fileURLToPath(import.meta.url)))
const pagePath = join(appDir, 'subPages/match/matchDetail.vue')
const real = (relative) => pathToFileURL(join(appDir, relative)).href

const moduleStubs = {
	'@dcloudio/uni-app': `
const register = (name) => (callback) => globalThis.__DETAIL_PAGE__.hooks[name].push(callback)
export const onLoad = register('onLoad')
export const onShow = register('onShow')
export const onHide = register('onHide')
export const onShareAppMessage = register('onShareAppMessage')
`,
	'@/api/match.js': `
export const getMatchDetail = () => globalThis.__DETAIL_PAGE__.request()
export const getPublicMatchDetail = () => globalThis.__DETAIL_PAGE__.request()
`,
	'@/utils/websocket.js': `
export const WS_MESSAGE_TYPES = { SCORE_UPDATE: 'score_update', ROUND_END: 'round_end', MATCH_END: 'match_end', SYNC: 'sync' }
export const matchWS = globalThis.__DETAIL_PAGE__.ws
`,
	'@/components/memberAvatar.vue': 'export default { render: () => null }\n',
	'@/components/matchGearCard.vue': 'export default { render: () => null }\n',
	'@/store/user.js': 'export const useUserStore = () => globalThis.__DETAIL_PAGE__.user\n',
	'@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`,
	'@/utils/share-card.js': 'export const buildMatchShareCard = () => ({})\n',
	'@/utils/gear-match.js': `export * from '${real('utils/gear-match.js')}'`,
	'@/utils/match-replay-entry.js': `export * from '${real('utils/match-replay-entry.js')}'`,
	'@/utils/match-detail.js': `export * from '${real('utils/match-detail.js')}'`,
	'@/utils/user-profile.js': `export * from '${real('utils/user-profile.js')}'`,
	'@/utils/match-referee-view.js': `export * from '${real('utils/match-referee-view.js')}'`
}

const buildDetail = ({ player1Score, player2Score, status = 1, foul1 = 0, foul2 = 0, events = [] }) => ({
	player1_id: 1,
	player2_id: 2,
	player1_name: '甲',
	player2_name: '乙',
	player1_score: player1Score,
	player2_score: player2Score,
	game_type: 2,
	status,
	viewer_role: 'player1',
	is_player1: true,
	score_breakdown: {
		available: true,
		player1: { score: player1Score, normal_wins: 1, small_golds: 0, big_golds: 0, foul_count: foul1, foul_points_given: foul1, other_points: 0 },
		player2: { score: player2Score, normal_wins: 0, small_golds: 0, big_golds: 0, foul_count: foul2, foul_points_given: foul2, other_points: 0 },
		events,
		events_truncated: false
	}
})

test('match detail page replaces nine ball round stats with a silently refreshed breakdown', async (t) => {
	t.mock.timers.enable({ apis: ['setTimeout'] })
	const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
	const previousGlobals = installDOMGlobals(dom.window)
	const tempDir = await mkdtemp(join(appDir, '.match-detail-page-mount-'))
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
			id: 'match-detail-page-mount',
			inlineTemplate: true,
			templateOptions: { compilerOptions: { isCustomElement: (tag) => ['view', 'text', 'image', 'button', 'uni-icons'].includes(tag) } }
		})
		let moduleSource = compiled.content
		const stubs = {
			...moduleStubs,
			'@/components/matchChasingScoreCard.vue': `export { default } from './MatchChasingScoreCard.mjs'\n`
		}
		let stubIndex = 0
		for (const [specifier, content] of Object.entries(stubs)) {
			const stubName = `stub-${stubIndex++}.mjs`
			await writeFile(join(tempDir, stubName), content)
			moduleSource = moduleSource
				.replaceAll(`from '${specifier}'`, `from './${stubName}'`)
				.replaceAll(`from "${specifier}"`, `from "./${stubName}"`)
		}
		await writeFile(join(tempDir, 'MatchDetailPage.mjs'), moduleSource)

		const requests = []
		const handlers = {}
		const toasts = []
		globalThis.__DETAIL_PAGE__ = {
			hooks: { onLoad: [], onShow: [], onHide: [], onShareAppMessage: [] },
			user: { isLoggedIn: true, userId: 1, authGeneration: 1 },
			request: () => {
				let resolve
				let reject
				const promise = new Promise((res, rej) => { resolve = res; reject = rej })
				requests.push({ resolve, reject })
				return promise
			},
			ws: {
				on: (type, handler) => { handlers[type] = handler },
				off() {},
				connect: async () => {},
				disconnect() {},
				isConnected: () => true,
				requestSync() {},
				setTargetActive() {}
			}
		}
		globalThis.uni = {
			setNavigationBarTitle() {},
			navigateTo() {},
			navigateBack() {},
			showToast: (options) => toasts.push(options),
			showModal() {}
		}

		const [{ mount }, { nextTick }] = await Promise.all([import('@vue/test-utils'), import('vue')])
		const componentModule = await import(`${pathToFileURL(join(tempDir, 'MatchDetailPage.mjs')).href}?v=${Date.now()}`)
		wrapper = mount(componentModule.default, {
			attachTo: document.body,
			global: { stubs: { 'uni-icons': true, memberAvatar: true, matchGearCard: true } }
		})

		// 忽略 flushPromises 内部依赖的 setTimeout，用微任务轮转等待请求落库。
		const settle = async () => {
			for (let i = 0; i < 6; i += 1) await Promise.resolve()
			await nextTick()
			for (let i = 0; i < 6; i += 1) await Promise.resolve()
		}

		globalThis.__DETAIL_PAGE__.hooks.onLoad.forEach((hook) => hook({ match_id: '9', perspective_user_id: '1' }))
		await settle()
		requests[0].resolve({
			success: true,
			match: buildDetail({
				player1Score: 10,
				player2Score: 10,
				events: [{ seq: 1, kind: 'foul', scorer: 1, points: 1, player1_score: 11, player2_score: 10, created_at: '2026-11-10 12:00:00' }]
			})
		})
		await settle()

		const initialText = wrapper.text()
		assert.match(initialText, /得分构成/)
		assert.match(initialText, /得分明细/)
		assert.match(initialText, /10 : 10/)
		assert.doesNotMatch(initialText, /总局数|当前局|局记录|第 \d+ 局/)
		assert.equal(wrapper.find('.round-history').exists(), false)
		assert.equal(wrapper.findAll('.match-chasing-score__event').length, 1)

		// 记分消息先更新比分，防抖窗口内不额外请求。
		handlers.score_update({ player1_score: 11, player2_score: 10, current_round: 2 })
		await settle()
		assert.match(wrapper.text(), /11 : 10/)
		assert.equal(requests.length, 1)

		t.mock.timers.tick(1000)
		await settle()
		assert.equal(requests.length, 2)
		assert.equal(wrapper.find('.loading-container').exists(), false)

		requests[1].resolve({
			success: true,
			match: buildDetail({
				player1Score: 11,
				player2Score: 10,
				foul1: 2,
				foul2: 4,
				events: [
					{ seq: 2, kind: 'foul', scorer: 1, points: 1, player1_score: 12, player2_score: 10, created_at: '2026-11-10 12:00:02' },
					{ seq: 1, kind: 'foul', scorer: 1, points: 1, player1_score: 11, player2_score: 10, created_at: '2026-11-10 12:00:00' }
				]
			})
		})
		await settle()
		assert.equal(wrapper.findAll('.match-chasing-score__event').length, 2)
		assert.ok(wrapper.findAll('.match-chasing-score__row').map((row) => row.text()).includes('2犯规4'))

		// 失败静默：保留旧数据，不弹错误提示。
		handlers.round_end({ player1_score: 11, player2_score: 10, current_round: 3, round_number: 3 })
		await settle()
		t.mock.timers.tick(1000)
		await settle()
		requests[2].reject(new Error('network'))
		await settle()
		assert.match(wrapper.text(), /得分构成/)
		assert.equal(wrapper.findAll('.match-chasing-score__event').length, 2)
		assert.equal(toasts.length, 0)

		// 迟到响应不得覆盖更新的数据。
		handlers.score_update({ player1_score: 12, player2_score: 10, current_round: 4 })
		await settle()
		t.mock.timers.tick(1000)
		await settle()
		handlers.score_update({ player1_score: 20, player2_score: 10, current_round: 5 })
		await settle()
		t.mock.timers.tick(1000)
		await settle()
		requests[4].resolve({ success: true, match: buildDetail({ player1Score: 20, player2Score: 10, foul1: 5 }) })
		await settle()
		requests[3].resolve({ success: true, match: buildDetail({ player1Score: 12, player2Score: 10 }) })
		await settle()
		assert.match(wrapper.text(), /20 : 10/)
		assert.doesNotMatch(wrapper.text(), /12 : 10/)

		// 对局结束立即再拉一次，不等防抖。
		handlers.match_end({ player1_score: 20, player2_score: 10, status: 2 })
		await settle()
		assert.equal(requests.length, 6)
		requests[5].resolve({ success: true, match: buildDetail({ player1Score: 20, player2Score: 10, status: 2, foul1: 5 }) })
		await settle()
		assert.match(wrapper.text(), /得分构成/)

		// 中式八球保持总局数、当前局与局记录。
		globalThis.__DETAIL_PAGE__.hooks.onLoad.forEach((hook) => hook({ match_id: '9', perspective_user_id: '1' }))
		await settle()
		requests[6].resolve({
			success: true,
			match: {
				...buildDetail({ player1Score: 3, player2Score: 1, status: 2 }),
				game_type: 3,
				total_rounds: 2,
				current_round: 2,
				rounds: [{ round_number: 1, player1_score: 21, player2_score: 10, winner: 1 }]
			}
		})
		await settle()
		const poolText = wrapper.text()
		assert.match(poolText, /总局数/)
		assert.match(poolText, /当前局/)
		assert.match(poolText, /局记录/)
		assert.match(poolText, /第 1 局/)
		assert.doesNotMatch(poolText, /得分构成/)
	} finally {
		wrapper?.unmount()
		delete globalThis.__DETAIL_PAGE__
		delete globalThis.uni
		await rm(tempDir, { recursive: true, force: true })
		restoreDOMGlobals(previousGlobals)
		dom.window.close()
	}
})

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
