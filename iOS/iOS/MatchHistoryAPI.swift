import Foundation

/// 对局记录与战报的读取入口。全部经由 `SessionStore.authorizedRequest`，身份校验由调用方的模型完成。
@MainActor
struct MatchHistoryAPI {
    static let pageSize = 20

    let session: SessionStore

    func participantPage(page: Int) async throws -> (items: [MatchListItem], total: Int) {
        let response: PagedResponse<MatchListItem> = try await session.authorizedRequest(
            "/api/match/list?page=\(page)&page_size=\(Self.pageSize)"
        )
        return (response.list, response.total)
    }

    func refereePage(page: Int) async throws -> (items: [RefereeHistoryItem], total: Int) {
        let response: PagedResponse<RefereeHistoryItem> = try await session.authorizedRequest(
            "/api/match/referee/history?page=\(page)&page_size=\(Self.pageSize)"
        )
        return (response.list, response.total)
    }

    func report(matchId: Int64) async throws -> MatchReport {
        let response: MatchDetailResponse = try await session.authorizedRequest(
            "/api/match/detail?match_id=\(matchId)"
        )
        return response.match
    }

    func rewardSummary(matchId: Int64) async throws -> MatchRewardSummary {
        try await session.authorizedRequest("/api/match/reward-summary?match_id=\(matchId)")
    }
}

private struct PagedResponse<Item: Decodable>: Decodable {
    let total: Int
    let list: [Item]
}

private struct MatchDetailResponse: Decodable {
    let match: MatchReport
}
