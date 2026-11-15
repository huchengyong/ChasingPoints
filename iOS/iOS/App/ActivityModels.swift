import Foundation

/// 聚合接口 `GET /api/user/bootstrap` 的响应，字段与 `GetUserBootstrapResp` 对齐。
struct ActivityBootstrapResponse: Decodable {
    let success: Bool
    let currentMatch: CurrentMatchInfo?
    let currentChallenge: ChallengeInfo?
    let receivedPendingChallengeCount: Int
    let serverTime: String
    let unreadCount: Int
    let pendingFriendRequestCount: Int
    let latestSeasonRollover: NotificationInfo?
    let availability: [String: Bool]?
}

/// `CurrentMatchInfo`：进行中的对局，`optional` 字段用可选类型。
struct CurrentMatchInfo: Decodable, Equatable {
    let id: Int64
    let challengeId: Int64?
    let gameType: Int
    let gameTypeName: String
    let gameMode: String
    let matchMode: String?
    let visibility: String?
    let finishState: String?
    let finishRequestedBy: Int64?
    let finishRequestExpiresAt: String?
    let finishRequestRemainingSeconds: Int?
    let viewerRole: String?
    let refereeBound: Bool?
    let refereeUserId: Int64?
    let refereeName: String?
    let refereeAvatar: String?
    let refereeJoinedAt: String?
    let completedByUserId: Int64?
    let completionSource: String?
    let canScore: Bool?
    let canUndo: Bool?
    let canFinish: Bool?
    let canRequestFinish: Bool?
    let canConfirmFinish: Bool?
    let canDisputeFinish: Bool?
    let canWithdrawFinish: Bool?
    let lastAction: MatchLastAction?
    let player1Id: Int64
    let player1Name: String
    let player1Avatar: String
    let player2Id: Int64
    let player2Name: String
    let player2Avatar: String
    let opponentId: Int64?
    let opponentName: String
    let opponentAvatar: String?
    let myScore: Int
    let opponentScore: Int
    let currentFrameStarted: Bool?
    let currentFrameMyScore: Int?
    let currentFrameOpponentScore: Int?
    let currentRound: Int
    let snookerRulesVersion: Int?
    let bestOfFrames: Int?
    let snookerFormat: String?
    let snookerTargetWins: Int?
    let canChangeSnookerFormat: Bool?
    let matchFormat: String?
    let targetWins: Int?
    let canChangeMatchFormat: Bool?
    let startingActor: Int?
    let snookerPhase: String?
    let snookerBallOn: String?
    let snookerStriker: Int?
    let snookerVisitNo: Int?
    let snookerCurrentBreak: Int?
    let snookerRedsRemaining: Int?
    let snookerFreeBallAvailable: Bool?
    let snookerCueBallInHand: Bool?
    let snookerMissWarningActive: Bool?
    let snookerRespottedBlackPending: Bool?
    let snookerPendingConcessionActor: Int?
    let snookerPendingConcessionScope: String?
    let snookerFrameEndReason: String?
    let serverRevision: Int64
    let matchTime: String
    let durationSeconds: Int64
    let refereeDurationSeconds: Int64?
}

/// `ChallengeInfo`：当前约球与收到的邀请。
struct ChallengeInfo: Decodable, Equatable, Identifiable {
    let id: Int64
    let matchId: Int64?
    let fromUserId: Int64
    let fromNickname: String
    let fromAvatar: String
    let fromVipLevel: Int?
    let toUserId: Int64
    let toNickname: String
    let toAvatar: String
    let toVipLevel: Int?
    let gameType: Int
    let matchMode: String?
    let message: String
    let status: Int
    let closeReason: String?
    let waitingUserId: Int64?
    let scheduledDate: String?
    let startHour: Int?
    let endHour: Int?
    let expiresAt: String?
    let createdAt: String
}

/// `NotificationInfo`：消息中心与换季提醒共用的通知条目。
struct NotificationInfo: Decodable, Equatable, Identifiable {
    let id: Int64
    let type: String
    let title: String
    let content: String
    let data: String
    var isRead: Bool
    let createdAt: String
}
