import test from 'node:test'
import assert from 'node:assert/strict'

import {
  parseMatchScanPayload,
  resolveMatchScanPayload,
  scanAndResolveMatchCode
} from '../utils/match-scan.js'

test('match scanner rejects mutable legacy profile QR payloads and accepts signed invite shape', () => {
  assert.deepEqual(parseMatchScanPayload(JSON.stringify({
    user_id: 2002,
    nickname: '可伪造昵称'
  })), {
    type: 'error',
    message: '该二维码已不再支持，请让对方刷新匹配二维码'
  })
  assert.deepEqual(parseMatchScanPayload('header.payload'), {
    type: 'preview_match_invite',
    inviteToken: 'header.payload'
  })
  assert.equal(parseMatchScanPayload('headerXpayload').type, 'error')
  assert.equal(parseMatchScanPayload('not a code').type, 'error')
})

test('match scanner gets trusted opponent preview before constructing a start action', async () => {
  const result = await resolveMatchScanPayload({
    rawValue: 'signed.invite',
    previewMatchInvite: async (token) => ({
      success: true,
      preview: {
        opponent_id: 2002,
        opponent_name: '可信对手',
        opponent_avatar: 'trusted.png'
      },
      token
    })
  })
  assert.deepEqual(result, {
    type: 'start_match',
    inviteToken: 'signed.invite',
    opponent: {
      id: 2002,
      nickname: '可信对手',
      avatar: 'trusted.png'
    }
  })
})

test('match scanner preserves referee join credentials and explains expired invites', async () => {
  assert.deepEqual(parseMatchScanPayload(JSON.stringify({
    type: 'match_referee',
    match_id: 88,
    join_token: 'server-issued-join-token'
  })), {
    type: 'join_referee',
    refereeJoin: {
      match_id: 88,
      join_token: 'server-issued-join-token'
    }
  })

  const expired = await resolveMatchScanPayload({
    rawValue: 'signed.expired',
    previewMatchInvite: async () => ({
      success: false,
      message: '匹配二维码已失效，请让对方刷新二维码'
    })
  })
  assert.deepEqual(expired, {
    type: 'error',
    message: '匹配二维码已失效，请让对方刷新二维码'
  })
})

test('scan cancellation and permission denial do not create a match action', async () => {
  const cancelled = await scanAndResolveMatchCode({
    uniApi: {
      authorize({ success }) {
        success()
      },
      scanCode({ fail }) {
        fail({ errMsg: 'scanCode:fail cancel' })
      }
    }
  })
  assert.deepEqual(cancelled, {
    type: 'cancelled',
    reason: 'cancelled',
    message: 'scanCode:fail cancel'
  })

  const denied = await scanAndResolveMatchCode({
    uniApi: {
      authorize() {},
      getSetting({ success }) {
        success({ authSetting: { 'scope.camera': false } })
      },
      showModal({ success }) {
        success({ confirm: false })
      }
    }
  })
  assert.equal(denied.type, 'error')
  assert.equal(denied.reason, 'permission-denied')
})
