import Foundation

// 过往对手的窄解码模型，字段与 `backend/chasing_points.api` 的 `GetOpponentListResp` 对齐。

struct OpponentRecord: Decodable, Identifiable, Equatable {
    let id: Int64
    let name: String
    let avatar: String
    let totalMatches: Int
    let wins: Int
    let losses: Int
    let winRate: Double
    let lastMatchAt: String
}

struct OpponentListResponse: Decodable {
    let success: Bool
    let message: String?
    let hidden: Bool
    let totalOpponents: Int
    let totalMatches: Int
    let totalWins: Int
    let total: Int64
    let list: [OpponentRecord]?
}
