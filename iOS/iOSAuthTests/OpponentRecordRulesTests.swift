import XCTest
@testable import iOS

@MainActor
final class OpponentRecordRulesTests: XCTestCase {
    func testRequestParamsCarryTargetUserIdInFriendMode() {
        let params = OpponentRecordRules.requestParams(
            page: 2,
            pageSize: 20,
            keyword: "阿杰",
            targetUserId: 52
        )
        let normalized = params.compactMapValues { $0.map { String(describing: $0) } }

        XCTAssertEqual(normalized, [
            "page": "2",
            "page_size": "20",
            "keyword": "阿杰",
            "target_user_id": "52",
        ])
    }

    func testViewModelKeepsSelfAndFriendCopy() {
        XCTAssertEqual(
            OpponentRecordRules.viewModel(targetName: "", targetUserId: 0),
            OpponentRecordViewModel(
                isTargetMode: false,
                subjectName: "你",
                navigationTitle: "过往对手",
                loginHint: "请登录后查看过往对手",
                searchPlaceholder: "搜索对手",
                emptyText: "暂无过往对手",
                emptyHint: "快去发起一场PK吧！"
            )
        )
        XCTAssertEqual(
            OpponentRecordRules.viewModel(targetName: "球友 B", targetUserId: 52),
            OpponentRecordViewModel(
                isTargetMode: true,
                subjectName: "球友 B",
                navigationTitle: "对方战绩",
                loginHint: "请登录后查看对方战绩",
                searchPlaceholder: "搜索 TA 的对手",
                emptyText: "暂无对方战绩",
                emptyHint: "还没看到球友 B 的历史过往对手"
            )
        )
    }

    func testH2HRouteKeepsTargetUserContext() {
        XCTAssertEqual(
            OpponentRecordRules.h2hRoute(
                opponent: record(id: 18, name: "阿杰"),
                targetUserId: 52,
                targetName: "球友 B",
                targetAvatar: "https://img.example/b.png"
            ),
            .h2hRecord(
                opponentId: 18,
                targetUserId: 52,
                targetName: "球友 B",
                targetAvatar: "https://img.example/b.png",
                opponentName: "阿杰"
            )
        )
        XCTAssertEqual(
            OpponentRecordRules.h2hRoute(
                opponent: record(id: 18, name: "阿杰"),
                targetUserId: 0,
                targetName: "",
                targetAvatar: ""
            ),
            .h2hRecord(
                opponentId: 18,
                targetUserId: 0,
                targetName: "",
                targetAvatar: "",
                opponentName: "阿杰"
            )
        )
    }

    func testStatsSummaryExplainsHowToUseTheList() {
        XCTAssertEqual(
            OpponentRecordRules.statsSummary(
                totalOpponents: 6,
                totalWins: 14,
                targetName: "",
                targetUserId: 0
            ),
            OpponentStatsSummary(
                title: "你的对手档案",
                subtitle: "已经形成 6 个对手样本，累计拿下 14 场胜利"
            )
        )
    }

    func testCardConvertsRecordIntoReplayFriendlyCopy() {
        let card = OpponentRecordRules.card(
            for: record(id: 18, name: "阿杰", wins: 7, losses: 5, winRate: 58.33),
            subjectName: "你"
        )

        XCTAssertEqual(card.relationshipText, "你略占上风")
        XCTAssertEqual(card.relationshipBadge, "领先")
        XCTAssertEqual(card.recordText, "总交锋 7 胜 5 负")
        XCTAssertEqual(card.winRateText, "58%")
        XCTAssertEqual(card.sampleText, "12 场交锋")
    }

    func testCardMarksCurrentUserAndLabelsWinRateInFriendLists() {
        let card = OpponentRecordRules.card(
            for: record(id: 18, name: "我自己", wins: 3, losses: 2, winRate: 60),
            subjectName: "球友 B",
            currentUserId: 18,
            showWinRateLabel: true
        )

        XCTAssertTrue(card.isMeOpponent)
        XCTAssertEqual(card.winRateText, "60% 胜率")
    }

    func testCardUsesBackendTotalMatchesAsSampleSource() {
        let card = OpponentRecordRules.card(
            for: record(id: 20, name: "阿峰", totalMatches: 6, wins: 4, losses: 0, winRate: 100),
            subjectName: "你"
        )

        XCTAssertEqual(card.sampleText, "6 场交锋")
        XCTAssertEqual(card.relationshipText, "你明显占优")
    }

    // MARK: - 辅助

    private func record(
        id: Int64,
        name: String = "阿杰",
        totalMatches: Int = 0,
        wins: Int = 7,
        losses: Int = 5,
        winRate: Double = 58.33,
        lastMatchAt: String = "2026-03-27 18:30:00"
    ) -> OpponentRecord {
        OpponentRecord(
            id: id,
            name: name,
            avatar: "",
            totalMatches: totalMatches,
            wins: wins,
            losses: losses,
            winRate: winRate,
            lastMatchAt: lastMatchAt
        )
    }
}
