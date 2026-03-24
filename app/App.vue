<script>
	import { useThemeStore, THEME_CHANGE_EVENT } from '@/store/theme.js'
	import { useUserStore } from '@/store/user.js'
	import { getCurrentMatch } from '@/api/match.js'
	import { post } from '@/utils/request.js'
	import { buildPlayingRoute, shouldPromptOngoingMatch } from '@/utils/ongoing-match-guard.js'

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
			const sysInfo = uni.getSystemInfoSync()
			themeStore.setThemeFromSystem(sysInfo.osTheme === 'dark')
			// 应用当前主题样式
			this.applyTheme()
			
			// 保存回调函数引用，以便后续取消监听
			const self = this
			this.themeChangeCallback = function (res) {
				themeStore.setThemeFromSystem(res?.theme === 'dark')
				self.applyTheme()
			}
			uni.onThemeChange(this.themeChangeCallback)
			this.checkOngoingMatchReminder()
		},
		onHide: function() {
			// 取消监听时需要传入与注册时相同的回调函数引用
			if (this.themeChangeCallback) {
				uni.offThemeChange(this.themeChangeCallback)
			}
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
			applyTheme(theme) {
				const themeStore = useThemeStore()
				
				// 如果没有传入 theme，从 store 或系统获取
				if (!theme) {
					// 优先使用 store 中的状态
					theme = themeStore.isDarkMode ? 'dark' : 'light'
				}

				console.log('[App] applyTheme:', theme)

				if (theme === 'dark') {
					// 暗色主题
					uni.setNavigationBarColor({
						frontColor: '#ffffff',
						backgroundColor: '#141109',
						animation: {
							duration: 400,
							timingFunc: 'easeIn'
						}
					})
					uni.setTabBarStyle({
						backgroundColor: '#141109',
						borderStyle: 'white',
						color: '#c6b78c',
						selectedColor: '#E0AE12'
					})
				} else {
					// 亮色主题
					uni.setNavigationBarColor({
						frontColor: '#000000',
						backgroundColor: '#faf8f2',
						animation: {
							duration: 400,
							timingFunc: 'easeIn'
						}
					})
					uni.setTabBarStyle({
						backgroundColor: '#faf8f2',
						borderStyle: 'black',
						color: '#6e6242',
						selectedColor: '#E0AE12'
					})
				}
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

						uni.showModal({
							title: '你有未结束的对局',
							content: `你和 ${currentMatch.opponent_name || '对手'} 的${currentMatch.game_type_name || 'PK'}对局仍在进行中，是否立即进入？`,
							confirmText: '进入对局',
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
	*,
	*::before,
	*::after {
		box-sizing: border-box;
	}

	/* 全局 CSS 变量定义 - 亮色主题（默认） */
	page {
		/* 主色调 */
		--primary-color: #e0ae12;
		--primary-color-light: rgba(224, 174, 18, 0.14);

		/* 背景色 */
		--bg-color: #faf8f2;
		--card-bg: #ffffff;
		--input-bg: #ffffff;

		/* 文字颜色 */
		--text-primary: #231c0b;
		--text-secondary: #6e6242;
		--text-tertiary: #9a8c67;

		/* 边框颜色 */
		--border-color: #e9e2cf;

		/* 其他 */
		--divider-color: #e9e2cf;
		--danger-color: #ef4444;
	}

	/* 暗色主题 */
		@media (prefers-color-scheme: dark) {
			page {
				/* 主色调 */
				--primary-color: #e0ae12;
				--primary-color-light: rgba(224, 174, 18, 0.2);

				/* 背景色 */
				--bg-color: #141109;
				--card-bg: #1e180d;
				--input-bg: #1e180d;

				/* 文字颜色 */
				--text-primary: #fff7e1;
				--text-secondary: #d7c89b;
				--text-tertiary: #9f926e;

				/* 边框颜色 */
				--border-color: #3a2e16;

				/* 其他 */
				--divider-color: #241d0f;
				--danger-color: #ef4444;
			}
		}
	</style>
