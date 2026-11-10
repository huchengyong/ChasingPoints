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
export const getMatchRewardSummary = () => Promise.resolve({ success: true, status: 'ready', list: [] })
`,
	'@/components/memberAvatar.vue': 'export default { render: () => null }\n',
	'@/components/matchGearCard.vue': 'export default { render: () => null }\n',
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
		return { wrapper, text: wrapper.text() }
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
