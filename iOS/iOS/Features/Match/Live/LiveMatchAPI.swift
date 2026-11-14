import Foundation

private struct UpdateMatchFormatRequest: Encodable {
    let matchId: Int64
    let matchFormat: String
    let targetWins: Int
    let baseRevision: Int64
}

/// 实时计分页的读写入口；写请求统一由模型附加 `client_action_id` 与 `base_revision`。
@MainActor
struct LiveMatchAPI {
    let session: SessionStore

    func detail(matchId: Int64) async throws -> LiveMatchDetailResponse {
        try await session.authorizedRequest(APIQuery.path("/api/match/detail", ["match_id": matchId]))
    }

    func current() async throws -> LiveCurrentMatchResponse {
        try await session.authorizedRequest("/api/match/current")
    }

    func publicDetail(matchId: Int64) async throws -> LiveMatchDetailResponse {
        try await session.optionalAuthRequest(APIQuery.path("/api/public/match/detail", ["match_id": matchId]))
    }

    func refereeQRCode(matchId: Int64) async throws -> LiveRefereeQRCodeResponse {
        try await session.authorizedRequest(APIQuery.path("/api/match/referee/qrcode", ["match_id": matchId]))
    }

    /// 所有带幂等键的写操作共用；路径与负载由页面模型给出。
    func perform(_ path: String, payload: [String: MatchActionValue]) async throws -> LiveMatchWriteResponse {
        try await session.authorizedPost(path, body: payload)
    }

    func endRound(_ payload: [String: MatchActionValue]) async throws -> LiveMatchWriteResponse {
        try await perform("/api/match/round/end", payload: payload)
    }

    /// 赛制接口没有幂等键，只带修订号；与其他写操作区分开。
    func updateMatchFormat(matchId: Int64, format: String, targetWins: Int, baseRevision: Int64) async throws -> LiveMatchWriteResponse {
        try await session.authorizedPost(
            "/api/match/format",
            body: UpdateMatchFormatRequest(
                matchId: matchId,
                matchFormat: format,
                targetWins: targetWins,
                baseRevision: baseRevision
            )
        )
    }
}
