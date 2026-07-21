<template>
	<view class="result-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="header" :style="{ paddingTop: statusBarHeight + 'px' }">
			<view class="header-side">
				<button class="header-icon-btn" @click="handleBack">
					<uni-icons type="left" size="22" :color="headerIconColor"></uni-icons>
				</button>
			</view>
			<view class="header-center">
				<text class="header-title">{{ refereeViewConfig.pageTitle }}</text>
				<text class="header-subtitle">{{ gameTypeName }}</text>
			</view>
			<view class="header-side header-side-right">
				<button v-if="!fromHistory" class="header-icon-btn" @click="handleClose">
					<uni-icons type="closeempty" size="22" :color="headerIconColor"></uni-icons>
				</button>
			</view>
		</view>

		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="28" :color="isDarkMode ? '#FCD34D' : '#E0AE12'"></uni-icons>
			<text class="loading-text">正在生成本场战报...</text>
		</view>

		<scroll-view v-else class="content-scroll" scroll-y>
			<view class="main-content">
				<view class="hero-card">
					<view :class="['hero-banner', isReferee ? 'is-neutral' : (isWin ? 'is-win' : 'is-lose')]">
						<view class="hero-topbar">
							<view class="hero-tag">
								<uni-icons :type="isReferee ? 'person-filled' : (isWin ? 'medal' : 'flag')" size="14" color="#ffffff"></uni-icons>
								<text>{{ resultTagText }}</text>
							</view>
							<text class="hero-time">{{ createdAtRelativeText }}</text>
						</view>

						<view class="hero-copy">
							<text class="hero-title">{{ resultTitle }}</text>
							<text class="hero-subtitle">{{ resultSubtitle }}</text>
						</view>

						<view class="versus-board">
							<view class="player-panel">
								<view :class="['avatar-shell', isReferee ? 'is-neutral' : (isWin ? 'is-winner' : 'is-neutral')]">
									<image class="avatar" :src="myAvatar" mode="aspectFill"></image>
								</view>
								<text class="player-name">{{ matchData.my_name || '我' }}</text>
								<text class="player-extra">{{ isReferee ? '选手' : (formatWinRate(matchData.my_win_rate) + ' 胜率') }}</text>
							</view>

							<view class="score-panel">
								<view class="score-line">
									<text class="score-text">{{ matchData.my_score }}</text>
									<text class="score-divider">:</text>
									<text class="score-text">{{ matchData.opponent_score }}</text>
								</view>
								<text class="score-caption">{{ scoreSummaryText }}</text>
							</view>

							<view class="player-panel">
								<view :class="['avatar-shell', isReferee ? 'is-neutral' : (!isWin ? 'is-winner' : 'is-neutral')]">
									<image class="avatar" :src="opponentAvatar" mode="aspectFill"></image>
								</view>
								<text class="player-name">{{ matchData.opponent_name || '对手' }}</text>
								<text class="player-extra">{{ isReferee ? '选手' : (formatWinRate(matchData.opponent_win_rate) + ' 胜率') }}</text>
							</view>
						</view>

						<view class="meta-strip">
							<view v-for="item in heroMetaItems" :key="item.label" class="meta-pill">
								<text class="meta-label">{{ item.label }}</text>
								<text class="meta-value">{{ item.value }}</text>
							</view>
						</view>
					</view>

					<view class="rank-section" v-if="refereeViewConfig.showRankChange">
						<view class="section-head compact">
							<view>
								<text class="section-title">排位变化</text>
								<text class="section-caption">这场对段位走势的直接影响</text>
							</view>
						</view>
						<view class="rank-grid">
							<view class="rank-card mine">
								<text class="rank-card-label">我的排位分</text>
								<text :class="['rank-card-value', myRankScoreClass]">{{ formatRankScore(matchData.my_rank_change) }}</text>
								<view v-if="rankDetailRows.length" class="rank-detail-list">
									<view v-for="detail in rankDetailRows" :key="detail.label" class="rank-detail-row">
										<text class="rank-detail-label">{{ detail.label }}</text>
										<text class="rank-detail-value">{{ formatRankScore(detail.value) }}</text>
									</view>
								</view>
								<text v-else class="rank-note">{{ myRankNote }}</text>
							</view>

							<view class="rank-card opponent">
								<text class="rank-card-label">对手排位分</text>
								<text :class="['rank-card-value', opponentRankScoreClass]">{{ formatRankScore(matchData.opponent_rank_change) }}</text>
								<text class="rank-note">{{ opponentRankNote }}</text>
							</view>
						</view>
					</view>
				</view>

				<view class="referee-card" v-if="refereeCard.hasReferee">
					<view class="referee-card-head">
						<uni-icons type="person-filled" size="18" color="#E0AE12"></uni-icons>
						<text class="referee-card-title">{{ refereeCard.neutralLabel }}</text>
					</view>
					<view class="referee-card-body">
						<image class="referee-card-avatar" :src="resolveAvatarUrl(refereeCard.refereeAvatar, refereeCard.refereeUserId)" mode="aspectFill"/>
						<view class="referee-card-info">
							<text class="referee-card-name">{{ refereeCard.refereeName }}</text>
							<text class="referee-card-duration" v-if="refereeCard.refereeDurationText">执裁时长 {{ refereeCard.refereeDurationText }}</text>
						</view>
					</view>
					<view class="referee-card-footer" v-if="refereeCard.hasReliableAttribution">
						<text class="referee-attribution-label">完成方式：{{ refereeCard.completionLabel }}</text>
					</view>
				</view>

				<view class="insight-section info-card">
					<view class="section-head">
						<view>
							<text class="section-title">本场亮点</text>
							<text class="section-caption">{{ highlightSectionCaption }}</text>
						</view>
					</view>
					<view v-if="highlightItems.length" class="highlight-grid">
						<view
							v-for="item in highlightItems"
							:key="item.label"
							:class="['highlight-card', item.tone]"
						>
							<view class="highlight-icon">
								<uni-icons :type="item.icon" size="20" :color="item.iconColor"></uni-icons>
							</view>
							<text class="highlight-value">{{ item.value }}</text>
							<text class="highlight-label">{{ item.label }}</text>
						</view>
					</view>
					<view v-else class="empty-state">
						<uni-icons type="info" size="18" :color="isDarkMode ? '#94a3b8' : '#64748b'"></uni-icons>
						<text class="empty-state-text">暂无特殊战绩，本场战绩已正常计入记录。</text>
					</view>
					<view :class="['ranking-rights-card', rankingRightsSummary.tone]">
						<view class="ranking-rights-head">
							<text class="ranking-rights-badge">{{ rankingRightsSummary.badgeText }}</text>
							<text class="ranking-rights-title">{{ rankingRightsSummary.title }}</text>
						</view>
						<text class="ranking-rights-desc">{{ rankingRightsSummary.description }}</text>
					</view>
				</view>

				<view class="stats-section info-card">
					<view class="section-head">
						<view>
							<text class="section-title">本场数据</text>
							<text class="section-caption">关键表现对比与记录摘要</text>
						</view>
					</view>
					<view class="stats-list">
						<view v-for="stat in statsData" :key="stat.label" class="stat-row">
							<view class="stat-main">
								<view class="stat-icon">
									<uni-icons :type="stat.icon" size="18" :color="stat.iconColor"></uni-icons>
								</view>
								<view class="stat-copy">
									<text class="stat-label">{{ stat.label }}</text>
									<text class="stat-desc">{{ stat.desc }}</text>
								</view>
							</view>
							<text class="stat-value">{{ stat.value }}</text>
						</view>
					</view>
				</view>
			</view>
		</scroll-view>

		<view class="footer" v-if="refereeViewConfig.showH2HActions">
			<button
				v-if="!fromHistory"
				class="footer-btn btn-primary"
				@click="handleRematch"
			>
				{{ primaryActionText }}
			</button>
			<button class="footer-btn btn-secondary" @click="handleShare">
				{{ secondaryActionText }}
			</button>
			<button class="footer-btn btn-link" @click="handleH2H">查看交锋记录</button>
		</view>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getMatchDetail } from '@/api/match.js'
import { formatDateTime, formatRelativeTime } from '@/utils/format.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { resolveMatchRankingRightsSummary } from '@/utils/member-ranking-rights.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'
import { buildRematchContext } from '@/utils/match-core-flow.js'
import { resolveRefereeIdentityCard, resolveRefereeResultViewConfig, resolveCompletionSourceLabel } from '@/utils/match-referee-view.js'

const SHARE_LINK = 'https://appgallery.huawei.com/app/detail?id=hm.dianzaozao.ballmall&channelId=SHARE&source=appshare'

const matchId = ref(null)
const fromHistory = ref(false)
const loading = ref(true)
const isRedirecting = ref(false)
const matchData = ref({
	my_score: 0,
	opponent_score: 0,
	my_name: '我',
	opponent_name: '对手',
	my_avatar: '',
	opponent_avatar: '',
	game_type: 3,
	my_win_rate: 0,
	opponent_win_rate: 0,
	my_max_score: 0,
	opponent_max_score: 0,
	my_rank_change: 0,
	opponent_rank_change: 0,
	my_rank_details: [],
	summary_highlights: [],
	summary_stats: [],
	achievements: {},
	created_at: ''
})

const statusBarHeight = ref(0)
const { isDarkMode } = usePageTheme()
const headerIconColor = computed(() => (isDarkMode.value ? '#f8fafc' : '#1f2937'))

const refereeCard = computed(() => resolveRefereeIdentityCard({
	refereeBound: matchData.value.referee_bound,
	refereeUserId: matchData.value.referee_user_id,
	refereeName: matchData.value.referee_name,
	refereeAvatar: matchData.value.referee_avatar,
	refereeJoinedAt: matchData.value.referee_joined_at,
	refereeDurationSeconds: matchData.value.referee_duration_seconds,
	completedByUserId: matchData.value.completed_by_user_id,
	completionSource: matchData.value.completion_source
}))

const refereeViewConfig = computed(() => resolveRefereeResultViewConfig({
	viewerRole: matchData.value.viewer_role,
	status: 2
}))

const completionLabel = computed(() => resolveCompletionSourceLabel(matchData.value.completion_source))

const systemInfo = uni.getSystemInfoSync()
statusBarHeight.value = systemInfo.statusBarHeight || 20

const isReferee = computed(() => matchData.value.viewer_role === 'referee')
const isWin = computed(() => !isReferee.value && matchData.value.my_score > matchData.value.opponent_score)
const scoreGap = computed(() => Math.abs((matchData.value.my_score || 0) - (matchData.value.opponent_score || 0)))

const gameTypeName = computed(() => {
	switch (matchData.value.game_type) {
		case 1:
			return '斯诺克'
		case 2:
			return '九球追分'
		case 3:
			return '中式八球'
		case 4:
			return '美式九球'
		default:
			return '台球'
	}
})

const resultTitle = computed(() => {
	if (isReferee.value) return '执裁记录'
	return isWin.value ? '拿下这一场' : '这场先记下'
})

const resultTagText = computed(() => {
	if (isReferee.value) return '裁判视角'
	return isWin.value ? '胜利战报' : '复盘战报'
})

const resultSubtitle = computed(() => {
	if (isReferee.value) return '你已完成本场执裁，对局结果已录入系统。'
	if (isWin.value) {
		if (scoreGap.value >= 3) {
			return '整场节奏控制稳定，关键分兑现得更彻底。'
		}
		return '比分咬得很紧，但你在收官阶段更稳。'
	}

	if (scoreGap.value <= 1) {
		return '差距很小，下一场把关键球处理得更坚决。'
	}

	return '这场被对手带走节奏，回看数据后更容易找到突破口。'
})

const createdAtRelativeText = computed(() => {
	if (!matchData.value.created_at) return '刚刚结束'
	return formatRelativeTime(matchData.value.created_at)
})

const createdAtFullText = computed(() => {
	if (!matchData.value.created_at) return '时间待同步'
	return formatDateTime(matchData.value.created_at)
})

const scoreGapUnit = computed(() => (matchData.value.game_type === 2 ? '分' : '局'))
const scoreGapLabel = computed(() => (matchData.value.game_type === 2 ? '分差' : '局差'))

const scoreSummaryText = computed(() => {
	if (isReferee.value) return matchData.value.game_type === 2 ? '本场最终比分' : '本场最终局数'
	if (scoreGap.value === 0) return matchData.value.game_type === 2 ? '双方比分持平' : '双方局数持平'
	return isWin.value
		? `领先 ${scoreGap.value} ${scoreGapUnit.value}收下对局`
		: `落后 ${scoreGap.value} ${scoreGapUnit.value}结束对局`
})

const heroMetaItems = computed(() => [
	{ label: '项目', value: gameTypeName.value },
	{ label: scoreGapLabel.value, value: scoreGap.value === 0 ? '平局' : `${scoreGap.value} ${scoreGapUnit.value}` },
	{ label: '结束时间', value: createdAtFullText.value }
])

const myRankScoreClass = computed(() => getRankClass(matchData.value.my_rank_change))
const opponentRankScoreClass = computed(() => getRankClass(matchData.value.opponent_rank_change))

const rankDetailRows = computed(() => {
	if (!Array.isArray(matchData.value.my_rank_details)) return []
	return matchData.value.my_rank_details.filter((detail) => detail && detail.label)
})

const myRankNote = computed(() => {
	const change = matchData.value.my_rank_change || 0
	if (change > 0) return '本场胜利带来稳定上分。'
	if (change < 0) return '本场失利导致排位小幅回落。'
	return '本场对排位暂无直接影响，可能触发了平局或 0 分保护。'
})

const opponentRankNote = computed(() => {
	const change = matchData.value.opponent_rank_change || 0
	if (change > 0) return '对手本场状态更受益于排位分规则。'
	if (change < 0) return '对手本场排位分出现回撤。'
	return '对手本场排位分保持不变。'
})

const highlightItems = computed(() => {
	const highlights = Array.isArray(matchData.value.summary_highlights) ? matchData.value.summary_highlights : []
	return highlights.map((item) => ({
		...item,
		tone: item.tone || 'blue',
		iconColor: getToneColor(item.tone)
	}))
})

const highlightSectionCaption = computed(() => {
	if (highlightItems.value.length) return `按${gameTypeName.value}模式提炼这场最值得记住的表现`
	return `当前${gameTypeName.value}模式暂无可展示亮点，但战绩已完成记录`
})

const rankingRightsSummary = computed(() => resolveMatchRankingRightsSummary(matchData.value))

const statsData = computed(() => {
	const stats = Array.isArray(matchData.value.summary_stats) ? matchData.value.summary_stats : []
	return stats.map((item) => ({
		...item,
		iconColor: getToneColor(item.tone)
	}))
})

const myAvatar = computed(() => resolveAvatarUrl(
	matchData.value.my_avatar,
	matchData.value.my_user_id || matchData.value.user_id
))
const opponentAvatar = computed(() => resolveAvatarUrl(
	matchData.value.opponent_avatar,
	matchData.value.opponent_id
))
const primaryActionText = computed(() => '再来一局')
const secondaryActionText = computed(() => (fromHistory.value ? '生成战绩海报' : '分享战绩'))

onLoad((options) => {
	if (options.match_id) {
		matchId.value = parseInt(options.match_id, 10)
	}
	fromHistory.value = options.from === 'history'
	loadMatchData()
})

const loadMatchData = async () => {
	if (!matchId.value) {
		loading.value = false
		return
	}

	loading.value = true

	try {
		const res = await getMatchDetail({ match_id: matchId.value })
		if (res.success && res.match) {
			matchData.value = {
				...matchData.value,
				...res.match
			}
			return
		}
		uni.showToast({
			title: '暂无可用的对局总结数据',
			icon: 'none'
		})
	} catch (error) {
		console.error('加载对局数据失败:', error)
		uni.showToast({
			title: '加载对局数据失败',
			icon: 'none'
		})
	} finally {
		loading.value = false
	}
}

const formatRankScore = (value) => {
	if (!value) return '+0'
	return value > 0 ? `+${value}` : `${value}`
}

const formatWinRate = (value) => {
	if (value === null || value === undefined || value === '') return '--'
	return `${Number(value).toFixed(0)}%`
}

const getRankClass = (value) => {
	if (value > 0) return 'is-positive'
	if (value < 0) return 'is-negative'
	return 'is-zero'
}

const getToneColor = (tone) => {
	switch (tone) {
		case 'gold':
			return '#f59e0b'
		case 'blue':
			return '#3b82f6'
		case 'red':
			return '#ef4444'
		case 'green':
			return '#22c55e'
		case 'emerald':
			return '#10b981'
		case 'purple':
			return '#8b5cf6'
		case 'orange':
			return '#f97316'
		case 'cyan':
			return '#06b6d4'
		case 'slate':
		default:
			return '#64748b'
	}
}

const handleBack = () => {
	if (fromHistory.value) {
		uni.navigateBack()
		return
	}
	redirectToMatchHistory()
}

const handleClose = () => {
	redirectToMatchHistory()
}

const redirectToMatchHistory = () => {
	if (isRedirecting.value) return
	isRedirecting.value = true

	uni.switchTab({
		url: '/pages/user/index',
		success: () => {
			setTimeout(() => {
				uni.navigateTo({
					url: '/subPages/user/matchHistory',
					fail: () => {
						isRedirecting.value = false
					}
				})
			}, 120)
		},
		fail: () => {
			isRedirecting.value = false
			uni.reLaunch({
				url: '/subPages/user/matchHistory'
			})
		}
	})
}

const handleRematch = () => {
	const context = buildRematchContext(matchData.value)
	if (!context.opponent_id) {
		uni.showToast({ title: '缺少对手信息，请从对局页重新扫码', icon: 'none' })
		return
	}
	uni.setStorageSync('pending_match_rematch', JSON.stringify(context))
	uni.switchTab({ url: '/pages/match/index' })
}

const handleH2H = () => {
	const opponentId = Number(matchData.value.opponent_id || 0)
	if (!opponentId) {
		uni.showToast({ title: '暂无可用的交锋记录', icon: 'none' })
		return
	}
	uni.navigateTo({
		url: `/subPages/user/h2hRecord?opponent_id=${opponentId}&opponent_name=${encodeURIComponent(matchData.value.opponent_name || '对手')}`
	})
}

const handleShare = () => {
	if (!matchId.value) {
		copyShareLink()
		return
	}

	uni.navigateTo({
		url: `/subPages/match/shareResult?match_id=${matchId.value}`,
		fail: () => {
			copyShareLink()
		}
	})
}

const copyShareLink = () => {
	uni.setClipboardData({
		data: SHARE_LINK,
		success: () => {
			uni.showToast({
				title: '分享链接已复制',
				icon: 'none'
			})
		}
	})
}
</script>

<style lang="scss" scoped>
@import './matchResult.scss';
</style>
