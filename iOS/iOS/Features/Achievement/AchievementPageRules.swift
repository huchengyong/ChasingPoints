import Foundation

// 迁移自 `uniapp/utils/achievement-page.js`，成就与称号的展示规则。

struct AchievementCategoryGroup: Equatable, Identifiable {
    let key: String
    let label: String

    var id: String { key }
}

struct AchievementGroup: Equatable, Identifiable {
    let key: String
    let label: String
    let items: [AchievementDef]

    var id: String { key }
}

enum TitleSelectionAction: Equatable {
    case close
    case equip(titleId: Int64)
    case unequip(titleId: Int64)
}

enum AchievementPageRules {
    static let categoryGroups: [AchievementCategoryGroup] = [
        AchievementCategoryGroup(key: "match", label: "对局"),
        AchievementCategoryGroup(key: "wins", label: "胜场"),
        AchievementCategoryGroup(key: "streak", label: "连胜"),
        AchievementCategoryGroup(key: "tournament", label: "赛事历程"),
    ]

    private static let categoryLabelMap = [
        "wins": "胜场",
        "streak": "连胜",
        "special": "特殊",
        "match": "对局",
        "tournament": "赛事",
        "social": "社交",
    ]

    private static let categoryEmojiMap = [
        "wins": "🏅",
        "streak": "🔥",
        "special": "⭐",
        "match": "🎱",
        "tournament": "🏆",
        "social": "👥",
    ]

    private static let gameTypeLabelMap = [
        0: "通用",
        1: "斯诺克",
        2: "九球追分",
        3: "中式八球",
        4: "美式九球",
    ]

    private static let gameTypeEmojiMap = [
        1: "🔴",
        2: "🎯",
        3: "🎱",
        4: "9️⃣",
    ]

    private static let titleSourceMap = [
        "achievement": "成就",
        "season": "赛季",
        "tournament": "赛事",
        "成就": "成就",
        "赛季": "赛季",
        "赛事": "赛事",
    ]

    private static let titleSourceClassMap = [
        "achievement": "achievement",
        "season": "season",
        "tournament": "tournament",
        "成就": "achievement",
        "赛季": "season",
        "赛事": "tournament",
    ]

    private static let titleSourceDescriptionMap = [
        "achievement": "生涯成就",
        "season": "赛季荣誉",
        "tournament": "赛事荣誉",
        "成就": "生涯成就",
        "赛季": "赛季荣誉",
        "赛事": "赛事荣誉",
    ]

    static func categoryLabel(_ category: String?) -> String {
        if let label = categoryLabelMap[category ?? ""], !label.isEmpty { return label }
        let text = category ?? ""
        return text.isEmpty ? "其他" : text
    }

    static func categoryEmoji(_ category: String?) -> String {
        categoryEmojiMap[category ?? ""] ?? "🎯"
    }

    static func gameTypeLabel(_ gameType: Int) -> String {
        gameTypeLabelMap[gameType] ?? "通用"
    }

    static func fallbackEmoji(_ achievement: AchievementDef) -> String {
        gameTypeEmojiMap[achievement.gameType] ?? categoryEmoji(achievement.category)
    }

    /// 按后端分类分组：已知分类按固定顺序在前，未知分类保持首次出现顺序排在后面。
    static func groupAchievementsByCategory(_ list: [AchievementDef]) -> [AchievementGroup] {
        var order: [String] = []
        var bucket: [String: [AchievementDef]] = [:]
        for item in list {
            let key = item.category.isEmpty ? "other" : item.category
            if bucket[key] == nil { order.append(key) }
            bucket[key, default: []].append(item)
        }

        var groups = categoryGroups.compactMap { group -> AchievementGroup? in
            guard let items = bucket[group.key] else { return nil }
            return AchievementGroup(key: group.key, label: group.label, items: items)
        }
        let knownKeys = Set(categoryGroups.map(\.key))
        for key in order where !knownKeys.contains(key) {
            groups.append(AchievementGroup(key: key, label: categoryLabel(key), items: bucket[key] ?? []))
        }
        return groups
    }

    /// 通用里程碑：`game_type == 0`，且不展示特殊分类。
    static func groupUniversalAchievements(_ list: [AchievementDef]) -> [AchievementGroup] {
        groupAchievementsByCategory(list.filter { $0.gameType == 0 })
            .filter { $0.key != "special" }
    }

    static func filterSpecialtyAchievements(
        _ list: [AchievementDef],
        gameType: Int,
        unlockedOnly: Bool = false
    ) -> [AchievementDef] {
        let selected = (1...4).contains(gameType) ? gameType : 3
        return list.filter { item in
            item.gameType == selected && (!unlockedOnly || item.unlocked)
        }
    }

    static func titleSourceLabel(_ source: String?) -> String {
        if let label = titleSourceMap[source ?? ""], !label.isEmpty { return label }
        let text = source ?? ""
        return text.isEmpty ? "其他" : text
    }

    static func titleSourceClass(_ source: String?) -> String {
        titleSourceClassMap[source ?? ""] ?? "default"
    }

    static func titleSourceDescription(_ title: TitleInfo) -> String {
        let source = title.sourceType ?? title.source
        let label = titleSourceDescriptionMap[source] ?? titleSourceLabel(source)
        let sourceName = (title.sourceRefName ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        return sourceName.isEmpty ? label : "\(label) · \(sourceName)"
    }

    /// 已佩戴称号排到首位，其余保持接口顺序。
    static func sortTitleOptions(_ list: [TitleInfo]) -> [TitleInfo] {
        guard let index = list.firstIndex(where: \.equipped), index > 0 else { return list }
        var result = list
        let equipped = result.remove(at: index)
        result.insert(equipped, at: 0)
        return result
    }

    /// 荣誉项图标：小程序只有这三类，其他类型按生涯成就处理。
    static func honorEmoji(_ sourceType: String) -> String {
        switch sourceType {
        case "season": "👑"
        case "tournament": "🏆"
        default: "🏅"
        }
    }

    /// 赛季挑战没有 CDN 图标时的兜底 emoji。
    static func challengeEmoji(_ key: String) -> String {
        if key.contains("wins") { return "🔥" }
        if key.contains("tournament") { return "🏆" }
        return "🎱"
    }

    /// 成就详情的进度：阈值缺失时按 1 计算，与小程序 `detail.vue` 一致。
    static func detailProgressPercent(progress: Int, threshold: Int, unlocked: Bool) -> Int {
        if unlocked { return 100 }
        let safeThreshold = threshold > 0 ? threshold : 1
        let percent = (Double(max(0, progress)) / Double(safeThreshold) * 100).rounded()
        return min(100, Int(percent))
    }

    static func resolveTitleSelection(currentTitleId: Int64, selectedTitleId: Int64) -> TitleSelectionAction {
        let current = max(0, currentTitleId)
        let selected = max(0, selectedTitleId)

        if current == selected { return .close }
        if selected > 0 { return .equip(titleId: selected) }
        return current > 0 ? .unequip(titleId: current) : .close
    }
}
