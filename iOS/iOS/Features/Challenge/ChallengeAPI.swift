import Foundation

/// 约球接口门面，全部经由 `SessionStore.authorizedRequest` / `authorizedPost`。
@MainActor
struct ChallengeAPI {
    static let historyPageSize = 20

    let session: SessionStore

    func summary() async throws -> ChallengeSummaryResponse {
        try await session.authorizedRequest("/api/challenge/summary")
    }

    func pending() async throws -> ChallengePendingResponse {
        try await session.authorizedRequest("/api/challenge/pending")
    }

    func detail(id: Int64) async throws -> ChallengeDetailResponse {
        try await session.authorizedRequest(APIQuery.path("/api/challenge/detail", ["id": id]))
    }

    func history(beforeId: Int64 = 0) async throws -> ChallengeHistoryResponse {
        var query: [String: CustomStringConvertible?] = ["page_size": Self.historyPageSize]
        if beforeId > 0 { query["before_id"] = beforeId }
        return try await session.authorizedRequest(APIQuery.path("/api/challenge/history", query))
    }

    func send(_ body: SendChallengeBody) async throws -> SendChallengeResponse {
        try await session.authorizedPost("/api/challenge/send", body: body)
    }

    func accept(challengeId: Int64, confirmCloseOwnPending: Bool = false) async throws -> AcceptChallengeResponse {
        try await session.authorizedPost(
            "/api/challenge/accept",
            body: AcceptChallengeBody(
                challengeId: challengeId,
                confirmCloseOwnPending: confirmCloseOwnPending ? true : nil
            )
        )
    }

    func reject(challengeId: Int64) async throws -> ChallengeActionResponse {
        try await session.authorizedPost("/api/challenge/reject", body: ChallengeIdBody(challengeId: challengeId))
    }

    func cancel(challengeId: Int64) async throws -> ChallengeActionResponse {
        try await session.authorizedPost("/api/challenge/cancel", body: ChallengeIdBody(challengeId: challengeId))
    }

    func abandon(challengeId: Int64) async throws -> ChallengeActionResponse {
        try await session.authorizedPost("/api/challenge/abandon", body: ChallengeIdBody(challengeId: challengeId))
    }

    func leave(challengeId: Int64) async throws -> ChallengeActionResponse {
        try await session.authorizedPost("/api/challenge/leave", body: ChallengeIdBody(challengeId: challengeId))
    }

    func enter(challengeId: Int64, confirmEarly: Bool = false) async throws -> EnterChallengeResponse {
        try await session.authorizedPost(
            "/api/challenge/enter",
            body: EnterChallengeBody(
                challengeId: challengeId,
                confirmEarly: confirmEarly ? true : nil
            )
        )
    }
}
