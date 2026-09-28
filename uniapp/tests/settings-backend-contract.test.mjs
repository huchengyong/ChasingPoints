import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const apiSource = readFileSync(
  new URL('../../backend/chasing_points.api', import.meta.url),
  'utf8'
)

const backendConfigSource = readFileSync(
  new URL('../../backend/internal/config/config.go', import.meta.url),
  'utf8'
)

const backendYamlSource = readFileSync(
  new URL('../../backend/etc/chasing_points-api.yaml', import.meta.url),
  'utf8'
)

test('backend user contract exposes qiniu upload token and profile update endpoints', () => {
  assert.match(apiSource, /type GetQiniuUploadTokenReq/)
  assert.match(apiSource, /type GetQiniuUploadTokenResp/)
  assert.match(apiSource, /type UpdateUserProfileReq/)
  assert.match(apiSource, /type UpdateUserProfileResp/)
  assert.match(apiSource, /@handler GetQiniuUploadToken/)
  assert.match(apiSource, /get \/upload-token/)
  assert.match(apiSource, /@handler UpdateUserProfile/)
  assert.match(apiSource, /post \/profile/)
})

test('backend config exposes qiniu credentials and public domain settings', () => {
  assert.match(backendConfigSource, /Qiniu struct/)
  assert.match(backendConfigSource, /AccessKey/)
  assert.match(backendConfigSource, /SecretKey/)
  assert.match(backendConfigSource, /Bucket/)
  assert.match(backendConfigSource, /UploadUrl/)
  assert.match(backendConfigSource, /PublicDomain/)
  assert.match(backendYamlSource, /Qiniu:/)
  assert.match(backendYamlSource, /AccessKey:/)
  assert.match(backendYamlSource, /SecretKey:/)
  assert.match(backendYamlSource, /Bucket:/)
  assert.match(backendYamlSource, /UploadUrl:/)
  assert.match(backendYamlSource, /PublicDomain:/)
})
