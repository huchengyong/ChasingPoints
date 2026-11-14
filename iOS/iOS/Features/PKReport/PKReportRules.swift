import Foundation

/// PK 报表的纯规则，迁移自 `uniapp/utils/pk-report-view-model.js` 与 `h2h-copywriting.js`。

struct PKReportHero: Equatable {
    let title: String
    let scoreText: String
    let metaText: String
}

struct PKPosterPayload: Equatable {
    let myName: String
    let opponentName: String
    let myWins: Int
    let opponentWins: Int
    let totalMatches: Int
    let winRateLabel: String
    let avgScoreDiffLabel: String
    let maxWinStreak: Int
    let summaryText: String
    let gameTypeLabel: String
}

enum PKReportRules {
    static let partialFailureMessage = "PK 报表部分数据加载失败，请稍后重试"
    static let defaultGameTypeLabel = "真实交锋数据"

    static func hero(stats: H2HStats?, opponentName: String, history: [MatchListItem]) -> PKReportHero {
        let totalMatches = stats?.totalMatches ?? 0
        let myWins = stats?.myWins ?? 0
        let opponentWins = stats?.opponentWins ?? 0
        let name = opponentName.isEmpty ? "对手" : opponentName
        let recent = countResults(history)

        return PKReportHero(
            title: H2HRules.title(
                totalMatches: totalMatches,
                myWins: myWins,
                opponentWins: opponentWins,
                opponentName: name
            ),
            scoreText: "\(String(myWins)) : \(String(opponentWins))",
            metaText: "胜率 \(percent(stats?.winRate ?? 0))% · 近 \(String(min(history.count, 5))) 场 \(String(recent.wins)) 胜 \(String(recent.losses)) 负"
        )
    }

    static func evidence(stats: H2HStats?, history: [MatchListItem]) -> [String] {
        let recent = countResults(history)
        var items: [String] = []
        if !history.isEmpty {
            items.append("近 \(String(min(history.count, 5))) 场 \(String(recent.wins)) 胜 \(String(recent.losses)) 负")
        }

        let streak = stats?.maxWinStreak ?? 0
        if streak > 0 {
            items.append("当前最长连胜 \(String(streak)) 场")
        } else {
            items.append("平均分差 \(signedDiff(stats?.avgScoreDiff ?? 0))")
        }
        return Array(items.prefix(2))
    }

    static func posterPayload(
        hero: PKReportHero,
        evidence: [String],
        stats: H2HStats?,
        userName: String,
        opponentName: String,
        gameTypeLabel: String
    ) -> PKPosterPayload {
        let diff = stats?.avgScoreDiff ?? 0
        let summary = ([hero.title] + evidence).filter { !$0.isEmpty }.joined(separator: " · ")
        return PKPosterPayload(
            myName: userName.isEmpty ? "我" : userName,
            opponentName: opponentName.isEmpty ? "对手" : opponentName,
            myWins: stats?.myWins ?? 0,
            opponentWins: stats?.opponentWins ?? 0,
            totalMatches: stats?.totalMatches ?? 0,
            winRateLabel: "\(percent(stats?.winRate ?? 0))%",
            avgScoreDiffLabel: signedDiff(diff),
            maxWinStreak: stats?.maxWinStreak ?? 0,
            summaryText: summary,
            gameTypeLabel: gameTypeLabel.isEmpty ? defaultGameTypeLabel : gameTypeLabel
        )
    }

    static func latestMatchSummary(_ item: MatchListItem?) -> String? {
        guard let item else { return nil }
        let resultText = item.result == 1 ? "你赢了" : item.result == 2 ? "你输了" : "打平了"
        let dateText = matchDate(item.matchTime)
        let gameType = item.gameTypeName.isEmpty ? "台球" : item.gameTypeName
        return "\(dateText) · \(gameType) · \(resultText) \(String(item.myScore)) : \(String(item.opponentScore))"
    }

    static func matchDate(_ text: String) -> String {
        guard let monthDay = AppFormat.monthDay(text) else { return "未知时间" }
        return "\(monthDay) · \(AppFormat.relativeTime(text))"
    }

    static func shareSummary(hero: PKReportHero, evidence: [String], totalMatches: Int) -> String {
        guard totalMatches > 0 else { return "" }
        let focus = evidence.first ?? hero.metaText
        return "\(hero.title)，历史交锋 \(hero.scoreText)。\(focus)。"
    }

    static func posterEmptyText(totalMatches: Int) -> String {
        totalMatches > 0 ? "海报暂未生成" : "至少完成一场真实交锋后再生成海报"
    }

    // MARK: - 私有

    private static func countResults(_ history: [MatchListItem]) -> (wins: Int, losses: Int) {
        history.prefix(5).reduce(into: (wins: 0, losses: 0)) { result, item in
            switch item.result {
            case 1: result.wins += 1
            case 2: result.losses += 1
            default: break
            }
        }
    }

    /// JS `Math.round` 对正数的行为。
    private static func percent(_ value: Double) -> Int {
        Int(value.rounded())
    }

    private static func signedDiff(_ value: Double) -> String {
        "\(value > 0 ? "+" : "")\(String(format: "%.1f", value))"
    }
}
