<template>
	<view class="agreement-container" :class="{ 'dark-mode': isDarkMode }">
		<!-- 内容区域 -->
		<scroll-view class="content" scroll-y>
			<view class="article">
				<text class="article-title">追分用户服务协议</text>
				<text class="update-time">更新日期：2024年12月1日</text>
				
				<view class="section">
					<text class="section-title">一、总则</text>
					<text class="section-content">
						欢迎您使用追分！本协议是您与追分之间关于您使用追分服务所订立的协议。请您仔细阅读本协议，您访问或使用追分，即表示您已阅读并同意受本协议的约束。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">二、服务内容</text>
					<text class="section-content">
						1. 追分为用户提供台球比赛记录、数据分析、智能匹配对手等服务。
						
						2. 用户可以通过本应用记录个人比赛数据，查看排行榜，与其他用户进行互动。
						
						3. 我们会不断完善和优化服务内容，为用户提供更好的体验。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">三、用户账号</text>
					<text class="section-content">
						1. 用户在使用本服务前需要注册账号。注册时，用户应当提供真实、准确、完整的信息。
						
						2. 用户应当妥善保管自己的账号和密码，对账号下的所有活动负责。
						
						3. 如发现账号被盗用或其他安全问题，用户应立即通知我们。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">四、用户行为规范</text>
					<text class="section-content">
						1. 用户在使用本服务时，应遵守国家法律法规和社会公德。
						
						2. 用户不得利用本服务从事任何违法违规活动。
						
						3. 用户不得发布虚假信息、恶意攻击其他用户或干扰服务正常运行。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">五、知识产权</text>
					<text class="section-content">
						1. 追分的所有内容，包括但不限于文字、图片、软件、音频、视频等，均受著作权法保护。
						
						2. 未经我们书面许可，用户不得复制、修改、传播或以其他方式使用上述内容。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">六、免责声明</text>
					<text class="section-content">
						1. 对于因不可抗力或非我们过错导致的服务中断，我们不承担责任。
						
						2. 用户因自身原因造成的损失，我们不承担责任。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">七、协议修改</text>
					<text class="section-content">
						我们保留随时修改本协议的权利。修改后的协议将在应用内公布，用户继续使用本服务即表示同意修改后的协议。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">八、联系我们</text>
					<text class="section-content">
						如您对本协议有任何疑问，请通过应用内的反馈渠道联系我们。
					</text>
				</view>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { ref, computed, onUnmounted } from 'vue'
import { onShow, onHide } from '@dcloudio/uni-app'
import { useThemeStore, THEME_CHANGE_EVENT } from '@/store/theme.js'

// ========== 状态管理 ==========
const themeStore = useThemeStore()

// ========== 计算属性 ==========
const isDarkMode = computed(() => themeStore.isDarkMode)

// ========== 生命周期 ==========
onShow(() => {
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
	// 监听主题变化事件
	uni.$on(THEME_CHANGE_EVENT, handleThemeChange)
})

onHide(() => {
	uni.$off(THEME_CHANGE_EVENT, handleThemeChange)
})

onUnmounted(() => {
	uni.$off(THEME_CHANGE_EVENT, handleThemeChange)
})

// ========== 方法 ==========
/**
 * 处理主题变化事件
 */
const handleThemeChange = () => {
	themeStore.applyNavigationBarTheme()
}
</script>

<style lang="scss">
/* 浅色模式（默认） */
.agreement-container {
	min-height: 100vh;
	background-color: #f5f5f5;
	display: flex;
	flex-direction: column;

	/* 深色模式 */
	&.dark-mode {
		background-color: #121212;

		.nav-bar {
			background-color: #1e1e1e;
		}

		.nav-back-icon {
			color: #ffffff;
		}

		.nav-title {
			color: #ffffff;
		}

		.article-title {
			color: #ffffff;
		}

		.update-time {
			color: #9dabb9;
		}

		.section-title {
			color: #ffffff;
		}

		.section-content {
			color: rgba(255, 255, 255, 0.8);
		}
	}
}

/* 导航栏 */
.nav-bar {
	display: flex;
	align-items: center;
	justify-content: space-between;
	height: 88rpx;
	padding: 0 32rpx;
	padding-top: var(--status-bar-height);
	background-color: #ffffff;
}

.nav-back {
	width: 60rpx;
	height: 60rpx;
	display: flex;
	align-items: center;
	justify-content: center;
}

.nav-back-icon {
	color: #333333;
	font-size: 40rpx;
}

.nav-title {
	color: #333333;
	font-size: 36rpx;
	font-weight: bold;
}

.nav-placeholder {
	width: 60rpx;
}

/* 内容区域 */
.content {
	flex: 1;
	width: 100%;
	padding: 32rpx;
	box-sizing: border-box;
}

.article {
	padding-bottom: 64rpx;
}

.article-title {
	display: block;
	color: #333333;
	font-size: 44rpx;
	font-weight: bold;
	text-align: center;
	margin-bottom: 16rpx;
}

.update-time {
	display: block;
	color: #666666;
	font-size: 26rpx;
	text-align: center;
	margin-bottom: 48rpx;
}

.section {
	margin-bottom: 40rpx;
}

.section-title {
	display: block;
	color: #333333;
	font-size: 32rpx;
	font-weight: bold;
	margin-bottom: 16rpx;
}

.section-content {
	display: block;
	color: rgba(0, 0, 0, 0.7);
	font-size: 28rpx;
	line-height: 1.8;
}
</style>
