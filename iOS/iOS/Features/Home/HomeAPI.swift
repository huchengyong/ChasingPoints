import Foundation

/// 首页各区块的读取入口：全部访客可读，有凭据时沿用会话刷新规则。
@MainActor
struct HomeAPI {
    static let livePreviewSize = 4

    let session: SessionStore

    func leaderboardSummary(gameType: Int = 3) async throws -> LeaderboardSummary {
        let path = APIQuery.path("/api/public/rank/leaderboard-summary", ["game_type": gameType])
        return try await session.optionalAuthRequest(path)
    }

    func topEventNews() async throws -> EventNewsInfo? {
        let page = try await EventNewsAPI(session: session).list(
            filter: EventNewsFilter(),
            page: 1,
            pageSize: 1
        )
        return page.items.first
    }

    /// 公开大厅进行中对局预览，对应小程序 page_size 4。
    func livePreview() async throws -> [SpectatorMatch] {
        let path = APIQuery.path("/api/public/matches", [
            "scope": "hall",
            "status": 1,
            "page": 1,
            "page_size": Self.livePreviewSize,
        ])
        let response: HomeLiveMatchResponse = try await session.optionalAuthRequest(path)
        return response.list
    }

    func nearbyVenues(_ params: HomeNearbyVenueParams) async throws -> [VenueInfo] {
        let page = try await VenueAPI(session: session).nearby(
            Coordinate(latitude: params.latitude, longitude: params.longitude),
            radius: params.radius,
            limit: params.limit
        )
        return Array(page.items.prefix(params.limit))
    }
}

private struct HomeLiveMatchResponse: Decodable {
    let list: [SpectatorMatch]
}
