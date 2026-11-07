import test from 'node:test'
import assert from 'node:assert/strict'

import { createPageHarness, createUniRecorder, defer } from './helpers/page-harness.mjs'

const GLOBAL_KEY = '__SCAN_CONFIRM__'

const stubSources = {
  '@dcloudio/uni-app': `
export const onLoad = (callback) => callback({})
export const onShow = (callback) => { globalThis.${GLOBAL_KEY}.showCallbacks.push(callback) }
`,
  '@/api/match.js': `
const call = (name) => (...args) => globalThis.${GLOBAL_KEY}.api[name](...args)
export const startMatch = call('startMatch')
export const previewMatchReferee = call('previewMatchReferee')
export const joinMatchReferee = call('joinMatchReferee')
`,
  '@/store/user.js': `
export const useUserStore = () => globalThis.${GLOBAL_KEY}.user
`,
  '@/utils/page-theme.js': `
import { ref } from 'vue'
export const usePageTheme = () => ({ isDarkMode: ref(false) })
`,
  '@/utils/user-profile.js': `
export const resolveAvatarUrl = (avatar, id) => avatar || \`default-\${id}\`
`,
  '@/utils/quick-scan-entry.js': `
export const takeScanHandoff = (identity) => globalThis.${GLOBAL_KEY}.takeHandoff(identity)
export const startQuickScan = (options) => { globalThis.${GLOBAL_KEY}.rescans.push(options) }
`
}

// 纯逻辑工具直接复用真实实现，避免在测试里复写开局与路由规则。
const realModules = [
  '@/utils/game-types.js',
  '@/utils/game-type-preference.js',
  '@/utils/ongoing-match-guard.js',
  '@/utils/start-match.js'
]

const harness = createPageHarness({
  pageFile: 'subPages/match/scanConfirm.vue',
  globalKey: GLOBAL_KEY,
  stubSources,
  realModules
})

const startHandoff = (overrides = {}) => ({
  identity: { userId: 7, authGeneration: 1 },
  mode: 'start',
  inviteToken: 'secret.token',
  opponent: { id: 9, nickname: '对手乙', avatar: '' },
  preset: null,
  rematch: null,
  ...overrides
})

const refereeHandoff = () => ({
  identity: { userId: 7, authGeneration: 1 },
  mode: 'referee',
  refereeJoin: { match_id: 77, join_token: 'referee-secret' },
  preset: null,
  rematch: null
})

const createdResponse = (matchId = 55) => ({
  success: true,
  action: 'created',
  match: { match_id: matchId, game_type: 3, opponent_name: '对手乙', opponent_avatar: '' }
})

const refereePreview = {
  player1_id: 1,
  player1_name: '甲',
  player1_score: 3,
  player2_id: 2,
  player2_name: '乙',
  player2_score: 2,
  game_type_name: '中式八球',
  match_mode: 'ranked'
}

const mountConfirm = async ({ handoff = startHandoff(), api = {}, storage = {}, configureUni } = {}) => {
  const { reactive } = await harness.vueRuntimePromise
  const user = reactive({ isLoggedIn: true, userId: 7, authGeneration: 1 })
  const recorder = createUniRecorder()
  for (const [key, value] of Object.entries(storage)) recorder.storage.set(key, value)
  configureUni?.(recorder)
  const context = {
    user,
    api,
    handoff,
    showCallbacks: [],
    rescans: [],
    takeHandoff: (identity) => {
      context.takenWith = identity
      return context.handoff ?? null
    }
  }
  const mounted = await harness.mountPage({ context, uni: recorder.uni })
  await mounted.flushPromises()
  return { ...mounted, user, context, recorder }
}

const chipByLabel = (wrapper, label) => wrapper.findAll('.chip').find((chip) => chip.text().includes(label))
const startButton = (wrapper) => wrapper.find('.primary-btn')
const tap = async (target, flushPromises) => {
  await target.trigger('tap')
  await flushPromises()
}

test('picking a game type and starting creates the match and replaces the page with the scoring page', async () => {
  const payloads = []
  const { wrapper, flushPromises, recorder, context, cleanup } = await mountConfirm({
    api: { startMatch: async (payload) => { payloads.push(payload); return createdResponse() } }
  })
  try {
    assert.deepEqual(context.takenWith, { userId: 7, authGeneration: 1 })
    assert.equal(wrapper.text().includes('对手乙'), true)
    assert.equal(startButton(wrapper).element.disabled, true, 'no default game type: start stays disabled')

    await tap(chipByLabel(wrapper, '中式八球'), flushPromises)
    assert.equal(startButton(wrapper).element.disabled, false)

    await tap(startButton(wrapper), flushPromises)

    assert.deepEqual(payloads, [{
      game_type: 3,
      match_mode: 'practice',
      visibility: 'private',
      invite_token: 'secret.token',
      match_format: 'free',
      target_wins: 0
    }])
    const redirect = recorder.lastOf('redirectTo')
    assert.match(redirect.options.url, /^\/subPages\/match\/playing\?match_id=55&game_type=3/)
    assert.doesNotMatch(redirect.options.url, /secret/)
    assert.equal(recorder.callsOf('navigateTo').length, 0)
    assert.deepEqual(recorder.lastOf('setNavigationBarTitle').options, { title: '确认对局' })
  } finally {
    await cleanup()
  }
})

test('the saved default game type is preselected and marked', async () => {
  const { wrapper, cleanup } = await mountConfirm({ storage: { 'default-game-type:7': 2 } })
  try {
    const active = wrapper.findAll('.chip.active')
    assert.equal(active.length, 1)
    assert.match(active[0].text(), /九球追分/)
    assert.match(active[0].text(), /默认/)
    assert.equal(startButton(wrapper).element.disabled, false)
    assert.equal(wrapper.find('.default-choice').exists(), false, 'the default choice itself needs no "set as default" prompt')
  } finally {
    await cleanup()
  }
})

test('a different choice can be saved as the new default', async () => {
  const { wrapper, flushPromises, recorder, cleanup } = await mountConfirm({
    storage: { 'default-game-type:7': 2 },
    api: { startMatch: async () => createdResponse() }
  })
  try {
    await tap(chipByLabel(wrapper, '斯诺克'), flushPromises)
    assert.equal(wrapper.find('.default-choice').exists(), true)
    await tap(wrapper.find('.default-choice'), flushPromises)
    await tap(startButton(wrapper), flushPromises)

    assert.equal(recorder.storage.get('default-game-type:7'), 1)
  } finally {
    await cleanup()
  }
})

test('without the checkbox the default stays untouched', async () => {
  const { wrapper, flushPromises, recorder, cleanup } = await mountConfirm({
    storage: { 'default-game-type:7': 2 },
    api: { startMatch: async () => createdResponse() }
  })
  try {
    await tap(chipByLabel(wrapper, '斯诺克'), flushPromises)
    await tap(startButton(wrapper), flushPromises)

    assert.equal(recorder.storage.get('default-game-type:7'), 2)
  } finally {
    await cleanup()
  }
})

test('the match type choice maps to the existing mode and visibility pairs', async () => {
  const payloads = []
  const { wrapper, flushPromises, cleanup } = await mountConfirm({
    storage: { 'default-game-type:7': 3 },
    api: { startMatch: async (payload) => { payloads.push(payload); return createdResponse() } }
  })
  try {
    const options = wrapper.findAll('.mode-option')
    assert.equal(options.length, 3)
    assert.equal(options[0].classes().includes('active'), true, 'practice + private is the default')

    await tap(options[2], flushPromises)
    await tap(startButton(wrapper), flushPromises)

    assert.equal(payloads[0].match_mode, 'ranked')
    assert.equal(payloads[0].visibility, 'public')
  } finally {
    await cleanup()
  }
})

test('a rematch preset preselects the game type, mode and visibility', async () => {
  const payloads = []
  const { wrapper, flushPromises, cleanup } = await mountConfirm({
    handoff: startHandoff({ preset: { gameType: 1, matchMode: 'practice', visibility: 'public' } }),
    api: { startMatch: async (payload) => { payloads.push(payload); return createdResponse() } }
  })
  try {
    assert.match(wrapper.find('.chip.active').text(), /斯诺克/)
    const options = wrapper.findAll('.mode-option')
    assert.equal(options[1].classes().includes('active'), true)

    await tap(startButton(wrapper), flushPromises)
    assert.equal(payloads[0].game_type, 1)
    assert.equal(payloads[0].match_mode, 'practice')
    assert.equal(payloads[0].visibility, 'public')
  } finally {
    await cleanup()
  }
})

test('a rejected invite shows the reason, creates nothing and offers a rescan that keeps the rematch request', async () => {
  const rematch = { opponent_id: 9, game_type: 3, match_mode: 'practice' }
  const { wrapper, flushPromises, recorder, context, cleanup } = await mountConfirm({
    handoff: startHandoff({ rematch, preset: { gameType: 3, matchMode: 'practice', visibility: 'private' } }),
    api: { startMatch: async () => ({ success: false, message: '匹配二维码已失效，请让对方刷新二维码' }) }
  })
  try {
    await tap(startButton(wrapper), flushPromises)

    assert.match(wrapper.find('.inline-error').text(), /匹配二维码已失效/)
    assert.equal(recorder.callsOf('redirectTo').length, 0)

    const rescan = wrapper.find('.ghost-btn')
    assert.equal(rescan.text(), '重新扫码')
    await tap(rescan, flushPromises)
    assert.deepEqual(context.rescans, [{ rematch, replace: true }])
  } finally {
    await cleanup()
  }
})

test('a network failure while starting is reported inline and the button recovers', async () => {
  const { wrapper, flushPromises, cleanup } = await mountConfirm({
    storage: { 'default-game-type:7': 3 },
    api: { startMatch: async () => { throw new Error('offline') } }
  })
  const originalError = console.error
  console.error = () => {}
  try {
    await tap(startButton(wrapper), flushPromises)

    assert.match(wrapper.find('.inline-error').text(), /匹配失败/)
    assert.equal(startButton(wrapper).element.disabled, false)
  } finally {
    console.error = originalError
    await cleanup()
  }
})

test('while a start request is in flight the button is locked and only one request is sent', async () => {
  const pending = defer()
  let requests = 0
  const { wrapper, flushPromises, recorder, cleanup } = await mountConfirm({
    storage: { 'default-game-type:7': 3 },
    api: { startMatch: () => { requests += 1; return pending.promise } }
  })
  try {
    await tap(startButton(wrapper), flushPromises)
    assert.equal(startButton(wrapper).element.disabled, true)
    assert.equal(startButton(wrapper).text(), '创建中...')
    assert.equal(wrapper.find('.ghost-btn').element.disabled, true)

    await startButton(wrapper).trigger('tap')
    assert.equal(requests, 1)

    pending.resolve(createdResponse())
    await flushPromises()
    assert.equal(recorder.callsOf('redirectTo').length, 1)
  } finally {
    await cleanup()
  }
})

test('an existing unfinished match prompts and enters that match on confirmation', async () => {
  const { wrapper, flushPromises, recorder, cleanup } = await mountConfirm({
    storage: { 'default-game-type:7': 3 },
    configureUni: (uniRecorder) => {
      uniRecorder.uni.showModal = (options) => {
        uniRecorder.calls.push({ name: 'showModal', options })
        options.success?.({ confirm: true })
      }
    },
    api: {
      startMatch: async () => ({
        success: true,
        action: 'blocked',
        block_reason: 'self_ongoing',
        message: '你还有未结束的对局，请先处理',
        ongoing_match: { match_id: 31, game_type: 3, opponent_name: '老对手' }
      })
    }
  })
  try {
    await tap(startButton(wrapper), flushPromises)

    assert.equal(recorder.lastOf('showModal').options.title, '你有未结束的对局')
    assert.match(recorder.lastOf('redirectTo').options.url, /match_id=31/)
  } finally {
    await cleanup()
  }
})

test('a busy opponent is a toast and the page stays', async () => {
  const { wrapper, flushPromises, recorder, cleanup } = await mountConfirm({
    storage: { 'default-game-type:7': 3 },
    api: {
      startMatch: async () => ({
        success: true,
        action: 'blocked',
        block_reason: 'opponent_ongoing',
        message: '对手还有未结束的对局，暂时无法开始新的 PK'
      })
    }
  })
  try {
    await tap(startButton(wrapper), flushPromises)

    assert.match(recorder.lastOf('showToast').options.title, /对手还有未结束/)
    assert.equal(recorder.callsOf('redirectTo').length, 0)
  } finally {
    await cleanup()
  }
})

test('a start result that arrives after the account changed neither navigates nor prompts', async () => {
  const pending = defer()
  const { wrapper, flushPromises, recorder, user, cleanup } = await mountConfirm({
    storage: { 'default-game-type:7': 3 },
    api: { startMatch: () => pending.promise }
  })
  try {
    await tap(startButton(wrapper), flushPromises)
    user.authGeneration = 2
    pending.resolve(createdResponse())
    await flushPromises()

    assert.equal(recorder.callsOf('redirectTo').length, 0)
    assert.equal(recorder.callsOf('showModal').length, 0)
    assert.equal(wrapper.find('.inline-error').exists(), false)
  } finally {
    await cleanup()
  }
})

test('coming back after the account changed discards the scan', async () => {
  const { wrapper, flushPromises, user, context, cleanup } = await mountConfirm()
  try {
    user.authGeneration = 2
    context.showCallbacks.forEach((callback) => callback())
    await flushPromises()

    assert.match(wrapper.text(), /登录状态已变更，请重新扫码/)
    assert.equal(wrapper.find('.primary-btn').text(), '重新扫码')
  } finally {
    await cleanup()
  }
})

test('without handoff data the page asks to scan again', async () => {
  const { wrapper, flushPromises, context, cleanup } = await mountConfirm({ handoff: null })
  try {
    assert.match(wrapper.text(), /扫码信息已失效，请重新扫码/)
    assert.equal(wrapper.find('.chip').exists(), false)

    await tap(wrapper.find('.primary-btn'), flushPromises)
    assert.deepEqual(context.rescans, [{ rematch: null, replace: true }])
  } finally {
    await cleanup()
  }
})

test('a logged out visitor never receives a handoff', async () => {
  const { reactive } = await harness.vueRuntimePromise
  const recorder = createUniRecorder()
  const context = {
    user: reactive({ isLoggedIn: false, userId: 0, authGeneration: 1 }),
    api: {},
    showCallbacks: [],
    rescans: [],
    takeHandoff: () => { throw new Error('must not be asked for a handoff') }
  }
  const { wrapper, flushPromises, cleanup } = await harness.mountPage({ context, uni: recorder.uni })
  try {
    await flushPromises()
    assert.match(wrapper.text(), /扫码信息已失效/)
  } finally {
    await cleanup()
  }
})

test('referee mode previews the match first and joins only after confirmation', async () => {
  const calls = []
  const { wrapper, flushPromises, recorder, cleanup } = await mountConfirm({
    handoff: refereeHandoff(),
    api: {
      previewMatchReferee: async (join) => { calls.push(['preview', join]); return { success: true, preview: refereePreview } },
      joinMatchReferee: async (join) => {
        calls.push(['join', join])
        return { success: true, match_id: 77, match: { game_type: 3, opponent_name: '乙', opponent_avatar: '' } }
      }
    }
  })
  try {
    assert.deepEqual(calls, [['preview', { match_id: 77, join_token: 'referee-secret' }]])
    assert.deepEqual(recorder.lastOf('setNavigationBarTitle').options, { title: '确认执裁' })
    const text = wrapper.text()
    assert.match(text, /确认担任本场裁判/)
    assert.match(text, /甲/)
    assert.match(text, /3 : 2/)
    assert.match(text, /中式八球 · 排位赛/)
    assert.equal(recorder.callsOf('redirectTo').length, 0)

    await tap(startButton(wrapper), flushPromises)

    assert.equal(calls.length, 2)
    assert.equal(calls[1][0], 'join')
    const url = recorder.lastOf('redirectTo').options.url
    assert.match(url, /^\/subPages\/match\/playing\?match_id=77&game_type=3/)
    assert.doesNotMatch(url, /referee-secret/)
  } finally {
    await cleanup()
  }
})

test('declining the referee role goes back without joining', async () => {
  let joins = 0
  const { wrapper, flushPromises, recorder, cleanup } = await mountConfirm({
    handoff: refereeHandoff(),
    api: {
      previewMatchReferee: async () => ({ success: true, preview: refereePreview }),
      joinMatchReferee: async () => { joins += 1; return { success: true } }
    }
  })
  try {
    await tap(wrapper.find('.ghost-btn'), flushPromises)

    assert.equal(joins, 0)
    assert.equal(recorder.callsOf('navigateBack').length, 1)
  } finally {
    await cleanup()
  }
})

test('an expired referee code is explained and can be rescanned', async () => {
  const { wrapper, flushPromises, context, cleanup } = await mountConfirm({
    handoff: refereeHandoff(),
    api: { previewMatchReferee: async () => ({ success: false, message: '裁判码已失效，请重新扫码' }) }
  })
  try {
    assert.match(wrapper.text(), /裁判码已失效，请重新扫码/)
    assert.equal(wrapper.find('.referee-card').exists(), false)

    await tap(wrapper.find('.primary-btn'), flushPromises)
    assert.deepEqual(context.rescans, [{ rematch: null, replace: true }])
  } finally {
    await cleanup()
  }
})

test('a referee code that fails at join time switches to the error state', async () => {
  const { wrapper, flushPromises, recorder, cleanup } = await mountConfirm({
    handoff: refereeHandoff(),
    api: {
      previewMatchReferee: async () => ({ success: true, preview: refereePreview }),
      joinMatchReferee: async () => ({ success: false, message: '该对局已有裁判' })
    }
  })
  try {
    await tap(startButton(wrapper), flushPromises)

    assert.match(wrapper.text(), /该对局已有裁判/)
    assert.equal(recorder.callsOf('redirectTo').length, 0)
  } finally {
    await cleanup()
  }
})
