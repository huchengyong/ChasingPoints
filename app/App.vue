<script>
	import { useThemeStore, THEME_CHANGE_EVENT } from '@/store/theme.js'
	import { useUserStore } from '@/store/user.js'
	import { useRankStore } from '@/store/rank.js'
	import { userWS, WS_MESSAGE_TYPES } from '@/utils/websocket.js'
	import { getCurrentMatch } from '@/api/match.js'
	import { post } from '@/utils/request.js'
	import { buildPlayingRoute, shouldPromptOngoingMatch } from '@/utils/ongoing-match-guard.js'
	import { applyRuntimeTheme } from '@/utils/theme-application.js'
	import { resolveSystemDarkMode } from '@/utils/theme-preference.js'

	export default {
		themeChangeCallback: null, // 保存主题变化回调函数引用
		ongoingMatchReminderShown: false,
		ongoingMatchReminderPending: false,
		ongoingMatchPromptVisible: false,
		ongoingMatchReminderRetryCount: 0,
		ongoingMatchReminderRetryTimer: null,
		maxOngoingMatchReminderRetries: 5,
		onLaunch: function() {
			console.log('App Launch')
			const themeStore = useThemeStore()
			themeStore.initializeTheme(this.getSystemThemeInfo())
			userWS.off(WS_MESSAGE_TYPES.RANK_INFO_UPDATED, this.handleRankInfoUpdated)
			userWS.on(WS_MESSAGE_TYPES.RANK_INFO_UPDATED, this.handleRankInfoUpdated)
			// 推送注册
			// #ifdef APP-PLUS
			try {
				uni.getPushClientId({
					success: (res) => {
						console.log('[Push] Got client ID:', res.cid)
						uni.setStorageSync('pushClientId', res.cid)
						const token = uni.getStorageSync('token')
						if (token && res.cid) {
							post('/api/user/push-token', { push_client_id: res.cid })
								.then(() => console.log('[Push] Token uploaded'))
								.catch(err => console.error('[Push] Token upload failed:', err))
						}
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
				rankStore.invalidate(userStore.userId)
				this.connectUserWS()
			} else {
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
			this.checkOngoingMatchReminder()
		},
		onHide: function() {
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
			this.ongoingMatchPromptVisible = false
			this.ongoingMatchReminderRetryCount = 0
		},
		methods: {
			connectUserWS() {
				userWS.connect().catch((error) => {
					console.error('[App] 用户WS连接失败:', error)
				})
			},
			handleRankInfoUpdated() {
				const userStore = useUserStore()
				if (userStore.isLoggedIn) {
					useRankStore().invalidate(userStore.userId)
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
				if (!userStore.isLoggedIn || this.ongoingMatchReminderPending || this.ongoingMatchPromptVisible) {
					return
				}

				const currentRoute = this.getCurrentRoute()
				if (!currentRoute) {
					this.scheduleOngoingMatchReminderRetry()
					return
				}

				this.ongoingMatchReminderRetryCount = 0

				this.ongoingMatchReminderPending = true

				getCurrentMatch({ silent: true })
					.then((res) => {
						const currentMatch = res?.success ? res.match : null
						if (!shouldPromptOngoingMatch({
							isLoggedIn: userStore.isLoggedIn,
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
						this.ongoingMatchReminderPending = false
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
		/* 主色调 */
		--primary-color: #e0ae12;
		--primary-color-light: rgba(224, 174, 18, 0.14);

		/* 背景色 */
		--bg-color: #ffffff;
		--card-bg: #ffffff;
		--input-bg: #ffffff;

		/* 文字颜色 */
		--text-primary: #0f172a;
		--text-secondary: #64748b;
		--text-tertiary: #94a3b8;

		/* 边框颜色 */
		--border-color: #e5e7eb;

		/* 其他 */
		--divider-color: #e5e7eb;
		--danger-color: #ef4444;
	}

	/* 暗色变量由应用最终计算出的主题控制，避免手动浅色与系统暗色互相覆盖。 */
	.dark-mode {
		--primary-color: #e0ae12;
		--primary-color-light: rgba(224, 174, 18, 0.2);
		--bg-color: #141109;
		--card-bg: #1e180d;
		--input-bg: #1e180d;
		--text-primary: #fff7e1;
		--text-secondary: #d7c89b;
		--text-tertiary: #9f926e;
		--border-color: #3a2e16;
		--divider-color: #241d0f;
		--danger-color: #ef4444;
	}
	</style>
