import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/match-result-page.test.mjs` 的行为断言；
/// 其中两条是 Vue / JS 源码正则检查，原生端没有对应实现，未迁移。
final class MatchResultPageRulesTests: XCTestCase {
    func testRankingRightsMarksHighlightsAsRecordedOnlyWhenNoRankDetailWasCounted() {
        let summary = MemberRankingRights.summary(MatchRankingRightsInput(
            achievements: ["break_clear": 1, "break_100": 1],
            myRankDetails: [MatchRankDetail(label: "基础分", value: 20)]
        ))

        XCTAssertFalse(summary.countedInRanking)
        XCTAssertEqual(summary.badgeText, "高光仅记录")
        XCTAssertTrue(summary.description.contains("普通用户高光会保留记录，VIP用户才会按等级倍率计入"))
    }

    func testRankingRightsExplainsCountedMemberHighlightsAndDailyCapClipping() {
        let summary = MemberRankingRights.summary(MatchRankingRightsInput(
            achievements: ["break_147": 1],
            myRankDetails: [
                MatchRankDetail(label: "基础分", value: 20),
                MatchRankDetail(label: "VIP特殊战绩分", value: 30),
                MatchRankDetail(label: "每日封顶", value: -6),
            ]
        ))

        XCTAssertTrue(summary.countedInRanking)
        XCTAssertTrue(summary.capped)
        XCTAssertEqual(summary.badgeText, "已计入并触发封顶")
        XCTAssertTrue(summary.description.contains("30 分特殊战绩已计入排位"))
        XCTAssertTrue(summary.description.contains("6 分受到VIP特殊战绩单日上限影响"))
    }

    func testRankingRightsFallsBackToBaseRankExplanationWithoutAchievements() {
        let summary = MemberRankingRights.summary(MatchRankingRightsInput(
            achievements: [:],
            myRankDetails: [MatchRankDetail(label: "基础分", value: 20)]
        ))

        XCTAssertFalse(summary.countedInRanking)
        XCTAssertEqual(summary.recordedAchievementsText, "")
        XCTAssertEqual(summary.title, "本场没有可计入排位的特殊战绩")
    }

    func testReadyRewardsMapIntoRewardItems() throws {
        let summary = try decodeRewardSummary([
            "success": true,
            "status": "ready",
            "list": [[
                "achievement_id": 12,
                "achievement_key": "first_win",
                "achievement_name": "首胜",
                "description": "赢得第一场有效排位",
                "icon": "/static/achievement/first-win.png",
                "reward_title_id": 7,
                "reward_title_name": "初露锋芒",
                "unlocked_at": "2026-07-23 20:00:00",
            ]],
        ])
        let view = MatchResultPage.resolveRewardSummary(summary)

        XCTAssertEqual(view.state, .ready(summary.list))
        XCTAssertTrue(view.visible)
        XCTAssertEqual(view.rewards.count, 1)
        XCTAssertEqual(view.rewards[0].achievementName, "首胜")
        XCTAssertEqual(view.rewards[0].rewardTitleName, "初露锋芒")
        XCTAssertFalse(view.shouldRetry)
    }

    func testReadyEmptySummaryIsHidden() throws {
        let summary = try decodeRewardSummary(["success": true, "status": "ready", "list": []])

        let view = MatchResultPage.resolveRewardSummary(summary)

        XCTAssertEqual(view.state, .hidden)
        XCTAssertFalse(view.visible)
        XCTAssertTrue(view.rewards.isEmpty)
    }

    func testPendingStaysDistinctFromEmptyAndLimitsRetries() throws {
        let pending = try decodeRewardSummary([
            "success": true,
            "status": "pending",
            "message": "正在结算",
            "list": [],
        ])
        let retrying = MatchResultPage.resolveRewardSummary(pending, attempt: 1, maxAttempts: 3)
        let exhausted = MatchResultPage.resolveRewardSummary(pending, attempt: 3, maxAttempts: 3)

        XCTAssertEqual(retrying.state, .pending)
        XCTAssertTrue(retrying.visible)
        XCTAssertEqual(retrying.message, "正在结算")
        XCTAssertTrue(retrying.shouldRetry)
        XCTAssertEqual(exhausted.state, .pending)
        XCTAssertFalse(exhausted.shouldRetry)
    }

    /// 原生端把 `success=false` 视为解码失败，由战报模型隐藏奖励块；这里覆盖同类的未知状态。
    func testUnknownRewardStatusHidesTheModule() throws {
        let summary = try decodeRewardSummary([
            "status": "error",
            "message": "奖励数据暂不可用",
            "list": [],
        ])

        let view = MatchResultPage.resolveRewardSummary(summary)

        XCTAssertEqual(view.state, .hidden)
        XCTAssertFalse(view.visible)
        XCTAssertEqual(view.message, "奖励数据暂不可用")
        XCTAssertFalse(view.shouldRetry)
    }

    func testHonorWallFallsBackToChineseEightForUnsupportedGameTypes() {
        XCTAssertEqual(MatchResultPage.honorWallGameType(4), 4)
        XCTAssertEqual(MatchResultPage.honorWallGameType(99), 3)
        XCTAssertEqual(MatchResultPage.honorWallGameType(nil), 3)
    }

    func testOutcomePrefersSettledResultOverFrameScores() {
        // 自由局数整场认输：认输方局分领先但结果为负，胜方局分落后但结果为胜。
        XCTAssertEqual(MatchResultPage.outcome(result: 2, myScore: 2, opponentScore: 1), .loss)
        XCTAssertEqual(MatchResultPage.outcome(result: 1, myScore: 1, opponentScore: 2), .win)
        // 已结算平局不按局分判胜负。
        XCTAssertEqual(MatchResultPage.outcome(result: 3, myScore: 2, opponentScore: 1), .draw)
        // 旧响应缺少 result 或对局未结算时按比分判断，比分相等为平局。
        XCTAssertEqual(MatchResultPage.outcome(result: nil, myScore: 3, opponentScore: 1), .win)
        XCTAssertEqual(MatchResultPage.outcome(result: nil, myScore: 1, opponentScore: 3), .loss)
        XCTAssertEqual(MatchResultPage.outcome(result: nil, myScore: 2, opponentScore: 2), .draw)
        XCTAssertEqual(MatchResultPage.outcome(result: 0, myScore: 1, opponentScore: 3), .loss)
    }

    func testRewardIconsStoredAsStaticPathsResolveToTheCDN() throws {
        let summary = try decodeRewardSummary([
            "success": true,
            "status": "ready",
            "list": [[
                "achievement_id": 1,
                "achievement_key": "wins_1",
                "icon": "/static/images/achievements/wins_1.png",
            ]],
        ])
        let view = MatchResultPage.resolveRewardSummary(summary)

        XCTAssertEqual(
            view.rewards[0].iconURL?.absoluteString,
            "https://cdn.zhuifen.cn/static/images/achievements/wins_1.png"
        )
    }
}

private func decodeRewardSummary(_ json: [String: Any]) throws -> MatchRewardSummary {
    let data = try JSONSerialization.data(withJSONObject: json)
    let decoder = JSONDecoder()
    decoder.keyDecodingStrategy = .convertFromSnakeCase
    return try decoder.decode(MatchRewardSummary.self, from: data)
}
