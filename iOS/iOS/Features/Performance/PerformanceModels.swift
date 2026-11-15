import Foundation

// 表现分析的窄解码模型，字段与 `backend/chasing_points.api` 的 GetMemberPerformanceResp 及其嵌套类型对齐。

struct PerformanceRating: Decodable, Equatable {
    let rating: Int
    /// at_least=至少 at_most=至多，缺省为精确值。
    let bound: String?
}

struct PerformanceMonthPreview: Decodable, Equatable {
    let month: String
    let sampleCount: Int
    /// 本月少于 3 场时为 null。
    let rating: PerformanceRating?
}

struct PerformanceSummary: Decodable, Equatable {
    let wins: Int
    let expectedWins: Double
    let winsDiff: Double
    /// 样本少于 5 场时为 null。
    let rating: PerformanceRating?
}

struct PerformanceMatchItem: Decodable, Equatable {
    let matchId: Int64
    let opponentName: String
    let opponentAvatar: String
    let opponentBeforeScore: Int
    let expectedWinRate: Double
    let playedAt: String
}

struct PerformanceMonthItem: Decodable, Equatable {
    let month: String
    let sampleCount: Int
    /// 该月少于 3 场时为 null。
    let rating: PerformanceRating?
    let endRankScore: Int
    /// 读取上限截断了该月数据。
    let incomplete: Bool?
}

struct PerformanceResponse: Decodable, Equatable {
    let success: Bool
    let gameType: Int
    /// true=非有效会员，只返回样本数与本月预览。
    let locked: Bool
    let sampleCount: Int
    /// 仅 locked 时返回。
    let monthPreview: PerformanceMonthPreview?
    /// 以下四项仅有效会员返回。
    let summary: PerformanceSummary?
    let bestWin: PerformanceMatchItem?
    let worstLoss: PerformanceMatchItem?
    /// 最近 6 个自然月。
    let months: [PerformanceMonthItem]?
}
