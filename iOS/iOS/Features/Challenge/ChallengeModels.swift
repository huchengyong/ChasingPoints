import Foundation

// 约球模块模型，字段与 `backend/chasing_points.api` 约球段对齐；`ChallengeInfo` 复用 App/ActivityModels.swift。

/// 发起页可预填的对手，对应小程序 `challengeCompose` 的 query 参数。
struct ChallengeComposeTarget: Hashable {
    let userId: Int64
    let nickname: String
    let avatar: String
    var gameType: Int? = nil
    var matchMode: String? = nil
}

struct SendChallengeBody: Encodable, Equatable {
    let toUserId: Int64
    let gameType: Int
    let matchMode: String
    let message: String
    let scheduledDate: String
    let startHour: Int
    let endHour: Int
}

struct AcceptChallengeBody: Encodable, Equatable {
    let challengeId: Int64
    let confirmCloseOwnPending: Bool?
}

struct ChallengeIdBody: Encodable, Equatable {
    let challengeId: Int64
}

struct EnterChallengeBody: Encodable, Equatable {
    let challengeId: Int64
    let confirmEarly: Bool?
}

/// `SendChallengeResp`；业务失败由请求层抛出，页面只读取成功结果。
struct SendChallengeResponse: Decodable, Equatable {
    let success: Bool
    let challengeId: Int64
    let message: String?
    let pendingChallengeId: Int64?
}

/// `AcceptChallengeResp`。
struct AcceptChallengeResponse: Decodable, Equatable {
    let success: Bool
    let challengeId: Int64
    let matchId: Int64?
    let message: String?
    let typeConflict: Bool?
    let needConfirmCloseOwn: Bool?
    let pendingChallengeId: Int64?
}

/// `ChallengeActionResp`：拒绝 / 取消 / 放弃 / 退出等待共用。
struct ChallengeActionResponse: Decodable, Equatable {
    let success: Bool
    let message: String?
    let challengeId: Int64?
    let matchId: Int64?
}

/// `EnterChallengeResp`。
struct EnterChallengeResponse: Decodable, Equatable {
    let success: Bool
    let action: String?
    let challengeId: Int64?
    let matchId: Int64?
    let message: String?
}

/// `GetChallengeSummaryResp`。
struct ChallengeSummaryResponse: Decodable, Equatable {
    let success: Bool
    let currentChallenge: ChallengeInfo?
    let receivedPendingCount: Int
    let serverTime: String
}

/// `GetPendingChallengesResp`。
struct ChallengePendingResponse: Decodable, Equatable {
    let success: Bool
    let list: [ChallengeInfo]
}

/// `GetChallengeDetailResp`。
struct ChallengeDetailResponse: Decodable, Equatable {
    let success: Bool
    let challenge: ChallengeInfo?
}

/// `GetChallengeHistoryResp`。
struct ChallengeHistoryResponse: Decodable, Equatable {
    let success: Bool
    let list: [ChallengeInfo]
    let nextBeforeId: Int64?
}

/// 约球业务失败响应体里的结构化字段；请求层抛出 `APIError` 后从这里取值，不按文案匹配。
struct ChallengeFailureDetails: Decodable, Equatable {
    let pendingChallengeId: Int64?
    let needConfirmCloseOwn: Bool?
    let typeConflict: Bool?
    let action: String?
}

// MARK: - 规则返回值

/// `challenge-entry.js` 的 `buildChallengePayload` 结果。
struct ChallengeEntryPayload: Equatable {
    let toUserId: Int64
    let gameType: Int
    let message: String
}

/// `normalizeChallengeListItem` 的方向与对手字段。
struct ChallengeListItemView: Equatable {
    enum Direction: Equatable {
        case sent
        case received
    }

    let direction: Direction
    let opponentId: Int64
    let nickname: String
    let avatar: String
}

/// `resolveChallengeAction` 的结果。
struct ChallengeEntryAction: Equatable {
    enum Kind: Equatable {
        case viewMatch
        case offlineStart
        case report
    }

    let kind: Kind
    let label: String
    let matchId: Int64
}

/// `buildChallengeStartContext` 的结果。
struct ChallengeStartContext: Equatable {
    let challengeId: Int64
    let opponentId: Int64
    let opponentName: String
    let opponentAvatar: String
    let gameType: Int
}

/// 首页约球待办卡，对应 `resolveChallengeQuickCard`。
struct ChallengeQuickCard: Equatable {
    let challengeId: Int64
    let matchId: Int64
    let title: String
    let needsAction: Bool
    let subText: String
    let actionLabel: String
}

/// 约球对局中一方的展示信息。
struct ChallengePlayer: Equatable {
    let userId: Int64
    let nickname: String
    let avatar: String
    let vipLevel: Int
    let isWaiting: Bool
}
