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

test('guest view keeps a single login action, public entries, and direct feedback access', () => {
  assert.match(source, /class="login-btn" @click="handleGoLogin"/)
  assert.doesNotMatch(source, /handleGuestStartPK|登录后发起/)
  assert.match(source, /title: '排行榜'/)
  assert.match(source, /title: '规则百科'/)
  assert.match(source, /title: '帮助、投诉与举报'[\s\S]*handler: \(\) => handleHelp\(\)/)
  // 赛讯、观赛已是 Tab，不在游客列表里重复。
  assert.doesNotMatch(source, /正在进行的对局|title: '赛讯'/)
})

test('logged-in header opens profile editing and keeps message and settings buttons', () => {
  assert.match(source, /<view class="profile-header">/)
  assert.match(source, /class="profile-main" @click="handleEditProfile"/)
  assert.match(source, /class="profile-avatar" :class="\{ 'has-rank-frame': rankAvatarFrame \}"/)
  assert.match(source, /@click\.stop="handleOpenMemberCenter"/)
  assert.match(source, /@click\.stop="handleOpenReputation"/)
  assert.match(source, /class="icon-btn" @click="handleNotificationCenter"/)
  assert.match(source, /class="icon-btn" @click="handleSettings"/)
  assert.match(source, /resolveReputationChip\(reputationStatus\.value, coreDataLoading\.value\)/)
  assert.match(source, /`VIP · \$\{memberLevelText\.value\}`/)
})

test('active members use the current rank avatar frame without replacing profile navigation', () => {
  assert.match(source, /v-if="rankAvatarFrame" class="profile-avatar-frame" :src="rankAvatarFrame"/)
  assert.match(source, /resolveMemberRankAvatarFrame\(\{/)
  assert.match(source, /rankLoading: rankLoading\.value/)
  assert.match(styleSource, /\.profile-avatar-frame\s*\{[\s\S]*pointer-events:\s*none;/)
  assert.match(styleSource, /\.profile-avatar\.has-rank-frame\s*\{[\s\S]*width:\s*124rpx;[\s\S]*height:\s*124rpx;/)
})

test('user page aggregates core loading and only shows placeholders before the first read', () => {
  assert.match(source, /const coreDataLoading = ref\(false\)/)
  assert.match(source, /Promise\.allSettled\(\[\s*activityStore\.fetch\(identity, \{ force, silent: true \}\),\s*userOverviewStore\.fetch\(identity, \{ force, silent: true \}\),\s*rankStore\.ensureFresh\(identity\)/)
  assert.match(source, /v-if="!hasRankCache && \(coreDataLoading \|\| rankLoading\)" class="rank-skeleton"/)
  assert.match(source, /if \(!memberStatus\.value && coreDataLoading\.value\) return '等级同步中'/)
  assert.match(styleSource, /@keyframes skeleton-shimmer/)
})

test('rank card puts the game tabs above the rank, score, and promotion progress', () => {
  assert.match(source, /class="rank-card"[\s\S]*class="rank-tabs"[\s\S]*class="rank-body" @click="handleRankExplain"/)
  assert.match(source, /resolveRankCardView\(rankInfo\.value\)/)
  assert.match(source, /:style="\{ width: `\$\{rankView\.progress\}%` \}"/)
  assert.match(source, /\{\{ rankView\.hint \}\}/)
  assert.match(source, /v-if="rankView\.ranked" class="rank-score"/)
  assert.match(source, /v-if="rankView\.ranked" class="rank-progress"/)
  assert.match(source, /readDefaultGameType\(uni, userStore\.userId\)/)
  assert.doesNotMatch(source, /member-hero-strip|memberHeroStrip/)
  assert.match(styleSource, /\.rank-tabs\s*\{[\s\S]*grid-template-columns:\s*repeat\(4, minmax\(0, 1fr\)\);/)
})

test('match launching lives on home and match tabs, not on the profile tab', () => {
  assert.doesNotMatch(source, /status-card|challenge-status-strip|qrcode-modal|gameTypeModal/)
  assert.doesNotMatch(source, /scanAndResolveMatchCode|startMatch|getMatchQRCode|userWS/)
  assert.match(source, /const goStartFirstMatch = \(\) => \{\s*openRoute\('\/pages\/index\/index', true\)/)
})

test('career stats are labelled as all game types and handle empty and failed reads', () => {
  assert.match(source, /<text class="section-title">生涯数据<\/text>\s*<text class="section-caption">全部球种<\/text>/)
  assert.match(source, /v-if="careerState === 'empty'" class="career-empty"/)
  assert.match(source, /v-if="careerState === 'error'" class="career-error" @click="retryHomepageData"/)
  assert.match(source, /statsFailed\.value && !coreDataLoading\.value \? 'error' : 'loading'/)
  assert.match(styleSource, /\.metrics-card\s*\{[\s\S]*grid-template-columns:\s*repeat\(4, minmax\(0, 1fr\)\);/)
})

test('shortcuts keep the four record entries in one brand color', () => {
  assert.match(source, /class="metrics-card"[\s\S]*class="shortcuts-card"[\s\S]*class="list-card"/)
  for (const label of ['比赛记录', '过往对手', '荣誉墙', '好友']) {
    assert.match(source, new RegExp(`label: '${label}'`))
  }
  assert.doesNotMatch(source, /iconClass|#2563eb|#7c3aed|#0f766e/)
  assert.match(styleSource, /\.shortcuts-card\s*\{[\s\S]*grid-template-columns:\s*repeat\(4, minmax\(0, 1fr\)\);/)
})

test('service list only uses existing member, season, rules, and venue routes', () => {
  assert.match(source, /title: '会员中心', icon: 'vip', value: memberEntry\.value\.text/)
  assert.match(source, /openRoute\('\/subPages\/season\/index'\)/)
  assert.match(source, /openRoute\('\/subPages\/rules\/index'\)/)
  assert.match(source, /openRoute\('\/subPages\/venue\/index'\)/)
  assert.match(source, /handleStatsDetail/)
  assert.doesNotMatch(source, /赛事情报|按需查看/)
})

test('privacy and logout controls are absent from the main user stream', () => {
  assert.doesNotMatch(source, /toggleHideMatch/)
  assert.doesNotMatch(source, /getUserPrivacy/)
  assert.doesNotMatch(source, /updateUserPrivacy/)
  assert.doesNotMatch(source, /handleLogout/)
  assert.doesNotMatch(source, /退出登录/)
})

test('long names truncate without moving the header buttons', () => {
  assert.match(styleSource, /\.profile-copy\s*\{[\s\S]*min-width:\s*0;[\s\S]*overflow:\s*hidden;/)
  assert.match(styleSource, /\.profile-name\s*\{[\s\S]*overflow:\s*hidden;[\s\S]*text-overflow:\s*ellipsis;[\s\S]*white-space:\s*nowrap;/)
  assert.match(styleSource, /\.profile-actions\s*\{[\s\S]*flex-shrink:\s*0;/)
})

test('floatable reward states render a single floating entry and the old full-screen modal is removed', () => {
  assert.match(source, /class="float-reward-entry"/)
  assert.match(source, /class="float-reward-body"/)
  assert.match(source, /class="float-reward-close"/)
  assert.match(source, /handleFloatRewardClose/)
  assert.match(source, /:class="\{ 'has-float-reward': floatRewardVisible \}"/)
  assert.match(styleSource, /bottom:\s*calc\(24rpx \+ var\(--window-bottom, 0px\)\)/)
  assert.doesNotMatch(source, /class="reward-modal-overlay"/)
})

test('pending_review stays as an in-page read-only card without floating entry or action navigation', () => {
  assert.match(source, /<view v-if="compactRewardEntry\.inline" class="reward-entry">/)
  assert.doesNotMatch(source, /compactRewardEntry\.visible/)
})

test('user page keeps dark theme tokens and UniApp-safe custom button rules', () => {
  assert.match(styleSource, /&\.dark-mode\s*\{[\s\S]*--me-surface:\s*#1E180D;[\s\S]*background:\s*#141109;/)
  assert.match(styleSource, /\.login-btn\s*\{[\s\S]*height:\s*88rpx;[\s\S]*line-height:\s*88rpx;[\s\S]*margin:\s*28rpx 0 0;[\s\S]*&::after\s*\{\s*display:\s*none;/)
  assert.match(styleSource, /\.icon-btn\s*\{[\s\S]*height:\s*72rpx;[\s\S]*line-height:\s*72rpx;[\s\S]*margin:\s*0;[\s\S]*&::after\s*\{\s*display:\s*none;/)
  assert.doesNotMatch(styleSource, /(^|[\s,{>])\*(?=\s|\{|:)/m)
  assert.doesNotMatch(styleSource, /:disabled/)
})
