import Foundation

// 字段与 `backend/chasing_points.api` 的赛季模块一致。

struct SeasonInfo: Decodable, Equatable, Identifiable {
    let id: Int64
    let name: String
    let startDate: String
    let endDate: String
    let startAt: String
    let endAtExclusive: String
    let status: Int
    let rankResetRatio: Double
    let qualifyMinMatches: Int
    let qualifyMinOpponents: Int
}

struct SeasonRecordInfo: Decodable, Equatable {
    let seasonId: Int64
    let seasonName: String
    let startRankScore: Int
    let endRankScore: Int
    let peakRankScore: Int
    let matchesPlayed: Int
    let wins: Int
    let finalRank: Int
    let distinctOpponents: Int
    let qualified: Bool
}

struct SeasonLeaderboardItem: Decodable, Equatable, Identifiable {
    let rank: Int
    let userId: Int64
    let nickname: String
    let avatar: String
    let rankScore: Int
    let peakScore: Int
    let matchesPlayed: Int
    let wins: Int
    /// 旧服务端不返回达标标记。
    let qualified: Bool?

    var id: Int64 { userId }
}

struct SeasonCurrentResponse: Decodable, Equatable {
    let seasonState: String
    let season: SeasonInfo?
}

struct SeasonLeaderboardResponse: Decodable, Equatable {
    let total: Int64
    let list: [SeasonLeaderboardItem]
}

struct SeasonOverviewResponse: Decodable, Equatable {
    let seasonState: String
    let season: SeasonInfo?
    let record: SeasonRecordInfo?
    let myRank: Int
    let leaderboard: [SeasonLeaderboardItem]
    let availability: [String: Bool]?
}

struct SeasonReportData: Decodable, Equatable {
    let record: SeasonRecordInfo?
    let rankTrend: [Int]?
    let winRateByType: [String: Int]?
    let topAchievements: [AchievementDef]?
}

struct SeasonReportResponse: Decodable, Equatable {
    let report: SeasonReportData?
}

struct SeasonHistoryItem: Decodable, Equatable {
    let season: SeasonInfo?
    let record: SeasonRecordInfo?
}

struct SeasonHistoryResponse: Decodable, Equatable {
    let total: Int64
    let list: [SeasonHistoryItem]
}
