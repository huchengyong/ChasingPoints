import Foundation

/// 会员成长卡：迁移小程序 `member-center.js` 的 `resolveMemberGrowthCard` 及其依赖。
struct MemberGrowthCard: Equatable {
    let statusText: String
    let level: Int
    let levelLabel: String
    let growthPoints: Int
    let growthPointsText: String
    let todayGrowthCount: Int
    let dailyCap: Int
    let todayProgressText: String
    let frozen: Bool
    let description: String
    let nextLevelText: String
    /// 已是最高成长等级时为 nil。
    let nextLevel: Int?
    let progressPercent: Int

    private static let defaultDailyCap = 8
    private static let levels: [(level: Int, threshold: Int)] = [
        (1, 0), (2, 10), (3, 40), (4, 120), (5, 300),
    ]

    /// `status` 为 nil 等同 JS 传入 `{}`。
    static func resolve(_ status: MemberStatus?, now: Date) -> MemberGrowthCard {
        let expiresAtText = (status?.memberExpiresAt ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let expiresAt = ServerTime.utc8(expiresAtText)
        let isActive = status?.isActive == true && expiresAt.map { $0 > now } == true

        let growthPoints = safeInteger(status?.growthPoints)
        let dailyCap = resolveDailyCap(status?.growthDailyCap)
        let todayGrowthCount = min(dailyCap, safeInteger(status?.todayGrowthCount))
        let currentLevel = resolveCurrentLevel(growthPoints: growthPoints, explicitLevel: status?.growthLevel)
        let nextLevel = resolveNextLevel(currentLevel: currentLevel, status: status)
        let explicitRemaining = safeInteger(status?.remainingGrowthPoints)
        let remainingPoints = explicitRemaining > 0
            ? explicitRemaining
            : (nextLevel.map { max(0, $0.threshold - growthPoints) } ?? 0)
        let frozen = !isActive && !expiresAtText.isEmpty
        let hasGrowthProfile = frozen || growthPoints > 0 || todayGrowthCount > 0

        var statusText = "待开启"
        var description = "开通VIP后，完成真实对局可累计成长。"
        var nextLevelText = "开通VIP后，完成真实对局可累计成长"

        if isActive {
            statusText = "成长中"
            description = "当前已累计 \(growthPoints) 点成长值，今日已计入 \(todayGrowthCount)/\(dailyCap) 场真实对局。"
            nextLevelText = nextLevel.map { "距 Lv\($0.level) 还差 \(remainingPoints) 场" }
                ?? "当前已达到最高成长等级"
        } else if frozen && hasGrowthProfile {
            statusText = "成长已冻结"
            description = "VIP已于 \(expiresAtText) 到期，当前成长已冻结，续开后恢复继续成长。"
            nextLevelText = nextLevel.map { "续开后，再完成 \(remainingPoints) 场可升到 Lv\($0.level)" }
                ?? "续开后继续保留当前最高成长等级"
        }

        return MemberGrowthCard(
            statusText: statusText,
            level: currentLevel.level,
            levelLabel: "Lv\(currentLevel.level)",
            growthPoints: growthPoints,
            growthPointsText: "成长值 \(growthPoints)",
            todayGrowthCount: todayGrowthCount,
            dailyCap: dailyCap,
            todayProgressText: "今日已计入 \(todayGrowthCount)/\(dailyCap)",
            frozen: frozen,
            description: description,
            nextLevelText: nextLevelText,
            nextLevel: nextLevel?.level,
            progressPercent: resolveProgressPercent(
                growthPoints: growthPoints, currentLevel: currentLevel, nextLevel: nextLevel
            )
        )
    }

    /// JS `toSafeInteger`：非有限值或负数按 0，其余向下取整。
    private static func safeInteger(_ value: Int?) -> Int {
        max(0, value ?? 0)
    }

    private static func resolveDailyCap(_ value: Int?) -> Int {
        let normalized = safeInteger(value)
        return max(1, normalized == 0 ? defaultDailyCap : normalized)
    }

    private static func resolveCurrentLevel(
        growthPoints: Int,
        explicitLevel: Int?
    ) -> (level: Int, threshold: Int) {
        let normalizedLevel = safeInteger(explicitLevel)
        if normalizedLevel > 0 {
            if let matched = levels.first(where: { $0.level == normalizedLevel }) { return matched }
            return (normalizedLevel, growthPoints)
        }

        var current = levels[0]
        for item in levels {
            guard growthPoints >= item.threshold else { break }
            current = item
        }
        return current
    }

    private static func resolveNextLevel(
        currentLevel: (level: Int, threshold: Int),
        status: MemberStatus?
    ) -> (level: Int, threshold: Int)? {
        let explicitLevel = safeInteger(status?.nextGrowthLevel)
        if explicitLevel > 0 {
            if let matched = levels.first(where: { $0.level == explicitLevel }) { return matched }
            return (explicitLevel, safeInteger(status?.nextGrowthLevelPoints))
        }
        return levels.first { $0.level == currentLevel.level + 1 }
    }

    private static func resolveProgressPercent(
        growthPoints: Int,
        currentLevel: (level: Int, threshold: Int),
        nextLevel: (level: Int, threshold: Int)?
    ) -> Int {
        guard let nextLevel else { return 100 }
        let segmentPoints = nextLevel.threshold - currentLevel.threshold
        guard segmentPoints > 0 else { return 100 }
        let completed = max(0, growthPoints - currentLevel.threshold)
        let percent = (Double(completed) / Double(segmentPoints) * 100).rounded()
        return min(100, Int(percent))
    }
}

/// 服务端时间文本解析：`yyyy-MM-dd HH:mm:ss` 按北京时间（+08:00）解释。
enum ServerTime {
    static func utc8(_ text: String?) -> Date? {
        guard let text else { return nil }
        let normalized = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalized.isEmpty else { return nil }
        return beijingFormatter().date(from: normalized)
    }

    /// 上述格式优先，其次按 ISO 8601 解析（对应页面里 `parseMemberExpiresAt` 的兜底）。
    static func flexible(_ text: String?) -> Date? {
        if let date = utc8(text) { return date }
        guard let text else { return nil }
        let normalized = text.trimmingCharacters(in: .whitespacesAndNewlines)
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter.date(from: normalized)
    }

    private static func beijingFormatter() -> DateFormatter {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(identifier: "Asia/Shanghai")
        formatter.dateFormat = "yyyy-MM-dd HH:mm:ss"
        return formatter
    }
}
