<script>
	import { useThemeStore, THEME_CHANGE_EVENT } from '@/store/theme.js'
	import { useActivityStore } from '@/store/activity.js'
	import { useUserStore } from '@/store/user.js'
	import { useRankStore } from '@/store/rank.js'
	import { useUserOverviewStore } from '@/store/userOverview.js'
	import { useUserDataInvalidationStore } from '@/store/userDataInvalidation.js'
	import { usePublicReadStore } from '@/store/publicRead.js'
	import { matchWS, userWS, WS_MESSAGE_TYPES } from '@/utils/websocket.js'
	import { getUserBootstrap } from '@/api/user.js'
	import { handleCurrentSessionInvalid, post } from '@/utils/request.js'
	import { buildPlayingRoute, shouldPromptOngoingMatch } from '@/utils/ongoing-match-guard.js'
	import { applyRuntimeTheme } from '@/utils/theme-application.js'
	import { resolveSystemDarkMode } from '@/utils/theme-preference.js'
	import {
		canApplySessionRecoveryResult,
		createSessionRecovery
	} from '@/utils/session-recovery.js'

	// 网络/5xx 静默保留凭证；SESSION_INVALID 仍由请求层统一清理并引导重新登录。
	const sessionRecovery = createSessionRecovery({
		getUserInfo: () => getUserBootstrap({ silent: true }),
		onSessionInvalid: () => {}
	})

	export default {
		themeChangeCallback: null, // 保存主题变化回调函数引用
		userSessionReadyCallback: null,
		websocketSessionInvalidCallback: null,
		bootstrapActivityReservation: null,
		ongoingMatchReminderShown: false,
		ongoingMatchReminderPending: false,
		ongoingMatchReminderPendingKey: '',
		ongoingMatchPromptVisible: false,
		ongoingMatchReminderRetryCount: 0,
		ongoingMatchReminderRetryTimer: null,
		maxOngoingMatchReminderRetries: 5,
		appIsForeground: false,
		sessionRecoveryLifecycle: 0,
		validatedAuthGeneration: -1,
		pushTokenUploadPendingKey: '',
		pushTokenUploadedKey: '',
		networkStatusCallback: null,
		onLaunch: function() {
			console.log('App Launch')
			const themeStore = useThemeStore()
			themeStore.initializeTheme(this.getSystemThemeInfo())
			userWS.off(WS_MESSAGE_TYPES.RANK_INFO_UPDATED, this.handleRankInfoUpdated)
			userWS.off(WS_MESSAGE_TYPES.USER_DATA_UPDATED, this.handleUserDataUpdated)
			userWS.off(WS_MESSAGE_TYPES.NOTIFICATION_UPDATE, this.handleNotificationUpdate)
			userWS.on(WS_MESSAGE_TYPES.RANK_INFO_UPDATED, this.handleRankInfoUpdated)
			userWS.on(WS_MESSAGE_TYPES.USER_DATA_UPDATED, this.handleUserDataUpdated)
			userWS.on(WS_MESSAGE_TYPES.NOTIFICATION_UPDATE, this.handleNotificationUpdate)
			if (this.userSessionReadyCallback && typeof uni.$off === 'function') {
				uni.$off('user-session-ready', this.userSessionReadyCallback)
			}
			this.userSessionReadyCallback = () => this.handleUserSessionReady()
			if (typeof uni.$on === 'function') {
				uni.$on('user-session-ready', this.userSessionReadyCallback)
			}
			if (this.websocketSessionInvalidCallback && typeof uni.$off === 'function') {
				uni.$off('session-invalid', this.websocketSessionInvalidCallback)
			}
			this.websocketSessionInvalidCallback = (payload) => this.handleWebSocketSessionInvalid(payload)
			if (typeof uni.$on === 'function') {
				uni.$on('session-invalid', this.websocketSessionInvalidCallback)
			}
			this.bindNetworkStatusListener()
			// 推送注册
			// #ifdef APP-PLUS
			try {
				uni.getPushClientId({
					success: (res) => {
						console.log('[Push] Got client ID:', res.cid)
						uni.setStorageSync('pushClientId', res.cid)
						this.uploadPushTokenIfValidated()
					},
					fail: (err) => {
						console.error('[Push] getPushClientId failed:', err)
					}
				})
				uni.onPushMessage((res) => {
					console.log('[Push] Message received:', res)
					if (res.type === 'click') {
						const payload = typeof res.data === 'string' ? JSON.parse(res.data) : res.data
						if (payload && payload.url) {
							uni.navigateTo({ url: payload.url })
						}
					}
				})
			} catch (e) {
				console.error('[Push] Init failed:', e)
			}
			// #endif
		},
		onShow: function() {
			console.log('App Show')
			this.appIsForeground = true
			this.sessionRecoveryLifecycle += 1
			userWS.setForeground(true)
			matchWS.setForeground(true)
			const themeStore = useThemeStore()
			themeStore.setThemeFromSystem(resolveSystemDarkMode(
				this.getSystemThemeInfo(),
				themeStore.systemIsDark
			))
			// 应用当前主题样式
			this.applyTheme()
			const userStore = useUserStore()
			const rankStore = useRankStore()
			if (userStore.isLoggedIn) {
				this.restoreUserSession()
			} else {
				this.validatedAuthGeneration = -1
				rankStore.clear()
				userWS.disconnect()
			}

			if (this.themeChangeCallback && typeof uni.offThemeChange === 'function') {
				uni.offThemeChange(this.themeChangeCallback)
			}

			// 保存回调函数引用，以便后续取消监听
			const self = this
			this.themeChangeCallback = function (res) {
				themeStore.setThemeFromSystem(resolveSystemDarkMode(res, themeStore.systemIsDark))
				self.applyTheme()
			}
			if (typeof uni.onThemeChange === 'function') {
				uni.onThemeChange(this.themeChangeCallback)
			}
		},
		onHide: function() {
			this.appIsForeground = false
			this.bootstrapActivityReservation?.release()
			this.bootstrapActivityReservation = null
			this.sessionRecoveryLifecycle += 1
			this.validatedAuthGeneration = -1
			userWS.setForeground(false)
			matchWS.setForeground(false)
			userWS.disconnect()
			// 取消监听时需要传入与注册时相同的回调函数引用
			if (this.themeChangeCallback && typeof uni.offThemeChange === 'function') {
				uni.offThemeChange(this.themeChangeCallback)
			}
			this.themeChangeCallback = null
			if (this.ongoingMatchReminderRetryTimer) {
				clearTimeout(this.ongoingMatchReminderRetryTimer)
				this.ongoingMatchReminderRetryTimer = null
			}
			this.ongoingMatchReminderShown = false
			this.ongoingMatchReminderPending = false
			this.ongoingMatchReminderPendingKey = ''
			this.ongoingMatchPromptVisible = false
			this.ongoingMatchReminderRetryCount = 0
		},
		onUnload: function() {
			if (this.networkStatusCallback && typeof uni.offNetworkStatusChange === 'function') {
				uni.offNetworkStatusChange(this.networkStatusCallback)
			}
			this.networkStatusCallback = null
			if (this.websocketSessionInvalidCallback && typeof uni.$off === 'function') {
				uni.$off('session-invalid', this.websocketSessionInvalidCallback)
			}
			this.websocketSessionInvalidCallback = null
			userWS.disconnect()
			matchWS.disconnect()
		},
		methods: {
			handleWebSocketSessionInvalid(payload = {}) {
				if (payload?.reason !== 'SESSION_INVALID') return
				handleCurrentSessionInvalid({ data: payload, message: payload.message }).catch(() => {})
			},
			bindNetworkStatusListener() {
				if (this.networkStatusCallback && typeof uni.offNetworkStatusChange === 'function') {
					uni.offNetworkStatusChange(this.networkStatusCallback)
				}
				this.networkStatusCallback = (status = {}) => {
					this.setRealtimeNetworkOnline(status.isConnected !== false)
				}
				if (typeof uni.onNetworkStatusChange === 'function') {
					uni.onNetworkStatusChange(this.networkStatusCallback)
				}
				if (typeof uni.getNetworkType === 'function') {
					uni.getNetworkType({
						success: (status = {}) => {
							this.setRealtimeNetworkOnline(status.networkType !== 'none')
						}
					})
				}
			},
			setRealtimeNetworkOnline(online) {
				userWS.setNetworkOnline(online)
				matchWS.setNetworkOnline(online)
			},
			// 恢复持久化会话：helper 只返回服务端资料；应用前再次校验
			// auth generation、App 前台状态和当前生命周期，拒绝旧账号/后台延迟结果。
			restoreUserSession() {
				const userStore = useUserStore()
				const expectedGeneration = userStore.authGeneration
				const expectedLifecycleGeneration = this.sessionRecoveryLifecycle
				const bootstrapReservation = useActivityStore().reserveBootstrap({
					userId: userStore.userId,
					authGeneration: expectedGeneration
				})
				if (bootstrapReservation) {
					this.bootstrapActivityReservation = bootstrapReservation
				}

				sessionRecovery.validate(expectedGeneration)
					.then((result) => {
						if (!result?.valid) {
							bootstrapReservation?.release()
							return
						}
						const currentUserStore = useUserStore()
						if (!canApplySessionRecoveryResult({
							expectedGeneration,
							currentGeneration: currentUserStore.authGeneration,
							expectedLifecycleGeneration,
							currentLifecycleGeneration: this.sessionRecoveryLifecycle,
							isLoggedIn: currentUserStore.isLoggedIn,
							isForeground: this.appIsForeground
						})) {
							bootstrapReservation?.release()
							return
						}

						currentUserStore.updateUserInfo(result.userInfo)
						const identity = {
							userId: currentUserStore.userId,
							authGeneration: currentUserStore.authGeneration
						}
						if (bootstrapReservation?.matches(identity)) {
							bootstrapReservation.apply(result.bootstrap)
						} else {
							bootstrapReservation?.release()
							useActivityStore().applyBootstrap(identity, result.bootstrap)
						}
						if (this.bootstrapActivityReservation === bootstrapReservation) {
							this.bootstrapActivityReservation = null
						}
						this.applyBootstrapCompetitiveRevision(identity, result.bootstrap)
						this.validatedAuthGeneration = expectedGeneration
						this.connectUserWS()
						this.uploadPushTokenIfValidated()
						this.checkOngoingMatchReminder()
					})
					.catch(() => {
						bootstrapReservation?.release()
						if (this.bootstrapActivityReservation === bootstrapReservation) {
							this.bootstrapActivityReservation = null
						}
					})
			},
			handleUserSessionReady() {
				if (this.appIsForeground && useUserStore().isLoggedIn) {
					this.restoreUserSession()
				}
			},
			applyBootstrapCompetitiveRevision(identity, bootstrap = {}) {
				if (bootstrap?.availability?.competitive_revision === false) return
				const invalidationStore = useUserDataInvalidationStore()
				const previousRevision = invalidationStore.matchesIdentity(identity)
					? invalidationStore.competitiveRevision
					: 0
				const revision = Number(bootstrap?.competitive_revision) || 0
				const scopes = previousRevision > 0 && revision > previousRevision
					? ['rank', 'stats', 'h2h', 'opponents', 'history', 'honor', 'season', 'leaderboard']
					: []
				invalidationStore.invalidate(identity, scopes, revision)
			},
			hasValidatedAppSession() {
				const userStore = useUserStore()
				return this.appIsForeground &&
					userStore.isLoggedIn &&
					this.validatedAuthGeneration === userStore.authGeneration
			},
			uploadPushTokenIfValidated() {
				if (!this.hasValidatedAppSession()) return
				const pushClientId = String(uni.getStorageSync('pushClientId') || '')
				if (!pushClientId) return

				const userStore = useUserStore()
				const uploadKey = `${userStore.authGeneration}:${pushClientId}`
				if (this.pushTokenUploadPendingKey === uploadKey || this.pushTokenUploadedKey === uploadKey) return

				this.pushTokenUploadPendingKey = uploadKey
				post('/api/user/push-token', { push_client_id: pushClientId }, { silent: true })
					.then(() => {
						if (this.hasValidatedAppSession()) {
							this.pushTokenUploadedKey = uploadKey
						}
						console.log('[Push] Token uploaded')
					})
					.catch(err => console.error('[Push] Token upload failed:', err))
					.finally(() => {
						if (this.pushTokenUploadPendingKey === uploadKey) {
							this.pushTokenUploadPendingKey = ''
						}
					})
			},
			connectUserWS() {
				const userStore = useUserStore()
				userWS.connect({ authGeneration: userStore.authGeneration }).catch((error) => {
					console.error('[App] 用户WS连接失败:', error)
				})
			},
			handleRankInfoUpdated(data = {}) {
				this.handleUserDataUpdated({ ...data, scopes: ['rank'] })
			},
			handleUserDataUpdated(data = {}) {
				const userStore = useUserStore()
				if (!userStore.isLoggedIn || !userStore.userId) return

				const scopes = Array.isArray(data.scopes) ? data.scopes : []
				const identity = {
					userId: userStore.userId,
					authGeneration: userStore.authGeneration
				}
				useUserDataInvalidationStore().invalidate(identity, scopes, data.competitive_revision)

				if (scopes.includes('rank')) {
					useRankStore().invalidate(identity)
				}
				if (scopes.some((scope) => ['stats', 'member', 'reputation'].includes(scope))) {
					useUserOverviewStore().markDirty()
				}
				if (scopes.includes('leaderboard')) {
					usePublicReadStore().invalidate('leaderboard')
				}
				if (Object.prototype.hasOwnProperty.call(data, 'pending_friend_request_count')) {
					useActivityStore().setPendingFriendRequestCount(data.pending_friend_request_count, identity)
				}
			},
			handleNotificationUpdate(data = {}) {
				const userStore = useUserStore()
				if (!userStore.isLoggedIn) return
				const identity = {
					userId: userStore.userId,
					authGeneration: userStore.authGeneration
				}
				useUserDataInvalidationStore().invalidate(identity, ['notification'])
				if (Object.prototype.hasOwnProperty.call(data, 'unread_count')) {
					useActivityStore().setUnreadCount(data.unread_count, identity)
				}
				if (data.category === 'season_rollover') {
					useActivityStore().markDirty()
				}
			},
			getSystemThemeInfo() {
				try {
					return uni.getSystemInfoSync()
				} catch (error) {
					console.warn('[App] 获取系统主题失败:', error)
					return {}
				}
			},
			applyTheme(theme) {
				const themeStore = useThemeStore()

				// 如果没有传入 theme，从 store 或系统获取
				if (!theme) {
					// 优先使用 store 中的状态
					theme = themeStore.isDarkMode ? 'dark' : 'light'
				}

				console.log('[App] applyTheme:', theme)

				applyRuntimeTheme({
					uniApi: uni,
					isDarkMode: theme === 'dark',
					animationDuration: 400
				})
			},
			getCurrentRoute() {
				const pages = getCurrentPages()
				const currentPage = pages[pages.length - 1]
				if (!currentPage?.route) {
					return ''
				}
				return `/${currentPage.route}`
			},
			scheduleOngoingMatchReminderRetry() {
				if (this.ongoingMatchReminderRetryCount >= this.maxOngoingMatchReminderRetries) {
					return
				}
				if (this.ongoingMatchReminderRetryTimer) {
					clearTimeout(this.ongoingMatchReminderRetryTimer)
				}
				this.ongoingMatchReminderRetryCount += 1
				this.ongoingMatchReminderRetryTimer = setTimeout(() => {
					this.ongoingMatchReminderRetryTimer = null
					this.checkOngoingMatchReminder()
				}, 80)
			},
			checkOngoingMatchReminder() {
				const userStore = useUserStore()
				if (!this.hasValidatedAppSession() || this.ongoingMatchReminderPending || this.ongoingMatchPromptVisible) {
					return
				}
				const expectedGeneration = userStore.authGeneration
				const expectedLifecycleGeneration = this.sessionRecoveryLifecycle
				const pendingKey = `${expectedGeneration}:${expectedLifecycleGeneration}`

				const currentRoute = this.getCurrentRoute()
				if (!currentRoute) {
					this.scheduleOngoingMatchReminderRetry()
					return
				}

				this.ongoingMatchReminderRetryCount = 0

				this.ongoingMatchReminderPending = true
				this.ongoingMatchReminderPendingKey = pendingKey

				useActivityStore().fetch({
					userId: userStore.userId,
					authGeneration: userStore.authGeneration
				}, { silent: true })
					.then((activity) => {
						const currentUserStore = useUserStore()
						if (!this.hasValidatedAppSession() || !canApplySessionRecoveryResult({
							expectedGeneration,
							currentGeneration: currentUserStore.authGeneration,
							expectedLifecycleGeneration,
							currentLifecycleGeneration: this.sessionRecoveryLifecycle,
							isLoggedIn: currentUserStore.isLoggedIn,
							isForeground: this.appIsForeground
						})) return
						const currentMatch = activity?.currentMatch || null
						if (!shouldPromptOngoingMatch({
							isLoggedIn: currentUserStore.isLoggedIn,
							currentRoute,
							currentMatch,
							hasPromptedInForeground: this.ongoingMatchReminderShown
						})) {
							return
						}

						this.ongoingMatchReminderShown = true
						this.ongoingMatchPromptVisible = true
						const finishPending = currentMatch.finish_state === 'pending_confirmation'

						uni.showModal({
							title: finishPending ? '有一场对局等待确认' : '你有未结束的对局',
							content: finishPending
								? `你和 ${currentMatch.opponent_name || '对手'} 的排位赛正在等待结束确认，是否立即处理？`
								: currentMatch.viewer_role === 'referee'
									? `你担任裁判的${currentMatch.game_type_name || 'PK'}对局仍在进行中，是否立即进入？`
									: `你和 ${currentMatch.opponent_name || '对手'} 的${currentMatch.game_type_name || 'PK'}对局仍在进行中，是否立即进入？`,
							confirmText: finishPending ? '处理确认' : '进入对局',
							cancelText: '暂不进入',
							success: ({ confirm }) => {
								if (confirm) {
									uni.navigateTo({
										url: buildPlayingRoute(currentMatch)
									})
								}
							},
							complete: () => {
								this.ongoingMatchPromptVisible = false
							}
						})
					})
					.catch(() => {})
					.finally(() => {
						if (this.ongoingMatchReminderPendingKey === pendingKey) {
							this.ongoingMatchReminderPending = false
							this.ongoingMatchReminderPendingKey = ''
						}
					})
			}
		}
	}
</script>

<style lang="scss">
	/*每个页面公共css */
	@font-face {
		font-family: CustomFont;
		src: url('./static/iconfont/iconfont.ttf');
	}

	/* 全局盒模型设置 - 避免 width: 100% + padding 导致元素超出容器 */
	/* #ifndef MP-WEIXIN */
	*,
	*::before,
	*::after {
		box-sizing: border-box;
	}
	/* #endif */

	/* 全局 CSS 变量定义 - 亮色主题（默认） */
	page {
		/* 品牌色 */
		--ui-brand-primary: #E0AE12;
		--ui-brand-strong: #C69200;
		--ui-brand-gradient-start: #D9A617;
		--ui-brand-gradient-end: #BB8400;
		--ui-brand-tint: rgba(224, 174, 18, 0.14);

		/* 表面色 */
		--ui-surface-page: #F7F4EC;
		--ui-surface-card: #FFFFFF;
		--ui-surface-subtle: #FAF8F2;

		/* 文字色 */
		--ui-text-primary: #231C0B;
		--ui-text-secondary: #6E6242;
		--ui-text-muted: #9A8C67;

		/* 边框 */
		--ui-border-default: #E9E2CF;

		/* 语义色 */
		--ui-success: #18B05B;
		--ui-warning: #F97316;
		--ui-danger: #EF4444;
		--ui-info: #3B82F6;

		/* 圆角 */
		--ui-radius-sm: 12rpx;
		--ui-radius-md: 18rpx;
		--ui-radius-lg: 24rpx;
		--ui-radius-xl: 32rpx;
		--ui-radius-pill: 999rpx;

		/* 阴影 */
		--ui-shadow-soft: 0 2rpx 8rpx rgba(31, 26, 16, 0.05);
		--ui-shadow-primary: 0 8rpx 32rpx rgba(224, 174, 18, 0.22);
		--ui-shadow-card: 0 16rpx 40rpx rgba(31, 26, 16, 0.08);

		/* 兼容别名：迁移期保留，指向语义 Token */
		--primary-color: var(--ui-brand-primary);
		--primary-color-light: var(--ui-brand-tint);
		--bg-color: var(--ui-surface-page);
		--card-bg: var(--ui-surface-card);
		--input-bg: var(--ui-surface-card);
		--text-primary: var(--ui-text-primary);
		--text-secondary: var(--ui-text-secondary);
		--text-tertiary: var(--ui-text-muted);
		--border-color: var(--ui-border-default);
		--divider-color: var(--ui-border-default);
		--danger-color: var(--ui-danger);
	}

	/* 暗色变量由应用最终计算出的主题控制，避免手动浅色与系统暗色互相覆盖。 */
	.dark-mode {
		--ui-brand-primary: #E0AE12;
		--ui-brand-strong: #F0C542;
		--ui-brand-gradient-start: #E0AE12;
		--ui-brand-gradient-end: #A97500;
		--ui-brand-tint: rgba(224, 174, 18, 0.20);

		--ui-surface-page: #141109;
		--ui-surface-card: #1E180D;
		--ui-surface-subtle: #241D10;

		--ui-text-primary: #FFF7E1;
		--ui-text-secondary: #D7C89B;
		--ui-text-muted: #9F926E;

		--ui-border-default: #3A2E16;

		--ui-success: #22C55E;
		--ui-warning: #F97316;
		--ui-danger: #EF4444;
		--ui-info: #60A5FA;

		--ui-shadow-soft: 0 2rpx 8rpx rgba(0, 0, 0, 0.20);
		--ui-shadow-primary: 0 8rpx 32rpx rgba(0, 0, 0, 0.28);
		--ui-shadow-card: 0 16rpx 40rpx rgba(0, 0, 0, 0.24);
	}
	</style>
