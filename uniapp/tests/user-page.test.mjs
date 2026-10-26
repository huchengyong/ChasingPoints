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

test('logged-in header opens profile editing and keeps the QR and settings buttons', () => {
  assert.match(source, /<view class="profile-header">/)
  assert.match(source, /class="profile-main" @click="handleEditProfile"/)
  assert.match(source, /class="profile-avatar" :class="\{ 'has-rank-frame': rankAvatarFrame \}"/)
  assert.match(source, /@click\.stop="handleOpenMemberCenter"/)
  assert.match(source, /@click\.stop="handleOpenReputation"/)
  assert.match(source, /class="icon-btn" @click="handleSettings"/)
  assert.match(source, /resolveReputationChip\(reputationStatus\.value, coreDataLoading\.value\)/)
  assert.match(source, /`\$\{identity\.badgeText\} · \$\{identity\.tierLabel\}`/)
})

test('active members use the current rank avatar frame without replacing profile navigation', () => {
  assert.match(source, /v-if="rankAvatarFrame" class="profile-avatar-frame" :src="rankAvatarFrame"/)
  assert.match(source, /resolveSelfMemberIdentity\(\{/)
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
  assert.doesNotMatch(source, /member-hero-strip|memberHeroStrip/)
  assert.match(styleSource, /\.rank-tabs\s*\{[\s\S]*display:\s*flex;/)
  assert.match(styleSource, /\.rank-tab\s*\{[\s\S]*flex:\s*1;/)
})

test('rank card shows the primary game type and only switches between relevant ones', () => {
  assert.match(source, /const preferredGameType = ref\(readDefaultGameType\(uni, userStore\.userId\)\)/)
  assert.match(source, /resolveRankGameTypeView\(\{\s*preferredGameType: preferredGameType\.value,\s*rankInfoMap: hasRankCache\.value \? rankStore\.rankInfoMap : \{\}/)
  assert.match(source, /v-if="rankGameView\.tabs\.length" class="rank-tabs"[\s\S]*v-for="item in rankGameView\.tabs"/)
  assert.match(source, /<text v-else-if="hasRankCache" class="rank-game-label">\{\{ rankGameView\.primaryLabel \}\}<\/text>/)
  // 临时切换只在当前可选球种内生效，偏好或账号变化后回到常用球种。
  assert.match(source, /rankGameView\.value\.tabs\.some\(\(item\) => item\.value === selectedRankGameType\.value\)/)
  assert.match(source, /onShow\(\(\) => \{\s*syncRankGamePreference\(\)/)
  assert.match(source, /if \(ownerId === rankPreferenceOwnerId && preferred === preferredGameType\.value\) return[\s\S]*selectedRankGameType\.value = 0/)
})

test('players without a preference or ranked games can pick a primary game type inline', () => {
  assert.match(source, /const showRankGamePicker = computed\(\(\) => hasRankCache\.value && rankGameView\.value\.needsPick\)/)
  assert.match(source, /class="rank-body" @click="handleRankExplain"[\s\S]*v-if="showRankGamePicker" class="rank-picker"/)
  assert.match(source, /@tap="handlePickRankGameType\(item\.value\)"/)
  assert.match(source, /preferredGameType\.value = saveDefaultGameType\(uni, userStore\.userId, gameType\)/)
  assert.doesNotMatch(source, /showModal\([^)]*常用球种/)
  assert.match(styleSource, /\.rank-picker-options\s*\{[\s\S]*grid-template-columns:\s*repeat\(4, minmax\(0, 1fr\)\);/)
})

test('profile header offers the match QR and settings; scanning and messages live elsewhere', () => {
  assert.match(source, /class="profile-actions">\s*<button class="icon-btn" aria-label="出示我的二维码" @click="showMyMatchQRCode">\s*<view class="qr-glyph">[\s\S]*@click="handleSettings"/)
  // 扫码由底部中间按钮承接，消息由消息 Tab 承接
  assert.doesNotMatch(source, /handleQuickScan|aria-label="扫码开局"|type="scan"/)
  assert.doesNotMatch(source, /handleNotificationCenter|notification-filled|pendingTotal|pendingBadgeText|icon-btn-badge/)
  assert.doesNotMatch(styleSource, /icon-btn-badge/)
  assert.doesNotMatch(source, /goStartFirstMatch|去发起对局|empty-action/)
  assert.doesNotMatch(styleSource, /\.empty-action/)
  // 开局与球种弹窗已移到扫码确认页；段位卡仍可直接设置常用球种
  assert.doesNotMatch(source, /gameTypeModal|:default-type=|handleGameTypeConfirm|scanAndResolveMatchCode|startMatch\(/)
  assert.match(source, /const handlePickRankGameType = \(gameType\) => \{\s*preferredGameType\.value = saveDefaultGameType\(uni, userStore\.userId, gameType\)\s*selectedRankGameType\.value = 0/)
  // 当前对局卡、约球条在首页
  assert.doesNotMatch(source, /status-card|challenge-status-strip|userWS/)
  assert.match(styleSource, /\.qr-glyph-finder\s*\{[\s\S]*border:\s*4rpx solid var\(--me-text-secondary\);/)
  // 图标挤占昵称区时，窄屏下标签换行而不是截成省略号
  assert.match(styleSource, /\.profile-chips\s*\{\s*display:\s*flex;\s*flex-wrap:\s*wrap;/)
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

test('service list opens the challenge history first, then the existing member, season, rules and venue routes', () => {
  assert.match(source, /serviceLinks = computed\(\(\) => \(\[\s*\{ title: '约球记录', icon: 'list', value: '', handler: \(\) => openRoute\('\/subPages\/social\/challenges\?tab=history'\) \},\s*\{ title: '会员中心'/)
  assert.match(source, /title: '会员中心', icon: 'vip', value: memberEntry\.value\.text/)
  assert.match(source, /title: '装备', icon: 'tune', value: '', handler: \(\) => openRoute\('\/subPages\/gear\/index'\)/)
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
  // 浮动入口让开 H5 页内 TabBar（--window-bottom）和小程序自定义 TabBar（--ui-tab-bar-space）
  assert.match(styleSource, /bottom:\s*calc\(24rpx \+ var\(--window-bottom, 0px\) \+ var\(--ui-tab-bar-space, 0px\)\)/)
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
