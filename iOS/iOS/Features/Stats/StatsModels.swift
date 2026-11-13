import Foundation

// 竞技分析的窄解码模型，字段与 `backend/chasing_points.api` 的 GetStatsOverviewResp 及其嵌套类型对齐。
// 概览接口按区块返回 availability：某区块读取失败时对应 slice 是 nil（JSON null），其余区块仍然成功，
// 所以列表都按可选解码，由页面模型决定沿用旧数据还是归一为空数组。

struct StatsGameTypeStats: Decodable, Equatable {
    let gameType: Int
    let gameTypeName: String
    let totalMatches: Int
    let wins: Int
    let losses: Int
    let winRate: Double
    let highestScore: Int
}

struct StatsTrendPoint: Decodable, Equatable {
    let matchId: Int64
    let date: String
    let winRate: Double
    /// 段位分趋势才有；`omitempty` 时缺省。
    let rankScore: Int?
    let result: Int
}

struct StatsHighScoreRecord: Decodable, Equatable {
    let matchId: Int64
    let score: Int
    let gameType: Int
    let gameTypeName: String
    let opponentName: String
    let date: String
}

struct StatsDurationStats: Decodable, Equatable {
    let averageSeconds: Int
    let fastestSeconds: Int
    let longestSeconds: Int
    let totalMatches: Int
}

struct StatsOpponentStrengthItem: Decodable, Equatable {
    let rankRange: String
    let matches: Int
    let wins: Int
    let winRate: Double
}

struct StatsOverviewResponse: Decodable, Equatable {
    let success: Bool
    let byGameType: [StatsGameTypeStats]?
    let recentTrend: [StatsTrendPoint]?
    let rankScoreTrend: [StatsTrendPoint]?
    let singleHighScores: [StatsHighScoreRecord]?
    let duration: StatsDurationStats?
    let opponentStrength: [StatsOpponentStrengthItem]?
    let availability: [String: Bool]?

    init(
        success: Bool,
        byGameType: [StatsGameTypeStats]?,
        recentTrend: [StatsTrendPoint]?,
        rankScoreTrend: [StatsTrendPoint]?,
        singleHighScores: [StatsHighScoreRecord]?,
        duration: StatsDurationStats?,
        opponentStrength: [StatsOpponentStrengthItem]?,
        availability: [String: Bool]?
    ) {
        self.success = success
        self.byGameType = byGameType
        self.recentTrend = recentTrend
        self.rankScoreTrend = rankScoreTrend
        self.singleHighScores = singleHighScores
        self.duration = duration
        self.opponentStrength = opponentStrength
        self.availability = availability
    }

    /// 与小程序一致：`availability[scope] === false` 的分区不覆盖已有数据。
    func merged(with previous: StatsOverviewResponse?) -> StatsOverviewResponse {
        func isAvailable(_ scope: String) -> Bool {
            availability?[scope] != false
        }
        return StatsOverviewResponse(
            success: success,
            byGameType: isAvailable("by_game_type") ? byGameType : previous?.byGameType,
            recentTrend: isAvailable("recent_trend") ? recentTrend : previous?.recentTrend,
            rankScoreTrend: isAvailable("rank_score_trend") ? rankScoreTrend : previous?.rankScoreTrend,
            singleHighScores: isAvailable("single_high_scores") ? singleHighScores : previous?.singleHighScores,
            duration: isAvailable("duration") ? duration : previous?.duration,
            opponentStrength: isAvailable("opponent_strength") ? opponentStrength : previous?.opponentStrength,
            availability: availability
        )
    }
}
