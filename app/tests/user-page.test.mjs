import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const source = readFileSync(
  new URL('../pages/user/index.vue', import.meta.url),
  'utf8'
)
const styleSource = readFileSync(
  new URL('../pages/user/index.scss', import.meta.url),
  'utf8'
)

test('user page guest hero keeps login as primary action and start PK as secondary action', () => {
  assert.match(source, /hero-btn hero-btn-primary" @click="handleGoLogin"/)
  assert.match(source, /hero-btn hero-btn-secondary" @click="handleGuestStartPK"/)
})

test('user page only renders the status action row when there are visible actions', () => {
  assert.match(
    source,
    /<view v-if="showPrimaryStatusAction \|\| showSecondaryStatusAction" class="status-actions">/
  )
})

test('user page rank chip renders the computed highest rank label instead of hardcoded copy', () => {
  assert.match(source, /<text>{{ highestRankChipText }}<\/text>/)
  assert.doesNotMatch(source, /<text>当前段位<\/text>/)
})

test('user page status eyebrow uses readable muted text on the light status card', () => {
  assert.doesNotMatch(
    styleSource,
    /\.guest-eyebrow,\s*\.status-eyebrow\s*\{[\s\S]*color:\s*rgba\(255,\s*255,\s*255,\s*0\.74\);/
  )
  assert.match(
    styleSource,
    /\.status-eyebrow\s*\{[\s\S]*color:\s*\$text-muted-light;/
  )
})

test('user page notification badge stays within the icon button bounds', () => {
  assert.match(source, /<view v-if="pendingTotal > 0" class="icon-btn-badge">/)
  assert.match(
    styleSource,
    /\.icon-btn-badge\s*\{[\s\S]*top:\s*[0-9]+rpx;[\s\S]*right:\s*[0-9]+rpx;/
  )
  assert.doesNotMatch(
    styleSource,
    /\.icon-btn-badge\s*\{[\s\S]*top:\s*-\d+rpx;[\s\S]*right:\s*-\d+rpx;/
  )
})

test('user page keeps long nicknames on one line with ellipsis in the identity hero', () => {
  assert.match(
    styleSource,
    /\.identity-copy\s*\{[\s\S]*min-width:\s*0;/
  )
  assert.match(
    styleSource,
    /\.identity-name\s*\{[\s\S]*max-width:\s*[0-9]+rpx;[\s\S]*overflow:\s*hidden;[\s\S]*text-overflow:\s*ellipsis;[\s\S]*white-space:\s*nowrap;/
  )
})

test('user page only renders member benefit rows when there are actual extra benefits', () => {
  assert.match(
    source,
    /<view v-if="favoriteVenueMemberCard\.benefits\.length" class="member-benefits">/
  )
})

test('user page merges active member cards into one clickable card', () => {
  assert.match(
    source,
    /const showMemberCenterEntryCard = computed\(\(\) => memberCenterCard\.value\.visible && !isActiveFavoriteVenueMember\.value\)/
  )
  assert.match(
    source,
    /<view v-if="favoriteVenueMemberCard\.visible" class="member-card" @click="handleOpenMemberCenter">/
  )
  assert.match(
    source,
    /<view v-if="showMemberCenterEntryCard" class="subscription-entry-card" @click="handleOpenMemberCenter">/
  )
})

test('user page does not render a separate view-benefits button for member cards', () => {
  assert.match(
    source,
    /const shouldShowMemberCenterButton = computed\(\(\) => memberCenterCard\.value\.actionText && memberCenterCard\.value\.actionText !== '查看权益'\)/
  )
  assert.match(
    source,
    /<button v-if="shouldShowMemberCenterButton" class="subscription-entry-btn">/
  )
})
