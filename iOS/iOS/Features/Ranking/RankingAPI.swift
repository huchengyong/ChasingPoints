import Foundation

/// 排行榜读取入口：公开可读，登录后额外带 `my_ranking`。
@MainActor
struct RankingAPI {
    let session: SessionStore

    static let pageSize = 20

    func leaderboard(gameType: Int, page: Int) async throws -> LeaderboardPage {
        let path = APIQuery.path("/api/public/rank/leaderboard", [
            "game_type": gameType,
            "page": page,
            "page_size": Self.pageSize,
        ])
        return try await session.optionalAuthRequest(path)
    }
}
