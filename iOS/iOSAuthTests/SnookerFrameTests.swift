import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/snooker-frame.test.mjs`。
@MainActor
final class SnookerFrameTests: XCTestCase {
    func testResolveSnookerNextFrameActionStartsNextFrameDirectlyWhenCurrentFrameIsAlreadyClosed() {
        XCTAssertEqual(
            SnookerFrame.nextFrameAction(SnookerFrameSettleInput(
                currentFrameStarted: false, myFrameScore: 0, opponentFrameScore: 0
            )),
            SnookerFrameAction(action: "start_next_round", winner: nil, message: nil)
        )
    }

    func testResolveSnookerNextFrameActionAutoAwardsCurrentFrameToTheHigherScore() {
        XCTAssertEqual(
            SnookerFrame.nextFrameAction(SnookerFrameSettleInput(
                currentFrameStarted: true, myFrameScore: 67, opponentFrameScore: 42
            )),
            SnookerFrameAction(action: "settle_and_start_next_round", winner: 1, message: nil)
        )
        XCTAssertEqual(
            SnookerFrame.nextFrameAction(SnookerFrameSettleInput(
                currentFrameStarted: true, myFrameScore: 18, opponentFrameScore: 39
            )),
            SnookerFrameAction(action: "settle_and_start_next_round", winner: 2, message: nil)
        )
    }

    func testResolveSnookerNextFrameActionBlocksAutoSettlementWhenFrameScoresAreTied() {
        XCTAssertEqual(
            SnookerFrame.nextFrameAction(SnookerFrameSettleInput(
                currentFrameStarted: true, myFrameScore: 35, opponentFrameScore: 35
            )),
            SnookerFrameAction(
                action: "blocked",
                winner: nil,
                message: "当前局比分相同，无法自动判定胜负，请继续记分或结束本场对局"
            )
        )
    }

    func testResolveSnookerFinishMatchActionAutoSettlesTheCurrentFrameBeforeFinishingMatch() {
        XCTAssertEqual(
            SnookerFrame.finishMatchAction(SnookerFrameSettleInput(
                currentFrameStarted: true, myFrameScore: 52, opponentFrameScore: 37
            )),
            SnookerFrameAction(action: "settle_and_finish_match", winner: 1, message: nil)
        )
    }

    func testResolveSnookerFinishMatchActionAllowsFinishingWhenCurrentFrameIsEmptyOrClosed() {
        XCTAssertEqual(
            SnookerFrame.finishMatchAction(SnookerFrameSettleInput(
                currentFrameStarted: false, myFrameScore: 0, opponentFrameScore: 0
            )),
            SnookerFrameAction(action: "finish_match", winner: nil, message: nil)
        )
        XCTAssertEqual(
            SnookerFrame.finishMatchAction(SnookerFrameSettleInput(
                currentFrameStarted: true, myFrameScore: 0, opponentFrameScore: 0
            )),
            SnookerFrameAction(action: "finish_match", winner: nil, message: nil)
        )
    }

    func testResolveSnookerFinishMatchActionBlocksFinishingWhenNonZeroFrameScoresAreTied() {
        XCTAssertEqual(
            SnookerFrame.finishMatchAction(SnookerFrameSettleInput(
                currentFrameStarted: true, myFrameScore: 41, opponentFrameScore: 41
            )),
            SnookerFrameAction(
                action: "blocked",
                winner: nil,
                message: "当前局比分相同，无法自动判定胜负，请继续记分后再结束本场对局"
            )
        )
    }
}
