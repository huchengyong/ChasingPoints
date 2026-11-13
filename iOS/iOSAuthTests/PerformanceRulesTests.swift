import XCTest
@testable import iOS

@MainActor
final class PerformanceRulesTests: XCTestCase {
    func testRatingViewShowsExactValuesBoundsAndEmptyState() {
        XCTAssertEqual(
            PerformanceRules.resolveRatingView(PerformanceRating(rating: 1678, bound: nil)),
            PerformanceRatingView(available: true, value: "1678", qualifier: "", text: "1678")
        )
        XCTAssertEqual(
            PerformanceRules.resolveRatingView(PerformanceRating(rating: 2415, bound: "at_least")),
            PerformanceRatingView(available: true, value: "2415", qualifier: "至少", text: "至少2415")
        )
        XCTAssertEqual(
            PerformanceRules.resolveRatingView(PerformanceRating(rating: 0, bound: "at_most")),
            PerformanceRatingView(available: true, value: "0", qualifier: "至多", text: "至多0")
        )
        XCTAssertEqual(
            PerformanceRules.resolveRatingView(nil),
            PerformanceRatingView(available: false, value: "", qualifier: "", text: "—")
        )
    }

    func testSampleCountsAreCappedAtServerLimitAndNeverNegative() {
        XCTAssertEqual(PerformanceRules.formatSampleCount(0), "0")
        XCTAssertEqual(PerformanceRules.formatSampleCount(37), "37")
        XCTAssertEqual(PerformanceRules.formatSampleCount(200), "200+")
        XCTAssertEqual(PerformanceRules.formatSampleCount(999), "200+")
        XCTAssertEqual(PerformanceRules.formatSampleCount(-3), "0")
    }

    func testSummaryViewExplainsActualVersusExpectedWins() {
        let better = PerformanceRules.resolveSummaryView(
            PerformanceSummary(wins: 14, expectedWins: 11.2, winsDiff: 2.8, rating: PerformanceRating(rating: 1720, bound: nil)),
            sampleCount: 20
        )
        XCTAssertEqual(better.tone, .up)
        XCTAssertEqual(better.diffText, "+2.8")
        XCTAssertEqual(better.expectedText, "11.2")
        XCTAssertEqual(better.wins, 14)
        XCTAssertEqual(better.rating.text, "1720")
        XCTAssertTrue(better.headline.contains("更好"))

        let worse = PerformanceRules.resolveSummaryView(
            PerformanceSummary(wins: 8, expectedWins: 9.5, winsDiff: -1.5, rating: nil),
            sampleCount: 12
        )
        XCTAssertEqual(worse.tone, .down)
        XCTAssertEqual(worse.diffText, "-1.5")
        XCTAssertFalse(worse.rating.available)
        XCTAssertTrue(worse.ratingHint.contains("样本不足 5 场"))

        let flat = PerformanceRules.resolveSummaryView(
            PerformanceSummary(wins: 9, expectedWins: 9.2, winsDiff: -0.2, rating: PerformanceRating(rating: 1500, bound: nil)),
            sampleCount: 20
        )
        XCTAssertEqual(flat.tone, .flat)
        XCTAssertEqual(flat.diffText, "-0.2")

        // 四舍五入后为 0 时不带正负号。
        XCTAssertEqual(
            PerformanceRules.resolveSummaryView(
                PerformanceSummary(wins: 9, expectedWins: 9.004, winsDiff: -0.004, rating: nil),
                sampleCount: 9
            ).diffText,
            "0.0"
        )
        XCTAssertFalse(PerformanceRules.resolveSummaryView(nil, sampleCount: 3).available)
    }

    func testMonthChartScalesBarsAndEndOfMonthDotsOnSharedAxis() {
        let chart = PerformanceRules.buildMonthChart([
            month("2026-05", samples: 0, rating: nil, end: 0),
            month("2026-06", samples: 8, rating: 1500, end: 1480),
            month("2026-07", samples: 6, rating: 1700, end: 1620),
            month("2026-08", samples: 2, rating: nil, end: 1630),
            month("2026-09", samples: 5, rating: 1900, end: 1700, bound: "at_least", incomplete: false),
            month("2026-10", samples: 4, rating: 1600, end: 1710),
        ])

        XCTAssertTrue(chart.hasData)
        XCTAssertEqual(chart.bars.count, 6)
        XCTAssertEqual(chart.bars.map(\.label), ["5月", "6月", "7月", "8月", "9月", "10月"])
        XCTAssertEqual(chart.bars.last?.isCurrent, true)
        XCTAssertEqual(chart.bars.filter(\.isCurrent).count, 1)

        let may = chart.bars[0]
        let june = chart.bars[1]
        let july = chart.bars[2]
        let august = chart.bars[3]
        let september = chart.bars[4]
        let october = chart.bars[5]
        XCTAssertEqual(may.status, .insufficient)
        XCTAssertEqual(may.heightPercent, 0)
        XCTAssertNil(may.dotPercent, "没有排位分就没有点")
        XCTAssertEqual(august.note, "样本不足")
        XCTAssertTrue((august.dotPercent ?? 0) > 0, "没有表现分的月份仍可展示月末排位分")
        XCTAssertEqual(september.qualifier, "至少")
        XCTAssertLessThan(june.heightPercent, july.heightPercent)
        XCTAssertLessThan(july.heightPercent, september.heightPercent)
        XCTAssertLessThan(october.heightPercent, july.heightPercent)
        XCTAssertGreaterThan(october.heightPercent, june.heightPercent)
        for bar in chart.bars {
            XCTAssertGreaterThanOrEqual(bar.heightPercent, 0)
            XCTAssertLessThanOrEqual(bar.heightPercent, 100)
            if let dot = bar.dotPercent {
                XCTAssertGreaterThanOrEqual(dot, 0)
                XCTAssertLessThanOrEqual(dot, 100)
            }
        }
        XCTAssertEqual(chart.scaleMin % 100, 0)
        XCTAssertEqual(chart.scaleMax % 100, 0)
    }

    func testMonthTruncatedByReadCapIsMarkedIncompleteAndNeverDrawnAsBar() {
        let chart = PerformanceRules.buildMonthChart([
            month("2026-09", samples: 12, rating: nil, end: 1500, incomplete: true),
            // 即使误带了表现分，也不能把残缺数据画成水平。
            month("2026-10", samples: 12, rating: 1800, end: 1520, incomplete: true),
        ])

        XCTAssertEqual(chart.bars.map(\.status), [.incomplete, .incomplete])
        XCTAssertEqual(chart.bars.map(\.heightPercent), [0, 0])
        XCTAssertEqual(chart.bars.map(\.note), ["数据不全", "数据不全"])
        XCTAssertNil(chart.bars[1].ratingValue)
    }

    func testMonthChartReportsNoDataWhenThereIsNothingToDraw() {
        XCTAssertFalse(PerformanceRules.buildMonthChart([]).hasData)
        XCTAssertFalse(PerformanceRules.buildMonthChart(nil).hasData)
        XCTAssertFalse(
            PerformanceRules.buildMonthChart([
                month("2026-10", samples: 1, rating: nil, end: 0),
            ]).hasData
        )
    }

    func testSingleFlatSeriesStillGetsUsableScale() {
        let chart = PerformanceRules.buildMonthChart([
            month("2026-10", samples: 6, rating: 1500, end: 1500),
        ])

        XCTAssertGreaterThanOrEqual(chart.scaleMax - chart.scaleMin, 200)
        XCTAssertGreaterThan(chart.bars[0].heightPercent, 0)
        XCTAssertLessThan(chart.bars[0].heightPercent, 100)
    }

    func testNotableMatchesDescribeOpponentAndPreMatchExpectation() {
        let best = PerformanceRules.resolveNotableMatchView(
            PerformanceMatchItem(
                matchId: 9,
                opponentName: "老王",
                opponentAvatar: "a.png",
                opponentBeforeScore: 1800,
                expectedWinRate: 23.5,
                playedAt: "2026-10-01 20:00:00"
            ),
            kind: .bestWin
        )
        XCTAssertEqual(best?.title, "最佳胜利")
        XCTAssertEqual(best?.headline, "击败 老王")
        XCTAssertTrue(best?.detail.contains("1800") == true)
        XCTAssertTrue(best?.detail.contains("只有 23.5%") == true)
        XCTAssertEqual(best?.matchId, 9)

        let loss = PerformanceRules.resolveNotableMatchView(
            PerformanceMatchItem(
                matchId: 10,
                opponentName: "",
                opponentAvatar: "",
                opponentBeforeScore: 1300,
                expectedWinRate: 80,
                playedAt: ""
            ),
            kind: .worstLoss
        )
        XCTAssertEqual(loss?.title, "爆冷失利")
        XCTAssertEqual(loss?.headline, "负于 对手")
        XCTAssertTrue(loss?.detail.contains("有 80%") == true)
        XCTAssertNil(PerformanceRules.resolveNotableMatchView(nil, kind: .bestWin))
    }

    func testLockedViewUsesServerSampleCountAndComplianceSwitch() {
        let response = lockedResponse(sampleCount: 200, preview: (6, 1678, nil))
        let hidden = PerformanceRules.resolveLockedView(response, canShowPurchase: false)
        XCTAssertFalse(hidden.showPurchase)
        XCTAssertEqual(hidden.sampleText, "200+")
        XCTAssertTrue(hidden.preview.available)
        XCTAssertEqual(hidden.preview.ratingText, "1678")
        XCTAssertEqual(hidden.preview.monthText, "10月")
        XCTAssertTrue(hidden.preview.hint.contains("6 场"))
        XCTAssertEqual(hidden.features.count, 4)
        XCTAssertTrue(PerformanceRules.resolveLockedView(response, canShowPurchase: true).showPurchase)
        XCTAssertFalse(PerformanceRules.resolveLockedView(response, canShowPurchase: false).showPurchase, "默认不展示开通入口")

        let sparse = PerformanceRules.resolveLockedView(
            lockedResponse(sampleCount: 2, preview: (2, nil, nil)),
            canShowPurchase: false
        )
        XCTAssertFalse(sparse.preview.available)
        XCTAssertEqual(sparse.preview.ratingText, "—")
        XCTAssertTrue(sparse.preview.hint.contains("满 3 场"))

        let bounded = PerformanceRules.resolveLockedView(
            lockedResponse(sampleCount: 5, preview: (3, 2300, "at_least")),
            canShowPurchase: false
        )
        XCTAssertEqual(bounded.preview.qualifier, "至少")

        let none = PerformanceRules.resolveLockedView(
            PerformanceResponse(
                success: true,
                gameType: 3,
                locked: true,
                sampleCount: 0,
                monthPreview: nil,
                summary: nil,
                bestWin: nil,
                worstLoss: nil,
                months: nil
            ),
            canShowPurchase: false
        )
        XCTAssertFalse(none.hasSamples)
        XCTAssertFalse(none.preview.available)
        XCTAssertEqual(none.preview.monthText, "本月")
    }

    func testLockedViewNeverCarriesAnalysisNumbersBeyondServerPreview() {
        let response = PerformanceResponse(
            success: true,
            gameType: 3,
            locked: true,
            sampleCount: 12,
            monthPreview: Self.preview(month: "2026-10", samples: 3, rating: nil),
            summary: PerformanceSummary(
                wins: 9,
                expectedWins: 7,
                winsDiff: 2,
                rating: PerformanceRating(rating: 1800, bound: nil)
            ),
            bestWin: PerformanceMatchItem(
                matchId: 1,
                opponentName: "老王",
                opponentAvatar: "",
                opponentBeforeScore: 1800,
                expectedWinRate: 20,
                playedAt: ""
            ),
            worstLoss: nil,
            months: [month("2026-10", samples: 4, rating: 1800, end: 1600)]
        )

        let view = PerformanceRules.resolveLockedView(response, canShowPurchase: false)

        XCTAssertEqual(view.sampleText, "12")
        XCTAssertFalse(view.preview.available)
        XCTAssertEqual(view.preview.ratingText, "—")
    }

    func testInvalidationVersionCombinesRankAndMemberScopes() {
        XCTAssertEqual(PerformanceRules.version(rankVersion: 2, memberVersion: 1), "2:1")
        XCTAssertEqual(PerformanceRules.version(rankVersion: 0, memberVersion: 0), "0:0")
        XCTAssertNotEqual(
            PerformanceRules.version(rankVersion: 1, memberVersion: 0),
            PerformanceRules.version(rankVersion: 0, memberVersion: 1)
        )
    }

    func testChartFootnoteExplainsMinimumSamplesPerMonth() {
        XCTAssertTrue(PerformanceRules.monthFootnote.contains("满 3 场"))
    }

    // MARK: - 辅助

    private func month(
        _ month: String,
        samples: Int,
        rating: Int?,
        end: Int,
        bound: String? = nil,
        incomplete: Bool? = nil
    ) -> PerformanceMonthItem {
        PerformanceMonthItem(
            month: month,
            sampleCount: samples,
            rating: rating.map { PerformanceRating(rating: $0, bound: bound) },
            endRankScore: end,
            incomplete: incomplete
        )
    }

    private func lockedResponse(
        sampleCount: Int,
        preview: (samples: Int, rating: Int?, bound: String?)
    ) -> PerformanceResponse {
        PerformanceResponse(
            success: true,
            gameType: 3,
            locked: true,
            sampleCount: sampleCount,
            monthPreview: Self.preview(month: "2026-10", samples: preview.samples, rating: preview.rating, bound: preview.bound),
            summary: nil,
            bestWin: nil,
            worstLoss: nil,
            months: nil
        )
    }

    private static func preview(
        month: String,
        samples: Int,
        rating: Int?,
        bound: String? = nil
    ) -> PerformanceMonthPreview {
        PerformanceMonthPreview(
            month: month,
            sampleCount: samples,
            rating: rating.map { PerformanceRating(rating: $0, bound: bound) }
        )
    }
}
