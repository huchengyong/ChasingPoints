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

test('user page keeps the guest login funnel, public entries, and direct feedback access', () => {
  assert.match(source, /hero-btn hero-btn-primary" @click="handleGoLogin"/)
  assert.match(source, /hero-btn hero-btn-secondary" @click="handleGuestStartPK"/)
  assert.match(source, /登录后解锁/)
  assert.match(source, /游客也能继续看/)
  assert.match(source, /帮助、投诉与举报/)
  assert.match(source, /@click="handleHelp"/)
})

test('logged-in user profile is a standalone header with profile, message, and settings navigation', () => {
  assert.match(source, /<view class="profile-header">/)
  assert.match(source, /class="profile-avatar" @click="handleEditProfile"/)
  assert.match(source, /class="icon-btn" @click="handleNotificationCenter"/)
  assert.match(source, /class="icon-btn" @click="handleSettings"/)
  assert.match(source, /profileLevelText/)
  assert.match(source, /reputationEntryStatusText/)
  assert.doesNotMatch(source, /class="identity-hero"/)
})

test('user page aggregates core loading and renders neutral rank and status skeletons', () => {
  assert.match(source, /const coreDataLoading = ref\(false\)/)
  assert.match(source, /resolveUserHomepageModel\(\{/)
  assert.match(source, /Promise\.allSettled\(\[\s*loadUserStats\(\),\s*loadRankInfo\(\),\s*loadCurrentMatch\(\)/)
  assert.match(source, /v-if="coreDataLoading \|\| rankLoading" class="rank-skeleton"/)
  assert.match(source, /<template v-if="statusCard\.loading">/)
  assert.match(styleSource, /@keyframes skeleton-shimmer/)
})

test('competitive identity card contains the single member strip, rank, score, and four game tabs', () => {
  assert.match(source, /class="identity-card"/)
  assert.match(source, /v-if="memberHeroStrip\.visible"/)
  assert.match(source, /memberHeroStrip\.description/)
  assert.match(source, /排位分 \{\{ displayRank\.score \}\}/)
  assert.match(source, /v-for="item in rankGameTabs"/)
  assert.match(styleSource, /\.rank-tabs\s*\{[\s\S]*grid-template-columns:\s*repeat\(4, minmax\(0, 1fr\)\);/)
})

test('player and referee ongoing states share one fixed-participant status card', () => {
  assert.match(source, /statusCard\.type === 'ongoing'/)
  assert.match(source, /statusPlayers\[0\]\.avatar/)
  assert.match(source, /statusPlayers\[0\]\.name/)
  assert.match(source, /statusPlayers\[1\]\.avatar/)
  assert.match(source, /statusPlayers\[1\]\.name/)
  assert.match(source, /homepageMode === 'ongoing-referee'/)
  assert.doesNotMatch(source, /referee-match-info/)
})

test('newcomer and idle states reuse the status card with PK and QR actions', () => {
  assert.match(source, /<template v-else>[\s\S]*class="empty-status-copy"/)
  assert.match(source, /handleStatusAction\(statusCard\.action\)/)
  assert.match(source, /handleStatusAction\(statusCard\.secondaryAction\)/)
  assert.match(source, /showPrimaryStatusAction/)
  assert.match(source, /showSecondaryStatusAction/)
})

test('core metrics and the four required shortcuts stay compact and ordered after status', () => {
  assert.match(source, /class="metrics-card"[\s\S]*class="shortcuts-card"/)
  assert.match(source, /label: '比赛记录'/)
  assert.match(source, /label: '过往对手'/)
  assert.match(source, /label: '荣誉墙'/)
  assert.match(source, /label: '好友'/)
  assert.match(styleSource, /\.metrics-card\s*\{[\s\S]*grid-template-columns:\s*repeat\(4, minmax\(0, 1fr\)\);/)
  assert.match(styleSource, /\.shortcut-grid\s*\{[\s\S]*grid-template-columns:\s*repeat\(4, minmax\(0, 1fr\)\);/)
})

test('membership and venue rewards no longer render duplicate large cards', () => {
  assert.match(source, /const memberHeroStrip = computed/)
  assert.match(source, /const compactRewardEntry = computed/)
  assert.match(source, /class="reward-entry"/)
  assert.doesNotMatch(source, /favoriteVenueMemberCard/)
  assert.doesNotMatch(source, /memberCenterCard/)
  assert.doesNotMatch(source, /subscription-entry-card/)
  assert.doesNotMatch(source, /reward-task-card/)
})

test('low-frequency services only use existing rules, season, venue, tournament, and analysis routes', () => {
  assert.match(source, /handleStatsDetail/)
  assert.match(source, /openRoute\('\/subPages\/rules\/index'\)/)
  assert.match(source, /openRoute\('\/subPages\/season\/index'\)/)
  assert.match(source, /openRoute\('\/subPages\/venue\/index'\)/)
  assert.match(source, /openRoute\('\/subPages\/tournament\/index'\)/)
  assert.doesNotMatch(source, /更多服务/)
})

test('privacy and logout controls are absent from the main user stream', () => {
  assert.doesNotMatch(source, /toggleHideMatch/)
  assert.doesNotMatch(source, /getUserPrivacy/)
  assert.doesNotMatch(source, /updateUserPrivacy/)
  assert.doesNotMatch(source, /handleLogout/)
  assert.doesNotMatch(source, /退出登录/)
})

test('long profile and participant names truncate without moving the action buttons', () => {
  assert.match(styleSource, /\.profile-copy\s*\{[\s\S]*min-width:\s*0;/)
  assert.match(styleSource, /\.profile-name\s*\{[\s\S]*overflow:\s*hidden;[\s\S]*text-overflow:\s*ellipsis;[\s\S]*white-space:\s*nowrap;/)
  assert.match(styleSource, /\.profile-actions\s*\{[\s\S]*flex-shrink:\s*0;/)
  assert.match(styleSource, /\.match-player-name\s*\{[\s\S]*overflow:\s*hidden;[\s\S]*text-overflow:\s*ellipsis;/)
})

test('user page keeps dark theme contrast and UniApp-safe custom button rules', () => {
  assert.match(styleSource, /\.dark-mode\s*\{/)
  assert.match(styleSource, /\$card-bg-dark:\s*#1e180d;/)
  assert.match(styleSource, /\.hero-btn,[\s\S]*\.status-action-btn,[\s\S]*\.reward-modal-btn\s*\{[\s\S]*margin:\s*0;/)
  assert.match(styleSource, /\.hero-btn,[\s\S]*\.status-action-btn,[\s\S]*\.reward-modal-btn\s*\{[\s\S]*height:\s*88rpx;[\s\S]*line-height:\s*88rpx;/)
  assert.doesNotMatch(styleSource, /(^|[\s,{>])\*(?=\s|\{|:)/m)
  assert.doesNotMatch(styleSource, /:disabled/)
})
