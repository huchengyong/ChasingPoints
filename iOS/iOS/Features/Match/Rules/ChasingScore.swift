import Foundation

struct ChasingScoreRow: Equatable {
    let key: String
    let label: String
    let left: String
    let right: String
}

struct ChasingScoreEventView: Equatable {
    let seq: Int
    let kind: String
    let kindLabel: String
    /// left / right。
    let side: String
    let sideLabel: String
    let pointsText: String
    let scoreText: String
    let timeText: String
    let desc: String
}

struct ChasingScoreView: Equatable {
    let available: Bool
    let leftLabel: String
    let rightLabel: String
    let leftScore: Int
    let rightScore: Int
    let rows: [ChasingScoreRow]
    let events: [ChasingScoreEventView]
    let eventsTruncated: Bool
    let notice: String
}

/// 逐条对应小程序 `utils/chasing-score.js`：九球追分得分构成的展示模型。
/// 数据固定按选手1 / 选手2 下发，这里只负责视角换边与文案。
enum ChasingScore {
    private static let kindLabels = [
        "normal": "普胜",
        "small_gold": "小金",
        "big_gold": "大金",
        "foul": "犯规",
        "other": "其他",
    ]

    private static let summaryRows: [(key: String, label: String)] = [
        ("score", "总分"),
        ("normal_wins", "普胜"),
        ("small_golds", "小金"),
        ("big_golds", "大金"),
        ("foul_count", "犯规"),
        ("foul_points_given", "犯规送分"),
        ("other_points", "其他得分"),
    ]

    private static let timePattern = try! NSRegularExpression(pattern: #"(\d{2}:\d{2}:\d{2})"#)

    /// `swap`=查看者是选手2；`neutral`=裁判 / 观战者，固定选手1 / 选手2 且不换边。
    static func resolveView(
        _ breakdown: ChasingScoreBreakdown?,
        swap: Bool = false,
        neutral: Bool = false
    ) -> ChasingScoreView? {
        guard let breakdown else { return nil }

        let available = breakdown.available
        let swapSides = swap && !neutral
        let leftLabel = neutral ? "选手1" : "我"
        let rightLabel = neutral ? "选手2" : "对手"
        let leftSide = swapSides ? breakdown.player2 : breakdown.player1
        let rightSide = swapSides ? breakdown.player1 : breakdown.player2

        var rows: [ChasingScoreRow] = []
        if available {
            for row in summaryRows {
                if row.key == "other_points" && breakdown.player1.otherPoints <= 0 && breakdown.player2.otherPoints <= 0 {
                    continue
                }
                rows.append(ChasingScoreRow(
                    key: row.key,
                    label: row.label,
                    left: String(value(of: leftSide, key: row.key)),
                    right: String(value(of: rightSide, key: row.key))
                ))
            }
        } else {
            rows.append(ChasingScoreRow(
                key: "score",
                label: "总分",
                left: String(leftSide.score),
                right: String(rightSide.score)
            ))
        }

        var events: [ChasingScoreEventView] = []
        if available {
            events = breakdown.events.sorted { $0.seq > $1.seq }.map { event in
                let kind = event.kind
                let kindLabel = kindLabels[kind] ?? "其他"
                // 犯规条目的 scorer 是非犯规方即得分方。
                let scorer = event.scorer == 2 ? 2 : 1
                let side = (scorer == 1) != swapSides ? "left" : "right"
                let sideLabel = side == "left" ? leftLabel : rightLabel
                let foulSideLabel = side == "left" ? rightLabel : leftLabel
                // 事件分数与 rows 的左右一致。
                let eventLeftScore = swapSides ? event.player2Score : event.player1Score
                let eventRightScore = swapSides ? event.player1Score : event.player2Score
                return ChasingScoreEventView(
                    seq: event.seq,
                    kind: kind,
                    kindLabel: kindLabel,
                    side: side,
                    sideLabel: sideLabel,
                    // 犯规的分记给对方，写明得分方，避免读成犯规方得分。
                    pointsText: kind == "foul" ? "\(sideLabel) +\(event.points)" : "+\(event.points)",
                    scoreText: "\(eventLeftScore) : \(eventRightScore)",
                    timeText: timeText(event.createdAt),
                    desc: kind == "foul"
                        ? "\(foulSideLabel)犯规"
                        : kind == "other"
                            ? "\(sideLabel) 得分"
                            : "\(sideLabel) \(kindLabel)"
                )
            }
        }

        let eventsTruncated = breakdown.eventsTruncated ?? false
        return ChasingScoreView(
            available: available,
            leftLabel: leftLabel,
            rightLabel: rightLabel,
            leftScore: leftSide.score,
            rightScore: rightSide.score,
            rows: rows,
            events: events,
            eventsTruncated: eventsTruncated,
            notice: !available
                ? "记分明细暂不可用，仅显示总分"
                : eventsTruncated
                    ? "仅显示最近 200 条记分"
                    : ""
        )
    }

    private static func value(of side: ChasingScoreSide, key: String) -> Int {
        switch key {
        case "score": side.score
        case "normal_wins": side.normalWins
        case "small_golds": side.smallGolds
        case "big_golds": side.bigGolds
        case "foul_count": side.foulCount
        case "foul_points_given": side.foulPointsGiven
        case "other_points": side.otherPoints
        default: 0
        }
    }

    /// 服务端时间形如 2026-11-10 12:34:56，没有时间时为空串。
    private static func timeText(_ createdAt: String) -> String {
        let range = NSRange(createdAt.startIndex..., in: createdAt)
        guard let matched = timePattern.firstMatch(in: createdAt, range: range),
              let matchRange = Range(matched.range(at: 1), in: createdAt) else {
            return ""
        }
        return String(createdAt[matchRange])
    }
}
