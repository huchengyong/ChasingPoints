import Foundation

/// 归一化后的对局信息；进行中引导与跳转只读取这些字段。
struct GuardedMatch: Equatable {
    let matchId: Int64
    let gameType: Int
    let opponentName: String
    let opponentAvatar: String
    let opponentId: Int64
    let matchMode: String
    let visibility: String
    let challengeId: Int64
}

/// `normalizeMatchPayload` 的兜底来源：扫码对手与服务端缺失字段的默认值。
struct GuardedMatchFallback: Equatable {
    var matchId: Int64 = 0
    var gameType: Int = 0
    var opponentName: String = ""
    var opponentAvatar: String = ""
    var opponentId: Int64 = 0
    var matchMode: String = ""
    var visibility: String = ""
    var challengeId: Int64 = 0
}

enum StartMatchGuardOutcome: Equatable {
    case navigate(GuardedMatch)
    case resume(GuardedMatch)
    case promptSelfOngoing(message: String, match: GuardedMatch)
    case toastOpponentOngoing(message: String)
    case error(message: String)
}

/// 逐条对应小程序 `utils/ongoing-match-guard.js` 的开局结果分流。
/// 原生端前台提醒与进行中页面尚未迁移，只迁移确认页实际使用的分支。
enum OngoingMatchGuard {
    static func normalizeMatchPayload(
        _ match: CurrentMatchInfo? = nil,
        fallback: GuardedMatchFallback = GuardedMatchFallback()
    ) -> GuardedMatch {
        GuardedMatch(
            matchId: firstNonZero(match?.matchId, match?.id, fallback.matchId),
            gameType: firstNonZero(match?.gameType, fallback.gameType, 3),
            opponentName: firstNonEmpty(match?.opponentName, fallback.opponentName, "对手"),
            opponentAvatar: firstNonEmpty(match?.opponentAvatar, fallback.opponentAvatar, ""),
            opponentId: firstNonZero(match?.opponentId, fallback.opponentId),
            matchMode: firstNonEmpty(match?.matchMode, fallback.matchMode, "ranked"),
            visibility: firstNonEmpty(match?.visibility, fallback.visibility, "public"),
            challengeId: firstNonZero(match?.challengeId, fallback.challengeId)
        )
    }

    static func resolveStartMatchGuardAction(
        response: StartMatchResponse,
        selectedGameType: Int,
        scannedOpponent: StartMatchOpponent = StartMatchOpponent()
    ) -> StartMatchGuardOutcome {
        guard response.success else {
            return .error(message: firstNonEmpty(response.message, "创建对局失败"))
        }

        let scannedFallback = GuardedMatchFallback(
            gameType: selectedGameType,
            opponentName: firstNonEmpty(
                scannedOpponent.nickname, scannedOpponent.name, scannedOpponent.opponentName, "对手"
            ),
            opponentAvatar: firstNonEmpty(scannedOpponent.avatar, scannedOpponent.opponentAvatar, ""),
            opponentId: firstNonZero(scannedOpponent.userId, scannedOpponent.id, scannedOpponent.opponentId)
        )

        switch response.action {
        case "created":
            let match = response.match ?? CurrentMatchInfo(matchId: response.matchId)
            return .navigate(normalizeMatchPayload(match, fallback: scannedFallback))
        case "resume_existing":
            var fallback = scannedFallback
            fallback.matchId = response.matchId ?? 0
            let match = response.ongoingMatch ?? CurrentMatchInfo(matchId: response.matchId)
            return .resume(normalizeMatchPayload(match, fallback: fallback))
        case "blocked" where response.blockReason == "self_ongoing":
            var fallback = scannedFallback
            fallback.matchId = response.matchId ?? 0
            let match = response.ongoingMatch ?? CurrentMatchInfo(matchId: response.matchId)
            return .promptSelfOngoing(
                message: firstNonEmpty(response.message, "你还有未结束的对局，请先处理"),
                match: normalizeMatchPayload(match, fallback: fallback)
            )
        case "blocked" where response.blockReason == "opponent_ongoing":
            return .toastOpponentOngoing(
                message: firstNonEmpty(response.message, "对手还有未结束的对局，暂时无法开始新的 PK")
            )
        default:
            return .error(message: firstNonEmpty(response.message, "创建对局失败"))
        }
    }

    private static func firstNonZero(_ values: Int64?...) -> Int64 {
        for value in values {
            if let value, value != 0 { return value }
        }
        return 0
    }

    private static func firstNonZero(_ values: Int?...) -> Int {
        for value in values {
            if let value, value != 0 { return value }
        }
        return 0
    }

    private static func firstNonEmpty(_ values: String?...) -> String {
        for value in values {
            if let value, !value.isEmpty { return value }
        }
        return ""
    }
}
