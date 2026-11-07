/**
 * 排位结束确认（30秒）展示逻辑。
 * 倒计时以服务端返回的剩余秒数为起点，按本地经过时间递减，不使用手机时钟的绝对值；
 * 能否撤回、确认、异议一律以服务端字段为准，这里只决定展示与文案。
 */

// 与后端 model.FinishRequestTTL 保持一致，仅用于文案。
export const FINISH_CONFIRM_SECONDS = 30
// 倒计时归零后等待结束消息的时长，超时则回源读取权威状态。
export const FINISH_SETTLE_GRACE_MS = 3000

const toWholeSeconds = (value) => {
	const seconds = Number(value)
	return Number.isFinite(seconds) && seconds > 0 ? Math.floor(seconds) : 0
}

/** 记录收到快照时的服务端剩余秒数与本地时刻。 */
export const createFinishCountdownAnchor = (remainingSeconds, receivedAtMs) => ({
	remaining: toWholeSeconds(remainingSeconds),
	receivedAtMs: Number(receivedAtMs) || 0
})

/** 按本地经过时间计算剩余秒数（最小为 0）；本地时钟回拨不会增加剩余时间。 */
export const finishCountdownSecondsLeft = (anchor, nowMs) => {
	if (!anchor) return 0
	const elapsedMs = Math.max(0, (Number(nowMs) || 0) - anchor.receivedAtMs)
	return Math.max(0, anchor.remaining - Math.floor(elapsedMs / 1000))
}

/**
 * 解析进行中页的结束确认视图。
 * @param {{
 *   finishState: string, secondsLeft: number, isRequester: boolean, opponentName: string,
 *   myScore: number, opponentScore: number,
 *   canWithdrawFinish: boolean, canConfirmFinish: boolean, canDisputeFinish: boolean
 * }} input
 */
export const resolveFinishConfirmView = ({
	finishState,
	secondsLeft,
	isRequester,
	opponentName,
	myScore,
	opponentScore,
	canWithdrawFinish,
	canConfirmFinish,
	canDisputeFinish
} = {}) => {
	const view = {
		visible: false,
		role: 'observer',
		finishing: false,
		secondsLeft: 0,
		title: '',
		desc: '',
		showWithdraw: false,
		showConfirm: false,
		showDispute: false
	}
	if (finishState !== 'pending_confirmation') return view
	const left = toWholeSeconds(secondsLeft)
	const other = opponentName || '对方'
	const score = `${Number(myScore) || 0} : ${Number(opponentScore) || 0}`
	view.visible = true
	view.secondsLeft = left
	view.role = isRequester ? 'requester' : (canConfirmFinish || canDisputeFinish) ? 'responder' : 'observer'
	// 本地倒计时归零，或服务端已不再提供任何操作：只等服务端按冻结比分结束。
	view.finishing = left <= 0 || !(canWithdrawFinish || canConfirmFinish || canDisputeFinish)
	if (view.finishing) {
		view.title = '正在结束…'
		view.desc = `确认期已满，正在按 ${score} 结束对局。`
		return view
	}
	if (view.role === 'requester') {
		view.title = `等待${other}确认`
		view.desc = `${FINISH_CONFIRM_SECONDS}秒内无人处理，将按 ${score} 自动结束，之后不能再提出异议。`
		view.showWithdraw = canWithdrawFinish === true
	} else if (view.role === 'responder') {
		view.title = `${other}申请结束对局`
		view.desc = `比分 ${score}。${left} 秒内未处理，将按此比分自动结束，结束后不能再提出异议。`
		view.showConfirm = canConfirmFinish === true
		view.showDispute = canDisputeFinish === true
	} else {
		view.title = '选手申请结束对局'
		view.desc = `${left} 秒内未处理，将按 ${score} 自动结束。`
	}
	return view
}

/**
 * 结束滑块旁的说明：练习写明无需对方确认；排位写明30秒确认、可撤销与超时按当前比分结束。
 * @param {{ ranked: boolean, opponentName: string, myScore: number, opponentScore: number }} input
 */
export const buildFinishSlideNote = ({ ranked, opponentName, myScore, opponentScore } = {}) => {
	if (!ranked) return '滑到终点后松手即可结束，无需对方确认。'
	return `松手后需${opponentName || '对方'}在${FINISH_CONFIRM_SECONDS}秒内确认，期间你可撤销；无人处理则按 ${Number(myScore) || 0} : ${Number(opponentScore) || 0} 自动结束，之后不能再提出异议。`
}

/**
 * 排位赛果页的“结束方式”：对方确认时完成人是确认方、发起方是另一名参赛者；超时时完成人是发起方。
 * 练习赛的手动结束与达标结束无法区分，不显示。
 * @param {Object} match getMatchDetail 返回的 match
 */
export const resolveFinishMethodText = (match) => {
	if (!match || match.match_mode !== 'ranked') return ''
	const source = match.completion_source
	if (source !== 'player_confirmed' && source !== 'finish_timeout') return ''
	const player1 = Number(match.player1_id) || 0
	const player2 = Number(match.player2_id) || 0
	const completedBy = Number(match.completed_by_user_id) || 0
	if (!completedBy || (completedBy !== player1 && completedBy !== player2)) return ''
	const viewer = match.viewer_role === 'referee' ? 0 : (match.is_player1 ? player1 : player2)
	const nameOf = (userId) => {
		if (viewer && userId === viewer) return '你'
		return (viewer ? match.opponent_name : '') || '对方'
	}
	if (source === 'finish_timeout') {
		return `${nameOf(completedBy)}发起 · ${FINISH_CONFIRM_SECONDS}秒未处理，自动结束`
	}
	const requester = completedBy === player1 ? player2 : player1
	return `${nameOf(requester)}发起 · ${nameOf(completedBy)}确认`
}

/** 记录发起异步动作时的账号与认证代次，响应返回时据此判断是否仍可回写。 */
export const captureFinishSession = (userStore) => ({
	userId: userStore?.userId ?? null,
	authGeneration: userStore?.authGeneration ?? null
})

export const isSameFinishSession = (session, userStore) => Boolean(session) &&
	session.userId === (userStore?.userId ?? null) &&
	session.authGeneration === (userStore?.authGeneration ?? null)
