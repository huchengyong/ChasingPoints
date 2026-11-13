import Foundation

// 交锋记录的窄解码模型，字段与 `backend/chasing_points.api` 的 H2HOverviewResp / H2HHistoryResp 对齐。
// `stats` / `opponent` 是 Go 指针，可分区块失败返回 null；`list` 是 nil slice 时编码成 null。

struct H2HOpponent: Decodable, Equatable {
    let id: Int64
    let name: String
    let avatar: String
}

struct H2HStats: Decodable, Equatable {
    let totalMatches: Int
    let myWins: Int
    let opponentWins: Int
    let winRate: Double
    let avgScoreDiff: Double
    let maxWinStreak: Int
}

struct H2HOverviewResponse: Decodable {
    let success: Bool
    let message: String?
    let hidden: Bool
    let opponent: H2HOpponent?
    let stats: H2HStats?
    let total: Int64
    let list: [MatchListItem]?
    let hasMore: Bool
    let availability: [String: Bool]?
}

struct H2HHistoryResponse: Decodable {
    let success: Bool
    let message: String?
    let hidden: Bool
    let total: Int64
    let list: [MatchListItem]?
}
