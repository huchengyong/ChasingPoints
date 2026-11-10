<template>
	<view v-if="view" class="match-chasing-score">
		<text class="match-chasing-score__title">得分构成</text>

		<view class="match-chasing-score__head">
			<text class="match-chasing-score__head-label">{{ view.leftLabel }}</text>
			<text class="match-chasing-score__head-label match-chasing-score__head-label--center">项目</text>
			<text class="match-chasing-score__head-label match-chasing-score__head-label--right">{{ view.rightLabel }}</text>
		</view>

		<view
			v-for="row in view.rows"
			:key="row.key"
			:class="['match-chasing-score__row', { 'is-total': row.key === 'score' }]"
			:aria-label="`${row.label}：${view.leftLabel} ${row.left}，${view.rightLabel} ${row.right}`"
		>
			<text class="match-chasing-score__value">{{ row.left }}</text>
			<text class="match-chasing-score__label">{{ row.label }}</text>
			<text class="match-chasing-score__value match-chasing-score__value--right">{{ row.right }}</text>
		</view>

		<view v-if="view.notice" class="match-chasing-score__notice">
			<text>{{ view.notice }}</text>
		</view>

		<template v-if="showEvents">
			<text class="match-chasing-score__events-title">得分明细</text>

			<view v-if="view.events.length" class="match-chasing-score__events">
				<view
					v-for="event in view.events"
					:key="event.seq"
					:class="['match-chasing-score__event', `is-${event.side}`, { 'is-foul': event.kind === 'foul' }]"
				>
					<view class="match-chasing-score__event-main">
						<text class="match-chasing-score__event-desc">{{ event.desc }}</text>
						<text class="match-chasing-score__event-points">{{ event.pointsText }}</text>
					</view>
					<view class="match-chasing-score__event-meta">
						<text class="match-chasing-score__event-score">{{ event.scoreText }}</text>
						<text v-if="event.timeText" class="match-chasing-score__event-time">{{ event.timeText }}</text>
					</view>
				</view>
			</view>
			<text v-else-if="view.available" class="match-chasing-score__empty">暂无记分</text>
		</template>
	</view>
</template>

<script setup>
// 九球追分双方对照表与得分明细。view 来自 utils/chasing-score.js，已按查看者视角换边。
defineProps({
	view: { type: Object, default: null },
	showEvents: { type: Boolean, default: true }
})
</script>

<style lang="scss" scoped>
.match-chasing-score {
	margin: 20rpx 0;
	padding: 28rpx;
	border-radius: var(--ui-radius-lg, 24rpx);
	background: var(--ui-surface-card, #FFFFFF);
	border: 1rpx solid var(--ui-border-default, #E9E2CF);
	box-sizing: border-box;

	.match-chasing-score__title {
		display: block;
		font-size: 30rpx;
		font-weight: 700;
		color: var(--ui-text-primary, #231C0B);
	}
}

.match-chasing-score__head {
	display: flex;
	align-items: center;
	margin-top: 20rpx;
	font-size: 22rpx;
	color: var(--ui-text-muted, #9A8C67);

	.match-chasing-score__head-label {
		flex: 0 0 160rpx;
	}

	.match-chasing-score__head-label--center {
		flex: 1;
		text-align: center;
	}

	.match-chasing-score__head-label--right {
		text-align: right;
	}
}

.match-chasing-score__row {
	display: flex;
	align-items: center;
	margin-top: 10rpx;
	padding: 10rpx 16rpx;
	border-radius: var(--ui-radius-sm, 12rpx);

	.match-chasing-score__value {
		flex: 0 0 160rpx;
		font-size: 30rpx;
		font-weight: 600;
		color: var(--ui-text-primary, #231C0B);
	}

	.match-chasing-score__value--right {
		text-align: right;
	}

	.match-chasing-score__label {
		flex: 1;
		text-align: center;
		font-size: 24rpx;
		color: var(--ui-text-secondary, #6E6242);
	}

	&.is-total {
		margin-top: 16rpx;
		background: var(--ui-brand-tint, rgba(224, 174, 18, 0.14));

		.match-chasing-score__value {
			font-size: 36rpx;
			font-weight: 800;
			color: var(--ui-brand-strong, #C69200);
		}

		.match-chasing-score__label {
			font-weight: 600;
			color: var(--ui-text-primary, #231C0B);
		}
	}
}

.match-chasing-score__notice {
	margin-top: 16rpx;
	padding: 12rpx 16rpx;
	border-radius: var(--ui-radius-sm, 12rpx);
	background: var(--ui-brand-tint, rgba(224, 174, 18, 0.14));
	font-size: 22rpx;
	line-height: 1.5;
	color: var(--ui-text-secondary, #6E6242);
}

.match-chasing-score__events-title {
	display: block;
	margin-top: 28rpx;
	font-size: 28rpx;
	font-weight: 600;
	color: var(--ui-text-primary, #231C0B);
}

.match-chasing-score__events {
	display: flex;
	flex-direction: column;
	gap: 12rpx;
	margin-top: 12rpx;
}

.match-chasing-score__event {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 16rpx;
	padding: 16rpx 18rpx;
	border-radius: var(--ui-radius-sm, 12rpx);
	border-left: 6rpx solid var(--ui-border-default, #E9E2CF);
	background: var(--ui-surface-subtle, #FAF8F2);
	box-sizing: border-box;

	&.is-left {
		border-left-color: var(--ui-brand-strong, #C69200);

		.match-chasing-score__event-points {
			color: var(--ui-brand-strong, #C69200);
		}
	}

	&.is-right {
		border-left-color: var(--ui-text-muted, #9A8C67);

		.match-chasing-score__event-points {
			color: var(--ui-text-secondary, #6E6242);
		}
	}

	&.is-foul {
		border-left-color: var(--ui-warning, #F97316);
		background: rgba(249, 115, 22, 0.1);
	}
}

.match-chasing-score__event-main {
	flex: 1;
	min-width: 0;
	display: flex;
	align-items: baseline;
	gap: 12rpx;
}

.match-chasing-score__event-desc {
	font-size: 26rpx;
	font-weight: 600;
	color: var(--ui-text-primary, #231C0B);
	overflow-wrap: break-word;
}

.match-chasing-score__event-points {
	flex-shrink: 0;
	font-size: 28rpx;
	font-weight: 700;
}

.match-chasing-score__event-meta {
	flex-shrink: 0;
	display: flex;
	flex-direction: column;
	align-items: flex-end;
	gap: 2rpx;
}

.match-chasing-score__event-score {
	font-size: 26rpx;
	font-weight: 700;
	color: var(--ui-text-primary, #231C0B);
}

.match-chasing-score__event-time {
	font-size: 20rpx;
	color: var(--ui-text-muted, #9A8C67);
}

.match-chasing-score__empty {
	display: block;
	margin-top: 12rpx;
	font-size: 24rpx;
	color: var(--ui-text-muted, #9A8C67);
}
</style>
