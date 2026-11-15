import Foundation

/// 表现分析的读取入口，经由 `SessionStore.authorizedRequest`。
@MainActor
struct PerformanceAPI {
    let session: SessionStore

    func performance(gameType: Int) async throws -> PerformanceResponse {
        let path = APIQuery.path("/api/member/performance", ["game_type": gameType])
        return try await session.authorizedRequest(path)
    }
}
