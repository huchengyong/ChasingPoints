import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'

const appRoot = path.resolve(new URL('..', import.meta.url).pathname)
const pagesJson = fs.readFileSync(path.join(appRoot, 'pages.json'), 'utf8')
const editProfileSource = fs.readFileSync(
  path.join(appRoot, 'subPages/user/editProfile.vue'),
  'utf8'
)

test('pages register the standalone edit profile route', () => {
  assert.match(pagesJson, /"root":\s*"subPages\/user"[\s\S]*?"path":\s*"editProfile"[\s\S]*?"navigationBarTitleText":\s*"编辑资料"/)
})

test('edit profile supports platform avatar selection and current profile fields', () => {
  assert.match(editProfileSource, /open-type="chooseAvatar"/)
  assert.match(editProfileSource, /chooseavatar/)
  assert.match(editProfileSource, /拍照/)
  assert.match(editProfileSource, /从手机相册选择/)
  assert.match(editProfileSource, /昵称/)
  assert.match(editProfileSource, /手机号/)
  assert.match(editProfileSource, /上传中/)
})
