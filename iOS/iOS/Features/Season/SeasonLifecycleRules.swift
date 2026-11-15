import Foundation

/// 迁移自 `uniapp/utils/season-lifecycle.js`：赛季剩余天数与进度。
enum SeasonLifecycleRules {
    struct Timeline: Equatable {
        let remainDays: Int
        let progressPercent: Double
    }

    private static let dayMilliseconds: Double = 24 * 60 * 60 * 1000

    /// 业务日期按服务端配置的东八区零点解析。
    private static let businessCalendar: Calendar = {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(secondsFromGMT: 8 * 60 * 60) ?? .current
        return calendar
    }()

    static func resolveTimeline(_ season: SeasonInfo?, now: Date = Date()) -> Timeline {
        guard let season else { return Timeline(remainDays: 0, progressPercent: 0) }

        let start = parseInstant(season.startAt) ?? businessDate(season.startDate)
        let endExclusive = parseInstant(season.endAtExclusive) ?? businessDate(season.endDate, addDays: 1)
        let current = now.timeIntervalSince1970 * 1000

        guard let start, let endExclusive, endExclusive > start else {
            return Timeline(remainDays: 0, progressPercent: 0)
        }

        let remainDays = max(0, Int(((endExclusive - current) / dayMilliseconds).rounded(.up)))
        let progressPercent = min(100, max(0, ((current - start) / (endExclusive - start)) * 100))
        return Timeline(remainDays: remainDays, progressPercent: progressPercent)
    }

    private static func parseInstant(_ value: String) -> Double? {
        guard !value.isEmpty else { return nil }
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: value) {
            return date.timeIntervalSince1970 * 1000
        }
        formatter.formatOptions = [.withInternetDateTime]
        return formatter.date(from: value).map { $0.timeIntervalSince1970 * 1000 }
    }

    private static func businessDate(_ value: String, addDays: Int = 0) -> Double? {
        let parts = value.split(separator: "-", omittingEmptySubsequences: false)
        guard parts.count == 3,
              parts[0].count == 4, parts[1].count == 2, parts[2].count == 2,
              let year = Int(parts[0]), let month = Int(parts[1]), let day = Int(parts[2]),
              let date = businessCalendar.date(from: DateComponents(year: year, month: month, day: day)) else {
            return nil
        }
        let check = businessCalendar.dateComponents([.year, .month, .day], from: date)
        guard check.year == year, check.month == month, check.day == day,
              let shifted = businessCalendar.date(byAdding: .day, value: addDays, to: date) else {
            return nil
        }
        return shifted.timeIntervalSince1970 * 1000
    }
}
