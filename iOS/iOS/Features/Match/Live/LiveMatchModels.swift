import Foundation

/// 进入实时计分页的三种来源：本人对局（含裁判）用登录态详情，观赛用公开详情。
enum LiveMatchEntry: Equatable {
    /// 按当前登录状态选择：未登录读公开详情，已登录先读详情，无权限时回退公开详情。
    case automatic
    case participant
    case spectator
}

/// 页面需要跳转到的目的地；`back` 由视图直接 dismiss。
enum LiveMatchDestination: Equatable {
    case result(Int64)
    case matchHistory
}

enum LiveMatchConfirmKind: Equatable {
    case finishMatch
    case requestFinish
    case nextRound
    case offerSnookerConcession
    case awardSnookerFrame
    case back
}

/// 与小程序 `uni.showActionSheet` 对应的单选弹层；选项按下标回传。
struct LiveMatchChoice: Equatable, Identifiable {
    struct Option: Equatable {
        let title: String
    }

    let id = UUID()
    let title: String
    let options: [Option]
}

/// 与小程序 `uni.showModal` 对应的确认弹层。
struct LiveMatchConfirm: Equatable, Identifiable {
    let kind: LiveMatchConfirmKind
    let title: String
    let message: String
    let confirmTitle: String
    let cancelTitle: String

    var id: String { "\(kind)-\(title)-\(message)" }
}

struct LiveMatchPlayerInfo: Equatable {
    var userId: Int64 = 0
    var nickname = ""
    var avatar = ""
    var vipLevel = 0
    var rankLevel = 0
    var winRate = 0
    var maxScore = 0
}

/// `GET /api/match/current`、`GET /api/match/detail`、`GET /api/public/match/detail`、
/// `MatchSyncData`、`MatchSyncSnapshot` 与各写接口 `snapshot` 的并集。
/// 三种入口和下一轮的斯诺克 / 九球追分都会读取它，所以一次性解全；缺省字段一律可选。
struct LiveMatchSnapshot: Decodable, Equatable {
    var id: Int64?
    var matchId: Int64?
    var challengeId: Int64?
    var gameType: Int?
    var gameTypeName: String?
    var matchMode: String?
    var visibility: String?
    var status: Int?
    var serverRevision: Int64?
    var isPlayer1: Bool?
    var viewerRole: String?
    var refereeBound: Bool?
    var refereeUserId: Int64?
    var refereeName: String?
    var refereeAvatar: String?
    var refereeJoinedAt: String?
    var completedByUserId: Int64?
    var completionSource: String?
    var canScore: Bool?
    var canUndo: Bool?
    var canFinish: Bool?
    var canRequestFinish: Bool?
    var canConfirmFinish: Bool?
    var canDisputeFinish: Bool?
    var canWithdrawFinish: Bool?
    var finishState: String?
    var finishRequestedBy: Int64?
    var finishRequestExpiresAt: String?
    var finishRequestRemainingSeconds: Int?
    var lastAction: MatchLastAction?
    var myScore: Int?
    var opponentScore: Int?
    var player1Score: Int?
    var player2Score: Int?
    var player1Id: Int64?
    var player2Id: Int64?
    var myName: String?
    var opponentName: String?
    var opponentId: Int64?
    var myAvatar: String?
    var opponentAvatar: String?
    // 公开详情的固定选手视角资料，观赛时作为左右两侧来源。
    var player1Name: String?
    var player1Avatar: String?
    var player2Name: String?
    var player2Avatar: String?
    var myVipLevel: Int?
    var opponentVipLevel: Int?
    var myRankLevel: Int?
    var opponentRankLevel: Int?
    var myWinRate: Double?
    var opponentWinRate: Double?
    var myMaxScore: Int?
    var opponentMaxScore: Int?
    var currentFrameStarted: Bool?
    var currentFrameMyScore: Int?
    var currentFrameOpponentScore: Int?
    var currentFramePlayer1Score: Int?
    var currentFramePlayer2Score: Int?
    var currentRound: Int?
    var totalRounds: Int?
    var redBallCount: Int?
    var snookerClearanceStarted: Bool?
    var snookerClearedColors: [Int]?
    var snookerExpectedClearanceScore: Int?
    var snookerClearanceCompleted: Bool?
    var snookerRulesVersion: Int?
    var bestOfFrames: Int?
    var snookerFormat: String?
    var snookerTargetWins: Int?
    var canChangeSnookerFormat: Bool?
    var matchFormat: String?
    var targetWins: Int?
    var canChangeMatchFormat: Bool?
    var startingActor: Int?
    var snookerPhase: String?
    var snookerBallOn: String?
    var snookerStriker: Int?
    var snookerVisitNo: Int?
    var snookerCurrentBreak: Int?
    var snookerRedsRemaining: Int?
    var snookerFreeBallAvailable: Bool?
    var snookerCueBallInHand: Bool?
    var snookerMissWarningActive: Bool?
    var snookerRespottedBlackPending: Bool?
    var snookerPendingConcessionActor: Int?
    var snookerPendingConcessionScope: String?
    var snookerFrameEndReason: String?
    var durationSeconds: Int64?
    var matchTime: String?
    var result: Int?
    // 比分推送专用
    var roundNumber: Int?
    var actionType: String?
    var actor: String?
    var winner: Int?

    var resolvedId: Int64 { matchId ?? id ?? 0 }
}

struct LiveMatchDetailResponse: Decodable, Equatable {
    let success: Bool
    let match: LiveMatchSnapshot?
}

/// 各写接口响应的并集；`snapshot` 在失败响应里也可能缺省。
struct LiveMatchWriteResponse: Decodable, Equatable {
    var accepted: Bool?
    var success: Bool?
    var message: String?
    var action: String?
    var clientActionId: String?
    var serverRevision: Int64?
    var snapshot: LiveMatchSnapshot?
    var myScore: Int?
    var opponentScore: Int?
    var player1Score: Int?
    var player2Score: Int?
    var currentFrameStarted: Bool?
    var currentFrameMyScore: Int?
    var currentFrameOpponentScore: Int?
    var roundNo: Int?
    var finishState: String?
    var finishRequestedBy: Int64?
    var result: Int?
}

struct LiveRefereeQRCodeResponse: Decodable, Equatable {
    let success: Bool
    let qrcodeData: String?
    let expiresInSeconds: Int64?
    let message: String?
}

/// `GET /api/match/current` 的响应；`CurrentMatchInfo` 含修订号与赛制，供加载时校验当前对局。
struct LiveCurrentMatchResponse: Decodable, Equatable {
    let success: Bool
    let match: CurrentMatchInfo?
}

/// 对应小程序 `utils/rank-cache.js` 的 `shouldInvalidateRankAfterSettlement`。
enum LiveMatchSettlement {
    static func shouldInvalidateRank(
        isParticipant: Bool,
        matchMode: String,
        status: Int?,
        result: Int?,
        player1Score: Int?,
        player2Score: Int?,
        myScore: Int,
        opponentScore: Int
    ) -> Bool {
        guard isParticipant, matchMode == "ranked", status == 2 else { return false }
        if let result, result > 0 { return result != 3 }
        let first = player1Score ?? myScore
        let second = player2Score ?? opponentScore
        return first != second
    }
}

/// 模板里与球种、视角相关的文案；与 `playing.vue` 逐字一致。
enum LiveMatchText {
    static func gameTitle(_ gameType: Int) -> String {
        switch gameType {
        case 1: "斯诺克"
        case 2: "九球追分"
        case 3: "中式八球"
        case 4: "美式九球"
        default: "未知"
        }
    }

    static func gameSubtitle(gameType: Int, isSnookerV2: Bool, isReferee: Bool) -> String {
        if gameType == 1, isSnookerV2 {
            return "后端权威维护击球方、目标球与上手"
        }
        if isReferee {
            switch gameType {
            case 1: return "上方为整场 frame，下方记录当前局得分"
            case 2: return "分别记录选手1与选手2得分"
            case 3, 4: return "明确判定双方的本局胜法"
            default: return ""
            }
        }
        switch gameType {
        case 1: return "上方为整场 frame，下方记录当前局得分"
        case 2: return "区分我方得分与对手得分"
        case 3, 4: return "明确判定对手的本局胜法"
        default: return ""
        }
    }

    static func roundText(gameType: Int, currentFrameStarted: Bool, currentRound: Int) -> String {
        if gameType == 1, !currentFrameStarted {
            return "待开始第 \(String(currentRound)) 局"
        }
        return "第 \(String(currentRound)) 局"
    }

    static func actionPanelDescription(canScore: Bool, readonlyHint: String, gameType: Int, isSnookerV2: Bool, isReferee: Bool) -> String {
        if !canScore { return readonlyHint.isEmpty ? "当前由裁判负责记分" : readonlyHint }
        if gameType == 1, isSnookerV2 { return "按当前击球方和目标球记录一次真实击球结果" }
        if isReferee {
            switch gameType {
            case 1: return "分别选择得分方和犯规方，再记录对应球值"
            case 2: return "为选手1或选手2记录得分与犯规"
            case 3, 4: return "以下按钮会判给对应选手本局胜"
            default: return "请按选手1 / 选手2 的实际结果记分"
            }
        }
        switch gameType {
        case 1: return "对手每进一个球，都在下方点击对应的球进行记分"
        case 2: return "左列记到我方，右列记到对手"
        case 3, 4: return "以下按钮都会判给对手本局胜"
        default: return "请确认分数归属后再点击"
        }
    }

    static func finishButtonLabel(isFreeNormalFinish: Bool) -> String {
        isFreeNormalFinish ? "结束整场对局" : "结束本场对局"
    }

    static func finishRequestButtonLabel(isFreeNormalFinish: Bool) -> String {
        isFreeNormalFinish ? "结束整场对局" : "发起结束确认"
    }

    /// 与 `matchDetail.vue` 的 `formatDuration` 一致。
    static func duration(_ seconds: Int64) -> String {
        guard seconds > 0 else { return "" }
        let totalMinutes = seconds / 60
        if totalMinutes < 1 { return "\(String(seconds))秒" }
        if totalMinutes < 60 { return "\(String(totalMinutes))分钟" }
        let hours = totalMinutes / 60
        let remainMinutes = totalMinutes % 60
        if hours < 24 {
            return remainMinutes > 0 ? "\(String(hours))小时\(String(remainMinutes))分钟" : "\(String(hours))小时"
        }
        let days = hours / 24
        let remainHours = hours % 24
        return remainHours > 0 ? "\(String(days))天\(String(remainHours))小时" : "\(String(days))天"
    }
}
