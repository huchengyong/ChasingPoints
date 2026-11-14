import Foundation

// 迁移自 `uniapp/utils/season-history.js`：历届赛季列表与分页停止条件。

struct SeasonHistoryRow: Equatable, Identifiable {
    let id: Int64
    let name: String
    let dateRange: String
    let hasRecord: Bool
    let rankText: String
    let matches: Int
    let wins: Int
    let winRate: Int
    /// 有本人记录时才给出报告目标；无记录不可进入报告。
    let reportURL: String
}

enum SeasonHistoryRules {
    static let pageSize = 10
    static let reportPath = "/subPages/season/report"

    static func reportURL(seasonId: Int64, gameType: Int) -> String {
        "\(reportPath)?id=\(String(seasonId))&game_type=\(String(gameType))"
    }

    static func row(_ item: SeasonHistoryItem, gameType: Int) -> SeasonHistoryRow? {
        guard let season = item.season, season.id > 0 else { return nil }
        let record = item.record
        let matches = max(0, record?.matchesPlayed ?? 0)
        let wins = max(0, record?.wins ?? 0)
        let finalRank = max(0, record?.finalRank ?? 0)
        let hasThreshold = season.qualifyMinMatches > 0

        let rankText: String
        if finalRank > 0 {
            rankText = "第 \(String(finalRank)) 名"
        } else if hasThreshold && record?.qualified == false {
            rankText = "未达标"
        } else {
            rankText = "未排名"
        }

        let name = season.name.isEmpty ? "赛季 \(String(season.id))" : season.name
        let dateRange = !season.startDate.isEmpty && !season.endDate.isEmpty
            ? "\(season.startDate) ~ \(season.endDate)"
            : ""

        return SeasonHistoryRow(
            id: season.id,
            name: name,
            dateRange: dateRange,
            hasRecord: record != nil,
            rankText: rankText,
            matches: matches,
            wins: wins,
            winRate: matches > 0 ? Int((Double(wins) / Double(matches) * 100).rounded()) : 0,
            reportURL: record == nil ? "" : reportURL(seasonId: season.id, gameType: gameType)
        )
    }

    /// 空页立即停止分页，避免数据收缩时无限加载。
    static func hasMore(loaded: Int = 0, total: Int64 = 0, lastPageCount: Int = 0) -> Bool {
        lastPageCount > 0 && Int64(loaded) < total
    }
}
