import Foundation

/// 单场复盘的读取入口，经由 `SessionStore.authorizedRequest`。
@MainActor
struct MatchReplayAPI {
    let session: SessionStore

    func replay(matchId: Int64) async throws -> MatchReplayResponse {
        try await session.authorizedRequest(
            APIQuery.path("/api/member/match-replay", ["match_id": matchId])
        )
    }
}
