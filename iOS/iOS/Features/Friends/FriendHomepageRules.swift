import Foundation

// 迁移自 `uniapp/utils/friend-homepage.js` 的纯规则。
struct FriendHomepageSummary: Equatable {
    let hasMatches: Bool
    let heroTitle: String
    let heroDesc: String
    let totalMatchesText: String
    let recordText: String
    let recentMatchText: String
    let emptyTitle: String
    let emptyDesc: String
    let battleSectionTitle: String
    let battleSectionTip: String
    let hiddenTitle: String
    let hiddenDesc: String
}

enum FriendHomepageRules {
    static let battleSectionTitle = "对方战绩"
    static let battleSectionTip = "直接查看 TA 最近的过往对手"
    static let hiddenTitle = "对方已隐藏战绩"
    static let hiddenDesc = "这位好友暂时没有公开自己的战绩列表"

    static func summary(
        totalMatches: Int,
        totalWins: Int,
        lastMatchAt: String,
        now: Date = Date()
    ) -> FriendHomepageSummary {
        let matches = max(0, totalMatches)
        let wins = max(0, totalWins)
        let relativeTime = lastMatchAt.isEmpty ? "" : AppFormat.relativeTime(lastMatchAt, now: now)
        let recentMatchLabel = relativeTime.isEmpty ? "暂无记录" : relativeTime
        let hasMatches = matches > 0

        return FriendHomepageSummary(
            hasMatches: hasMatches,
            heroTitle: hasMatches
                ? "这位球友最近有 \(String(matches)) 场真实对局"
                : "这位球友还没有公开的对局样本",
            heroDesc: hasMatches
                ? "可以直接往下看 TA 的对手战绩列表，PK 报表里再看双方对比。"
                : "等 TA 完成更多真实对局后，这里会直接更新最新战绩列表。",
            totalMatchesText: "真实对局 \(String(matches)) 场",
            recordText: "累计胜场 \(String(wins)) 场",
            recentMatchText: "最近更新 \(recentMatchLabel)",
            emptyTitle: hasMatches ? "" : "暂时还没有可展示的战绩",
            emptyDesc: hasMatches ? "" : "等这位好友完成真实对局后，再回来看看",
            battleSectionTitle: battleSectionTitle,
            battleSectionTip: battleSectionTip,
            hiddenTitle: hiddenTitle,
            hiddenDesc: hiddenDesc
        )
    }
}
