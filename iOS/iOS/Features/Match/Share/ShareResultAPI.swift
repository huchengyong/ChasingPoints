import Foundation

/// `GET /api/share/match` 的数据，字段与 backend/chasing_points.api 对齐。
/// APIClient 会自动解包响应的 `data` 层，因此这里直接解码分享数据本身。
struct ShareResultData: Decodable, Equatable {
    let matchId: Int64
    let gameType: Int
    let gameTypeName: String
    let myName: String
    let myAvatar: String
    let myScore: Int
    let opponentName: String
    let opponentAvatar: String
    let opponentScore: Int
    let myVipLevel: Int?
    let opponentVipLevel: Int?
    let myRankLevel: Int?
    let opponentRankLevel: Int?
    let result: String
    let matchTime: String
    let rankChange: Int
    let visibility: String?
    let matchMode: String?
    let summaryHighlights: [MatchSummaryItem]?
    let summaryStats: [MatchSummaryItem]?
}

@MainActor
struct ShareResultAPI {
    let session: SessionStore

    func matchShareData(matchId: Int64) async throws -> ShareResultData {
        let path = APIQuery.path("/api/share/match", ["match_id": matchId])
        return try await session.authorizedRequest(path)
    }
}
