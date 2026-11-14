import Foundation

/// 单场特殊战绩。
struct AchievementRecord: Equatable, Identifiable {
    let key: String
    let label: String
    let count: Int

    var id: String { key }
}

/// 战报里的排位权益说明，迁移小程序 `utils/member-ranking-rights.js`。
struct RankingRightsSummary: Equatable {
    let badgeText: String
    let title: String
    let description: String
    /// slate / blue / emerald / gold。
    let tone: String
    let countedInRanking: Bool
    let capped: Bool
    let recordedAchievementsText: String
}

enum MemberRankingRights {
    private static let definitions: [(key: String, label: String)] = [
        ("break_clear", "炸清"),
        ("continue_clear", "接清"),
        ("golden_break", "小金"),
        ("nine_on_break", "大金"),
        ("break_50", "单杆50+"),
        ("break_100", "单杆100+"),
        ("break_147", "147"),
    ]

    static func achievementRecords(_ achievements: [String: Int]?) -> [AchievementRecord] {
        definitions.compactMap { item in
            let count = max(0, achievements?[item.key] ?? 0)
            return count > 0 ? AchievementRecord(key: item.key, label: item.label, count: count) : nil
        }
    }

    static func summary(match: MatchReport) -> RankingRightsSummary {
        let records = achievementRecords(match.achievements)
        let recordText = formatRecords(records)

        if match.matchMode == "practice" {
            return RankingRightsSummary(
                badgeText: "练习赛",
                title: "练习赛不计入排位",
                description: recordText.isEmpty
                    ? "练习赛的胜负和战绩不影响排位分。"
                    : "本场记录了 \(recordText)，高光会保留在战报和成就里。",
                tone: "slate",
                countedInRanking: false,
                capped: false,
                recordedAchievementsText: recordText
            )
        }

        if match.rankRulesVersion == 2 {
            return recordText.isEmpty
                ? RankingRightsSummary(
                    badgeText: "只看胜负",
                    title: "本场排位只看胜负与对手实力",
                    description: "排位分由赛前胜率、赛制长度和胜负决定，会员与普通用户规则相同。",
                    tone: "slate",
                    countedInRanking: false,
                    capped: false,
                    recordedAchievementsText: ""
                )
                : RankingRightsSummary(
                    badgeText: "高光不计分",
                    title: "本场高光已记录",
                    description: "本场记录了 \(recordText)，高光会保留在战报和成就里；排位分只看胜负与对手实力，会员与普通用户规则相同。",
                    tone: "blue",
                    countedInRanking: false,
                    capped: false,
                    recordedAchievementsText: recordText
                )
        }

        let details = match.myRankDetails ?? []
        let countedScore = details
            .filter(isAchievementRankDetail)
            .reduce(0) { $0 + max(0, $1.value) }
        let capReduction = details
            .filter(isDailyCapDetail)
            .reduce(0) { $0 + abs(min(0, $1.value)) }

        guard !records.isEmpty else {
            return RankingRightsSummary(
                badgeText: "无高光排位分",
                title: "本场没有可计入排位的特殊战绩",
                description: "本场排位只按基础分结算。",
                tone: "slate",
                countedInRanking: false,
                capped: false,
                recordedAchievementsText: ""
            )
        }

        if countedScore > 0 {
            let capped = capReduction > 0
            return RankingRightsSummary(
                badgeText: capped ? "已计入并触发封顶" : "高光已计入排位",
                title: capped ? "本场高光已部分计入排位" : "本场高光已计入排位",
                description: capped
                    ? "本场记录了 \(recordText)，其中 \(String(countedScore)) 分特殊战绩已计入排位，另有 \(String(capReduction)) 分受到VIP特殊战绩单日上限影响。"
                    : "本场记录了 \(recordText)，特殊战绩分已计入排位，具体结算以排位变化明细为准。",
                tone: capped ? "gold" : "emerald",
                countedInRanking: true,
                capped: capped,
                recordedAchievementsText: recordText
            )
        }

        return RankingRightsSummary(
            badgeText: "高光仅记录",
            title: "本场高光未计入排位",
            description: "本场记录了 \(recordText)，但这场排位未计入高光分。普通用户高光会保留记录，VIP用户才会按等级倍率计入。",
            tone: "blue",
            countedInRanking: false,
            capped: false,
            recordedAchievementsText: recordText
        )
    }

    private static func isAchievementRankDetail(_ detail: MatchRankDetail) -> Bool {
        detail.label.contains("VIP特殊战绩分")
            || detail.label.contains("成就奖励")
            || detail.label.contains("特殊战绩")
    }

    private static func isDailyCapDetail(_ detail: MatchRankDetail) -> Bool {
        detail.label.contains("每日封顶")
    }

    private static func formatRecords(_ records: [AchievementRecord]) -> String {
        records.map { "\($0.label)x\(String($0.count))" }.joined(separator: "、")
    }
}
