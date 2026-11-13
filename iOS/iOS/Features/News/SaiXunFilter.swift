import Foundation

/// 赛讯列表筛选条件，对应小程序页面的 `filter` 对象。
struct EventNewsFilter: Equatable {
    var year: Int?
    var from: String
    var to: String
    var gameType: Int
    var status: Int?
    var city: String

    init(
        year: Int? = nil,
        from: String = "",
        to: String = "",
        gameType: Int = 0,
        status: Int? = nil,
        city: String = ""
    ) {
        self.year = year
        self.from = from
        self.to = to
        self.gameType = gameType
        self.status = status
        self.city = city
    }
}

struct YearDateRange: Equatable {
    let from: String
    let to: String
}

struct CurrentYearFilter: Equatable {
    let year: Int
    let from: String
    let to: String
    let preset: String
}

/// 赛讯筛选规则，逐条对应小程序 `utils/saixun-filter.js`。
enum SaiXunFilter {
    private static let defaultPreset = "full_year"
    private static let minYear = 2023

    static func buildYearDateRange(_ year: Int?) -> YearDateRange {
        guard let year, year > 0 else { return YearDateRange(from: "", to: "") }
        return YearDateRange(
            from: buildDateText(year, month: 1, day: 1),
            to: buildDateText(year, month: 12, day: 31)
        )
    }

    static func buildCurrentYearFilter(
        now: Date,
        calendar: Calendar = SaiXunCalendar.shanghai
    ) -> CurrentYearFilter {
        let year = calendar.component(.year, from: now)
        let range = buildYearDateRange(year)
        return CurrentYearFilter(year: year, from: range.from, to: range.to, preset: defaultPreset)
    }

    /// 显式起止日期优先；否则按自然年。参数与 `buildEventNewsListParams` 一致（page / page_size 总会带上）。
    static func buildEventNewsListParams(
        _ filter: EventNewsFilter,
        page: Int = 1,
        pageSize: Int = 10
    ) -> [String: CustomStringConvertible?] {
        var params: [String: CustomStringConvertible?] = [
            "page": page,
            "page_size": pageSize,
        ]

        let year = filter.year ?? 0
        let from = filter.from.trimmingCharacters(in: .whitespacesAndNewlines)
        let to = filter.to.trimmingCharacters(in: .whitespacesAndNewlines)

        if !from.isEmpty && !to.isEmpty {
            params["from"] = from
            params["to"] = to
        } else if year > 0 {
            params["year"] = year
        }

        if filter.gameType > 0 { params["game_type"] = filter.gameType }
        if let status = filter.status, status >= 0 { params["status"] = status }

        // 小程序筛选条暂未提供城市输入，接口本身支持该参数。
        let city = filter.city.trimmingCharacters(in: .whitespacesAndNewlines)
        if !city.isEmpty { params["city"] = city }

        return params
    }

    static func buildYearOptions(
        _ centerYear: Int?,
        span: Int = 2,
        now: Date,
        calendar: Calendar = SaiXunCalendar.shanghai
    ) -> [Int] {
        guard let year = centerYear, year > 0 else { return [] }

        let currentYear = calendar.component(.year, from: now)
        let maxYear = min(year + span, currentYear)
        let minYearBound = max(year - span, minYear)
        guard minYearBound <= maxYear else { return [] }
        return Array(minYearBound...maxYear)
    }

    static func buildYearBoundaryDate(year: Int?, month: Int?, dayHint: Int?) -> String {
        guard let year, let month else { return "" }

        let maxDay = lastDayOfMonth(year: year, month: month)
        let normalizedDay = min(max(dayHint ?? 0, 1), maxDay)
        return buildDateText(year, month: month, day: normalizedDay)
    }

    private static func buildDateText(_ year: Int, month: Int, day: Int) -> String {
        "\(year)-\(pad2(month))-\(pad2(day))"
    }

    private static func pad2(_ value: Int) -> String {
        value < 10 ? "0\(value)" : "\(value)"
    }

    /// 对应 `new Date(Date.UTC(year, month, 0)).getUTCDate()`。
    private static func lastDayOfMonth(year: Int, month: Int) -> Int {
        let calendar = SaiXunCalendar.shanghai
        guard let firstDay = calendar.date(from: DateComponents(year: year, month: month, day: 1)),
              let range = calendar.range(of: .day, in: .month, for: firstDay) else {
            return 31
        }
        return range.count
    }
}
