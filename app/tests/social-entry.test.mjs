import test from 'node:test'
import assert from 'node:assert/strict'

import {
  SOCIAL_TABS,
  buildFeedUrl,
  filterReportPosts,
  resolveFeedTab,
  resolveSocialEmptyState
} from '../utils/social-entry.js'

test('SOCIAL_TABS keeps one canonical order for social surfaces', () => {
  assert.deepEqual(SOCIAL_TABS, [
    { label: '推荐', value: 'recommend' },
    { label: '好友', value: 'friends' },
    { label: '战报', value: 'reports' }
  ])
})

test('resolveFeedTab maps social-home tabs to full-feed tabs', () => {
  assert.equal(resolveFeedTab('recommend'), 'public')
  assert.equal(resolveFeedTab('friends'), 'following')
  assert.equal(resolveFeedTab('reports'), 'reports')
  assert.equal(resolveFeedTab('unknown'), 'public')
})

test('buildFeedUrl preserves the selected tab in the destination url', () => {
  assert.equal(buildFeedUrl('recommend'), '/subPages/social/feed?tab=recommend')
  assert.equal(buildFeedUrl('friends'), '/subPages/social/feed?tab=friends')
  assert.equal(buildFeedUrl('reports'), '/subPages/social/feed?tab=reports')
})

test('resolveSocialEmptyState adds a login cta for guest friends tab', () => {
  assert.deepEqual(resolveSocialEmptyState({ tab: 'friends', isLoggedIn: false }), {
    icon: '👥',
    title: '登录后查看好友动态',
    desc: '登录后可查看关注与好友的真实战绩分享。',
    ctaText: '立即登录'
  })
})

test('resolveSocialEmptyState keeps report copy actionless when no login is required', () => {
  assert.deepEqual(resolveSocialEmptyState({ tab: 'reports', isLoggedIn: true }), {
    icon: '🏆',
    title: '暂时还没有战报',
    desc: '真实对局结束后分享的战绩，会优先出现在这里。',
    ctaText: ''
  })
})

test('filterReportPosts only keeps match share items', () => {
  assert.deepEqual(filterReportPosts([
    { id: 1, post_type: 3 },
    { id: 2, post_type: 1 },
    { id: 3, post_type: 2 },
    { id: 4, post_type: 1 }
  ]), [
    { id: 2, post_type: 1 },
    { id: 4, post_type: 1 }
  ])
})
