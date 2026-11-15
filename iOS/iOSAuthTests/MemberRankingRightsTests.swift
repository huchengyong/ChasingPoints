import Foundation
import XCTest
@testable import iOS

/// 迁移自 `uniapp/tests/member-ranking-rights.test.mjs`。
@MainActor
final class MemberRankingRightsTests: XCTestCase {
    func testAchievementRecordsKeepOnlyRecordedSpecialAchievements() {
        let records = MemberRankingRights.achievementRecords([
            "break_clear": 1,
            "break_100": 2,
            "golden_break": 0,
        ])

        XCTAssertEqual(records.map { [$0.label, "\($0.count)"] }, [["炸清", "1"], ["单杆100+", "2"]])
    }

    func testV2SummaryExplainsMatchWithoutMemberWording() throws {
        let recorded = MemberRankingRights.summary(match: try report([
            "rank_rules_version": 2,
            "achievements": ["break_clear": 1],
            "my_rank_details": [["label": "胜负结算（赛前胜率 36%）", "value": 57]],
        ]))
        XCTAssertEqual(recorded.badgeText, "高光不计分")
        XCTAssertFalse(recorded.countedInRanking)
        XCTAssertTrue(recorded.description.contains("炸清x1"))
        XCTAssertTrue(recorded.description.contains("排位分只看胜负与对手实力"))
        XCTAssertFalse(recorded.description.contains("VIP"))

        let plain = MemberRankingRights.summary(match: try report([
            "rank_rules_version": 2,
            "achievements": [:],
        ]))
        XCTAssertEqual(plain.badgeText, "只看胜负")
        XCTAssertTrue(plain.description.contains("会员与普通用户规则相同"))
    }

    func testPracticeMatchIsNotCountedInRanking() throws {
        let recorded = MemberRankingRights.summary(match: try report([
            "match_mode": "practice",
            "achievements": ["break_clear": 1],
        ]))
        XCTAssertEqual(recorded.badgeText, "练习赛")
        XCTAssertEqual(recorded.title, "练习赛不计入排位")
        XCTAssertEqual(recorded.tone, "slate")
        XCTAssertFalse(recorded.countedInRanking)
        XCTAssertEqual(recorded.recordedAchievementsText, "炸清x1")
        XCTAssertEqual(recorded.description, "本场记录了 炸清x1，高光会保留在战报和成就里。")

        let plain = MemberRankingRights.summary(match: try report(["match_mode": "practice"]))
        XCTAssertEqual(plain.recordedAchievementsText, "")
        XCTAssertEqual(plain.description, "练习赛的胜负和战绩不影响排位分。")

        let ranked = MemberRankingRights.summary(match: try report([
            "match_mode": "ranked",
            "rank_rules_version": 2,
            "achievements": [:],
        ]))
        XCTAssertEqual(ranked.badgeText, "只看胜负")
    }

    func testLegacyRankedMatchCountsAchievementScore() throws {
        let counted = MemberRankingRights.summary(match: try report([
            "achievements": ["break_clear": 1],
            "my_rank_details": [
                ["label": "基础分", "value": 10],
                ["label": "VIP特殊战绩分（炸清）", "value": 6],
            ],
        ]))
        XCTAssertTrue(counted.countedInRanking)
        XCTAssertFalse(counted.capped)
        XCTAssertEqual(counted.badgeText, "高光已计入排位")
        XCTAssertEqual(counted.tone, "emerald")
        XCTAssertTrue(counted.description.contains("特殊战绩分已计入排位"))

        let capped = MemberRankingRights.summary(match: try report([
            "achievements": ["break_clear": 2],
            "my_rank_details": [
                ["label": "VIP特殊战绩分", "value": 12],
                ["label": "每日封顶扣减", "value": -5],
            ],
        ]))
        XCTAssertTrue(capped.capped)
        XCTAssertEqual(capped.tone, "gold")
        XCTAssertTrue(capped.badgeText.contains("触发封顶"))
        XCTAssertTrue(capped.description.contains("5 分受到VIP特殊战绩单日上限影响"))
    }

    func testLegacyMatchWithoutAchievementsOnlySettlesBaseScore() throws {
        let summary = MemberRankingRights.summary(match: try report(["achievements": [:]]))
        XCTAssertEqual(summary.badgeText, "无高光排位分")
        XCTAssertEqual(summary.description, "本场排位只按基础分结算。")
    }

    private func report(_ overrides: [String: Any] = [:]) throws -> MatchReport {
        guard var match = MatchHistoryFixtures.playerDetail()["match"] as? [String: Any] else {
            throw NSError(domain: "MemberRankingRightsTests", code: 1)
        }
        for (key, value) in overrides {
            match[key] = value
        }
        let data = try JSONSerialization.data(withJSONObject: match)
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        return try decoder.decode(MatchReport.self, from: data)
    }
}
