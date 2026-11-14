import Foundation

// 扫码开局与执裁共用的一组接口；字段与 backend/chasing_points.api 对齐，统一 convertFromSnakeCase 解码。

struct PreviewMatchInviteRequest: Encodable, Equatable {
    let inviteToken: String
}

struct ResolveMatchInviteCodeRequest: Encodable, Equatable {
    let inviteCode: String
}

/// `StartMatchResp` 的安全子集：只保留开局结果分流实际读取的字段。
struct StartMatchResponse: Decodable, Equatable {
    let success: Bool
    let action: String?
    let blockReason: String?
    let message: String?
    let matchId: Int64?
    let match: OngoingMatchRef?
    let ongoingMatch: OngoingMatchRef?
}

/// `GetCurrentMatchResp`。
struct GetCurrentMatchResponse: Decodable, Equatable {
    let success: Bool
    let match: OngoingMatchRef?
}

/// 进行中对局的窄解码（`CurrentMatchInfo` 的子集）：进行中引导与跳转只用到这些字段，全部可选以兼容缺失值。
struct OngoingMatchRef: Decodable, Equatable {
    var id: Int64? = nil
    var matchId: Int64? = nil
    var gameType: Int? = nil
    var opponentId: Int64? = nil
    var opponentName: String? = nil
    var opponentAvatar: String? = nil
    var matchMode: String? = nil
    var visibility: String? = nil
    var challengeId: Int64? = nil
}

struct RefereePreviewInfo: Decodable, Equatable {
    let player1Id: Int64
    let player1Name: String
    let player1Avatar: String
    let player2Id: Int64
    let player2Name: String
    let player2Avatar: String
    let player1Score: Int
    let player2Score: Int
    let gameType: Int
    let gameTypeName: String
    let matchMode: String?
    let visibility: String?
    let status: Int
    let statusText: String
}

struct RefereePreviewResponse: Decodable, Equatable {
    let success: Bool
    let message: String?
    let preview: RefereePreviewInfo?
}

struct JoinMatchRefereeResponse: Decodable, Equatable {
    let success: Bool
    let message: String?
    let matchId: Int64?
    let match: OngoingMatchRef?
}

/// `OpponentCandidate`：选择球友页展示的候选对手。
struct OpponentCandidate: Decodable, Equatable, Identifiable {
    let userId: Int64
    let name: String
    let avatar: String
    let rankName: String?
    let source: String
    let lastMatchAt: String?

    var id: Int64 { userId }

    /// 服务端 `source` 只有 friend / recent 两种；其余按最近对手展示。
    var sourceText: String { source == "friend" ? "好友" : "最近对手" }
}

struct OpponentCandidatesResponse: Decodable, Equatable {
    let success: Bool
    let list: [OpponentCandidate]
}

/// 扫码开局、执裁与选择球友的请求门面；需要登录的接口统一走 authorizedPost / authorizedRequest。
@MainActor
struct MatchStartAPI {
    static let opponentCandidateLimit = 30

    let session: SessionStore

    func previewMatchInvite(_ inviteToken: String) async throws -> MatchInvitePreviewResponse {
        try await session.authorizedPost(
            "/api/match/invite/preview",
            body: PreviewMatchInviteRequest(
                inviteToken: inviteToken.trimmingCharacters(in: .whitespacesAndNewlines)
            )
        )
    }

    func resolveMatchInviteCode(_ inviteCode: String) async throws -> MatchInviteResolveResponse {
        try await session.authorizedPost(
            "/api/match/invite/resolve",
            body: ResolveMatchInviteCodeRequest(
                inviteCode: inviteCode.trimmingCharacters(in: .whitespacesAndNewlines)
            )
        )
    }

    func previewMatchReferee(_ join: RefereeJoin) async throws -> RefereePreviewResponse {
        try await session.authorizedPost("/api/match/referee/preview", body: join)
    }

    func joinMatchReferee(_ join: RefereeJoin) async throws -> JoinMatchRefereeResponse {
        try await session.authorizedPost("/api/match/referee/join", body: join)
    }

    func startMatch(_ payload: StartMatchRequest) async throws -> StartMatchResponse {
        try await session.authorizedPost("/api/match/start", body: payload)
    }

    func currentMatch() async throws -> GetCurrentMatchResponse {
        try await session.authorizedRequest("/api/match/current")
    }

    func opponentCandidates(limit: Int = Self.opponentCandidateLimit) async throws -> [OpponentCandidate] {
        let path = APIQuery.path("/api/opponent/candidates", ["limit": limit])
        let response: OpponentCandidatesResponse = try await session.authorizedRequest(path)
        return response.list
    }
}
