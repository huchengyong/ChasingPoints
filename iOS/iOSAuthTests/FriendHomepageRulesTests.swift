import Foundation
import XCTest
@testable import iOS

/// 好友主页规则测试，搬运 `uniapp/tests/friend-homepage.test.mjs` 的摘要与隐藏态用例。
@MainActor
final class FriendHomepageRulesTests: XCTestCase {
    private let now = AppFormat.parseDateTime("2026-10-11 12:00:00")!

    func testSummaryForFriendWithMatches() {
        let summary = FriendHomepageRules.summary(
            totalMatches: 12,
            totalWins: 7,
            lastMatchAt: "2026-10-08 12:00:00",
            now: now
        )

        XCTAssertTrue(summary.hasMatches)
        XCTAssertEqual(summary.heroTitle, "这位球友最近有 12 场真实对局")
        XCTAssertEqual(summary.heroDesc, "可以直接往下看 TA 的对手战绩列表，PK 报表里再看双方对比。")
        XCTAssertEqual(summary.totalMatchesText, "真实对局 12 场")
        XCTAssertEqual(summary.recordText, "累计胜场 7 场")
        XCTAssertEqual(summary.recentMatchText, "最近更新 3天前")
        XCTAssertEqual(summary.emptyTitle, "")
        XCTAssertEqual(summary.emptyDesc, "")
        XCTAssertEqual(summary.battleSectionTitle, "对方战绩")
        XCTAssertEqual(summary.battleSectionTip, "直接查看 TA 最近的过往对手")
    }

    func testSummaryFallsBackToCalmEmptyState() {
        let summary = FriendHomepageRules.summary(
            totalMatches: 0,
            totalWins: 0,
            lastMatchAt: "",
            now: now
        )

        XCTAssertFalse(summary.hasMatches)
        XCTAssertEqual(summary.heroTitle, "这位球友还没有公开的对局样本")
        XCTAssertEqual(summary.heroDesc, "等 TA 完成更多真实对局后，这里会直接更新最新战绩列表。")
        XCTAssertEqual(summary.totalMatchesText, "真实对局 0 场")
        XCTAssertEqual(summary.recordText, "累计胜场 0 场")
        XCTAssertEqual(summary.recentMatchText, "最近更新 暂无记录")
        XCTAssertEqual(summary.emptyTitle, "暂时还没有可展示的战绩")
        XCTAssertEqual(summary.emptyDesc, "等这位好友完成真实对局后，再回来看看")
    }

    func testSummaryKeepsCopyFreeFromMySideComparison() {
        let summary = FriendHomepageRules.summary(
            totalMatches: 9,
            totalWins: 4,
            lastMatchAt: "2026-10-09 12:00:00",
            now: now
        )
        let banned = ["你", "我们", "交手", "领先", "落后", "追回", "胶着"]

        for word in banned {
            XCTAssertFalse(summary.heroTitle.contains(word), "摘要标题不应出现「\(word)」")
            XCTAssertFalse(summary.heroDesc.contains(word), "摘要描述不应出现「\(word)」")
        }
        XCTAssertEqual(summary.recordText, "累计胜场 4 场")
        XCTAssertEqual(summary.recentMatchText, "最近更新 2天前")
    }

    func testHiddenCopyStaysDedicated() {
        let summary = FriendHomepageRules.summary(totalMatches: 9, totalWins: 4, lastMatchAt: "", now: now)

        XCTAssertEqual(summary.hiddenTitle, "对方已隐藏战绩")
        XCTAssertEqual(summary.hiddenDesc, "这位好友暂时没有公开自己的战绩列表")
    }

    func testNegativeTotalsClampToZero() {
        let summary = FriendHomepageRules.summary(totalMatches: -2, totalWins: -5, lastMatchAt: "", now: now)

        XCTAssertFalse(summary.hasMatches)
        XCTAssertEqual(summary.totalMatchesText, "真实对局 0 场")
        XCTAssertEqual(summary.recordText, "累计胜场 0 场")
    }
}
