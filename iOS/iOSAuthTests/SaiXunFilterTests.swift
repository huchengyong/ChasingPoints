import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/saixun-filter.test.mjs` 的 6 条用例。
@MainActor
final class SaiXunFilterTests: XCTestCase {
    private func time(_ iso: String) -> Date { EventNewsFixtures.time(iso) }

    private func params(
        _ filter: EventNewsFilter,
        page: Int = 1,
        pageSize: Int = 10
    ) -> [String: String] {
        var result: [String: String] = [:]
        for (key, value) in SaiXunFilter.buildEventNewsListParams(filter, page: page, pageSize: pageSize) {
            if let value { result[key] = String(describing: value) }
        }
        return result
    }

    func testBuildCurrentYearFilterDefaultsToCurrentNaturalYearFullRange() {
        let filter = SaiXunFilter.buildCurrentYearFilter(now: time("2026-04-07T10:00:00+08:00"))

        XCTAssertEqual(
            filter,
            CurrentYearFilter(year: 2026, from: "2026-01-01", to: "2026-12-31", preset: "full_year")
        )
    }

    func testBuildYearDateRangeReturnsFullYearBoundaries() {
        XCTAssertEqual(
            SaiXunFilter.buildYearDateRange(2025),
            YearDateRange(from: "2025-01-01", to: "2025-12-31")
        )
    }

    func testBuildEventNewsListParamsPrefersExplicitFromToOverYear() {
        XCTAssertEqual(
            params(EventNewsFilter(year: 2026, from: "2026-01-01", to: "2026-04-30"), page: 2, pageSize: 20),
            ["page": "2", "page_size": "20", "from": "2026-01-01", "to": "2026-04-30"]
        )
        XCTAssertEqual(
            params(EventNewsFilter(year: 2025)),
            ["page": "1", "page_size": "10", "year": "2025"]
        )
    }

    func testBuildEventNewsListParamsMergesOptionalGameTypeAndStatusFilters() {
        XCTAssertEqual(
            params(EventNewsFilter(year: 2026, gameType: 1, status: 2)),
            ["page": "1", "page_size": "10", "year": "2026", "game_type": "1", "status": "2"]
        )
        XCTAssertEqual(
            params(EventNewsFilter(year: 2026, gameType: 0, status: -1)),
            ["page": "1", "page_size": "10", "year": "2026"]
        )
        XCTAssertEqual(
            params(EventNewsFilter(year: 2026, status: 0)),
            ["page": "1", "page_size": "10", "year": "2026", "status": "0"]
        )
    }

    func testBuildYearOptionsExposesBoundedYearChoices() {
        let now = time("2026-06-01T00:00:00Z")
        XCTAssertEqual(SaiXunFilter.buildYearOptions(2026, span: 1, now: now), [2025, 2026])
        XCTAssertEqual(SaiXunFilter.buildYearOptions(2024, span: 2, now: now), [2023, 2024, 2025, 2026])
        XCTAssertEqual(SaiXunFilter.buildYearOptions(2023, span: 2, now: now), [2023, 2024, 2025])
        XCTAssertEqual(SaiXunFilter.buildYearOptions(2021, span: 2, now: now), [2023])
    }

    func testBuildYearBoundaryDateClampsTheDayIntoTheMonth() {
        XCTAssertEqual(SaiXunFilter.buildYearBoundaryDate(year: 2026, month: 2, dayHint: 31), "2026-02-28")
    }
}
