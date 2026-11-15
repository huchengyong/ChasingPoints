import XCTest
@testable import iOS

/// 迁移自 `uniapp/tests/pk-report-view-model.test.mjs`。
final class PKReportRulesTests: XCTestCase {
    private let stats = H2HStats(
        totalMatches: 12,
        myWins: 7,
        opponentWins: 5,
        winRate: 58.33,
        avgScoreDiff: 1.8,
        maxWinStreak: 3
    )
    private let history = [1, 1, 2, 1, 1].enumerated().map { index, result in
        MatchListItem(
            id: Int64(index + 1),
            opponentId: 20,
            gameType: 3,
            gameTypeName: "中式八球",
            opponentName: "阿杰",
            opponentAvatar: "",
            myScore: 7,
            opponentScore: 5,
            result: result,
            matchTime: "2026-10-09T20:30:00+08:00"
        )
    }

    func testHeroKeepsSocialCopyShortAndScreenshotFriendly() {
        let hero = PKReportRules.hero(stats: stats, opponentName: "阿杰", history: history)

        XCTAssertEqual(hero.title, "你对阿杰略占上风")
        XCTAssertEqual(hero.scoreText, "7 : 5")
        XCTAssertEqual(hero.metaText, "胜率 58% · 近 5 场 4 胜 1 负")
    }

    func testEvidenceReturnsExactlyTwoRowsForTheMVP() {
        let evidence = PKReportRules.evidence(stats: stats, history: history)

        XCTAssertEqual(evidence.count, 2)
        XCTAssertEqual(evidence.first, "近 5 场 4 胜 1 负")
        XCTAssertEqual(evidence.last, "当前最长连胜 3 场")
    }

    func testEvidenceFallsBackToAverageDiffWithoutStreak() {
        let noStreak = H2HStats(
            totalMatches: 2,
            myWins: 1,
            opponentWins: 1,
            winRate: 50,
            avgScoreDiff: -0.4,
            maxWinStreak: 0
        )

        let evidence = PKReportRules.evidence(stats: noStreak, history: [])

        XCTAssertEqual(evidence, ["平均分差 -0.4"])
    }

    func testPosterPayloadKeepsSummaryAndWinRateHumanized() {
        let hero = PKReportRules.hero(stats: stats, opponentName: "阿杰", history: history)
        let payload = PKReportRules.posterPayload(
            hero: hero,
            evidence: PKReportRules.evidence(stats: stats, history: history),
            stats: stats,
            userName: "我",
            opponentName: "阿杰",
            gameTypeLabel: "中八"
        )

        XCTAssertEqual(payload.winRateLabel, "58%")
        XCTAssertEqual(payload.avgScoreDiffLabel, "+1.8")
        XCTAssertEqual(payload.totalMatches, 12)
        XCTAssertEqual(payload.summaryText, "你对阿杰略占上风 · 近 5 场 4 胜 1 负 · 当前最长连胜 3 场")
        XCTAssertEqual(payload.gameTypeLabel, "中八")
    }

    func testStatusSeparatesPosterFailureFromDataReadiness() {
        XCTAssertEqual(PKReportRules.posterEmptyText(totalMatches: 6), "海报暂未生成")
        XCTAssertEqual(PKReportRules.posterEmptyText(totalMatches: 0), "至少完成一场真实交锋后再生成海报")
        XCTAssertEqual(PKReportRules.shareSummary(hero: hero(), evidence: ["近 5 场 4 胜 1 负"], totalMatches: 6),
                       "你对阿杰略占上风，历史交锋 7 : 5。近 5 场 4 胜 1 负。")
        XCTAssertEqual(PKReportRules.shareSummary(hero: hero(), evidence: [], totalMatches: 0), "")
    }

    func testLatestMatchSummaryUsesRelativeDateAndResult() {
        let summary = PKReportRules.latestMatchSummary(history.first)

        XCTAssertNotNil(summary)
        XCTAssertTrue(summary?.contains("中式八球") == true)
        XCTAssertTrue(summary?.contains("你赢了 7 : 5") == true)
        XCTAssertNil(PKReportRules.latestMatchSummary(nil))
    }

    private func hero() -> PKReportHero {
        PKReportRules.hero(stats: stats, opponentName: "阿杰", history: history)
    }
}
