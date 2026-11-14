import Foundation

/// 首页展示规则，逐条对应小程序 `utils/home-index.js`、`challenge-view.js`
/// 与 `user-homepage.js` 中首页用到的部分。
enum HomeRules {
    static let nearbyRadiusMeters = 5000
    static let nearbyLimit = 3
    static let nearbyRefreshInterval: TimeInterval = 3 * 60

    private static let matchModeText = ["ranked": "排位", "practice": "练习"]

    // MARK: - 问候语

    /// 按本地小时划分时段，对应 `resolveHomeGreeting`。
    static func greeting(hour: Int) -> String {
        if hour >= 5 && hour < 11 { return "早上好" }
        if hour >= 11 && hour < 13 { return "中午好" }
        if hour >= 13 && hour < 18 { return "下午好" }
        return "晚上好"
    }

    // MARK: - 我的对局

    /// 时长文案，对应 `utils/user-homepage.js` 的 `formatDuration`。
    static func formatDuration(_ durationSeconds: Int64) -> String {
        guard durationSeconds > 0 else { return "0分" }

        let totalMinutes = durationSeconds / 60
        if totalMinutes < 1 { return "\(durationSeconds)秒" }
        if totalMinutes < 60 { return "\(totalMinutes)分" }

        let hours = totalMinutes / 60
        let remainMinutes = totalMinutes % 60
        if hours < 24 {
            return remainMinutes > 0 ? "\(hours)小时\(remainMinutes)分" : "\(hours)小时"
        }

        let days = hours / 24
        let remainHours = hours % 24
        return remainHours > 0 ? "\(days)天\(remainHours)小时" : "\(days)天"
    }

    /// 当前对局卡；`nil` 表示没有需要继续的对局。
    static func ongoingMatchCard(_ match: CurrentMatchInfo?, currentUser: UserInfo?) -> HomeMatchCard? {
        guard let match else { return nil }

        let viewerRole = match.viewerRole ?? "player1"
        let isReferee = viewerRole == "referee"
        let isPending = match.finishState == "pending_confirmation"
        let viewerIndex = viewerRole == "player2" ? 1 : 0
        let players = participants(match, currentUser: currentUser, viewerRole: viewerRole)
        let modeText = match.matchMode.flatMap { matchModeText[$0] } ?? ""
        let gameTypeName = trimmed(match.gameTypeName)
        let metaText = [gameTypeName.isEmpty ? "台球" : gameTypeName, modeText]
            .filter { !$0.isEmpty }
            .joined(separator: " · ")

        return HomeMatchCard(
            id: match.id,
            isReferee: isReferee,
            isPending: isPending,
            statusText: isPending ? "待确认结束" : isReferee ? "执裁中" : "进行中",
            metaText: metaText,
            durationText: formatDuration(match.durationSeconds),
            players: players.enumerated().map { index, player in
                HomeMatchPlayer(
                    id: player.id,
                    name: player.name,
                    avatar: player.avatar,
                    score: player.score,
                    isMe: !isReferee && index == viewerIndex
                )
            },
            actionLabel: isPending ? "处理结束确认" : isReferee ? "进入裁判记分" : "继续对局"
        )
    }

    /// 对应 `resolveMatchParticipants`：玩家顺序固定为 player1 / player2。
    private static func participants(
        _ match: CurrentMatchInfo,
        currentUser: UserInfo?,
        viewerRole: String
    ) -> [(id: Int64, name: String, avatar: String, score: Int)] {
        let viewerId = currentUser?.id ?? 0
        let viewerName = trimmed(currentUser?.nickname ?? "")
        let viewerAvatar = trimmed(currentUser?.avatar ?? "")

        var player1Id = match.player1Id
        var player1Name = trimmed(match.player1Name)
        var player1Avatar = trimmed(match.player1Avatar)
        var player2Id = match.player2Id
        var player2Name = trimmed(match.player2Name)
        var player2Avatar = trimmed(match.player2Avatar)

        if player1Name.isEmpty && viewerRole == "player1" {
            player1Id = player1Id == 0 ? viewerId : player1Id
            player1Name = viewerName.isEmpty ? "我" : viewerName
            player1Avatar = player1Avatar.isEmpty ? viewerAvatar : player1Avatar
        }
        if player2Name.isEmpty && viewerRole == "player2" {
            player2Id = player2Id == 0 ? viewerId : player2Id
            player2Name = viewerName.isEmpty ? "我" : viewerName
            player2Avatar = player2Avatar.isEmpty ? viewerAvatar : player2Avatar
        }
        if player1Name.isEmpty && viewerRole == "player2" {
            player1Id = player1Id == 0 ? (match.opponentId ?? 0) : player1Id
            let opponentName = trimmed(match.opponentName)
            player1Name = opponentName.isEmpty ? "玩家1" : opponentName
            player1Avatar = player1Avatar.isEmpty ? trimmed(match.opponentAvatar ?? "") : player1Avatar
        }
        if player2Name.isEmpty && viewerRole != "player2" {
            player2Id = player2Id == 0 ? (match.opponentId ?? 0) : player2Id
            let opponentName = trimmed(match.opponentName)
            player2Name = opponentName.isEmpty ? "玩家2" : opponentName
            player2Avatar = player2Avatar.isEmpty ? trimmed(match.opponentAvatar ?? "") : player2Avatar
        }

        if player1Name.isEmpty { player1Name = "玩家1" }
        if player2Name.isEmpty { player2Name = "玩家2" }

        var player1Score = match.myScore
        var player2Score = match.opponentScore
        if viewerRole == "player2" {
            player1Score = match.opponentScore
            player2Score = match.myScore
        }

        return [
            (player1Id, player1Name, player1Avatar, player1Score),
            (player2Id, player2Name, player2Avatar, player2Score),
        ]
    }

    // MARK: - 约球

    /// 约球待办卡；不认识的状态返回 `nil`，不展示空卡。
    static func challengeQuickCard(
        _ challenge: ChallengeInfo?,
        userId: Int64,
        serverTime: String
    ) -> HomeChallengeCard? {
        guard let card = ChallengeRules.quickCard(challenge, userId: userId, serverTime: serverTime) else {
            return nil
        }
        return HomeChallengeCard(
            challengeId: card.challengeId,
            matchId: card.matchId,
            title: card.title,
            needsAction: card.needsAction,
            subText: card.subText,
            actionLabel: card.actionLabel
        )
    }

    /// `今天16点—17点`；日期按服务端时间换算。
    static func challengeSchedule(_ challenge: ChallengeInfo, serverTime: String) -> String {
        ChallengeRules.schedule(challenge, serverTime: serverTime)
    }

    /// 相对日期文案：今天 / 明天 / 昨天 / 月日（跨年带年份）。
    static func challengeDayLabel(_ scheduledDate: String, serverTime: String) -> String {
        ChallengeRules.dayLabel(scheduledDate, serverTime: serverTime)
    }

    // MARK: - 排行榜

    /// 「我」的名次文案，对应 `formatHomeMyRanking`。
    static func myRankingText(_ ranking: LeaderboardItem?) -> String {
        guard let ranking, ranking.rank > 0 else {
            return "暂未上榜，打一场排位赛开始积分"
        }
        var parts = ["第 \(ranking.rank) 名"]
        if !ranking.rankName.isEmpty { parts.append(ranking.rankName) }
        parts.append("\(String(ranking.rankScore)) 分")
        return parts.joined(separator: " · ")
    }

    // MARK: - 附近球房

    /// 经纬度为 0 视为没有位置信息，不请求附近球房。
    static func nearbyVenueParams(_ coordinate: Coordinate?) -> HomeNearbyVenueParams? {
        guard let coordinate, coordinate.latitude != 0, coordinate.longitude != 0 else { return nil }
        return HomeNearbyVenueParams(
            latitude: coordinate.latitude,
            longitude: coordinate.longitude,
            radius: nearbyRadiusMeters,
            limit: nearbyLimit
        )
    }

    /// 首次进入、超过刷新间隔或用户主动刷新时才重新定位。
    static func shouldReloadNearby(force: Bool, lastAttemptAt: Date?, now: Date) -> Bool {
        guard !force else { return true }
        guard let lastAttemptAt else { return true }
        return now.timeIntervalSince(lastAttemptAt) >= nearbyRefreshInterval
    }

    /// 距离文案，对应 `formatHomeVenueDistance`。
    static func venueDistance(_ meters: Double?) -> String {
        guard let meters, meters > 0 else { return "" }
        if meters < 1000 { return "\(Int(meters.rounded()))m" }
        return String(format: "%.1fkm", meters / 1000)
    }

    /// 附近球房空态；奖励活动是否可领取决定是否引导领取 VIP。
    static func venueEmptyAction(_ rewardStatus: FavoriteVenueRewardStatus?) -> HomeVenueEmptyAction {
        FavoriteVenueReward.isClaimable(rewardStatus)
            ? HomeVenueEmptyAction(
                text: "添加球馆领取VIP",
                desc: "你可以提交常玩球馆，审核通过后VIP会自动到账。"
            )
            : HomeVenueEmptyAction(
                text: "添加附近球馆",
                desc: "你可以提交常去球馆，审核通过后会展示给附近球友。"
            )
    }

    private static func trimmed(_ value: String) -> String {
        value.trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
