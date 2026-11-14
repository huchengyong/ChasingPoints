import Foundation

// 迁移自 `uniapp/utils/season-qualification.js`：赛季达标进度、本季净变化与榜单合并。

struct SeasonQualification: Equatable {
    let hasThreshold: Bool
    let minMatches: Int
    let minOpponents: Int
    let qualified: Bool
    let progressText: String
    let thresholdText: String
}

struct SeasonLeaderboardRank: Equatable {
    let rank: Int
    let text: String
    let unqualified: Bool
}

/// 榜单展示行：服务端名次与原始条目组合，未达标记录不给名次。
struct SeasonLeaderboardRow: Equatable, Identifiable {
    let item: SeasonLeaderboardItem
    let rank: SeasonLeaderboardRank

    var id: Int64 { item.userId }
}

enum SeasonQualificationRules {
    /// 门槛来自服务端赛季信息，旧赛季门槛为 0（全员参与排名）。
    static func qualification(season: SeasonInfo?, record: SeasonRecordInfo?) -> SeasonQualification {
        let minMatches = max(0, season?.qualifyMinMatches ?? 0)
        let minOpponents = max(0, season?.qualifyMinOpponents ?? 0)
        let hasThreshold = minMatches > 0
        let matches = max(0, record?.matchesPlayed ?? 0)
        let opponents = max(0, record?.distinctOpponents ?? 0)

        return SeasonQualification(
            hasThreshold: hasThreshold,
            minMatches: minMatches,
            minOpponents: minOpponents,
            qualified: !hasThreshold || record?.qualified == true,
            progressText: hasThreshold
                ? "场次 \(String(min(matches, minMatches)))/\(String(minMatches)) · 不同对手 \(String(min(opponents, minOpponents)))/\(String(minOpponents))"
                : "",
            thresholdText: hasThreshold
                ? "本季打满 \(String(minMatches)) 场、遇到 \(String(minOpponents)) 位不同对手即参与赛季排名"
                : ""
        )
    }

    static func netChange(_ record: SeasonRecordInfo?) -> String {
        guard let record else { return "" }
        let delta = record.endRankScore - record.startRankScore
        return delta > 0 ? "+\(String(delta))" : String(delta)
    }

    /// 分页之间名次可能变化，同一用户只保留首次出现，避免重复行。
    static func mergeLeaderboard(
        _ current: [SeasonLeaderboardItem],
        _ incoming: [SeasonLeaderboardItem]
    ) -> [SeasonLeaderboardItem] {
        var seen = Set(current.map(\.userId))
        var merged = current
        for item in incoming where !seen.contains(item.userId) {
            seen.insert(item.userId)
            merged.append(item)
        }
        return merged
    }

    /// 服务端对未达标记录返回 rank = 0，不显示名次。
    static func leaderboardRank(_ item: SeasonLeaderboardItem) -> SeasonLeaderboardRank {
        let rank = max(0, item.rank)
        return SeasonLeaderboardRank(
            rank: rank,
            text: rank > 0 ? String(rank) : "—",
            unqualified: item.qualified == false
        )
    }
}
