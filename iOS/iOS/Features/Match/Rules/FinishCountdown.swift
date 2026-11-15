import Foundation

/// 收到快照时的服务端剩余秒数与本地时刻。
struct FinishCountdownAnchor: Equatable {
    let remaining: Int
    let receivedAt: Date
}

struct FinishConfirmInput: Equatable {
    var finishState: String = ""
    var secondsLeft: Int = 0
    var isRequester: Bool = false
    var opponentName: String = ""
    var myScore: Int = 0
    var opponentScore: Int = 0
    var canWithdrawFinish: Bool = false
    var canConfirmFinish: Bool = false
    var canDisputeFinish: Bool = false
}

struct FinishConfirmView: Equatable {
    let visible: Bool
    /// requester / responder / observer。
    let role: String
    let finishing: Bool
    let secondsLeft: Int
    let title: String
    let desc: String
    let showWithdraw: Bool
    let showConfirm: Bool
    let showDispute: Bool
}

/// 记录发起异步动作时的账号与认证代次，响应返回时据此判断是否仍可回写。
struct FinishSession: Equatable {
    let userId: Int64?
    let authGeneration: Int?
}

/// 逐条对应小程序 `utils/finish-countdown.js`（排位结束确认 30 秒的展示逻辑）。
/// 倒计时以服务端返回的剩余秒数为起点，按本地经过时间递减，不使用手机时钟的绝对值。
enum FinishCountdown {
    /// 与后端 model.FinishRequestTTL 保持一致，仅用于文案。
    static let confirmSeconds = 30
    /// 倒计时归零后等待结束消息的时长，超时则回源读取权威状态。
    static let settleGraceMs = 3000

    static func makeAnchor(remainingSeconds: Int, receivedAt: Date) -> FinishCountdownAnchor {
        FinishCountdownAnchor(remaining: max(0, remainingSeconds), receivedAt: receivedAt)
    }

    /// 按本地经过时间计算剩余秒数（最小为 0）；本地时钟回拨不会增加剩余时间。
    static func secondsLeft(_ anchor: FinishCountdownAnchor?, now: Date) -> Int {
        guard let anchor else { return 0 }
        let elapsedMs = max(0, now.timeIntervalSince(anchor.receivedAt) * 1000)
        return max(0, anchor.remaining - Int(elapsedMs / 1000))
    }

    static func resolveConfirmView(_ input: FinishConfirmInput = FinishConfirmInput()) -> FinishConfirmView {
        let hidden = FinishConfirmView(
            visible: false,
            role: "observer",
            finishing: false,
            secondsLeft: 0,
            title: "",
            desc: "",
            showWithdraw: false,
            showConfirm: false,
            showDispute: false
        )
        if input.finishState != "pending_confirmation" { return hidden }

        let left = max(0, input.secondsLeft)
        let other = input.opponentName.isEmpty ? "对方" : input.opponentName
        let score = "\(input.myScore) : \(input.opponentScore)"
        let role = input.isRequester
            ? "requester"
            : (input.canConfirmFinish || input.canDisputeFinish) ? "responder" : "observer"
        // 本地倒计时归零，或服务端已不再提供任何操作：只等服务端按冻结比分结束。
        let finishing = left <= 0
            || !(input.canWithdrawFinish || input.canConfirmFinish || input.canDisputeFinish)

        if finishing {
            return FinishConfirmView(
                visible: true,
                role: role,
                finishing: true,
                secondsLeft: left,
                title: "正在结束…",
                desc: "确认期已满，正在按 \(score) 结束对局。",
                showWithdraw: false,
                showConfirm: false,
                showDispute: false
            )
        }

        if role == "requester" {
            return FinishConfirmView(
                visible: true,
                role: role,
                finishing: false,
                secondsLeft: left,
                title: "等待\(other)确认",
                desc: "\(confirmSeconds)秒内无人处理，将按 \(score) 自动结束，之后不能再提出异议。",
                showWithdraw: input.canWithdrawFinish,
                showConfirm: false,
                showDispute: false
            )
        }

        if role == "responder" {
            return FinishConfirmView(
                visible: true,
                role: role,
                finishing: false,
                secondsLeft: left,
                title: "\(other)申请结束对局",
                desc: "比分 \(score)。\(left) 秒内未处理，将按此比分自动结束，结束后不能再提出异议。",
                showWithdraw: false,
                showConfirm: input.canConfirmFinish,
                showDispute: input.canDisputeFinish
            )
        }

        return FinishConfirmView(
            visible: true,
            role: role,
            finishing: false,
            secondsLeft: left,
            title: "选手申请结束对局",
            desc: "\(left) 秒内未处理，将按 \(score) 自动结束。",
            showWithdraw: false,
            showConfirm: false,
            showDispute: false
        )
    }

    /// 练习写明无需对方确认；排位写明 30 秒确认、可撤销与超时按当前比分结束。
    static func buildSlideNote(ranked: Bool, opponentName: String = "", myScore: Int = 0, opponentScore: Int = 0) -> String {
        if !ranked { return "滑到终点后松手即可结束，无需对方确认。" }
        let other = opponentName.isEmpty ? "对方" : opponentName
        return "松手后需\(other)在\(confirmSeconds)秒内确认，期间你可撤销；无人处理则按 \(myScore) : \(opponentScore) 自动结束，之后不能再提出异议。"
    }

    /// 排位赛果页的「结束方式」：对方确认时完成人是确认方、发起方是另一名参赛者；超时时完成人是发起方。
    /// 练习赛的手动结束与达标结束无法区分，不显示。
    static func resolveMethodText(_ match: FinishMatchDetailInput?) -> String {
        guard let match, match.matchMode == "ranked" else { return "" }
        let source = match.completionSource
        guard source == "player_confirmed" || source == "finish_timeout" else { return "" }
        let player1 = match.player1Id
        let player2 = match.player2Id
        let completedBy = match.completedByUserId ?? 0
        guard completedBy != 0, completedBy == player1 || completedBy == player2 else { return "" }
        let viewer: Int64 = match.viewerRole == "referee" ? 0 : (match.isPlayer1 ? player1 : player2)
        func nameOf(_ userId: Int64) -> String {
            if viewer != 0 && userId == viewer { return "你" }
            let opponentName = viewer != 0 ? match.opponentName : ""
            return opponentName.isEmpty ? "对方" : opponentName
        }
        if source == "finish_timeout" {
            return "\(nameOf(completedBy))发起 · \(confirmSeconds)秒未处理，自动结束"
        }
        let requester = completedBy == player1 ? player2 : player1
        return "\(nameOf(requester))发起 · \(nameOf(completedBy))确认"
    }

    static func captureSession(userId: Int64?, authGeneration: Int?) -> FinishSession {
        FinishSession(userId: userId, authGeneration: authGeneration)
    }

    static func isSameSession(_ session: FinishSession?, userId: Int64?, authGeneration: Int?) -> Bool {
        guard let session else { return false }
        return session.userId == userId && session.authGeneration == authGeneration
    }
}
