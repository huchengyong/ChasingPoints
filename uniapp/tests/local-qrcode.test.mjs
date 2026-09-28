import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildLocalQRCodePlan,
  renderLocalQRCode
} from '../utils/local-qrcode.js'

test('local QR render plan keeps arbitrary match or referee payload local and bounded', () => {
  assert.deepEqual(buildLocalQRCodePlan({
    canvasId: 'match-invite-code',
    content: 'signed.match.invite',
    size: 240
  }), {
    canvasId: 'match-invite-code',
    content: 'signed.match.invite',
    size: 240
  })
  assert.equal(buildLocalQRCodePlan({
    canvasId: 'referee-code',
    content: 'referee-join-token',
    size: 99999
  }).size, 1024)
  assert.throws(() => buildLocalQRCodePlan({ content: 'x' }), /画布/)
  assert.throws(() => buildLocalQRCodePlan({ canvasId: 'x' }), /内容/)
})

test('local QR renderer draws through Uni canvas without a third-party image request', () => {
  const calls = []
  class FakeQRCode {
    make() {
      calls.push('make')
    }

    drawCanvas() {
      calls.push('draw')
    }
  }
  const context = { kind: 'canvas' }
  const plan = renderLocalQRCode({
    canvasId: 'match-invite-code',
    content: 'signed.match.invite',
    size: 200,
    QRCode: FakeQRCode,
    uniApi: {
      createCanvasContext(canvasId) {
        calls.push(canvasId)
        return context
      }
    }
  })

  assert.deepEqual(plan, {
    canvasId: 'match-invite-code',
    content: 'signed.match.invite',
    size: 200
  })
  assert.deepEqual(calls, ['make', 'match-invite-code', 'draw'])
})
