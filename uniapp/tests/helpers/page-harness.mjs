import assert from 'node:assert/strict'
import { after } from 'node:test'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'

const appDir = dirname(dirname(dirname(fileURLToPath(import.meta.url))))

const DEFAULT_CUSTOM_ELEMENTS = ['view', 'text', 'image', 'canvas', 'scroll-view', 'uni-icons']

const installDOMGlobals = (window) => {
  const keys = ['window', 'document', 'navigator', 'Node', 'Element', 'HTMLElement', 'HTMLInputElement', 'HTMLButtonElement', 'SVGElement', 'Event', 'Document', 'DocumentFragment', 'ShadowRoot', 'CustomEvent']
  const previous = new Map()
  for (const key of keys) {
    previous.set(key, Object.getOwnPropertyDescriptor(globalThis, key))
    Object.defineProperty(globalThis, key, { configurable: true, writable: true, value: window[key] })
  }
  return previous
}

const restoreDOMGlobals = (previous) => {
  for (const [key, descriptor] of previous) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor)
    else delete globalThis[key]
  }
}

export const defer = () => {
  let resolve
  let reject
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

// 记录调用的 uni 桩；overrides 可替换或补充单个接口（例如让 showModal 自动点确认）。
export const createUniRecorder = (overrides = {}) => {
  const calls = []
  const storage = new Map()
  const record = (name) => (options) => { calls.push({ name, options }) }
  const names = [
    'navigateTo', 'redirectTo', 'switchTab', 'navigateBack', 'reLaunch', 'showToast', 'showModal',
    'showLoading', 'hideLoading', 'showActionSheet', 'setNavigationBarTitle', 'hideTabBar', 'showTabBar',
    'stopPullDownRefresh'
  ]
  const uni = {
    getStorageSync: (key) => storage.get(key) ?? '',
    setStorageSync: (key, value) => { storage.set(key, value) },
    removeStorageSync: (key) => { storage.delete(key) },
    $on: () => {},
    $off: () => {},
    $emit: () => {},
    ...Object.fromEntries(names.map((name) => [name, record(name)])),
    ...overrides
  }
  return {
    uni,
    calls,
    storage,
    callsOf: (name) => calls.filter((item) => item.name === name),
    lastOf: (name) => calls.filter((item) => item.name === name).at(-1)
  }
}

// 把页面的 .vue 编译成可在 Node 里挂载的模块：@/ 依赖要么换成桩（行为由 globalThis[globalKey] 注入），要么指向真实实现。
export const createPageHarness = ({
  pageFile,
  globalKey,
  stubSources,
  realModules = [],
  customElements = DEFAULT_CUSTOM_ELEMENTS
}) => {
  // Vue 的 runtime-dom 在模块加载时抓取 document，因此必须先装好 jsdom 再引入 vue。
  const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
  const previousGlobals = installDOMGlobals(dom.window)
  const vueRuntimePromise = import('vue')
  const testUtilsPromise = import('@vue/test-utils')

  after(() => {
    restoreDOMGlobals(previousGlobals)
    dom.window.close()
  })

  const compilePage = async (tempDir) => {
    const source = await readFile(join(appDir, pageFile), 'utf8')
    const { descriptor, errors } = parse(source, { filename: pageFile })
    assert.deepEqual(errors, [])
    const compiled = compileScript(descriptor, {
      id: `harness-${globalKey}`,
      inlineTemplate: true,
      templateOptions: {
        compilerOptions: { isCustomElement: (tag) => customElements.includes(tag) }
      }
    })
    let moduleSource = compiled.content
    for (const [specifier, content] of Object.entries(stubSources)) {
      if (!moduleSource.includes(`'${specifier}'`)) continue
      const stubName = `stub-${specifier.replace(/\W+/g, '_')}.mjs`
      await writeFile(join(tempDir, stubName), content)
      moduleSource = moduleSource.replaceAll(`from '${specifier}'`, `from './${stubName}'`)
    }
    for (const specifier of realModules) {
      moduleSource = moduleSource.replaceAll(`from '${specifier}'`, `from '../${specifier.replace('@/', '')}'`)
    }
    assert.doesNotMatch(moduleSource, /from '@\//, '页面的 @/ 依赖都必须有桩或真实实现')
    const fileName = 'page.mjs'
    await writeFile(join(tempDir, fileName), moduleSource)
    return pathToFileURL(join(tempDir, fileName)).href
  }

  // context 里放测试注入的行为与状态；props 传给页面；返回的 cleanup 必须在 finally 里调用。
  const mountPage = async ({ context, uni, props } = {}) => {
    const tempDir = await mkdtemp(join(appDir, `.harness-${globalKey}-`))
    globalThis.uni = uni
    globalThis[globalKey] = context
    const cleanupGlobals = () => {
      delete globalThis[globalKey]
      delete globalThis.uni
    }
    try {
      const moduleUrl = await compilePage(tempDir)
      const [{ mount, flushPromises }, componentModule] = await Promise.all([
        testUtilsPromise,
        import(`${moduleUrl}?v=${Date.now()}-${Math.random()}`)
      ])
      const wrapper = mount(componentModule.default, { props, global: { stubs: { 'uni-icons': true } } })
      return {
        wrapper,
        flushPromises,
        cleanup: async () => {
          wrapper.unmount()
          cleanupGlobals()
          await rm(tempDir, { recursive: true, force: true })
        }
      }
    } catch (error) {
      cleanupGlobals()
      await rm(tempDir, { recursive: true, force: true })
      throw error
    }
  }

  return { appDir, mountPage, vueRuntimePromise }
}

// 加载依赖 @/ 别名的 .js 模块（如组合式函数）：每个 @/ 依赖要么换成桩，要么递归改写成可在 Node 里解析的副本；
// 相对路径依赖直接指向原文件。行为由 globalThis[globalKey] 注入，与页面 harness 一致。
export const createModuleHarness = ({ entry, globalKey, stubSources = {} }) => {
  const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
  const previousGlobals = installDOMGlobals(dom.window)
  const vueRuntimePromise = import('vue')

  after(() => {
    restoreDOMGlobals(previousGlobals)
    dom.window.close()
  })

  const materialize = async (specifier, tempDir, seen) => {
    const fileName = `${specifier.replace(/\W+/g, '_')}.mjs`
    if (seen.has(specifier)) return fileName
    seen.add(specifier)
    if (Object.hasOwn(stubSources, specifier)) {
      await writeFile(join(tempDir, fileName), stubSources[specifier])
      return fileName
    }
    const originalPath = join(appDir, specifier.replace('@/', ''))
    let source = await readFile(originalPath, 'utf8')
    // 先把原有的相对路径依赖指向原文件，再处理别名与桩（顺序不能反，否则会误改刚生成的 ./ 路径）
    source = source.replace(/from '(\.{1,2}\/[^']+)'/g, (_match, relative) => (
      `from '${pathToFileURL(join(dirname(originalPath), relative)).href}'`
    ))
    // 桩可以替换任意包名（如 @dcloudio/uni-app），其余只处理 @/ 别名
    for (const [, dependency] of [...source.matchAll(/from '([^']+)'/g)]) {
      if (!dependency.startsWith('@/') && !Object.hasOwn(stubSources, dependency)) continue
      const dependencyFile = await materialize(dependency, tempDir, seen)
      source = source.replaceAll(`from '${dependency}'`, `from './${dependencyFile}'`)
    }
    assert.doesNotMatch(source, /from '@\//, `${specifier}: 未解析的 @/ 依赖`)
    await writeFile(join(tempDir, fileName), source)
    return fileName
  }

  const load = async ({ context, uni }) => {
    const tempDir = await mkdtemp(join(appDir, `.harness-${globalKey}-`))
    globalThis.uni = uni
    globalThis[globalKey] = context
    try {
      const entryFile = await materialize(entry, tempDir, new Set())
      const module = await import(`${pathToFileURL(join(tempDir, entryFile)).href}?v=${Date.now()}-${Math.random()}`)
      return {
        module,
        cleanup: async () => {
          delete globalThis[globalKey]
          delete globalThis.uni
          await rm(tempDir, { recursive: true, force: true })
        }
      }
    } catch (error) {
      delete globalThis[globalKey]
      delete globalThis.uni
      await rm(tempDir, { recursive: true, force: true })
      throw error
    }
  }

  return { load, vueRuntimePromise }
}
