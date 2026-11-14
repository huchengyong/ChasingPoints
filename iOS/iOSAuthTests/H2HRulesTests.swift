import XCTest
@testable import iOS

@MainActor
final class H2HRulesTests: XCTestCase {
    func testAdvantageLevelDowngradesTinySamplesAndUpgradesLargeLeads() {
        XCTAssertEqual(
            H2HRules.advantageLevel(totalMatches: 1, myWins: 1, opponentWins: 0),
            .slightLead
        )
        XCTAssertEqual(
            H2HRules.advantageLevel(totalMatches: 12, myWins: 9, opponentWins: 3),
            .clearLead
        )
        XCTAssertEqual(
            H2HRules.advantageLevel(totalMatches: 12, myWins: 3, opponentWins: 9),
            .clearTrail
        )
        XCTAssertEqual(
            H2HRules.advantageLevel(totalMatches: 6, myWins: 3, opponentWins: 3),
            .balanced
        )
        XCTAssertEqual(
            H2HRules.advantageLevel(totalMatches: 0, myWins: 0, opponentWins: 0),
            .noSample
        )
    }

    func testTitleUsesRelationshipCopyInsteadOfRawPercentages() {
        XCTAssertEqual(
            H2HRules.title(totalMatches: 8, myWins: 5, opponentWins: 3, opponentName: "阿杰"),
            "你对阿杰略占上风"
        )
        XCTAssertEqual(
            H2HRules.title(totalMatches: 0, myWins: 0, opponentWins: 0, opponentName: "阿杰"),
            "你和阿杰还没形成交锋样本"
        )
    }

    func testSubtitleIncludesTotalRecordAndRecentForm() {
        let history = [
            matchItem(id: 1, result: 1),
            matchItem(id: 2, result: 1),
            matchItem(id: 3, result: 2),
            matchItem(id: 4, result: 1),
            matchItem(id: 5, result: 1),
        ]

        XCTAssertEqual(
            H2HRules.subtitle(myWins: 5, opponentWins: 3, recentHistory: history),
            "总交锋 5 胜 3 负，近 5 场 4 胜 1 负"
        )
    }

    func testPrimaryActionTextMapsBalancedRecordsToFinalMatchCopy() {
        XCTAssertEqual(H2HRules.primaryActionText(.balanced), "约一场决胜局")
        XCTAssertEqual(H2HRules.primaryActionText(.clearLead), "再约一场，扩大优势")
        XCTAssertEqual(H2HRules.primaryActionText(.slightTrail), "发起复仇局")
        XCTAssertEqual(H2HRules.primaryActionText(.noSample), "约一场见真章")
    }

    func testHeroExposesTitleSubtitleScoreAndBadges() {
        let hero = H2HRules.hero(
            stats: H2HStats(
                totalMatches: 8,
                myWins: 5,
                opponentWins: 3,
                winRate: 62.5,
                avgScoreDiff: 1.8,
                maxWinStreak: 3
            ),
            opponent: H2HOpponent(id: 18, name: "阿杰", avatar: "https://img.example/a.png"),
            history: [
                matchItem(id: 1, result: 1),
                matchItem(id: 2, result: 1),
                matchItem(id: 3, result: 2),
                matchItem(id: 4, result: 1),
                matchItem(id: 5, result: 1),
            ],
            subjectName: "你",
            subjectAvatar: "https://img.example/me.png"
        )

        XCTAssertEqual(hero.title, "你对阿杰略占上风")
        XCTAssertEqual(hero.subtitle, "总交锋 5 胜 3 负，近 5 场 4 胜 1 负")
        XCTAssertEqual(hero.scoreText, "5 : 3")
        XCTAssertEqual(hero.leftName, "你")
        XCTAssertEqual(hero.rightName, "阿杰")
        XCTAssertEqual(hero.badges, ["近期强势", "当前 3 连胜", "平均分差 +1.8"])
    }

    func testCalendarAggregatesSameDayWinsAndLosses() {
        let calendar = H2HRules.calendar(
            history: [
                matchItem(id: 1, result: 1, time: "2026-04-15T10:00:00+08:00"),
                matchItem(id: 2, result: 1, time: "2026-04-15T12:00:00+08:00"),
                matchItem(id: 3, result: 2, time: "2026-04-15T14:00:00+08:00"),
                matchItem(id: 4, result: 2, time: "2026-04-17T18:00:00+08:00"),
            ],
            monthKey: "2026-04"
        )

        XCTAssertEqual(calendar.title, "2026年4月")
        XCTAssertEqual(calendar.cells.count, 42)

        let day15 = calendar.cells.first { $0.dateKey == "2026-04-15" }
        XCTAssertEqual(day15?.day, 15)
        XCTAssertEqual(day15?.isCurrentMonth, true)
        XCTAssertEqual(day15?.winCount, 2)
        XCTAssertEqual(day15?.lossCount, 1)
        XCTAssertEqual(day15?.matchCount, 3)
        XCTAssertEqual(day15?.matchIds, [1, 2, 3])

        XCTAssertEqual(calendar.cells.first?.dateKey, "2026-03-29")
        XCTAssertEqual(calendar.cells.first?.isCurrentMonth, false)
    }

    func testMatchCardsProduceScoreDiffAndTags() {
        let cards = H2HRules.matchCards([
            matchItem(id: 10, result: 1, myScore: 7, opponentScore: 5, time: "2026-03-28T10:00:00Z"),
            matchItem(id: 11, result: 1, myScore: 7, opponentScore: 1),
            matchItem(id: 12, result: 2, myScore: 1, opponentScore: 7),
            matchItem(id: 13, result: 3, myScore: 4, opponentScore: 4),
        ])

        XCTAssertEqual(cards[0].scoreText, "7 : 5")
        XCTAssertEqual(cards[0].diffText, "+2")
        XCTAssertEqual(cards[0].tags, ["险胜"])
        XCTAssertEqual(cards[1].tags, ["大胜"])
        XCTAssertEqual(cards[2].tags, ["被压制"])
        XCTAssertEqual(cards[3].tags, ["平局"])
        XCTAssertEqual(cards[3].resultText, "平")
    }

    func testPageStatusPartialEmptyAndPartialOverReady() {
        XCTAssertEqual(
            H2HRules.pageStatus(
                statsLoaded: true,
                historyLoaded: false,
                totalMatches: 6,
                historyLength: 0,
                hasError: true
            ),
            .partial
        )
        XCTAssertEqual(
            H2HRules.pageStatus(
                statsLoaded: true,
                historyLoaded: true,
                totalMatches: 0,
                historyLength: 0,
                hasError: false
            ),
            .empty
        )
        XCTAssertEqual(
            H2HRules.pageStatus(
                statsLoaded: true,
                historyLoaded: true,
                totalMatches: 6,
                historyLength: 6,
                hasError: true
            ),
            .partial
        )
        XCTAssertEqual(
            H2HRules.pageStatus(
                statsLoaded: false,
                historyLoaded: false,
                totalMatches: 0,
                historyLength: 0,
                hasError: true
            ),
            .error
        )
    }

    func testSummaryCardRequiresLoadedStats() {
        XCTAssertFalse(
            H2HRules.shouldShowSummaryCard(pageStatus: .partial, statsLoaded: false)
        )
        XCTAssertTrue(
            H2HRules.shouldShowSummaryCard(pageStatus: .partial, statsLoaded: true)
        )
        XCTAssertFalse(
            H2HRules.shouldShowSummaryCard(pageStatus: .loading, statsLoaded: true)
        )
    }

    func testHistoryParamsPreferOpponentIdAndFallBackToName() {
        assertParams(
            H2HRules.historyParams(opponentId: 18, opponentName: "球友 A", page: 2, result: 1),
            [
                "opponent_id": "18",
                "page": "2",
                "page_size": "20",
                "result": "1",
            ]
        )
        assertParams(
            H2HRules.historyParams(opponentName: "线下朋友"),
            [
                "opponent_name": "线下朋友",
                "page": "1",
                "page_size": "20",
            ]
        )
        assertParams(
            H2HRules.historyParams(targetUserId: 52, opponentId: 18, page: 1, result: 2),
            [
                "target_user_id": "52",
                "opponent_id": "18",
                "page": "1",
                "page_size": "20",
                "result": "2",
            ]
        )
        assertParams(
            H2HRules.historyParams(
                opponentId: 18,
                pageSize: 100,
                startDate: "2026-04-01",
                endDate: "2026-04-30"
            ),
            [
                "opponent_id": "18",
                "page": "1",
                "page_size": "100",
                "start_date": "2026-04-01",
                "end_date": "2026-04-30",
            ]
        )
    }

    func testMonthRangeCurrentMonthAndYearBoundaryShift() {
        let range = H2HRules.monthDateRange("2026-04")
        XCTAssertEqual(range?.startDate, "2026-04-01")
        XCTAssertEqual(range?.endDate, "2026-04-30")
        XCTAssertNil(H2HRules.monthDateRange("2026-13"))

        let april30 = AppFormat.parseDateTime("2026-04-30T10:00:00+08:00")!
        XCTAssertEqual(H2HRules.currentMonthKey(april30), "2026-04")
        XCTAssertEqual(H2HRules.shiftMonthKey("2026-01", offset: -1), "2025-12")
        XCTAssertEqual(H2HRules.shiftMonthKey("2026-12", offset: 1), "2027-01")
    }

    func testViewModelKeepsSelfCopyAndSwitchesToFriendCopy() {
        XCTAssertEqual(
            H2HRules.viewModel(targetUserId: 0, targetName: "", opponentId: 18, opponentName: "阿杰"),
            H2HViewModel(
                isTargetMode: false,
                navigationTitle: "交锋记录 - 阿杰",
                subjectName: "你",
                winRateLabel: "你对阿杰的胜率是",
                isOpponentMe: false
            )
        )
        XCTAssertEqual(
            H2HRules.viewModel(targetUserId: 52, targetName: "球友 B", opponentId: 18, opponentName: "阿杰"),
            H2HViewModel(
                isTargetMode: true,
                navigationTitle: "球友 B的交锋记录",
                subjectName: "球友 B",
                winRateLabel: "球友 B 对阿杰的胜率是",
                isOpponentMe: false
            )
        )
        XCTAssertTrue(
            H2HRules.viewModel(
                targetUserId: 52,
                targetName: "球友 B",
                opponentId: 18,
                currentUserId: 18,
                opponentName: "我自己"
            ).isOpponentMe
        )
    }

    func testLoadFailureActionOnlyHandlesTargetAccessFailure() {
        XCTAssertEqual(
            H2HRules.loadFailureAction(targetUserId: 52, message: "仅可查看好友的对方战绩"),
            H2HLoadFailureAction(toastMessage: "仅可查看好友的对方战绩")
        )
        XCTAssertNil(H2HRules.loadFailureAction(targetUserId: 0, message: "网络请求失败"))
        XCTAssertNil(H2HRules.loadFailureAction(targetUserId: 52, message: "网络请求失败"))
    }

    func testHiddenEmptyStateCopy() {
        XCTAssertEqual(H2HRules.hiddenTitle, "对方已隐藏战绩")
        XCTAssertEqual(H2HRules.hiddenDescription, "这位好友暂时没有公开自己的战绩")
    }

    // MARK: - 辅助

    private func matchItem(
        id: Int64,
        result: Int = 1,
        myScore: Int = 7,
        opponentScore: Int = 5,
        time: String = SubPageFixtures.matchTime
    ) -> MatchListItem {
        MatchListItem(
            id: id,
            opponentId: 2,
            gameType: 3,
            gameTypeName: "中八",
            opponentName: "阿杰",
            opponentAvatar: "",
            myScore: myScore,
            opponentScore: opponentScore,
            result: result,
            matchTime: time
        )
    }

    private func assertParams(
        _ actual: [String: CustomStringConvertible?],
        _ expected: [String: String],
        file: StaticString = #filePath,
        line: UInt = #line
    ) {
        let normalized = actual.compactMapValues { $0.map { String(describing: $0) } }
        XCTAssertEqual(normalized, expected, file: file, line: line)
    }
}
