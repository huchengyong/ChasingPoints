import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildHuaweiOAuthLoginPayload,
  resolveHuaweiCredential
} from '../utils/oauth-login.js'

const authApiSource = readFileSync(new URL('../api/auth.js', import.meta.url), 'utf8')
const loginSource = readFileSync(new URL('../pages/login/login.vue', import.meta.url), 'utf8')

test('Huawei OAuth payload uses provider credential instead of OpenID identity fields', () => {
  const payload = buildHuaweiOAuthLoginPayload({
    loginResult: {
      authResult: {
        authorizationCode: ' auth-code-1 ',
        openid: 'client-openid',
        unionID: 'client-union'
      }
    },
    userInfo: {
      userInfo: {
        nickName: '华为用户',
        avatarUrl: 'https://example.com/avatar.png'
      }
    },
    platform: 'app-harmony'
  })

  assert.equal(payload.provider, 'huawei')
  assert.equal(payload.credential, 'auth-code-1')
  assert.equal(payload.credential_type, 'authorization_code')
  assert.equal(payload.platform, 'app-harmony')
  assert.equal(payload.nick_name, '华为用户')
  assert.equal(payload.avatar_url, 'https://example.com/avatar.png')
  assert.equal(Object.hasOwn(payload, 'open_id'), false)
  assert.equal(Object.hasOwn(payload, 'union_id'), false)
  assert.equal(Object.hasOwn(payload, 'openId'), false)
  assert.equal(Object.hasOwn(payload, 'unionID'), false)
})

test('Huawei credential resolver falls back to token credentials when no auth code exists', () => {
  assert.deepEqual(resolveHuaweiCredential({
    authResult: { idToken: 'id-token-1' }
  }), {
    credential: 'id-token-1',
    credentialType: 'id_token'
  })

  assert.deepEqual(resolveHuaweiCredential({
    authResult: { access_token: 'access-token-1' }
  }), {
    credential: 'access-token-1',
    credentialType: 'access_token'
  })
})

test('Huawei OAuth payload rejects missing provider credential before API call', () => {
  assert.throws(() => buildHuaweiOAuthLoginPayload({
    loginResult: { authResult: { openid: 'client-openid-only' } },
    userInfo: {},
    platform: 'app-harmony'
  }), /授权凭据缺失/)
})

test('auth API and Harmony login page no longer send client OpenID identity fields', () => {
  assert.doesNotMatch(authApiSource, /open_id|union_id|OpenID|UnionID/)
  assert.doesNotMatch(loginSource, /open_id\s*:|union_id\s*:/)
  assert.match(loginSource, /buildHuaweiOAuthLoginPayload/)
  assert.match(loginSource, /loginByOauth\(buildHuaweiOAuthLoginPayload/)
})
