import Foundation

/// 赛季接口门面：当前赛季与排行榜公开可读，其余需要登录。
@MainActor
struct SeasonAPI {
    let session: SessionStore

    static let leaderboardPageSize = 20

    func current() async throws -> SeasonCurrentResponse {
        try await session.optionalAuthRequest("/api/season/current")
    }

    func leaderboard(seasonId: Int64, gameType: Int, page: Int) async throws -> SeasonLeaderboardResponse {
        let path = APIQuery.path("/api/season/leaderboard", [
            "season_id": seasonId > 0 ? seasonId : nil,
            "game_type": gameType,
            "page": page,
            "page_size": Self.leaderboardPageSize,
        ])
        return try await session.optionalAuthRequest(path)
    }

    func overview(gameType: Int) async throws -> SeasonOverviewResponse {
        let path = APIQuery.path("/api/season/overview", ["game_type": gameType])
        return try await session.authorizedRequest(path)
    }

    func report(seasonId: Int64, gameType: Int) async throws -> SeasonReportResponse {
        let path = APIQuery.path("/api/season/report", [
            "season_id": seasonId,
            "game_type": gameType,
        ])
        return try await session.authorizedRequest(path)
    }

    func history(gameType: Int, page: Int) async throws -> SeasonHistoryResponse {
        let path = APIQuery.path("/api/season/history", [
            "game_type": gameType,
            "page": page,
            "page_size": SeasonHistoryRules.pageSize,
        ])
        return try await session.authorizedRequest(path)
    }
}
