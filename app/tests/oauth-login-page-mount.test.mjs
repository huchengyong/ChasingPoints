import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'

const appDir = dirname(dirname(fileURLToPath(import.meta.url)))
const loginPagePath = join(appDir, 'pages/login/login.vue')

const baseStubs = {
  '@dcloudio/uni-app': `
export const onLoad = (callback) => globalThis.__OAUTH_LOGIN_PAGE_MOUNT__.loaded.push(callback)
export const onShow = (callback) => globalThis.__OAUTH_LOGIN_PAGE_MOUNT__.shown.push(callback)
export const onUnload = (callback) => globalThis.__OAUTH_LOGIN_PAGE_MOUNT__.unloaded.push(callback)
`,
  '@/store/user.js': `
export const useUserStore = () => globalThis.__OAUTH_LOGIN_PAGE_MOUNT__.userStore
`,
  '@/api/auth.js': `
export const sendSms = (...args) => globalThis.__OAUTH_LOGIN_PAGE_MOUNT__.sendSms(...args)
export const login = (...args) => globalThis.__OAUTH_LOGIN_PAGE_MOUNT__.login(...args)
export const loginByOauth = (...args) => globalThis.__OAUTH_LOGIN_PAGE_MOUNT__.loginByOauth(...args)
export const wechatMiniLogin = (...args) => globalThis.__OAUTH_LOGIN_PAGE_MOUNT__.wechatMiniLogin(...args)
`,
  '@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`,
  '@/components/agreementConsentSheet.vue': `
import { h } from 'vue'
export default {
  props: ['show'],
  emits: ['close', 'agree', 'open-user', 'open-privacy'],
  setup(props) {
    return () => props.show ? h('view', { class: 'agreement-consent-sheet-stub' }, '需要先同意协议') : null
  }
}
`,
  '@/components/bindPhone.vue': `
import { h } from 'vue'
export default {
  props: ['show'],
  emits: ['close', 'success'],
  setup(props) {
    return () => props.show ? h('view', { class: 'bind-phone-stub' }, '绑定手机号') : null
  }
}
`
}

const realModuleReplacements = {
  '@/utils/entry-funnel.js': '../utils/entry-funnel.js',
  '@/utils/post-login-intent.js': '../utils/post-login-intent.js',
  '@/utils/oauth-login.js': '../utils/oauth-login.js'
}

test('Harmony OAuth login submits provider credential and finishes the login flow', async () => {
  const { wrapper, context, flushPromises, cleanup } = await mountLoginPage()
  try {
    await enterPhoneMode(wrapper, context)
    await triggerClick(wrapper, '.phone-agreement .agreement-row')
    await triggerClick(wrapper, '.third-party-btn')
    await flushPromises()

    assert.equal(context.uniCalls.getUserInfo, 1)
    assert.equal(context.uniCalls.login, 1)
    assert.deepEqual(context.oauthPayloads, [{
      provider: 'huawei',
      credential: 'huawei-auth-code',
      credential_type: 'authorization_code',
      platform: 'app-harmony',
      nick_name: '华为用户',
      avatar_url: 'https://example.com/avatar.png'
    }])
    assert.equal(context.userStore.loginCalls.length, 1)
    assert.deepEqual(context.switchTabs, ['/pages/index/index'])
    assert.equal(context.toasts.at(-1)?.title, '登录成功')
  } finally {
    await cleanup()
  }
})

test('Harmony OAuth login is blocked by agreement consent before requesting provider credentials', async () => {
  const { wrapper, context, cleanup } = await mountLoginPage()
  try {
    await enterPhoneMode(wrapper, context)
    await triggerClick(wrapper, '.third-party-btn')

    assert.equal(context.uniCalls.getUserInfo, 0)
    assert.equal(context.uniCalls.login, 0)
    assert.equal(context.oauthPayloads.length, 0)
    assert.ok(wrapper.element.querySelector('.agreement-consent-sheet-stub'), wrapper.html())
  } finally {
    await cleanup()
  }
})

test('Harmony OAuth login shows authorization-expired feedback without creating a session', async () => {
  const { wrapper, context, flushPromises, cleanup } = await mountLoginPage({
    loginByOauth: async (payload) => {
      context.oauthPayloads.push(payload)
      throw new Error('授权已失效，请重新授权')
    }
  })
  try {
    await enterPhoneMode(wrapper, context)
    await triggerClick(wrapper, '.phone-agreement .agreement-row')
    await triggerClick(wrapper, '.third-party-btn')
    await flushPromises()

    assert.equal(context.oauthPayloads.length, 1)
    assert.equal(context.userStore.loginCalls.length, 0)
    assert.equal(context.toasts.at(-1)?.title, '授权已失效，请重新授权')
  } finally {
    await cleanup()
  }
})

test('Harmony OAuth login shows upstream-unavailable feedback without creating a session', async () => {
  const { wrapper, context, flushPromises, cleanup } = await mountLoginPage({
    loginByOauth: async (payload) => {
      context.oauthPayloads.push(payload)
      throw new Error('华为登录服务暂不可用，请稍后重试')
    }
  })
  try {
    await enterPhoneMode(wrapper, context)
    await triggerClick(wrapper, '.phone-agreement .agreement-row')
    await triggerClick(wrapper, '.third-party-btn')
    await flushPromises()

    assert.equal(context.oauthPayloads.length, 1)
    assert.equal(context.userStore.loginCalls.length, 0)
    assert.equal(context.toasts.at(-1)?.title, '华为登录服务暂不可用，请稍后重试')
  } finally {
    await cleanup()
  }
})

async function mountLoginPage(overrides = {}) {
  const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
  const previousGlobals = installDOMGlobals(dom.window)
  const tempDir = await mkdtemp(join(appDir, '.oauth-login-page-mount-'))
  let wrapper

  const context = createMountContext(overrides)
  globalThis.__OAUTH_LOGIN_PAGE_MOUNT__ = context
  globalThis.uni = createUniStub(context)
  globalThis.getCurrentPages = () => [{ route: 'pages/login/login' }]

  try {
    const source = await readFile(loginPagePath, 'utf8')
    const { descriptor, errors } = parse(source, { filename: loginPagePath })
    assert.deepEqual(errors, [])

    const compiled = compileScript(descriptor, {
      id: 'oauth-login-page-real-mount',
      inlineTemplate: true,
      templateOptions: {
        compilerOptions: {
          isCustomElement: (tag) => ['view', 'text', 'image', 'uni-icons'].includes(tag)
        }
      }
    })

    let moduleSource = compiled.content
    let stubIndex = 0
    for (const [specifier, content] of Object.entries(baseStubs)) {
      const stubName = `stub-${stubIndex++}.mjs`
      await writeFile(join(tempDir, stubName), content)
      moduleSource = replaceModuleSpecifier(moduleSource, specifier, `./${stubName}`)
    }
    for (const [specifier, replacement] of Object.entries(realModuleReplacements)) {
      moduleSource = replaceModuleSpecifier(moduleSource, specifier, replacement)
    }
    await writeFile(join(tempDir, 'LoginPage.mjs'), moduleSource)

    const [{ mount, flushPromises }, { nextTick }] = await Promise.all([
      import('@vue/test-utils'),
      import('vue')
    ])
    const componentModule = await import(`${pathToFileURL(join(tempDir, 'LoginPage.mjs')).href}?v=${Date.now()}`)
    wrapper = mount(componentModule.default, {
      attachTo: document.body,
      global: { stubs: { 'uni-icons': true } }
    })

    return {
      wrapper,
      context,
      flushPromises: async () => {
        await flushPromises()
        await nextTick()
      },
      cleanup: async () => {
        wrapper?.unmount()
        delete globalThis.__OAUTH_LOGIN_PAGE_MOUNT__
        delete globalThis.uni
        delete globalThis.getCurrentPages
        await rm(tempDir, { recursive: true, force: true })
        restoreDOMGlobals(previousGlobals)
        dom.window.close()
      }
    }
  } catch (error) {
    wrapper?.unmount()
    delete globalThis.__OAUTH_LOGIN_PAGE_MOUNT__
    delete globalThis.uni
    delete globalThis.getCurrentPages
    await rm(tempDir, { recursive: true, force: true })
    restoreDOMGlobals(previousGlobals)
    dom.window.close()
    throw error
  }
}

async function enterPhoneMode(wrapper, context) {
  context.loaded[0]?.({ method: 'phone' })
  context.shown[0]?.()
  await wrapper.vm.$nextTick()
  assert.ok(wrapper.element.querySelector('.third-party-btn'), wrapper.html())
}

async function triggerClick(wrapper, selector) {
  const element = wrapper.element.querySelector(selector)
  assert.ok(element, `${selector} not found\n${wrapper.html()}`)
  element.dispatchEvent(new window.Event('click', { bubbles: true, cancelable: true }))
  await wrapper.vm.$nextTick()
}

function createMountContext(overrides = {}) {
  const storage = new Map()
  const context = {
    loaded: [],
    shown: [],
    unloaded: [],
    storage,
    toasts: [],
    switchTabs: [],
    oauthPayloads: [],
    uniCalls: {
      getUserInfo: 0,
      login: 0
    },
    userStore: {
      loginCalls: [],
      logoutCalls: 0,
      needBindPhone: false,
      login(payload) {
        this.loginCalls.push(payload)
      },
      logout() {
        this.logoutCalls += 1
      },
      setNeedBindPhone(value) {
        this.needBindPhone = Boolean(value)
      }
    },
    sendSms: async () => ({ success: true }),
    login: async () => ({ success: true }),
    loginByOauth: async (payload) => {
      context.oauthPayloads.push(payload)
      return {
        success: true,
        access_token: 'access-token',
        refresh_token: 'refresh-token',
        need_bind_phone: false,
        user_info: { id: 1, nickname: '华为用户' }
      }
    },
    wechatMiniLogin: async () => ({ success: true }),
    ...overrides
  }
  return context
}

function createUniStub(context) {
  return {
    getStorageSync(key) {
      return context.storage.get(key) || ''
    },
    setStorageSync(key, value) {
      context.storage.set(key, value)
    },
    removeStorageSync(key) {
      context.storage.delete(key)
    },
    showToast(payload) {
      context.toasts.push(payload)
    },
    switchTab(payload) {
      context.switchTabs.push(payload.url)
    },
    navigateBack() {},
    navigateTo() {},
    getSystemInfoSync() {
      return { uniPlatform: 'app-harmony' }
    },
    getUserInfo(payload) {
      context.uniCalls.getUserInfo += 1
      payload.success({
        userInfo: {
          nickName: '华为用户',
          avatarUrl: 'https://example.com/avatar.png'
        }
      })
    },
    login(payload) {
      context.uniCalls.login += 1
      payload.success({
        authResult: {
          authorizationCode: 'huawei-auth-code',
          openid: 'client-openid',
          unionID: 'client-union'
        }
      })
    }
  }
}

function replaceModuleSpecifier(source, specifier, replacement) {
  return source
    .replaceAll(`from '${specifier}'`, `from '${replacement}'`)
    .replaceAll(`from "${specifier}"`, `from "${replacement}"`)
}

function installDOMGlobals(window) {
  const keys = [
    'window',
    'document',
    'navigator',
    'Node',
    'Element',
    'HTMLElement',
    'HTMLButtonElement',
    'HTMLInputElement',
    'XMLSerializer',
    'SVGElement',
    'Event'
  ]
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
    if (descriptor) {
      Object.defineProperty(globalThis, key, descriptor)
    } else {
      delete globalThis[key]
    }
  }
}
