import Foundation

/// 竞技分析的读取入口：一次概览接口取回页面所有区块。
@MainActor
struct StatsAPI {
    let session: SessionStore

    func overview(gameType: Int) async throws -> StatsOverviewResponse {
        let path = APIQuery.path("/api/stats/overview", [
            "game_type": gameType,
            "trend_limit": 100,
        ])
        return try await session.authorizedRequest(path)
    }
}
