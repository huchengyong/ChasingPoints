import Foundation

/// 赛果页奖励区的展示模型；`state == .hidden` 时整块不渲染。
struct MatchRewardView: Equatable {
    let state: MatchRewardState
    let message: String
    let shouldRetry: Bool

    var visible: Bool {
        if case .hidden = state { return false }
        return true
    }

    var rewards: [MatchRewardItem] {
        if case .ready(let items) = state { return items }
        return []
    }
}

/// 逐条对应小程序 `utils/match-result-page.js` 以及 `matchResult.vue` 的赛果页规则。
enum MatchResultPage {
    static let rewardMaxAttempts = 3
    static let rewardRetryDelayMilliseconds = 800
    static let rewardMessageFallback = "本场荣誉奖励仍在处理中"

    /// 已结算结果优先：整场认输时胜方可能局分较少；旧响应缺少 `result` 时按比分判断。
    static func outcome(result: Int?, myScore: Int, opponentScore: Int) -> MatchOutcome {
        if let settled = result.flatMap(MatchOutcome.init(result:)) { return settled }
        return MatchOutcome(myScore: myScore, opponentScore: opponentScore)
    }

    /// 荣誉墙只支持四个球种，缺失或不支持时回落到中式八球。
    static func honorWallGameType(_ gameType: Int?) -> Int {
        guard let gameType, GameTypeCatalog.isSupported(gameType) else { return 3 }
        return gameType
    }

    static func resolveRewardSummary(
        _ summary: MatchRewardSummary,
        attempt: Int = 1,
        maxAttempts: Int = rewardMaxAttempts
    ) -> MatchRewardView {
        let status = summary.status.lowercased()
        let message = summary.message ?? ""

        if status == "pending" {
            return MatchRewardView(
                state: .pending,
                message: message.isEmpty ? rewardMessageFallback : message,
                shouldRetry: attempt < maxAttempts
            )
        }

        if status == "ready", !summary.list.isEmpty {
            return MatchRewardView(
                state: .ready(summary.list),
                message: "本场首次解锁 \(String(summary.list.count)) 项荣誉",
                shouldRetry: false
            )
        }

        if status == "ready" {
            return MatchRewardView(state: .hidden, message: "", shouldRetry: false)
        }

        return MatchRewardView(
            state: .hidden,
            message: message.isEmpty ? "奖励摘要暂时不可用" : message,
            shouldRetry: false
        )
    }

    /// 赛果页的「结束方式」只对排位赛展示，练习赛无法区分手动与达标结束。
    static func makeFinishInput(_ report: MatchReport) -> FinishMatchDetailInput {
        FinishMatchDetailInput(
            matchMode: report.matchMode,
            completionSource: report.completionSource,
            player1Id: report.player1Id ?? 0,
            player2Id: report.player2Id ?? 0,
            completedByUserId: report.completedByUserId,
            viewerRole: report.viewerRole,
            isPlayer1: report.isPlayer1 ?? false,
            opponentName: report.opponentName
        )
    }

    /// 与服务端时间一致的结束时刻：接口只给开始时间，已完成且有时长时按「开始 + 时长」还原。
    static func endTime(_ report: MatchReport) -> Date? {
        guard report.status == 2, let seconds = report.durationSeconds, seconds > 0 else { return nil }
        guard let start = AppFormat.parseDateTime(report.createdAt) else { return nil }
        return start.addingTimeInterval(TimeInterval(seconds))
    }

    /// 结束时间缺失时回落到开始时间；都不可用时显示“刚刚结束”。
    static func relativeTimeText(_ report: MatchReport) -> String {
        if let end = endTime(report) {
            return AppFormat.relativeTime(isoString(end))
        }
        let relative = AppFormat.relativeTime(report.createdAt)
        return relative.isEmpty ? "刚刚结束" : relative
    }

    static func fullTimeText(_ report: MatchReport) -> String {
        if let end = endTime(report) {
            return AppFormat.dateTime(isoString(end))
        }
        let full = AppFormat.dateTime(report.createdAt)
        return full.isEmpty ? "时间待同步" : full
    }

    static func timeMetaLabel(_ report: MatchReport) -> String {
        endTime(report) == nil ? "开始时间" : "结束时间"
    }

    private static func isoString(_ date: Date) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        formatter.timeZone = TimeZone(identifier: "Asia/Shanghai")
        return formatter.string(from: date)
    }
}
