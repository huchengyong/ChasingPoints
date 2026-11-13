import XCTest
@testable import iOS

@MainActor
final class StatsDetailRulesTests: XCTestCase {
    private let april = StatsDetailRulesTests.date("2026-04-15 12:00:00")

    func testLoadingModeUsesFullPageOnlyBeforeFirstRender() {
        XCTAssertEqual(StatsDetailRules.loadingMode(hasLoadedOnce: false, isFetching: true), .initial)
        XCTAssertEqual(StatsDetailRules.loadingMode(hasLoadedOnce: true, isFetching: true), .refreshing)
        XCTAssertEqual(StatsDetailRules.loadingMode(hasLoadedOnce: true, isFetching: false), .idle)
    }

    func testStaleGameSwitchResponseIsIgnored() {
        XCTAssertFalse(StatsDetailRules.shouldApplyResponse(requestId: 6, latestRequestId: 7))
        XCTAssertTrue(StatsDetailRules.shouldApplyResponse(requestId: 7, latestRequestId: 7))
    }

    func testTrendCalendarGroupsWinsAndLossesByDayInSelectedMonth() {
        let matches = [
            point(id: 1, date: "2026-04-03", result: 1),
            point(id: 2, date: "2026-04-03", result: 1),
            point(id: 3, date: "2026-04-03", result: 2),
            point(id: 4, date: "2026-05-01", result: 1),
        ]

        let calendar = StatsDetailRules.trendCalendar(matches: matches, monthKey: "2026-04", now: april)

        let day = calendar.cells.first { $0.dateKey == "2026-04-03" }
        XCTAssertEqual(calendar.title, "2026年4月")
        XCTAssertEqual(calendar.summary.matchCount, 3)
        XCTAssertEqual(calendar.summary.winCount, 2)
        XCTAssertEqual(calendar.summary.lossCount, 1)
        XCTAssertEqual(calendar.summary.winRate, 67)
        XCTAssertEqual(day?.winCount, 2)
        XCTAssertEqual(day?.lossCount, 1)
        XCTAssertEqual(day?.hasMatches, true)
        XCTAssertEqual(calendar.cells.count, 42)
        XCTAssertEqual(calendar.weekdays, ["日", "一", "二", "三", "四", "五", "六"])
    }

    func testRankChartAggregatesDailyRankScoreDeltasForSelectedMonth() {
        let chart = StatsDetailRules.rankDeltaChart(
            points: [
                point(id: 1, date: "2026-04-03", rankScore: 1030),
                point(id: 2, date: "2026-04-03", rankScore: 1010),
                point(id: 3, date: "2026-04-02", rankScore: 1020),
                point(id: 4, date: "2026-03-31", rankScore: 1000),
            ],
            monthKey: "2026-04",
            now: april
        )

        let aprilSecond = chart.days.first { $0.dateKey == "2026-04-02" }
        let aprilThird = chart.days.first { $0.dateKey == "2026-04-03" }

        XCTAssertEqual(chart.summary.netDelta, 30)
        XCTAssertEqual(chart.summary.upDays, 2)
        XCTAssertEqual(chart.summary.downDays, 0)
        XCTAssertEqual(aprilSecond?.delta, 20)
        XCTAssertEqual(aprilThird?.delta, 10)
        XCTAssertEqual(aprilThird?.label, "3日")
        XCTAssertEqual(aprilThird?.showTick, false)
        XCTAssertEqual(chart.days.count, 30)
    }

    func testRankTrendUsesNewestScoreAndPeak() {
        let trend = StatsDetailRules.rankTrend(points: [
            point(id: 1, date: "2026-04-11", rankScore: 1040),
            point(id: 2, date: "2026-04-03", rankScore: 1055),
            point(id: 3, date: "2026-04-02"),
        ])

        XCTAssertEqual(trend.currentScore, 1040)
        XCTAssertEqual(trend.peakScore, 1055)
        XCTAssertEqual(StatsDetailRules.rankTrend(points: []).peakScore, 0)
    }

    func testDurationStatsNormalizeBackendFields() {
        let values = StatsDetailRules.normalizeDurationStats(
            StatsDurationStats(averageSeconds: 125, fastestSeconds: 40, longestSeconds: 360, totalMatches: 3)
        )

        XCTAssertEqual(values, StatsDurationValues(average: 125, fastest: 40, longest: 360, total: 3))
        XCTAssertEqual(
            StatsDetailRules.normalizeDurationStats(nil),
            StatsDurationValues(average: 0, fastest: 0, longest: 0, total: 0)
        )
    }

    func testOpponentStrengthNormalizesRankRangesForDisplay() {
        let rows = StatsDetailRules.normalizeOpponentStrengthStats([
            StatsOpponentStrengthItem(rankRange: "中级(1001-2000)", matches: 25, wins: 22, winRate: 88),
            StatsOpponentStrengthItem(rankRange: "", matches: 1, wins: 0, winRate: 0),
        ])

        XCTAssertEqual(rows.first?.tierName, "中级(1001-2000)")
        XCTAssertEqual(rows.first?.matches, 25)
        XCTAssertEqual(rows.first?.wins, 22)
        XCTAssertEqual(rows.first?.winRate, 88)
        XCTAssertEqual(rows.last?.tierName, "段位2")
    }

    func testHighScoreUsesFirstRecordAsPersonalBestAndFiltersMonthByDatePrefix() {
        let summary = StatsDetailRules.normalizeHighScoreStats([
            StatsHighScoreRecord(matchId: 1, score: 87, gameType: 3, gameTypeName: "中式八球", opponentName: "阿杰", date: "2026-04-08"),
            StatsHighScoreRecord(matchId: 2, score: 72, gameType: 3, gameTypeName: "中式八球", opponentName: "阿杰", date: "2026-03-30"),
            StatsHighScoreRecord(matchId: 3, score: 90, gameType: 3, gameTypeName: "中式八球", opponentName: "阿杰", date: "2026-04-20"),
        ], now: april)

        XCTAssertEqual(summary.personalBest, 87)
        XCTAssertEqual(summary.monthBest, 90)
    }

    func testPercentFormattingDropsTrailingZeros() {
        XCTAssertEqual(StatsDetailRules.formatPercent(61.9), "61.9")
        XCTAssertEqual(StatsDetailRules.formatPercent(58.33), "58.33")
        XCTAssertEqual(StatsDetailRules.formatPercent(58.3), "58.3")
        XCTAssertEqual(StatsDetailRules.formatPercent(100), "100")
        XCTAssertEqual(StatsDetailRules.formatPercent(0), "0")
        XCTAssertEqual(StatsDetailRules.formatPercent(.nan), "0")
    }

    func testDurationFormattingMatchesMiniProgram() {
        XCTAssertEqual(StatsDetailRules.formatDuration(seconds: 0), "0分")
        XCTAssertEqual(StatsDetailRules.formatDuration(seconds: 40), "40秒")
        XCTAssertEqual(StatsDetailRules.formatDuration(seconds: 125), "2分")
        XCTAssertEqual(StatsDetailRules.formatDuration(seconds: 3600), "1小时")
        XCTAssertEqual(StatsDetailRules.formatDuration(seconds: 3720), "1小时2分")
        XCTAssertEqual(StatsDetailRules.formatDuration(seconds: 86400), "1天")
        XCTAssertEqual(StatsDetailRules.formatDuration(seconds: 90000), "1天1小时")
    }

    func testShiftMonthKeyCrossesYearBoundary() {
        XCTAssertEqual(StatsDetailRules.shiftMonthKey("2026-04", offset: 1, now: april), "2026-05")
        XCTAssertEqual(StatsDetailRules.shiftMonthKey("2026-04", offset: -1, now: april), "2026-03")
        XCTAssertEqual(StatsDetailRules.shiftMonthKey("2026-01", offset: -1, now: april), "2025-12")
    }

    // MARK: - 辅助

    private func point(id: Int64, date: String, winRate: Double = 50, rankScore: Int? = nil, result: Int = 1) -> StatsTrendPoint {
        StatsTrendPoint(matchId: id, date: date, winRate: winRate, rankScore: rankScore, result: result)
    }

    private static func date(_ text: String) -> Date {
        AppFormat.parseDateTime(text)!
    }
}
