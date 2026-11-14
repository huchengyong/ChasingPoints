import Foundation

/// 「本场用杆」的读取与修改入口。
@MainActor
struct GearAPI {
    let session: SessionStore

    func match(matchId: Int64) async throws -> GearMatchResponse {
        try await session.authorizedRequest(APIQuery.path("/api/gear/match", ["match_id": matchId]))
    }

    func updateMatch(_ payload: GearMatchPayload) async throws -> GearMatchResponse {
        try await session.authorizedPost("/api/gear/match/update", body: payload)
    }

    func mine() async throws -> GearMineResponse {
        try await session.authorizedRequest("/api/gear/mine")
    }
}
