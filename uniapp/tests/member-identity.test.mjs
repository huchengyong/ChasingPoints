import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  MEMBER_IDENTITY_TIERS,
  normalizeVipLevel,
  resolveMemberAvatarMetrics,
  resolveMemberIdentity,
  resolveMemberTier,
  resolveRankAvatarFramePath
} from '../utils/member-identity.js'

test('growth levels map onto the three confirmed tiers', () => {
  const tiers = [1, 2, 3, 4, 5].map(level => resolveMemberIdentity({ vipLevel: level }))
  assert.deepEqual(tiers.map(item => item.tier), ['standard', 'standard', 'advanced', 'advanced', 'premium'])
  assert.deepEqual(tiers.map(item => item.tierLabel), ['标准', '标准', '进阶', '进阶', '尊享'])
  assert.deepEqual(tiers.map(item => item.badgeText), ['VIP1', 'VIP2', 'VIP3', 'VIP4', 'VIP5'])
  assert.ok(tiers.every(item => item.showBadge))
})

test('tier table covers every level exactly once for the member center', () => {
  assert.deepEqual(
    MEMBER_IDENTITY_TIERS.map(item => [item.label, item.levelText]),
    [['标准', 'Lv1–Lv2'], ['进阶', 'Lv3–Lv4'], ['尊享', 'Lv5']]
  )
  for (let level = 1; level <= 5; level += 1) {
    const matched = MEMBER_IDENTITY_TIERS.filter(item => level >= item.minLevel && level <= item.maxLevel)
    assert.equal(matched.length, 1, `Lv${level} must belong to exactly one tier`)
  }
})

test('missing, zero and invalid vip levels have no member identity at all', () => {
  for (const vipLevel of [undefined, null, 0, -1, NaN, 'abc', '']) {
    const identity = resolveMemberIdentity({ vipLevel, rankLevel: 6 })
    assert.equal(identity.showBadge, false, `vipLevel=${String(vipLevel)}`)
    assert.equal(identity.badgeText, '')
    assert.equal(identity.frameUrl, '', 'a non-member never gets a frame, even with a rank')
    assert.equal(identity.tier, '')
  }
  assert.equal(resolveMemberIdentity().showBadge, false)
  assert.equal(resolveMemberTier(0), null)
})

test('levels above the maximum are shown as the top tier and numeric strings are accepted', () => {
  assert.equal(normalizeVipLevel(9), 5)
  assert.equal(resolveMemberIdentity({ vipLevel: 9 }).badgeText, 'VIP5')
  assert.equal(resolveMemberIdentity({ vipLevel: '3' }).badgeText, 'VIP3')
  assert.equal(normalizeVipLevel(2.9), 2)
})

test('frame follows the rank level and is empty when the rank is unknown', () => {
  assert.equal(
    resolveMemberIdentity({ vipLevel: 3, rankLevel: 4 }).frameUrl,
    'https://cdn.zhuifen.cn/static/images/avatar-frames/frame_platinum.png'
  )
  assert.equal(resolveMemberIdentity({ vipLevel: 3, rankLevel: 4 }).frameUrl, resolveRankAvatarFramePath(4))
  for (const rankLevel of [0, 7, undefined, null, 'x']) {
    const identity = resolveMemberIdentity({ vipLevel: 3, rankLevel })
    assert.equal(identity.frameUrl, '', `rankLevel=${String(rankLevel)}`)
    assert.equal(identity.showBadge, true, 'an unranked member still shows the badge')
  }
})

test('rank frame paths map all six rank levels to unique CDN assets', () => {
  const urls = [1, 2, 3, 4, 5, 6].map(resolveRankAvatarFramePath)
  assert.equal(new Set(urls).size, 6)
  assert.ok(urls.every(url => url.startsWith('https://cdn.zhuifen.cn/static/images/avatar-frames/')))
})

test('avatar metrics scale the frame and badge with the avatar and optionally reserve the frame box', () => {
  assert.deepEqual(resolveMemberAvatarMetrics(96), {
    avatarSize: 96,
    boxSize: 96,
    frameSize: 156,
    badgeFontSize: 21,
    badgeHeight: 32,
    badgePaddingX: 9
  })
  // 预留模式的布局盒与本人页一致：96rpx 头像配 124rpx 盒。
  assert.equal(resolveMemberAvatarMetrics(96, { reserveFrame: true }).boxSize, 124)
  assert.equal(resolveMemberAvatarMetrics(96, { reserveFrame: true }).avatarSize, 96)
  // fitFrame：size 是含头像框的总占位，有框时头像缩进框内，没有框时保持原大小。
  const fitted = resolveMemberAvatarMetrics(224, { fitFrame: true, hasFrame: true })
  assert.equal(fitted.boxSize, 224)
  assert.equal(fitted.frameSize, 224)
  assert.equal(fitted.avatarSize, 138)
  assert.ok(fitted.avatarSize * 1.625 <= fitted.frameSize + 1, 'the frame must not overflow the requested footprint')
  const unframed = resolveMemberAvatarMetrics(224, { fitFrame: true, hasFrame: false })
  assert.equal(unframed.avatarSize, 224)
  assert.equal(unframed.boxSize, 224)
  // 小头像的标识字号不会小于可读下限，大头像不会无限放大。
  assert.equal(resolveMemberAvatarMetrics(40).badgeFontSize, 14)
  assert.equal(resolveMemberAvatarMetrics(240).badgeFontSize, 22)
  assert.equal(resolveMemberAvatarMetrics(undefined).avatarSize, 0)
})

const componentSource = readFileSync(new URL('../components/memberAvatar.vue', import.meta.url), 'utf8')

test('memberAvatar renders the frame and badge only from the resolved identity', () => {
  assert.match(componentSource, /resolveMemberIdentity\(\{ vipLevel: props\.vipLevel, rankLevel: props\.rankLevel \}\)/)
  assert.match(componentSource, /v-if="identity\.frameUrl"/)
  assert.match(componentSource, /v-if="showBadge && identity\.showBadge"/)
  assert.match(componentSource, /resolveAvatarUrl\(props\.avatar, props\.userId\)/)
  assert.match(componentSource, /resolveMemberAvatarMetrics\(props\.size, \{\s*reserveFrame: props\.reserveFrame,\s*fitFrame: props\.fitFrame,\s*hasFrame: Boolean\(identity\.value\.frameUrl\)\s*\}\)/)
  for (const tier of ['standard', 'advanced', 'premium']) {
    assert.match(componentSource, new RegExp(`\\.member-avatar-badge--${tier}\\s*\\{`))
  }
})

test('memberAvatar styles are theme-token driven and WXSS safe', () => {
  assert.match(componentSource, /var\(--ui-brand-tint/)
  assert.match(componentSource, /var\(--ui-brand-strong/)
  assert.match(componentSource, /\.member-avatar-frame\s*\{[\s\S]*pointer-events:\s*none;/)
  // 微信小程序 WXSS 不支持通配选择器。
  const style = componentSource.slice(componentSource.indexOf('<style'))
  assert.doesNotMatch(style, /(^|[\s,{>+~])\*(?![\w/])/m)
  assert.doesNotMatch(style, /prefers-color-scheme/)
})
