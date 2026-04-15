<template>
	<view class="privacy-container" :class="{ 'dark-mode': isDarkMode }">
		<!-- 内容区域 -->
		<scroll-view class="content" scroll-y>
			<view class="article">
				<text class="article-title">追分隐私政策</text>
				<text class="update-time">更新日期：2024年12月1日</text>
				
				<view class="section">
					<text class="section-title">引言</text>
					<text class="section-content">
						追分非常重视用户的隐私保护。本隐私政策旨在向您说明我们如何收集、使用、存储和保护您的个人信息。请您仔细阅读本政策，以便更好地了解我们的隐私保护措施。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">一、我们收集的信息</text>
					<text class="section-content">
						1. 账号信息：当您注册账号时，我们会收集您的手机号码、昵称、头像等基本信息。
						
						2. 比赛数据：您在使用应用时记录的比赛信息，包括比分、对手、时间等。
						
						3. 设备信息：为了提供更好的服务，我们可能会收集您的设备型号、操作系统版本等信息。
						
						4. 位置信息：如您授权，我们可能会收集您的位置信息用于附近球友匹配功能。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">二、信息的使用</text>
					<text class="section-content">
						1. 提供服务：我们使用收集的信息为您提供核心功能服务，如比赛记录、数据分析、对手匹配等。
						
						2. 改进服务：通过分析用户行为数据，我们不断优化产品功能和用户体验。
						
						3. 安全保障：我们使用这些信息来识别和防范安全风险，保护您的账号安全。
						
						4. 通知推送：在您授权的情况下，我们会向您发送服务通知和活动信息。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">三、信息的存储与保护</text>
					<text class="section-content">
						1. 我们采用业界标准的安全技术措施来保护您的个人信息。
						
						2. 您的数据存储在安全的服务器上，并进行加密处理。
						
						3. 我们会定期进行安全审计，确保信息安全。
						
						4. 我们只在必要的期限内保留您的个人信息。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">四、信息的共享</text>
					<text class="section-content">
						1. 未经您的同意，我们不会向第三方共享您的个人信息。
						
						2. 以下情况除外：
						   - 获得您的明确授权
						   - 法律法规要求
						   - 保护公共利益
						   - 与关联公司共享（需遵守相同隐私政策）
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">五、您的权利</text>
					<text class="section-content">
						1. 您可以随时查看和修改您的个人信息。
						
						2. 您可以选择关闭某些数据收集功能。
						
						3. 您可以申请删除您的账号和相关数据。
						
						4. 您可以撤回之前给予的授权同意。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">六、未成年人保护</text>
					<text class="section-content">
						我们非常重视对未成年人信息的保护。如果您是未成年人，请在监护人的陪同下阅读本政策，并在监护人同意后使用我们的服务。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">七、政策更新</text>
					<text class="section-content">
						我们可能会不时更新本隐私政策。更新后的政策将在应用内公布，并注明更新日期。重大变更时，我们会向您发送通知。
					</text>
				</view>
				
				<view class="section">
					<text class="section-title">八、联系我们</text>
					<text class="section-content">
						如您对本隐私政策有任何疑问或建议，请通过应用内的反馈渠道与我们联系。我们将在15个工作日内回复您的请求。
					</text>
				</view>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { usePageTheme } from '@/utils/page-theme.js'

// ========== 状态管理 ==========
const { isDarkMode } = usePageTheme()

</script>

<style lang="scss">
/* 浅色模式（默认） */
.privacy-container {
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
