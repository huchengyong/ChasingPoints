import Foundation

/// 归一化后的斯诺克赛制。
struct SnookerMatchFormatState: Equatable {
    let format: String
    let targetWins: Int
}

/// 逐条对应小程序 `utils/snooker-match-format.js`。
enum SnookerMatchFormat {
    static let free = "free"
    static let raceTo = "race_to"
    static let maxFrames = 49
    static let maxTargetWins = 25
    static let targetOptions = Array(1...maxTargetWins)

    static func normalize(_ payload: SnookerMatchFormatInput = SnookerMatchFormatInput()) -> SnookerMatchFormatState {
        let format = payload.snookerFormat == raceTo ? raceTo : free
        let target = payload.snookerTargetWins ?? 0
        return SnookerMatchFormatState(
            format: format,
            targetWins: format == raceTo ? clampedTarget(target) : 0
        )
    }

    static func label(_ format: String, targetWins: Int = 0) -> String {
        format == raceTo ? "抢\(clampedTarget(targetWins))局制" : "自由局数"
    }

    static func hint(_ format: String, targetWins: Int = 0) -> String {
        if format != raceTo {
            return "不预设胜局，最多进行\(maxFrames)局"
        }
        let target = clampedTarget(targetWins)
        return "先达到\(target)胜，最多可能进行\(target * 2 - 1)局"
    }

    /// 自由局数必须打完当前局才能结束整场。
    static func canFinishFreeSnookerMatch(
        format: String,
        currentFrameStarted: Bool = false,
        myScore: Int?,
        opponentScore: Int?
    ) -> Bool {
        format == free && !currentFrameStarted && (myScore ?? 0) + (opponentScore ?? 0) >= 1
    }

    static func canStartNextSnookerFrame(
        format: String,
        targetWins: Int? = nil,
        myScore: Int?,
        opponentScore: Int?
    ) -> Bool {
        let myWins = myScore ?? 0
        let opponentWins = opponentScore ?? 0
        if format == free { return myWins + opponentWins < maxFrames }
        let target = clampedTarget(targetWins ?? 0)
        return myWins < target && opponentWins < target
    }

    /// JS 的 `Math.min(Math.max(Number(value) || 1, 1), 25)`：0 和负数都按 1 处理。
    private static func clampedTarget(_ value: Int) -> Int {
        min(max(value == 0 ? 1 : value, 1), maxTargetWins)
    }
}
