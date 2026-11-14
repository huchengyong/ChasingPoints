import Foundation

/// `RoundRecord`：局记录固定按选手1 / 选手2 下发，不随查看者交换。
struct MatchRoundRecord: Decodable, Equatable {
    var roundNumber: Int?
    var player1Score: Int?
    var player2Score: Int?
    var winner: Int?
    var winType: String?
}

/// 局记录换边后的展示行；`resultText` 用「胜 / 负」表示视角方输赢。
struct MatchRoundRecordView: Equatable {
    let roundNumber: Int
    let player1Score: Int
    let player2Score: Int
    let winner: Int
    let winType: String
    let winTypeLabel: String
    let result: String
    let resultText: String
}

struct MatchDetailSide: Equatable {
    let name: String
    let avatar: String
    let vipLevel: Int
    let rankLevel: Int
}

/// 详情页比分与对局信息；`player1/2` 是该页面的固定左右视角。
struct MatchDetailMatch: Equatable {
    let player1Score: Int
    let player2Score: Int
    let gameType: Int
    let status: Int
    let durationSeconds: Int64
    let currentRound: Int
    let totalRounds: Int
    let viewerRole: String
    let refereeBound: Bool
    let refereeUserId: Int64
    let refereeName: String
    let refereeAvatar: String
    let refereeJoinedAt: String
    let refereeDurationSeconds: Int64
    let completedByUserId: Int64
    let completionSource: String
}

struct MatchDetailViewData: Equatable {
    let matchData: MatchDetailMatch
    let player1Info: MatchDetailSide
    let player2Info: MatchDetailSide
    let player1UserId: Int64
    let player2UserId: Int64
    let roundRecords: [MatchRoundRecordView]
    let roundRecordsSwapped: Bool
    let roundRecordsNeutral: Bool
    let scoreBreakdown: ChasingScoreView?
}

/// 逐条对应小程序 `utils/match-detail.js`。
enum MatchDetailRules {
    static func shouldUsePublicMatchDetail(mode: String = "", source: String = "") -> Bool {
        mode == "spectate" || source == "history"
    }

    static func shouldShowSpectateBadge(mode: String = "") -> Bool {
        mode == "spectate"
    }

    /// 只标注四个九球胜法；普胜、斯诺克取值和缺失都留空。
    private static let roundWinTypes: [String: String] = [
        "break_clear": MemberReplayView.winTypeLabels["break_clear"] ?? "",
        "continue_clear": MemberReplayView.winTypeLabels["continue_clear"] ?? "",
        "small_gold": MemberReplayView.winTypeLabels["small_gold"] ?? "",
        "big_gold": MemberReplayView.winTypeLabels["big_gold"] ?? "",
    ]

    static func normalizeRoundRecords(
        _ rounds: [MatchRoundRecord],
        shouldSwapPerspective: Bool = false
    ) -> [MatchRoundRecordView] {
        rounds.map { round in
            let winner = round.winner ?? 0
            let perspectiveWinner = shouldSwapPerspective && winner > 0 ? (winner == 1 ? 2 : 1) : winner
            let winType = round.winType ?? ""
            return MatchRoundRecordView(
                roundNumber: round.roundNumber ?? 0,
                player1Score: shouldSwapPerspective ? (round.player2Score ?? 0) : (round.player1Score ?? 0),
                player2Score: shouldSwapPerspective ? (round.player1Score ?? 0) : (round.player2Score ?? 0),
                winner: perspectiveWinner,
                winType: winType,
                winTypeLabel: roundWinTypes[winType] ?? "",
                result: perspectiveWinner == 1 ? "win" : "loss",
                resultText: perspectiveWinner == 1 ? "胜" : "负"
            )
        }
    }

    static func normalize(
        _ match: LiveMatchSnapshot,
        perspectiveUserId: Int64 = 0
    ) -> MatchDetailViewData {
        let player1Id = match.player1Id ?? 0
        let player2Id = match.player2Id ?? 0
        let shouldSwapPerspective = perspectiveUserId > 0
            && player2Id == perspectiveUserId
            && player1Id != perspectiveUserId
        // 参与者详情已按查看者返回「我 / 对手」并带 is_player1；公开详情靠 perspectiveUserId 判断。
        let isReferee = match.viewerRole == "referee"
        let isOwnMatchPayload = match.isPlayer1 != nil && !isReferee
        let isPerspectivePlayer = perspectiveUserId > 0
            && (player1Id == perspectiveUserId || player2Id == perspectiveUserId)
        // 未登录、观战者或裁判看固定选手1 / 选手2，不跟随查看者换边。
        let neutralPerspective = isReferee || !(isOwnMatchPayload || isPerspectivePlayer)
        let swapScoreBreakdown = shouldSwapPerspective || (isOwnMatchPayload && match.isPlayer1 == false)
        // 局记录是选手1 / 选手2 固定视角：参与者详情跟随查看者换边，裁判和公开详情保持选手1在左。
        let swapRoundRecords = !isReferee && (shouldSwapPerspective || (isOwnMatchPayload && match.isPlayer1 == false))

        let gameType = match.gameType ?? 3
        let matchData = MatchDetailMatch(
            player1Score: shouldSwapPerspective
                ? pickNumber(match.player2Score, match.opponentScore)
                : pickNumber(match.player1Score, match.myScore),
            player2Score: shouldSwapPerspective
                ? pickNumber(match.player1Score, match.myScore)
                : pickNumber(match.player2Score, match.opponentScore),
            gameType: gameType,
            status: match.status ?? 1,
            durationSeconds: match.durationSeconds ?? 0,
            currentRound: match.currentRound ?? 1,
            totalRounds: match.totalRounds ?? 0,
            viewerRole: match.viewerRole ?? "",
            refereeBound: match.refereeBound ?? false,
            refereeUserId: match.refereeUserId ?? 0,
            refereeName: match.refereeName ?? "",
            refereeAvatar: match.refereeAvatar ?? "",
            refereeJoinedAt: match.refereeJoinedAt ?? "",
            refereeDurationSeconds: match.refereeDurationSeconds ?? 0,
            completedByUserId: match.completedByUserId ?? 0,
            completionSource: (match.completionSource?.isEmpty == false) ? match.completionSource! : "unknown"
        )

        let player1Info = MatchDetailSide(
            name: shouldSwapPerspective
                ? firstNonEmpty(match.player2Name, match.opponentName, "玩家1")
                : firstNonEmpty(match.player1Name, match.myName, "玩家1"),
            avatar: shouldSwapPerspective
                ? firstNonEmpty(match.player2Avatar, match.opponentAvatar, "")
                : firstNonEmpty(match.player1Avatar, match.myAvatar, ""),
            vipLevel: shouldSwapPerspective
                ? (match.player2VipLevel ?? match.opponentVipLevel ?? 0)
                : (match.player1VipLevel ?? match.myVipLevel ?? 0),
            rankLevel: shouldSwapPerspective
                ? (match.player2RankLevel ?? match.opponentRankLevel ?? 0)
                : (match.player1RankLevel ?? match.myRankLevel ?? 0)
        )
        let player2Info = MatchDetailSide(
            name: shouldSwapPerspective
                ? firstNonEmpty(match.player1Name, match.myName, "玩家2")
                : firstNonEmpty(match.player2Name, match.opponentName, "玩家2"),
            avatar: shouldSwapPerspective
                ? firstNonEmpty(match.player1Avatar, match.myAvatar, "")
                : firstNonEmpty(match.player2Avatar, match.opponentAvatar, ""),
            vipLevel: shouldSwapPerspective
                ? (match.player1VipLevel ?? match.myVipLevel ?? 0)
                : (match.player2VipLevel ?? match.opponentVipLevel ?? 0),
            rankLevel: shouldSwapPerspective
                ? (match.player1RankLevel ?? match.myRankLevel ?? 0)
                : (match.player2RankLevel ?? match.opponentRankLevel ?? 0)
        )

        return MatchDetailViewData(
            matchData: matchData,
            player1Info: player1Info,
            player2Info: player2Info,
            player1UserId: shouldSwapPerspective ? player2Id : player1Id,
            player2UserId: shouldSwapPerspective ? player1Id : player2Id,
            roundRecords: normalizeRoundRecords(match.rounds ?? [], shouldSwapPerspective: swapRoundRecords),
            roundRecordsSwapped: swapRoundRecords,
            roundRecordsNeutral: neutralPerspective,
            scoreBreakdown: gameType == 2
                ? ChasingScore.resolveView(match.scoreBreakdown, swap: swapScoreBreakdown, neutral: neutralPerspective)
                : nil
        )
    }

    private static func firstNonEmpty(_ values: String?...) -> String {
        for value in values {
            if let value, !value.isEmpty { return value }
        }
        return ""
    }

    /// 详情接口按角色下发 `player*_score` 或 `my_score / opponent_score`，取第一个有值的。
    private static func pickNumber(_ values: Int?...) -> Int {
        for value in values {
            if let value { return value }
        }
        return 0
    }
}
