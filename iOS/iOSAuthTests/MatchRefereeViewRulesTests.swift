import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/match-referee-view.test.mjs` 的行为断言。
/// 原用例里 `resolveRefereeHistoryCard` 的部分（列表卡状态文案）原生端已有自己的历史列表实现，未迁移。
final class MatchRefereeViewRulesTests: XCTestCase {
    func testPlayerAndRefereeResultModesStayDistinct() {
        XCTAssertTrue(MatchRefereeView.resultViewConfig(viewerRole: "player1", status: 2).showH2HActions)
        XCTAssertTrue(MatchRefereeView.resultViewConfig(viewerRole: "player2", status: 2).showH2HActions)

        let referee = MatchRefereeView.resultViewConfig(viewerRole: "referee", status: 2)
        XCTAssertEqual(referee.pageTitle, "执裁详情")
        XCTAssertFalse(referee.showH2HActions)
        XCTAssertFalse(referee.showRematchButton)
    }

    func testPersonalInsightsOnlyBelongToPlayerViews() {
        XCTAssertFalse(MatchRefereeView.resultViewConfig(viewerRole: "referee", status: 2).showPersonalInsights)
        XCTAssertFalse(MatchRefereeView.resultViewConfig(viewerRole: "referee", status: 1).showPersonalInsights)
        XCTAssertTrue(MatchRefereeView.resultViewConfig(viewerRole: "player1", status: 2).showPersonalInsights)
        XCTAssertTrue(MatchRefereeView.resultViewConfig(viewerRole: "player2", status: 2).showPersonalInsights)
    }

    func testUnknownAttributionNeverClaimsRefereeSettlement() {
        let unknown = MatchRefereeView.identityCard(
            refereeBound: true,
            refereeUserId: 3,
            completionSource: "unknown",
            status: 2
        )

        XCTAssertFalse(unknown.hasReliableAttribution)
        XCTAssertEqual(unknown.completionLabel, "")
    }

    func testOrdinaryRefereeCopyStaysNeutralAndKnownCompletionLabelsStaySafe() {
        let card = MatchRefereeView.identityCard(
            refereeBound: true,
            refereeUserId: 3,
            refereeName: "裁判丙",
            completionSource: "referee",
            status: 2
        )

        XCTAssertEqual(card.neutralLabel, "本场裁判")
        XCTAssertEqual(card.completionLabel, "裁判结算")
        XCTAssertEqual(MatchRefereeView.completionSourceLabel("player_cancelled"), "未知")
        XCTAssertEqual(MatchRefereeView.completionSourceLabel("finish_timeout"), "确认超时自动完成")
    }
}
