import Foundation

/// 观赛列表标签：「好友」看好友与自己的对局（含进行中和已结束），「进行中/已结束」看公开大厅。
enum SpectatorLobbyTab: String, CaseIterable, Identifiable {
    case friends
    case live
    case finished

    var id: String { rawValue }

    var label: String {
        switch self {
        case .friends: "好友"
        case .live: "进行中"
        case .finished: "已结束"
        }
    }

    var scope: String {
        self == .friends ? "friends" : "hall"
    }

    /// 仅大厅使用；好友标签混合两种状态。
    var status: Int {
        switch self {
        case .friends: 0
        case .live: 1
        case .finished: 2
        }
    }

    var requiresLogin: Bool { self == .friends }
}

struct SpectatorLobbyQuery: Equatable {
    let scope: String
    let status: Int
}

struct SpectatorLobbyEmptyState: Equatable {
    let title: String
    let subtitle: String
}

struct SpectatorLobbySection: Equatable {
    let key: String
    let title: String
    let items: [SpectatorMatchCard]
}

struct SpectatorGameTypeFilter: Identifiable, Equatable {
    let value: Int
    let label: String

    var id: Int { value }
}

/// 对局卡片点击后的目的地；由页面映射成具体路由。
enum SpectatorMatchDestination: Equatable {
    case report
    case publicDetail
    case live
}

enum SpectatorLobbyRules {
    static let pageSize = 20
    static let homeLiveMatchLimit = 3

    private static let matchModeText = ["ranked": "排位", "practice": "练习"]

    static let gameTypeOptions: [SpectatorGameTypeFilter] = [SpectatorGameTypeFilter(value: 0, label: "全部球种")]
        + GameTypeCatalog.ordered.map { SpectatorGameTypeFilter(value: $0.value, label: $0.label) }

    static func tabs(isLoggedIn: Bool) -> [SpectatorLobbyTab] {
        SpectatorLobbyTab.allCases.filter { isLoggedIn || !$0.requiresLogin }
    }

    static func query(for tabValue: String) -> SpectatorLobbyQuery {
        let tab = SpectatorLobbyTab(rawValue: tabValue) ?? .live
        return SpectatorLobbyQuery(scope: tab.scope, status: tab.status)
    }

    static func listParams(
        scope: String = "hall",
        status: Int = 1,
        gameType: Int = 0,
        page: Int = 1,
        pageSize: Int = SpectatorLobbyRules.pageSize
    ) -> [String: String] {
        let resolvedScope = scope == "friends" ? "friends" : "hall"
        var params = [
            "scope": resolvedScope,
            "page": String(page),
            "page_size": String(pageSize),
        ]

        if resolvedScope == "hall" {
            params["status"] = status == 2 ? "2" : "1"
        }

        if gameType > 0 {
            params["game_type"] = String(gameType)
        }

        return params
    }

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

    static func statusText(status: Int, durationSeconds: Int64) -> String {
        if status == 2 { return "已结束" }
        return "进行中 \(formatDuration(durationSeconds))"
    }

    /// 列表卡片：双方各占一行；已结束时按比分标记胜负，平局与进行中不标记。
    static func normalize(_ match: SpectatorMatch, userId: Int64 = 0, now: Date = Date()) -> SpectatorMatchCard {
        let viewerId = max(userId, 0)
        let isLive = match.status != 2
        let scores = [match.player1Score, match.player2Score]

        let players = [1, 2].map { side -> SpectatorPlayer in
            let id = side == 1 ? match.player1Id : match.player2Id
            let name = side == 1 ? match.player1Name : match.player2Name
            let avatar = side == 1 ? match.player1Avatar : match.player2Avatar
            let index = side - 1
            let score = scores[index]
            let otherScore = scores[1 - index]
            var outcome: SpectatorPlayer.Outcome?
            if !isLive, score != otherScore {
                outcome = score > otherScore ? .winner : .loser
            }
            return SpectatorPlayer(
                id: id,
                name: name.isEmpty ? "选手\(side)" : name,
                avatar: avatar,
                score: score,
                isMe: viewerId > 0 && id == viewerId,
                outcome: outcome
            )
        }

        let endedAgo = !isLive ? relativeTime(match.endTime, now: now) : ""
        let gameTypeName = match.gameTypeName.isEmpty
            ? GameTypeCatalog.label(for: match.gameType)
            : match.gameTypeName
        let metaText = [gameTypeName, match.matchMode.flatMap { matchModeText[$0] } ?? ""]
            .filter { !$0.isEmpty }
            .joined(separator: " · ")
        let resolvedStatus = match.status == 0 ? 1 : match.status

        return SpectatorMatchCard(
            id: match.id,
            rawStatus: match.status,
            isLive: isLive,
            isMine: players.contains { $0.isMe },
            metaText: metaText,
            statusText: endedAgo.isEmpty
                ? statusText(status: resolvedStatus, durationSeconds: match.durationSeconds)
                : "\(endedAgo)结束",
            players: players
        )
    }

    /// 首页「正在进行」：去掉本人的当前对局（已在开局卡展示）再取前 3 场，卡片形态与观赛列表一致。
    static func buildHomeLiveMatches(
        _ list: [SpectatorMatch]?,
        userId: Int64 = 0,
        currentMatchId: Int64 = 0,
        now: Date = Date()
    ) -> [SpectatorMatchCard] {
        let currentId = max(currentMatchId, 0)
        return (list ?? [])
            .filter { currentId == 0 || $0.id != currentId }
            .prefix(homeLiveMatchLimit)
            .map { normalize($0, userId: userId, now: now) }
    }

    /// 好友标签混合两种状态，按「进行中 / 已结束」分组；大厅标签只有一种状态，不加分组标题。
    static func sections(_ list: [SpectatorMatchCard], tabValue: String) -> [SpectatorLobbySection] {
        guard tabValue == "friends" else {
            return list.isEmpty ? [] : [SpectatorLobbySection(key: tabValue, title: "", items: list)]
        }

        return [
            SpectatorLobbySection(key: "live", title: "进行中", items: list.filter(\.isLive)),
            SpectatorLobbySection(key: "finished", title: "已结束", items: list.filter { !$0.isLive }),
        ]
        .filter { !$0.items.isEmpty }
    }

    static func emptyState(scope: String = "hall", status: Int = 1, gameType: Int = 0) -> SpectatorLobbyEmptyState {
        let gameTypeLabel = GameTypeCatalog.ordered.first { $0.value == gameType }?.label ?? ""

        if !gameTypeLabel.isEmpty {
            return SpectatorLobbyEmptyState(
                title: "暂无\(gameTypeLabel)\(scope == "friends" ? "好友" : "")对局",
                subtitle: scope == "friends"
                    ? "好友参与该球种的对局会集中展示在这里"
                    : "该球种有新的公开对局后会出现在这里"
            )
        }

        if scope == "friends" {
            return SpectatorLobbyEmptyState(
                title: "暂无好友对局",
                subtitle: "好友参与的进行中和已结束对局会展示在这里"
            )
        }

        if status == 2 {
            return SpectatorLobbyEmptyState(
                title: "暂无已结束对局",
                subtitle: "有新的完赛记录后会出现在这里"
            )
        }

        return SpectatorLobbyEmptyState(
            title: "暂无正在进行的对局",
            subtitle: "公开对局开打后会出现在这里"
        )
    }

    static func shouldOpenPlaying(_ card: SpectatorMatchCard, userId: Int64, isLoggedIn: Bool) -> Bool {
        guard isLoggedIn, userId > 0, card.rawStatus == 1 else { return false }
        return card.players.contains { $0.id == userId }
    }

    /// 与小程序点击处理一致：已结束先看本人战报，否则公开详情；未结束进入对局。
    static func destination(
        for card: SpectatorMatchCard,
        userId: Int64,
        isLoggedIn: Bool
    ) -> SpectatorMatchDestination? {
        if card.rawStatus == 2 {
            guard card.id > 0 else { return nil }
            if userId > 0, card.players.contains(where: { $0.id == userId }) {
                return .report
            }
            return .publicDetail
        }
        return .live
    }

    /// 与 `utils/format.js` 的 formatRelativeTime 一致：空或无法解析返回空字符串。
    private static func relativeTime(_ text: String?, now: Date) -> String {
        guard let date = MatchTimeFormatting.parse(text) else { return "" }
        let seconds = Int(now.timeIntervalSince(date))
        if seconds < 60 { return "刚刚" }
        let minutes = seconds / 60
        if minutes < 60 { return "\(minutes)分钟前" }
        let hours = minutes / 60
        if hours < 24 { return "\(hours)小时前" }
        let days = hours / 24
        if days < 30 { return "\(days)天前" }
        let months = days / 30
        if months < 12 { return "\(months)个月前" }
        return "\(days / 365)年前"
    }
}
