import Foundation

// 迁移自 `uniapp/utils/honor-wall.js`，荣誉墙的展示与加载判定。

enum HonorWallTab: String, CaseIterable, Equatable, Identifiable {
    case career
    case season
    case history

    var id: String { rawValue }

    var label: String {
        switch self {
        case .career: "生涯成就"
        case .season: "当前赛季"
        case .history: "历届荣誉"
        }
    }
}

struct HonorWallOptions: Equatable {
    let userId: Int64
    let gameType: Int
    let tab: HonorWallTab
    let historySeasonId: Int64
}

struct HonorWallSummaryItem: Equatable, Identifiable {
    let key: String
    let label: String
    let value: String

    var id: String { key }
}

struct HonorWallUpcomingItem: Equatable, Identifiable {
    let achievement: AchievementDef
    let progress: Int
    let threshold: Int
    let progressPercent: Int
    let progressText: String
    let remainingText: String

    var id: Int64 { achievement.id }
}

struct HonorWallUpcomingSection: Equatable {
    let hidden: Bool
    let completed: Bool
    let items: [HonorWallUpcomingItem]
}

struct HonorWallChallengeItem: Equatable, Identifiable {
    let key: String
    let name: String
    let description: String
    let progress: Int
    let threshold: Int
    let completed: Bool
    /// 小程序 static 目录里的挑战图标路径；空串表示用 emoji。
    let iconPath: String
    let progressPercent: Int
    let progressText: String
    let remainingText: String

    var id: String { key }
}

enum HonorWallSeasonMode: Equatable {
    case unavailable
    case notStarted
    case active
}

struct HonorWallSeasonState: Equatable {
    let mode: HonorWallSeasonMode
    let title: String
    let description: String
}

struct HonorWallRecentHonor: Equatable, Identifiable {
    let id: Int64
    let sourceType: String
    let name: String
    let sourceLabel: String
    let detailText: String
    let earnedAt: String
}

struct HonorWallSeasonOption: Equatable, Identifiable {
    let id: Int64
    let label: String
}

enum HonorWallLoadErrorKind: Equatable {
    case network
    case permission
    case session
    case server
}

struct HonorWallLoadError: Equatable {
    let kind: HonorWallLoadErrorKind
    let title: String
    let description: String
    let showRetry: Bool
}

struct HonorWallSeasonRollover: Equatable {
    let title: String
    let content: String
    let confirmText: String
    let cancelText: String
    let showCancel: Bool
    /// 可选的赛季回顾路径；空串表示不提供回顾入口。
    let reportURL: String
}

/// 只认最后一次请求，旧响应晚到时直接丢弃。
struct LatestRequestGuard {
    private var latestRequestId = 0

    mutating func next() -> Int {
        latestRequestId += 1
        return latestRequestId
    }

    func isLatest(_ requestId: Int) -> Bool {
        requestId == latestRequestId
    }
}

enum HonorWallRules {
    static let seasonReportPath = "/subPages/season/report"

    private static let supportedGameTypes: Set<Int> = [1, 2, 3, 4]

    private static let sourceLabels = [
        "achievement": "生涯成就",
        "season": "赛季荣誉",
        "tournament": "赛事荣誉",
    ]

    private static let gameTypeLabels = [
        1: "斯诺克",
        2: "九球追分",
        3: "中式八球",
        4: "美式九球",
    ]

    private static let challengeIconPaths = [
        "season_matches_20": "/static/images/achievements/season_matches_20.png",
        "season_wins_10": "/static/images/achievements/season_wins_10.png",
        "season_tournament_finish_1": "/static/images/achievements/season_tournament_finish_1.png",
    ]

    static func normalizeGameType(_ value: Int?) -> Int {
        guard let value, supportedGameTypes.contains(value) else { return 3 }
        return value
    }

    static func normalizeOptions(
        userId: Int64 = 0,
        gameType: Int? = nil,
        tab: String? = nil,
        historySeasonId: Int64 = 0
    ) -> HonorWallOptions {
        HonorWallOptions(
            userId: userId,
            gameType: normalizeGameType(gameType),
            tab: HonorWallTab(rawValue: tab ?? "") ?? .career,
            historySeasonId: historySeasonId
        )
    }

    /// 取代小程序页面路径：只返回跳转该页需要的路由参数，有值才带用户与赛季。
    static func routeQuery(
        userId: Int64 = 0,
        gameType: Int = 3,
        tab: HonorWallTab = .career,
        historySeasonId: Int64 = 0
    ) -> [String: String] {
        var query = [
            "game_type": String(normalizeGameType(gameType)),
            "tab": tab.rawValue,
        ]
        if userId > 0 { query["user_id"] = String(userId) }
        if historySeasonId > 0 { query["history_season_id"] = String(historySeasonId) }
        return query
    }

    static func tabs(viewerScope: String) -> [HonorWallTab] {
        viewerScope == "friend" ? [.career, .history] : HonorWallTab.allCases
    }

    static func careerSummaryItems(
        _ summary: HonorWallSummary,
        gameType: Int,
        gameTypeLabel: String
    ) -> [HonorWallSummaryItem] {
        let selected = normalizeGameType(gameType)
        let specialtyLabel = gameTypeLabel.isEmpty ? (gameTypeLabels[selected] ?? "") : gameTypeLabel

        return [
            HonorWallSummaryItem(
                key: "universal",
                label: "通用成就",
                value: "\(String(max(0, summary.universalUnlocked)))/\(String(max(0, summary.universalTotal)))"
            ),
            HonorWallSummaryItem(
                key: "specialty",
                label: "\(specialtyLabel)专精",
                value: "\(String(max(0, summary.specialtyUnlocked)))/\(String(max(0, summary.specialtyTotal)))"
            ),
            HonorWallSummaryItem(
                key: "season",
                label: "赛季荣誉",
                value: String(max(0, summary.seasonHonors))
            ),
            HonorWallSummaryItem(
                key: "tournament",
                label: "赛事荣誉",
                value: String(max(0, summary.tournamentHonors))
            ),
        ]
    }

    static func upcomingSection(
        _ achievements: [AchievementDef],
        gameType: Int,
        viewerScope: String
    ) -> HonorWallUpcomingSection {
        if viewerScope == "friend" {
            return HonorWallUpcomingSection(hidden: true, completed: false, items: [])
        }

        let selected = normalizeGameType(gameType)
        let applicable = achievements.filter { $0.gameType == 0 || $0.gameType == selected }
        let locked = applicable.enumerated().compactMap { index, item -> (order: Int, item: HonorWallUpcomingItem)? in
            guard !item.unlocked, item.threshold > 0 else { return nil }
            let progress = max(0, item.progress)
            let threshold = max(0, item.threshold)
            return (
                order: index,
                item: HonorWallUpcomingItem(
                    achievement: item,
                    progress: progress,
                    threshold: threshold,
                    progressPercent: progressPercent(progress: progress, threshold: threshold),
                    progressText: "\(String(progress))/\(String(threshold))",
                    remainingText: "还差 \(String(max(0, threshold - progress))) 达成"
                )
            )
        }
        .sorted { left, right in
            if left.item.progressPercent != right.item.progressPercent {
                return left.item.progressPercent > right.item.progressPercent
            }
            if left.order != right.order { return left.order < right.order }
            return left.item.id < right.item.id
        }

        return HonorWallUpcomingSection(
            hidden: false,
            completed: !applicable.isEmpty && locked.isEmpty,
            items: locked.prefix(3).map(\.item)
        )
    }

    static func upcomingEmptyText(_ section: HonorWallUpcomingSection, gameTypeLabel: String) -> String {
        section.completed
            ? "通用成就与\(gameTypeLabel)绝技已全部达成"
            : "暂无可追踪的成就目标"
    }

    static func loadError(category: LoadFailure.Category) -> HonorWallLoadError {
        switch category {
        case .session:
            return HonorWallLoadError(
                kind: .session,
                title: "登录状态已失效",
                description: "请重新登录后查看荣誉墙",
                showRetry: false
            )
        case .network:
            return HonorWallLoadError(
                kind: .network,
                title: "荣誉墙暂时没加载出来",
                description: "请检查网络后重试，已有荣誉不会受到影响",
                showRetry: true
            )
        case .permission:
            return HonorWallLoadError(
                kind: .permission,
                title: "暂无查看权限",
                description: "仅本人或好友可以查看该荣誉墙",
                showRetry: false
            )
        default:
            return HonorWallLoadError(
                kind: .server,
                title: "荣誉墙暂时无法加载",
                description: "服务开小差了，请稍后重试",
                showRetry: true
            )
        }
    }

    static func progressPercent(
        progress: Int = 0,
        threshold: Int = 0,
        completed: Bool = false,
        unlocked: Bool = false
    ) -> Int {
        if completed || unlocked { return 100 }
        guard threshold > 0 else { return 0 }
        let percent = ((Double(max(0, progress)) / Double(threshold)) * 100).rounded()
        return min(100, Int(percent))
    }

    static func challengeItem(_ challenge: SeasonChallengeInfo) -> HonorWallChallengeItem {
        let progress = max(0, challenge.progress)
        let threshold = max(0, challenge.threshold)
        let completed = challenge.completed || (threshold > 0 && progress >= threshold)

        return HonorWallChallengeItem(
            key: challenge.key,
            name: challenge.name,
            description: challenge.description,
            progress: progress,
            threshold: threshold,
            completed: completed,
            iconPath: challengeIconPaths[challenge.key] ?? "",
            progressPercent: progressPercent(progress: progress, threshold: threshold, completed: completed),
            progressText: completed ? "已完成" : "\(String(progress))/\(String(threshold))",
            remainingText: completed ? "本赛季已达成" : "还差 \(String(max(0, threshold - progress))) 完成"
        )
    }

    static func currentSeasonState(_ currentSeason: HonorWallSeasonInfo?, seasonState: String) -> HonorWallSeasonState {
        let normalizedState = seasonState.trimmingCharacters(in: .whitespacesAndNewlines)
        let hasCurrentSeason = currentSeason.map { $0.seasonId > 0 } ?? false

        if normalizedState == "unavailable" {
            return HonorWallSeasonState(
                mode: .unavailable,
                title: "赛季数据更新中",
                description: "赛季排期正在校验，生涯成就与历届荣誉不会受到影响。"
            )
        }
        if normalizedState == "not_started" || !hasCurrentSeason {
            return HonorWallSeasonState(
                mode: .notStarted,
                title: "赛季尚未开启",
                description: "首个赛季开启后，这里会展示赛季挑战与当前进度。"
            )
        }
        let name = (currentSeason?.seasonName ?? "").isEmpty ? "当前赛季" : (currentSeason?.seasonName ?? "")
        return HonorWallSeasonState(
            mode: .active,
            title: name,
            description: "挑战按当前赛季与球种独立累计，新赛季会从零开始。"
        )
    }

    static func recentHonor(_ item: HonorItem) -> HonorWallRecentHonor {
        let sourceLabel = sourceLabels[item.sourceType] ?? "荣誉"
        let detailText: String
        if let title = item.rewardTitleName, !title.isEmpty {
            detailText = "同时获得称号「\(title)」"
        } else if let description = item.description, !description.isEmpty {
            detailText = description
        } else if let sourceName = item.sourceRefName, !sourceName.isEmpty {
            detailText = sourceName
        } else {
            detailText = "已永久记录"
        }

        return HonorWallRecentHonor(
            id: item.id,
            sourceType: item.sourceType,
            name: item.name,
            sourceLabel: sourceLabel,
            detailText: detailText,
            earnedAt: item.earnedAt
        )
    }

    static func historySeasonOptions(_ honors: [HonorItem]) -> [HonorWallSeasonOption] {
        var seen: Set<Int64> = []
        var result: [HonorWallSeasonOption] = []
        for item in honors {
            guard item.sourceType == "season", let seasonId = item.sourceRefId, seasonId > 0, !seen.contains(seasonId) else {
                continue
            }
            seen.insert(seasonId)
            let name = item.sourceRefName ?? item.name
            result.append(HonorWallSeasonOption(
                id: seasonId,
                label: name.isEmpty ? "赛季 \(String(seasonId))" : name
            ))
        }
        return result
    }

    static func seasonDateText(_ season: HonorWallSeasonInfo?) -> String {
        guard let season else { return "" }
        return "\(formatDate(season.startDate)) - \(formatDate(season.endDate))"
    }

    static func formatDate(_ value: String) -> String {
        String(value.prefix(10)).replacingOccurrences(of: "-", with: ".")
    }

    static func latestUnreadSeasonRollover(_ list: [NotificationInfo]) -> NotificationInfo? {
        list
            .filter { $0.type == "season_rollover" && !$0.isRead }
            .sorted { left, right in
                if left.createdAt != right.createdAt { return left.createdAt > right.createdAt }
                return left.id > right.id
            }
            .first
    }

    static func seasonRollover(
        _ notification: NotificationInfo,
        allowReport: Bool = true
    ) -> HonorWallSeasonRollover {
        let data = decodeRolloverData(notification.data)
        let fromName = nonEmpty(data?.fromSeasonName) ?? "上一赛季"
        let toName = nonEmpty(data?.toSeasonName) ?? ""
        let intermission = (data?.intermission ?? false) || ((data?.toSeasonId ?? 0) == 0)

        var lines = [
            "\(fromName) 已归档，生涯成就保持不变。",
            "赛季挑战已归档 \(String(data?.challengeArchivedCount ?? 0)) 项，其中完成 \(String(data?.challengeCompletedCount ?? 0)) 项。",
            "赛季荣誉已记录 \(String(data?.seasonTitleCount ?? 0)) 枚。",
        ]
        if intermission {
            lines.append("当前处于赛季间歇期，新赛季开启后会自动更新。")
        } else {
            lines.append("\(toName.isEmpty ? "新赛季" : toName) 已自动开启，赛季挑战从零开始。")
        }

        let url = data?.url ?? ""
        let reportURL = allowReport && url.hasPrefix("\(seasonReportPath)?") ? url : ""

        return HonorWallSeasonRollover(
            title: intermission ? "\(fromName) 已结算" : "\(toName.isEmpty ? "新赛季" : toName) 已开启",
            content: lines.joined(separator: "\n"),
            confirmText: reportURL.isEmpty ? "知道了" : "查看回顾",
            cancelText: "知道了",
            showCancel: !reportURL.isEmpty,
            reportURL: reportURL
        )
    }

    private static func nonEmpty(_ value: String?) -> String? {
        guard let value, !value.isEmpty else { return nil }
        return value
    }
}

private struct SeasonRolloverData: Decodable {
    let fromSeasonName: String?
    let toSeasonId: Int64?
    let toSeasonName: String?
    let intermission: Bool?
    let challengeArchivedCount: Int?
    let challengeCompletedCount: Int?
    let seasonTitleCount: Int?
    let url: String?
}

private func decodeRolloverData(_ raw: String) -> SeasonRolloverData? {
    guard let data = raw.data(using: .utf8) else { return nil }
    let decoder = JSONDecoder()
    decoder.keyDecodingStrategy = .convertFromSnakeCase
    return try? decoder.decode(SeasonRolloverData.self, from: data)
}
