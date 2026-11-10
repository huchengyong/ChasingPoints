import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'

const appDir = dirname(dirname(fileURLToPath(import.meta.url)))
const componentPath = join(appDir, 'components/matchChasingScoreCard.vue')

const availableView = {
	available: true,
	leftLabel: '我',
	rightLabel: '对手',
	leftScore: 13,
	rightScore: 16,
	rows: [
		{ key: 'score', label: '总分', left: '13', right: '16' },
		{ key: 'normal_wins', label: '普胜', left: '0', right: '2' },
		{ key: 'small_golds', label: '小金', left: '0', right: '1' },
		{ key: 'big_golds', label: '大金', left: '1', right: '0' },
		{ key: 'foul_count', label: '犯规', left: '1', right: '1' },
		{ key: 'foul_points_given', label: '犯规送分', left: '1', right: '1' }
	],
	events: [
		{ seq: 2, kind: 'foul', kindLabel: '犯规', side: 'left', sideLabel: '对手', pointsText: '+1', scoreText: '10 : 11', timeText: '12:00:04', desc: '我犯规' },
		{ seq: 1, kind: 'big_gold', kindLabel: '大金', side: 'right', sideLabel: '我', pointsText: '+10', scoreText: '10 : 5', timeText: '', desc: '对手 大金' }
	],
	eventsTruncated: false,
	notice: ''
}

async function mountCard(props = {}) {
	const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
	const previousGlobals = installDOMGlobals(dom.window)
	const tempDir = await mkdtemp(join(appDir, '.match-chasing-score-card-'))
	const source = await readFile(componentPath, 'utf8')
	const { descriptor, errors } = parse(source, { filename: componentPath })
	assert.deepEqual(errors, [])
	const compiled = compileScript(descriptor, {
		id: 'match-chasing-score-card',
		inlineTemplate: true,
		templateOptions: { compilerOptions: { isCustomElement: (tag) => ['view', 'text'].includes(tag) } }
	})
	await writeFile(join(tempDir, 'MatchChasingScoreCard.mjs'), compiled.content)
	const [{ mount }, componentModule] = await Promise.all([
		import('@vue/test-utils'),
		import(`${pathToFileURL(join(tempDir, 'MatchChasingScoreCard.mjs')).href}?v=${Date.now()}`)
	])
	const wrapper = mount(componentModule.default, { attachTo: document.body, props })
	return {
		wrapper,
		async close() {
			wrapper.unmount()
			await rm(tempDir, { recursive: true, force: true })
			restoreDOMGlobals(previousGlobals)
			dom.window.close()
		}
	}
}

test('chasing score card renders the comparison table and score events', async () => {
	const card = await mountCard({ view: availableView })
	try {
		const text = card.wrapper.text()
		assert.match(text, /得分构成/)
		assert.match(text, /我/)
		assert.match(text, /对手/)
		assert.match(text, /项目/)
		assert.match(text, /总分/)
		assert.deepEqual(
			card.wrapper.findAll('.match-chasing-score__row').map((row) => row.text()),
			['13总分16', '0普胜2', '0小金1', '1大金0', '1犯规1', '1犯规送分1']
		)
		assert.equal(card.wrapper.find('.match-chasing-score__row.is-total').exists(), true)

		assert.match(text, /得分明细/)
		assert.deepEqual(
			card.wrapper.findAll('.match-chasing-score__event').map((event) => event.text()),
			['我犯规+110 : 1112:00:04', '对手 大金+1010 : 5']
		)
		assert.equal(card.wrapper.findAll('.match-chasing-score__event.is-foul').length, 1)
		assert.equal(card.wrapper.findAll('.match-chasing-score__event.is-left').length, 1)
		assert.equal(card.wrapper.findAll('.match-chasing-score__event.is-right').length, 1)
		assert.equal(card.wrapper.findAll('.match-chasing-score__event-time').length, 1)
	} finally {
		await card.close()
	}
})

test('chasing score card hides events for the result page and keeps the notice', async () => {
	const card = await mountCard({ view: availableView })
	try {
		await card.wrapper.setProps({ view: { ...availableView, notice: '仅显示最近 200 条记分' }, showEvents: false })
		const text = card.wrapper.text()
		assert.match(text, /仅显示最近 200 条记分/)
		assert.match(text, /总分/)
		assert.doesNotMatch(text, /得分明细/)
		assert.doesNotMatch(text, /我犯规/)
	} finally {
		await card.close()
	}
})

test('chasing score card falls back to scores only and marks an empty event list', async () => {
	const card = await mountCard({ view: availableView })
	try {
		await card.wrapper.setProps({
			view: {
				available: false,
				leftLabel: '选手1',
				rightLabel: '选手2',
				rows: [{ key: 'score', label: '总分', left: '16', right: '13' }],
				events: [],
				notice: '记分明细暂不可用，仅显示总分'
			}
		})
		const text = card.wrapper.text()
		assert.match(text, /记分明细暂不可用，仅显示总分/)
		assert.doesNotMatch(text, /暂无记分/)
		assert.equal(card.wrapper.findAll('.match-chasing-score__event').length, 0)

		await card.wrapper.setProps({ view: availableView, showEvents: true })
		await card.wrapper.setProps({ view: { ...availableView, events: [], notice: '' } })
		assert.match(card.wrapper.text(), /暂无记分/)
	} finally {
		await card.close()
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
