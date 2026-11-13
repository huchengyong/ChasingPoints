import Foundation

/// 对局规则模块共用的窄输入类型：只保留规则实际读取的字段，字段名与可选性以 backend/chasing_points.api 为准。
/// 统一用 convertFromSnakeCase 解码。

// MARK: - 匹配二维码（MatchScan）

/// `MatchInvitePreviewInfo`：扫码或邀请解析后返回的对手预览。
struct MatchInvitePreviewInfo: Decodable, Equatable {
    let opponentId: Int64
    let opponentName: String
    let opponentAvatar: String
}

/// `MatchInviteResolveResp`：把入口链接短码换成服务端签名的 invite_token。
struct MatchInviteResolveResponse: Decodable, Equatable {
    let success: Bool
    let message: String?
    let inviteToken: String?
    let preview: MatchInvitePreviewInfo?
}

/// `MatchInvitePreviewResp`：校验签名 invite_token 并返回对手预览。
struct MatchInvitePreviewResponse: Decodable, Equatable {
    let success: Bool
    let message: String?
    let preview: MatchInvitePreviewInfo?
}

// MARK: - 快照里的赛制（PoolMatchFormat / SnookerMatchFormat）

/// 中式八球、美式九球的快照职赛制字段。
struct PoolMatchFormatInput: Decodable, Equatable {
    var matchFormat: String? = nil
    var targetWins: Int? = nil
}

/// 斯诺克的快照赛制字段。
struct SnookerMatchFormatInput: Decodable, Equatable {
    var snookerFormat: String? = nil
    var snookerTargetWins: Int? = nil
}

// MARK: - 斯诺克 V2 局面（SnookerV2）

/// `MatchSyncSnapshot` 里斯诺克 V2 规则用到的字段。
struct SnookerV2SnapshotInput: Decodable, Equatable {
    var snookerRulesVersion: Int? = nil
    var bestOfFrames: Int? = nil
    var startingActor: Int? = nil
    var snookerPhase: String? = nil
    var snookerBallOn: String? = nil
    var snookerStriker: Int? = nil
    var snookerVisitNo: Int? = nil
    var snookerCurrentBreak: Int? = nil
    var snookerRedsRemaining: Int? = nil
    var snookerFreeBallAvailable: Bool? = nil
    var snookerCueBallInHand: Bool? = nil
    var snookerMissWarningActive: Bool? = nil
    var snookerRespottedBlackPending: Bool? = nil
    var snookerPendingConcessionActor: Int? = nil
    var snookerPendingConcessionScope: String? = nil
    var snookerFrameEndReason: String? = nil
    var currentFrameStarted: Bool? = nil
    var status: Int? = nil
}

// MARK: - 对局最后动作（MatchRoleView）

/// `MatchLastAction`：进行中页面透传展示的最后一次动作。
struct MatchLastAction: Decodable, Equatable {
    let actionType: String?
    let actor: Int?
    let scoreChange: Int?
    let serverRevision: Int64?
    let description: String?
}

// MARK: - 九球追分得分构成（ChasingScore / MemberReplayView）

struct ChasingScoreSide: Decodable, Equatable {
    var score: Int = 0
    var normalWins: Int = 0
    var smallGolds: Int = 0
    var bigGolds: Int = 0
    var foulCount: Int = 0
    var foulPointsGiven: Int = 0
    var otherPoints: Int = 0
}

struct ChasingScoreEvent: Decodable, Equatable {
    var seq: Int = 0
    var kind: String = ""
    var scorer: Int = 0
    var points: Int = 0
    var player1Score: Int = 0
    var player2Score: Int = 0
    var createdAt: String = ""
}

/// `ChasingScoreBreakdown`，固定按选手1 / 选手2 下发。
struct ChasingScoreBreakdown: Decodable, Equatable {
    var available: Bool = false
    var player1: ChasingScoreSide = ChasingScoreSide()
    var player2: ChasingScoreSide = ChasingScoreSide()
    var events: [ChasingScoreEvent] = []
    var eventsTruncated: Bool? = nil
}

// MARK: - 单场复盘（MemberReplayView）

struct ReplayTrendPoint: Decodable, Equatable {
    var step: Int = 0
    var diff: Int = 0
}

struct ReplayFrame: Decodable, Equatable {
    var frameNo: Int = 0
    var winner: Int = 0
    var myScore: Int = 0
    var opponentScore: Int = 0
    var winType: String? = nil
    var durationSeconds: Int? = nil
    var maxDeficit: Int = 0
    var keyTags: [String]? = nil
    var myMaxRun: Int? = nil
    var opponentMaxRun: Int? = nil
    var myFoulCount: Int? = nil
    var opponentFoulCount: Int? = nil
    var myFoulPoints: Int? = nil
    var opponentFoulPoints: Int? = nil
    var trend: [ReplayTrendPoint]? = nil
}

struct ReplayWinTypeStat: Decodable, Equatable {
    var winType: String = ""
    var label: String = ""
    var myCount: Int = 0
    var opponentCount: Int = 0
}

struct ReplaySummary: Decodable, Equatable {
    var myFrameWins: Int = 0
    var opponentFrameWins: Int = 0
    var myMaxRun: Int? = nil
    var opponentMaxRun: Int? = nil
    var myLongestStreak: Int? = nil
    var opponentLongestStreak: Int? = nil
    var winTypeStats: [ReplayWinTypeStat]? = nil
    var scoreBreakdown: ChasingScoreBreakdown? = nil
}

/// `GetMemberMatchReplayResp` 里复盘页实际读取的字段。
struct MatchReplayResponse: Decodable, Equatable {
    var gameType: Int = 0
    var locked: Bool = false
    var viewerActor: Int? = nil
    var frameCount: Int = 0
    var frameOnly: Bool? = nil
    var detailTruncated: Bool? = nil
    var summary: ReplaySummary? = nil
    var frames: [ReplayFrame]? = nil
}

// MARK: - 赛果页结束方式（FinishCountdown）

/// `MatchDetailData` 里判断「结束方式」需要的字段。
struct FinishMatchDetailInput: Decodable, Equatable {
    var matchMode: String? = nil
    var completionSource: String? = nil
    var player1Id: Int64 = 0
    var player2Id: Int64 = 0
    var completedByUserId: Int64? = nil
    var viewerRole: String? = nil
    var isPlayer1: Bool = false
    var opponentName: String = ""
}
