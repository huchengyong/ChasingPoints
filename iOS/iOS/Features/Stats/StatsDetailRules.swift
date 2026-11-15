import Foundation

// 迁移自 `uniapp/utils/stats-detail.js` 与 `statsDetail.vue` 页内纯函数。

enum StatsLoadingMode: Equatable {
    case initial
    case refreshing
    case idle
}

struct StatsDurationValues: Equatable {
    let average: Int
    let fastest: Int
    let longest: Int
    let total: Int
}

struct StatsOpponentStrengthRow: Equatable, Identifiable {
    let index: Int
    let tierName: String
    let matches: Int
    let wins: Int
    let winRate: Double

    var id: Int { index }
}

struct StatsHighScoreSummary: Equatable {
    let personalBest: Int
    let monthBest: Int
}

struct StatsTrendCell: Equatable, Identifiable {
    let dateKey: String
    let day: Int
    let isCurrentMonth: Bool
    let winCount: Int
    let lossCount: Int
    let drawCount: Int
    let matchCount: Int

    var id: String { dateKey }
    var hasMatches: Bool { matchCount > 0 }
}

struct StatsTrendSummary: Equatable {
    let matchCount: Int
    let winCount: Int
    let lossCount: Int
    let drawCount: Int
    let winRate: Int
}

struct StatsTrendCalendar: Equatable {
    let monthKey: String
    let title: String
    let weekdays: [String]
    let summary: StatsTrendSummary
    let cells: [StatsTrendCell]
}

struct StatsRankDay: Equatable, Identifiable {
    let dateKey: String
    let day: Int
    let label: String
    let delta: Int
    let hasChange: Bool
    let isUp: Bool
    let isDown: Bool
    let heightPercent: Int
    let showTick: Bool

    var id: String { dateKey }
}

struct StatsRankSummary: Equatable {
    let netDelta: Int
    let upDays: Int
    let downDays: Int
    let maxGain: Int
    let maxLoss: Int
}

struct StatsRankDeltaChart: Equatable {
    let monthKey: String
    let title: String
    let maxAbsDelta: Int
    let summary: StatsRankSummary
    let days: [StatsRankDay]
}

struct StatsRankTrend: Equatable {
    let currentScore: Int
    let peakScore: Int
}

enum StatsDetailRules {
    static let weekdays = ["日", "一", "二", "三", "四", "五", "六"]

    static func loadingMode(hasLoadedOnce: Bool, isFetching: Bool) -> StatsLoadingMode {
        guard isFetching else { return .idle }
        return hasLoadedOnce ? .refreshing : .initial
    }

    static func shouldApplyResponse(requestId: Int, latestRequestId: Int) -> Bool {
        requestId == latestRequestId
    }

    static func normalizeDurationStats(_ stats: StatsDurationStats?) -> StatsDurationValues {
        StatsDurationValues(
            average: stats?.averageSeconds ?? 0,
            fastest: stats?.fastestSeconds ?? 0,
            longest: stats?.longestSeconds ?? 0,
            total: stats?.totalMatches ?? 0
        )
    }

    static func normalizeOpponentStrengthStats(_ list: [StatsOpponentStrengthItem]?) -> [StatsOpponentStrengthRow] {
        (list ?? []).enumerated().map { index, item in
            StatsOpponentStrengthRow(
                index: index,
                tierName: item.rankRange.isEmpty ? "段位\(String(index + 1))" : item.rankRange,
                matches: item.matches,
                wins: item.wins,
                winRate: item.winRate
            )
        }
    }

    static func normalizeGameTypeStats(_ list: [StatsGameTypeStats]?) -> [Int: StatsGameTypeStats] {
        var map: [Int: StatsGameTypeStats] = [:]
        for item in list ?? [] where GameTypeCatalog.isSupported(item.gameType) {
            map[item.gameType] = item
        }
        return map
    }

    /// 个人最佳取列表首项（服务端按分数倒序返回），本月最佳按当前自然月前缀筛选。
    static func normalizeHighScoreStats(_ list: [StatsHighScoreRecord]?, now: Date) -> StatsHighScoreSummary {
        let records = list ?? []
        let monthKey = AppFormat.monthKey(now)
        let monthBest = records.reduce(0) { best, item in
            guard !item.date.isEmpty, item.date.hasPrefix(monthKey) else { return best }
            return max(best, item.score)
        }
        return StatsHighScoreSummary(personalBest: records.first?.score ?? 0, monthBest: monthBest)
    }

    static func shiftMonthKey(_ monthKey: String, offset: Int, now: Date = Date()) -> String {
        let calendar = AppFormat.beijingCalendar
        let parts = monthComponents(monthKey) ?? monthComponents(AppFormat.monthKey(now))!
        guard let date = calendar.date(from: DateComponents(year: parts.year, month: parts.month, day: 1)),
              let shifted = calendar.date(byAdding: .month, value: offset, to: date) else {
            return AppFormat.monthKey(now)
        }
        return AppFormat.monthKey(shifted)
    }

    static func trendCalendar(
        matches: [StatsTrendPoint],
        monthKey: String,
        now: Date = Date()
    ) -> StatsTrendCalendar {
        let resolved = resolveMonth(matches.map(\.date), monthKey: monthKey, now: now)
        let currentKey = String(format: "%04d-%02d", resolved.year, resolved.month)

        var byDay: [String: (wins: Int, losses: Int, draws: Int, count: Int)] = [:]
        var winCount = 0
        var lossCount = 0
        var drawCount = 0

        for item in matches {
            guard let date = AppFormat.parseDateTime(item.date),
                  AppFormat.monthKey(date) == currentKey else { continue }
            let key = AppFormat.dayKey(date)
            var stats = byDay[key] ?? (0, 0, 0, 0)
            switch item.result {
            case 1:
                stats.wins += 1
                winCount += 1
            case 2:
                stats.losses += 1
                lossCount += 1
            default:
                stats.draws += 1
                drawCount += 1
            }
            stats.count += 1
            byDay[key] = stats
        }

        let matchCount = winCount + lossCount + drawCount
        let summary = StatsTrendSummary(
            matchCount: matchCount,
            winCount: winCount,
            lossCount: lossCount,
            drawCount: drawCount,
            winRate: matchCount > 0 ? Int((Double(winCount) / Double(matchCount) * 100).rounded()) : 0
        )

        let calendar = AppFormat.beijingCalendar
        let firstDay = calendar.date(from: DateComponents(year: resolved.year, month: resolved.month, day: 1))!
        // 周首为周日（Calendar 的 weekday 1 = 周日），与小程序 `getDay()` 一致。
        let weekday = calendar.component(.weekday, from: firstDay)
        let startDate = calendar.date(byAdding: .day, value: -(weekday - 1), to: firstDay)!

        let cells = (0..<42).map { index -> StatsTrendCell in
            let date = calendar.date(byAdding: .day, value: index, to: startDate)!
            let key = AppFormat.dayKey(date)
            let stats = byDay[key] ?? (0, 0, 0, 0)
            return StatsTrendCell(
                dateKey: key,
                day: calendar.component(.day, from: date),
                isCurrentMonth: AppFormat.monthKey(date) == currentKey,
                winCount: stats.wins,
                lossCount: stats.losses,
                drawCount: stats.draws,
                matchCount: stats.count
            )
        }

        return StatsTrendCalendar(
            monthKey: currentKey,
            title: "\(String(resolved.year))年\(String(resolved.month))月",
            weekdays: weekdays,
            summary: summary,
            cells: cells
        )
    }

    static func rankDeltaChart(
        points: [StatsTrendPoint],
        monthKey: String,
        now: Date = Date()
    ) -> StatsRankDeltaChart {
        let changes = normalizeRankChanges(points)
        let resolved = resolveMonth(changes.map(\.date), monthKey: monthKey, now: now)
        let currentKey = String(format: "%04d-%02d", resolved.year, resolved.month)
        let calendar = AppFormat.beijingCalendar
        let firstDay = calendar.date(from: DateComponents(year: resolved.year, month: resolved.month, day: 1))!
        let daysInMonth = calendar.range(of: .day, in: .month, for: firstDay)?.count ?? 0

        var byDate: [String: Int] = [:]
        for item in changes {
            guard let date = AppFormat.parseDateTime(item.date),
                  AppFormat.monthKey(date) == currentKey else { continue }
            let key = AppFormat.dayKey(date)
            byDate[key, default: 0] += item.delta
        }

        let deltas = Array(byDate.values)
        let maxAbsDelta = deltas.reduce(0) { max($0, abs($1)) }
        let days = (1...max(daysInMonth, 1)).map { day -> StatsRankDay in
            let dateKey = "\(currentKey)-\(String(format: "%02d", day))"
            let delta = byDate[dateKey] ?? 0
            let absDelta = abs(delta)
            return StatsRankDay(
                dateKey: dateKey,
                day: day,
                label: "\(String(day))日",
                delta: delta,
                hasChange: delta != 0,
                isUp: delta > 0,
                isDown: delta < 0,
                heightPercent: maxAbsDelta > 0
                    ? max(8, Int((Double(absDelta) / Double(maxAbsDelta) * 100).rounded()))
                    : 0,
                showTick: day == 1 || day == 10 || day == 20 || day == daysInMonth
            )
        }

        return StatsRankDeltaChart(
            monthKey: currentKey,
            title: "\(String(resolved.year))年\(String(resolved.month))月",
            maxAbsDelta: maxAbsDelta,
            summary: StatsRankSummary(
                netDelta: deltas.reduce(0, +),
                upDays: deltas.filter { $0 > 0 }.count,
                downDays: deltas.filter { $0 < 0 }.count,
                maxGain: deltas.reduce(0) { max($0, $1) },
                maxLoss: deltas.reduce(0) { min($0, $1) }
            ),
            days: days
        )
    }

    static func rankTrend(points: [StatsTrendPoint]) -> StatsRankTrend {
        let scores = points.map { $0.rankScore ?? 0 }
        return StatsRankTrend(currentScore: scores.first ?? 0, peakScore: scores.max() ?? 0)
    }

    static func formatDuration(seconds: Int) -> String {
        guard seconds > 0 else { return "0分" }
        let totalMinutes = seconds / 60
        if totalMinutes < 1 { return "\(String(seconds))秒" }
        if totalMinutes < 60 { return "\(String(totalMinutes))分" }

        let hours = totalMinutes / 60
        let remainMinutes = totalMinutes % 60
        if hours < 24 {
            return remainMinutes > 0
                ? "\(String(hours))小时\(String(remainMinutes))分"
                : "\(String(hours))小时"
        }

        let days = hours / 24
        let remainHours = hours % 24
        return remainHours > 0
            ? "\(String(days))天\(String(remainHours))小时"
            : "\(String(days))天"
    }

    /// 对应页内 `formatPercent`：保留两位小数，去掉无意义的尾零，整数不带小数点。
    static func formatPercent(_ value: Double) -> String {
        guard value.isFinite else { return "0" }
        let rounded = (value * 100).rounded() / 100
        if rounded == rounded.rounded() {
            return String(Int(rounded))
        }
        var text = String(format: "%.2f", rounded)
        while text.last == "0" { text.removeLast() }
        if text.last == "." { text.removeLast() }
        return text
    }

    // MARK: - 私有

    private static func normalizeRankChanges(_ points: [StatsTrendPoint]) -> [(date: String, delta: Int)] {
        points.enumerated().map { index, item in
            let previous = index + 1 < points.count ? points[index + 1] : nil
            let previousScore = previous?.rankScore ?? item.rankScore ?? 0
            let score = item.rankScore ?? 0
            return (item.date, score - previousScore)
        }
    }

    private static func resolveMonth(
        _ dates: [String],
        monthKey: String,
        now: Date
    ) -> (year: Int, month: Int) {
        if let parts = monthComponents(monthKey) { return parts }

        let parsed = dates.lazy.compactMap { AppFormat.parseDateTime($0) }.first
        let date = parsed ?? now
        let parts = AppFormat.beijingCalendar.dateComponents([.year, .month], from: date)
        return (parts.year ?? 0, parts.month ?? 0)
    }

    private static func monthComponents(_ monthKey: String) -> (year: Int, month: Int)? {
        guard monthKey.range(of: #"^\d{4}-\d{2}$"#, options: .regularExpression) != nil else { return nil }
        let parts = monthKey.split(separator: "-")
        guard parts.count == 2,
              let year = Int(parts[0]),
              let month = Int(parts[1]),
              (1...12).contains(month) else { return nil }
        return (year, month)
    }
}
