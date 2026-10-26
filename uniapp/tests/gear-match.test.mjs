import test from 'node:test'
import assert from 'node:assert/strict'

import {
  applyCueToDraft,
  buildGearMatchDraft,
  buildGearMatchPayload,
  canEditGearMatch,
  createGearMatchLoader,
  describeGearMatchSide,
  formatGearEditDeadline,
  isGearMatchDraftEmpty,
  isGearMatchHiddenResponse,
  listMatchBreakOptions,
  listMatchCueOptions,
  shouldShowMatchGear
} from '../utils/gear-match.js'

const deferred = () => {
  let resolve
  let reject
  const promise = new Promise((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

test('only completed matches viewed by a player request the gear record', () => {
  assert.equal(shouldShowMatchGear({ status: 2, viewerRole: 'player1' }), true)
  assert.equal(shouldShowMatchGear({ status: 2, viewerRole: 'player2' }), true)
  assert.equal(shouldShowMatchGear({ status: 2, isPlayer1: true }), true)
  assert.equal(shouldShowMatchGear({ status: 2, viewerRole: 'referee' }), false)
  assert.equal(shouldShowMatchGear({ status: 2, viewerRole: 'spectator' }), false)
  assert.equal(shouldShowMatchGear({ status: 2 }), false)
  assert.equal(shouldShowMatchGear({ status: 1, viewerRole: 'player1' }), false)
  assert.equal(shouldShowMatchGear({ status: 2, viewerRole: 'player1', isSpectate: true }), false)
  assert.equal(shouldShowMatchGear(), false)
})

test('refusals that mean "nothing to show" hide the card instead of showing an error', () => {
  for (const reason of ['not_participant', 'not_completed', 'not_found']) {
    assert.equal(isGearMatchHiddenResponse({ success: false, reason }), true)
  }
  assert.equal(isGearMatchHiddenResponse({ success: false, reason: '' }), false)
  assert.equal(isGearMatchHiddenResponse({ success: false, message: '读取本场用杆失败' }), false)
  assert.equal(isGearMatchHiddenResponse({ success: true }), false)
  assert.equal(isGearMatchHiddenResponse(null), false)
})

test('an unrecorded side has no rows and a recorded side lists cue, tip, chalk and break cue in order', () => {
  assert.deepEqual(describeGearMatchSide({ recorded: false }), { recorded: false, rows: [] })
  assert.deepEqual(describeGearMatchSide(null), { recorded: false, rows: [] })

  const side = {
    recorded: true,
    cue: { id: 1, category: 'cue', brand_name: 'Alpha', model_name: 'A-100', cue_style: 'pool' },
    tip_product: { id: 5, brand_name: 'Alpha', name: 'A-Tip' },
    chalk_product: { id: 6, brand_name: '', name: '球房巧克' },
    break_cue: { id: 2, category: 'break_jump', brand_name: 'Alpha', model_name: 'A-Break', roles: ['break'] }
  }
  const view = describeGearMatchSide(side)
  assert.equal(view.recorded, true)
  assert.deepEqual(view.rows.map((row) => [row.key, row.label, row.title]), [
    ['cue', '球杆', 'Alpha A-100'],
    ['tip', '皮头', 'Alpha A-Tip'],
    ['chalk', '巧克', '球房巧克'],
    ['break', '冲杆', 'Alpha A-Break']
  ])
  assert.equal(view.rows[0].subtitle, '美式杆')

  // 记录里只有球杆：不凭空展示其他行
  assert.deepEqual(describeGearMatchSide({ recorded: true, cue: side.cue }).rows.map((row) => row.key), ['cue'])
})

test('the edit deadline reads as a month and day', () => {
  assert.equal(formatGearEditDeadline(new Date(2026, 4, 3, 12).toISOString()), '5月3日前可修改')
  assert.equal(formatGearEditDeadline(new Date(2026, 11, 31, 8).toISOString()), '12月31日前可修改')
  assert.equal(formatGearEditDeadline(''), '')
  assert.equal(formatGearEditDeadline('not a date'), '')
  assert.equal(formatGearEditDeadline(undefined), '')
})

test('editing needs the server flag and an unexpired deadline', () => {
  const deadline = new Date(2026, 4, 3).getTime()
  const response = { success: true, editable: true, edit_deadline: new Date(deadline).toISOString() }
  assert.equal(canEditGearMatch(response, deadline - 1000), true)
  assert.equal(canEditGearMatch(response, deadline), false, '到点即不可改')
  assert.equal(canEditGearMatch(response, deadline + 1000), false)
  assert.equal(canEditGearMatch({ ...response, editable: false }, deadline - 1000), false)
  assert.equal(canEditGearMatch({ success: false }, 0), false)
  assert.equal(canEditGearMatch(null), false)
  // 没有可解析的截止时间时信任服务端的判断
  assert.equal(canEditGearMatch({ success: true, editable: true, edit_deadline: '' }, 123), true)
})

test('the draft starts from the current record and switching cues prefills tip and chalk from that cue', () => {
  const side = {
    recorded: true,
    cue: { id: 1 }, tip_product: { id: 5 }, chalk_product: { id: 6 }, break_cue: { id: 2 }
  }
  const draft = buildGearMatchDraft(side)
  assert.deepEqual(draft, { cueItemId: 1, tipProduct: { id: 5 }, chalkProduct: { id: 6 }, breakItemId: 2 })
  assert.deepEqual(buildGearMatchDraft({ recorded: false }), { cueItemId: 0, tipProduct: null, chalkProduct: null, breakItemId: 0 })

  const other = { id: 9, tip_product: { id: 15 }, chalk_product: null }
  const switched = applyCueToDraft(draft, other)
  assert.equal(switched.cueItemId, 9)
  assert.deepEqual(switched.tipProduct, { id: 15 })
  assert.equal(switched.chalkProduct, null, '新球杆没有巧克就清空，而不是沿用旧的')
  assert.equal(switched.breakItemId, 2, '换球杆不影响冲杆')

  const cleared = applyCueToDraft(draft, null)
  assert.equal(cleared.cueItemId, 0)
  assert.equal(cleared.tipProduct, null)
})

test('the request body carries ids only and drops the break cue where the game has none', () => {
  const draft = { cueItemId: 1, tipProduct: { id: 5 }, chalkProduct: null, breakItemId: 2 }
  assert.deepEqual(buildGearMatchPayload(77, draft), {
    match_id: 77, cue_item_id: 1, tip_product_id: 5, chalk_product_id: 0, break_item_id: 2
  })
  assert.equal(buildGearMatchPayload(77, draft, { supportsBreak: false }).break_item_id, 0)
  assert.equal(buildGearMatchPayload('abc', draft).match_id, 0)
  assert.equal(isGearMatchDraftEmpty(buildGearMatchDraft(null)), true)
  assert.equal(isGearMatchDraftEmpty(draft), false)
  assert.equal(isGearMatchDraftEmpty({ cueItemId: 0, tipProduct: null, chalkProduct: { id: 6 }, breakItemId: 0 }), false)
})

test('editor options include retired cues but never deleted ones, and break options need the break role', () => {
  const items = [
    { id: 1, category: 'cue', status: 1 },
    { id: 2, category: 'cue', status: 2 },
    { id: 3, category: 'cue', status: 3 },
    { id: 4, category: 'break_jump', status: 1, roles: ['break'] },
    { id: 5, category: 'break_jump', status: 2, roles: ['break', 'jump'] },
    { id: 6, category: 'break_jump', status: 1, roles: ['jump'] },
    { id: 7, category: 'break_jump', status: 3, roles: ['break'] }
  ]
  assert.deepEqual(listMatchCueOptions(items).map((item) => item.id), [1, 2])
  assert.deepEqual(listMatchBreakOptions(items).map((item) => item.id), [4, 5])
  assert.deepEqual(listMatchCueOptions(), [])
})

// ========== 读取器：首次进入、重复读取、切号与迟到响应 ==========

const createLoader = (handler) => {
  const identity = { userId: 1, authGeneration: 1 }
  const calls = []
  const loader = createGearMatchLoader({
    request: (params) => {
      calls.push(params)
      return handler(calls.length)
    },
    readIdentity: () => ({ ...identity })
  })
  return { loader, calls, identity }
}

test('the first read returns its response and carries the match id', async () => {
  const { loader, calls } = createLoader(async () => ({ success: true, marker: 'ok' }))
  const result = await loader.load(500)
  assert.deepEqual(calls, [{ match_id: 500 }])
  assert.equal(result.stale, false)
  assert.equal(result.response.marker, 'ok')
})

test('only the newest of two overlapping reads is accepted', async () => {
  const gates = [deferred(), deferred()]
  const { loader } = createLoader((n) => gates[n - 1].promise)
  const first = loader.load(500)
  const second = loader.load(500)
  gates[1].resolve({ success: true, marker: 'second' })
  gates[0].resolve({ success: true, marker: 'first' })
  assert.equal((await first).stale, true, 'the older read must not overwrite the newer one')
  const accepted = await second
  assert.equal(accepted.stale, false)
  assert.equal(accepted.response.marker, 'second')
})

test('a response that arrives after the account switched is dropped', async () => {
  const gate = deferred()
  const { loader, identity } = createLoader(() => gate.promise)
  const pending = loader.load(500)
  identity.userId = 2
  identity.authGeneration = 2
  gate.resolve({ success: true, marker: 'old-account' })
  assert.equal((await pending).stale, true)
})

test('a response after the auth generation changed or after cancel is dropped', async () => {
  const gate = deferred()
  const { loader, identity } = createLoader(() => gate.promise)
  const pending = loader.load(500)
  identity.authGeneration = 9
  gate.resolve({ success: true })
  assert.equal((await pending).stale, true)

  const gate2 = deferred()
  const second = createLoader(() => gate2.promise)
  const cancelled = second.loader.load(500)
  second.loader.cancel()
  gate2.resolve({ success: true })
  assert.equal((await cancelled).stale, true)
})

test('a failed read throws so the card can show a retry state', async () => {
  const { loader } = createLoader(async () => { throw new Error('网络错误') })
  await assert.rejects(() => loader.load(500), /网络错误/)
})
