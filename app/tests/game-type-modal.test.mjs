import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'

const appDir = dirname(dirname(fileURLToPath(import.meta.url)))
const modalPath = join(appDir, 'components/gameTypeModal.vue')

test('game type modal covers default preselection, temporary switching and explicit default metadata', async () => {
	await withMountedModal(async ({ wrapper, nextTick }) => {
		await wrapper.setProps({ visible: true, defaultType: 3 })
		await nextTick()

		assert.match(wrapper.text(), /中式八球默认/)
		assert.equal(wrapper.findAll('.game-type-option.selected').length, 1)
		assert.match(wrapper.find('.game-type-option.selected').text(), /中式八球/)

		const nineBall = wrapper.findAll('.game-type-option').find((option) => option.text().includes('九球'))
		assert.ok(nineBall)
		await nineBall.trigger('click')
		await wrapper.find('.modal-btn.confirm').trigger('click')

		assert.deepEqual(wrapper.emitted('confirm')?.[0], [{ gameType: 2, setAsDefault: false }])

		await wrapper.setProps({ visible: false })
		await wrapper.setProps({ visible: true, defaultType: 0 })
		await nextTick()

		assert.equal(wrapper.findAll('.game-type-option.selected').length, 0)
		assert.equal(wrapper.find('.modal-btn.confirm').attributes('disabled'), '')

		const snooker = wrapper.findAll('.game-type-option').find((option) => option.text().includes('斯诺克'))
		assert.ok(snooker)
		await snooker.trigger('click')
		await wrapper.find('.default-choice').trigger('click')
		await wrapper.find('.modal-btn.confirm').trigger('click')

		assert.deepEqual(wrapper.emitted('confirm')?.[1], [{ gameType: 1, setAsDefault: true }])
	})
})

const withMountedModal = async (run) => {
	const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
	const previousGlobals = installDOMGlobals(dom.window)
	const tempDir = await mkdtemp(join(appDir, '.game-type-modal-'))
	let wrapper

	try {
		const source = await readFile(modalPath, 'utf8')
		const { descriptor, errors } = parse(source, { filename: modalPath })
		assert.deepEqual(errors, [])
		const compiled = compileScript(descriptor, {
			id: 'game-type-modal-test',
			inlineTemplate: true,
			templateOptions: {
				compilerOptions: {
					isCustomElement: (tag) => ['view', 'text', 'uni-icons'].includes(tag)
				}
			}
		})

		await writeFile(join(tempDir, 'theme-store.mjs'), `
export const useThemeStore = () => ({ isDarkMode: false })
`)
		await writeFile(join(tempDir, 'game-types.mjs'), `
export const GAME_TYPE_OPTIONS = [
  { value: 3, label: '中式八球' },
  { value: 2, label: '九球追分' },
  { value: 1, label: '斯诺克' },
  { value: 4, label: '美式九球' }
]
`)
		const moduleSource = compiled.content
			.replaceAll("from '@/store/theme.js'", "from './theme-store.mjs'")
			.replaceAll("from '@/utils/game-types.js'", "from './game-types.mjs'")
		await writeFile(join(tempDir, 'GameTypeModal.mjs'), moduleSource)

		const [{ mount }, { nextTick }] = await Promise.all([
			import('@vue/test-utils'),
			import('vue')
		])
		const componentModule = await import(`${pathToFileURL(join(tempDir, 'GameTypeModal.mjs')).href}?v=${Date.now()}`)
		wrapper = mount(componentModule.default, {
			attachTo: document.body,
			props: { visible: false, defaultType: 0 },
			global: { stubs: { 'uni-icons': true } }
		})
		await run({ wrapper, nextTick })
	} finally {
		wrapper?.unmount()
		await rm(tempDir, { recursive: true, force: true })
		restoreDOMGlobals(previousGlobals)
		dom.window.close()
	}
}

function installDOMGlobals(window) {
	const keys = ['window', 'document', 'navigator', 'Node', 'Element', 'HTMLElement', 'SVGElement', 'Event']
	const previous = new Map()
	for (const key of keys) {
		previous.set(key, Object.getOwnPropertyDescriptor(globalThis, key))
		Object.defineProperty(globalThis, key, {
			configurable: true,
			writable: true,
			value: window[key]
		})
	}
	return previous
}

function restoreDOMGlobals(previous) {
	for (const [key, descriptor] of previous) {
		if (descriptor) Object.defineProperty(globalThis, key, descriptor)
		else delete globalThis[key]
	}
}
