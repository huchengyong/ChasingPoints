import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8')

const feedSource = read('subPages/social/feed.vue')
const postCreateSource = read('subPages/social/postCreate.vue')
const myPostsSource = read('subPages/social/myPosts.vue')
const socialApiSource = read('api/social.js')

test('feed blocks requests and the publish entry in compliance mode', () => {
  assert.match(feedSource, /canShowSocialPublishing/)
  assert.match(feedSource, /const socialEnabled = canShowSocialPublishing\(\)/)
  assert.match(feedSource, /if \(!socialEnabled\) \{\s*loading\.value = false\s*return/)
  assert.match(feedSource, /v-if="!socialEnabled" class="empty-state compliance-state"/)
  assert.match(feedSource, /SOCIAL_COMPLIANCE_TITLE/)
  assert.match(feedSource, /v-if="socialEnabled" class="fab-btn"/)
})

test('feed fixes dark mode nickname and separators and supports comment deletion', () => {
  assert.match(feedSource, /\.post-header \.post-user-info \.post-nickname \{\s*color: #fff7e1/)
  assert.match(feedSource, /\.post-actions,\s*\.comment-section \{\s*border-top-color: #3a2e16/)
  assert.match(feedSource, /deleteComment/)
  assert.match(feedSource, /@longpress="handleCommentLongPress\(item, comment\)"/)
  assert.match(feedSource, /canDeleteComment/)
  assert.match(feedSource, /post\.comments_count = Math\.max/)
})

test('social api exposes comment deletion', () => {
  assert.match(socialApiSource, /export const deleteComment = \(data\) =>/)
  assert.match(socialApiSource, /\/api\/social\/comment\/delete/)
})

test('post creation and my posts pages exit when social publishing is closed', () => {
  for (const source of [postCreateSource, myPostsSource]) {
    assert.match(source, /canShowSocialPublishing/)
    assert.match(source, /blockByCompliance/)
    assert.match(source, /SOCIAL_COMPLIANCE_TITLE/)
    assert.match(source, /uni\.navigateBack/)
    assert.match(source, /uni\.switchTab\(\{ url: '\/pages\/index\/index' \}\)/)
  }
})

test('post creation surfaces the backend failure message', () => {
  assert.match(postCreateSource, /res\.message \|\| '发布失败'/)
  assert.match(postCreateSource, /e\?\.message \|\| '发布失败'/)
})
