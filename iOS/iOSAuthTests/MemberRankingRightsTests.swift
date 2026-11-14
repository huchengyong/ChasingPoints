import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/member-ranking-rights.test.mjs` 的行为断言。
/// 其中「旧会员卡常量与导出已移除」是 JS 模块导出检查，原生端没有对应 API，未迁移。
final class MemberRankingRightsTests: XCTestCase {
    func testAchievementRecordsKeepOnlyRecordedSpecialAchievements() {
        let records = MemberRankingRights.achievementRecords([
            "break_clear": 1,
            "break_100": 2,
            "golden_break": 0,
        ])

        XCTAssertEqual(records.map(\.label), ["炸清", "单杆100+"])
        XCTAssertEqual(records.map(\.count), [1, 2])
    }

    func testV2SummaryExplainsMatchesWithoutMemberWording() {
        let recorded = MemberRankingRights.summary(MatchRankingRightsInput(
            rankRulesVersion: 2,
            achievements: ["break_clear": 1],
            myRankDetails: [MatchRankDetail(label: "胜负结算（赛前胜率 36%）", value: 57)]
        ))
        XCTAssertEqual(recorded.badgeText, "高光不计分")
        XCTAssertFalse(recorded.countedInRanking)
        XCTAssertTrue(recorded.description.contains("炸清x1"))
        XCTAssertTrue(recorded.description.contains("排位分只看胜负与对手实力"))
        XCTAssertFalse(recorded.description.contains("VIP"))

        let plain = MemberRankingRights.summary(MatchRankingRightsInput(
            rankRulesVersion: 2,
            achievements: [:]
        ))
        XCTAssertEqual(plain.badgeText, "只看胜负")
        XCTAssertTrue(plain.description.contains("会员与普通用户规则相同"))
    }

    func testPracticeMatchesAreMarkedAsNotCountedInRanking() {
        let recorded = MemberRankingRights.summary(MatchRankingRightsInput(
            matchMode: "practice",
            achievements: ["break_clear": 1]
        ))
        XCTAssertEqual(recorded.badgeText, "练习赛")
        XCTAssertEqual(recorded.title, "练习赛不计入排位")
        XCTAssertEqual(recorded.tone, "slate")
        XCTAssertFalse(recorded.countedInRanking)
        XCTAssertEqual(recorded.recordedAchievementsText, "炸清x1")
        XCTAssertEqual(recorded.description, "本场记录了 炸清x1，高光会保留在战报和成就里。")

        let plain = MemberRankingRights.summary(MatchRankingRightsInput(matchMode: "practice"))
        XCTAssertEqual(plain.recordedAchievementsText, "")
        XCTAssertEqual(plain.description, "练习赛的胜负和战绩不影响排位分。")

        // 排位赛规则保持原样。
        let ranked = MemberRankingRights.summary(MatchRankingRightsInput(
            matchMode: "ranked",
            rankRulesVersion: 2,
            achievements: [:]
        ))
        XCTAssertEqual(ranked.badgeText, "只看胜负")
    }
}
