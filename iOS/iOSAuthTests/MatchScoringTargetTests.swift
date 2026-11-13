import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/match-scoring-target.test.mjs`（读取 `.vue` 源码的两条不搬）。
@MainActor
final class MatchScoringTargetTests: XCTestCase {
    func testMatchScoringActorKeepsRefereeFixedAndConvertsBothPlayerPerspectives() {
        XCTAssertEqual(MatchScoringTarget.resolveMatchApiActor(viewerRole: "referee", isPlayer1: true, uiActor: 1), 1)
        XCTAssertEqual(MatchScoringTarget.resolveMatchApiActor(viewerRole: "referee", isPlayer1: true, uiActor: 2), 2)
        XCTAssertEqual(MatchScoringTarget.resolveMatchApiActor(viewerRole: "player1", isPlayer1: true, uiActor: 1), 1)
        XCTAssertEqual(MatchScoringTarget.resolveMatchApiActor(viewerRole: "player1", isPlayer1: true, uiActor: 2), 2)
        XCTAssertEqual(MatchScoringTarget.resolveMatchApiActor(viewerRole: "player2", isPlayer1: false, uiActor: 1), 2)
        XCTAssertEqual(MatchScoringTarget.resolveMatchApiActor(viewerRole: "player2", isPlayer1: false, uiActor: 2), 1)
    }

    func testRefereeOptionsUseFixedParticipantLabelsForBothRoundWinGameTypes() {
        let chineseEightBall = MatchScoringTarget.buildRoundWinActionOptions(gameType: 3, viewerRole: "referee", uiActor: 1)
        let americanNineBall = MatchScoringTarget.buildRoundWinActionOptions(gameType: 4, viewerRole: "referee", uiActor: 2)

        XCTAssertEqual(chineseEightBall.map(\.title), ["选手1普胜", "选手1炸清", "选手1接清"])
        XCTAssertEqual(americanNineBall.map(\.title), ["选手2普胜", "选手2小金", "选手2大金"])
    }

    func testJiuqiuOptionsAndScoreFeedbackDescribeTheSelectedParticipant() {
        XCTAssertEqual(
            MatchScoringTarget.buildJiuqiuWinOptions(viewerRole: "referee", uiActor: 1).map(\.desc),
            ["选手1 +4 分", "选手1 +7 分", "选手1 +10 分"]
        )
        XCTAssertEqual(MatchScoringTarget.resolveScoreFeedback(viewerRole: "referee", uiActor: 2, score: 7), "已给选手2 +7 分")
        XCTAssertEqual(MatchScoringTarget.resolveWinFeedback(viewerRole: "referee", uiActor: 1), "已判给选手1")
    }

    func testFoulFeedbackDistinguishesTheFoulingParticipantFromTheScoringParticipant() {
        XCTAssertEqual(
            MatchScoringTarget.resolveFoulFeedback(viewerRole: "referee", foulingActor: 1, score: 4),
            "选手1犯规，选手2 +4 分"
        )
        XCTAssertEqual(
            MatchScoringTarget.resolveFoulFeedback(viewerRole: "referee", foulingActor: 2, score: 7),
            "选手2犯规，选手1 +7 分"
        )
        XCTAssertEqual(
            MatchScoringTarget.resolveFoulFeedback(viewerRole: "player2", foulingActor: 2, score: 1),
            "对手犯规，我方 +1 分"
        )
        XCTAssertEqual(MatchScoringTarget.resolveScoringParticipantLabel(viewerRole: "player1", uiActor: 1), "我方")
    }
}
