import Foundation

/// 归一化后的台球赛制。
struct PoolMatchFormatState: Equatable {
    let format: String
    let targetWins: Int
}

/// 逐条对应小程序 `utils/pool-match-format.js`：中式八球、美式九球的赛制与结束规则。
enum PoolMatchFormat {
    static let legacy = "legacy"
    static let free = "free"
    static let raceTo = "race_to"
    static let maxRounds = 129
    static let maxTargetWins = 65
    static let targetOptions = Array(1...maxTargetWins)

    static func isPoolMatchFormatGameType(_ gameType: Int?) -> Bool {
        [3, 4].contains(gameType ?? 0)
    }

    static func normalize(_ payload: PoolMatchFormatInput = PoolMatchFormatInput()) -> PoolMatchFormatState {
        let rawFormat = payload.matchFormat ?? ""
        if rawFormat == legacy {
            return PoolMatchFormatState(format: legacy, targetWins: 0)
        }
        let format = rawFormat == raceTo ? raceTo : free
        let target = payload.targetWins ?? 0
        return PoolMatchFormatState(
            format: format,
            targetWins: format == raceTo ? clampedTarget(target) : 0
        )
    }

    static func label(_ format: String, targetWins: Int = 0) -> String {
        if format == legacy { return "传统赛制" }
        return format == raceTo ? "抢\(clampedTarget(targetWins))局制" : "自由局数"
    }

    static func hint(_ format: String, targetWins: Int = 0) -> String {
        if format == legacy { return "按旧版手动结束规则进行" }
        if format != raceTo {
            return "不预设胜局，最多进行\(maxRounds)局，允许平局"
        }
        let target = clampedTarget(targetWins)
        return "先达到\(target)胜，最多可能进行\(target * 2 - 1)局"
    }

    /// 自由局数至少打出一局才能结束。
    static func canFinishFreePoolMatch(format: String, myScore: Int?, opponentScore: Int?) -> Bool {
        format == free && (myScore ?? 0) + (opponentScore ?? 0) >= 1
    }

    static func canContinuePoolMatch(format: String, targetWins: Int? = nil, myScore: Int?, opponentScore: Int?) -> Bool {
        let myWins = myScore ?? 0
        let opponentWins = opponentScore ?? 0
        if format == free { return myWins + opponentWins < maxRounds }
        if format == raceTo {
            let target = clampedTarget(targetWins ?? 0)
            return myWins < target && opponentWins < target
        }
        return true
    }

    /// JS 的 `Math.min(Math.max(Number(value) || 1, 1), 65)`：0 和负数都按 1 处理。
    private static func clampedTarget(_ value: Int) -> Int {
        min(max(value == 0 ? 1 : value, 1), maxTargetWins)
    }
}
