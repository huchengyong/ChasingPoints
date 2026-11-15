import Foundation

/// 下一局 / 结束整场前的判定结果；action 取值与小程序实际执行的动作名一致。
struct SnookerFrameAction: Equatable {
    let action: String
    let winner: Int?
    let message: String?
}

struct SnookerFrameSettleInput: Equatable {
    var currentFrameStarted: Bool = false
    var myFrameScore: Int? = nil
    var opponentFrameScore: Int? = nil
    var continueOnZeroTie: Bool = false
}

/// 逐条对应小程序 `utils/snooker-frame.js`。
enum SnookerFrame {
    static func nextFrameAction(_ input: SnookerFrameSettleInput = SnookerFrameSettleInput()) -> SnookerFrameAction {
        settle(
            input,
            continueOnZeroTie: input.continueOnZeroTie,
            successAction: "start_next_round",
            blockedMessage: "当前局比分相同，无法自动判定胜负，请继续记分或结束本场对局"
        )
    }

    static func finishMatchAction(_ input: SnookerFrameSettleInput = SnookerFrameSettleInput()) -> SnookerFrameAction {
        settle(
            input,
            continueOnZeroTie: true,
            successAction: "finish_match",
            blockedMessage: "当前局比分相同，无法自动判定胜负，请继续记分后再结束本场对局"
        )
    }

    private static func settle(
        _ input: SnookerFrameSettleInput,
        continueOnZeroTie: Bool,
        successAction: String,
        blockedMessage: String
    ) -> SnookerFrameAction {
        if !input.currentFrameStarted {
            return SnookerFrameAction(action: successAction, winner: nil, message: nil)
        }

        let myScore = input.myFrameScore ?? 0
        let opponentScore = input.opponentFrameScore ?? 0

        if myScore == opponentScore {
            // 0 : 0 视为当前局还没开始记分，结束整场时允许直接结束。
            if continueOnZeroTie && myScore == 0 {
                return SnookerFrameAction(action: successAction, winner: nil, message: nil)
            }
            return SnookerFrameAction(action: "blocked", winner: nil, message: blockedMessage)
        }

        return SnookerFrameAction(
            action: "settle_and_\(successAction)",
            winner: myScore > opponentScore ? 1 : 2,
            message: nil
        )
    }
}
