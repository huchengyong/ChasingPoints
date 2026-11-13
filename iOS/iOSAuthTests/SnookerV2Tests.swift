import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/snooker-v2.test.mjs`（读取 `.vue` 源码的两条不搬）。
@MainActor
final class SnookerV2Tests: XCTestCase {
    func testNormalizeSnookerV2StateExposesAuthoritativeVisitAndRefereeState() {
        let state = SnookerV2.normalizeState(SnookerV2SnapshotInput(
            snookerRulesVersion: 2,
            bestOfFrames: 7,
            startingActor: 2,
            snookerPhase: "colors",
            snookerBallOn: "blue",
            snookerStriker: 2,
            snookerVisitNo: 6,
            snookerCurrentBreak: 38,
            snookerRedsRemaining: 0,
            snookerFreeBallAvailable: true,
            snookerCueBallInHand: true,
            snookerMissWarningActive: true,
            currentFrameStarted: true
        ))
        XCTAssertTrue(SnookerV2.isRulesV2(SnookerV2SnapshotInput(snookerRulesVersion: 2)))
        XCTAssertEqual(state.bestOfFrames, 7)
        XCTAssertEqual(state.startingActor, 2)
        XCTAssertEqual(state.striker, 2)
        XCTAssertEqual(state.visitNo, 6)
        XCTAssertEqual(state.currentBreak, 38)
        XCTAssertTrue(state.freeBallAvailable)
        XCTAssertTrue(state.missWarningActive)
    }

    func testPotOptionsFollowRedColorChoiceAndOrderedClearanceBallOn() {
        XCTAssertEqual(
            SnookerV2.potOptions(SnookerV2State(
                phase: "reds", ballOn: "red", redsRemaining: 15, currentFrameStarted: true, status: 1
            )).map(\.value),
            [1]
        )
        XCTAssertEqual(
            SnookerV2.potOptions(SnookerV2State(
                phase: "color_after_red", ballOn: "color_choice", currentFrameStarted: true, status: 1
            )).map(\.value),
            [2, 3, 4, 5, 6, 7]
        )
        XCTAssertEqual(
            SnookerV2.potOptions(SnookerV2State(
                phase: "colors", ballOn: "pink", currentFrameStarted: true, status: 1
            )).map(\.value),
            [6]
        )
        XCTAssertEqual(
            SnookerV2.potOptions(SnookerV2State(
                phase: "respotted_black_pending", ballOn: "black", currentFrameStarted: true, status: 1
            )).map(\.value),
            []
        )
    }

    func testRedCountPagesNeverExceedTheCrossPlatformActionSheetLimit() {
        XCTAssertEqual(SnookerV2.countPages(minimum: 1, maximum: 15), [
            [1, 2, 3, 4, 5, 6],
            [7, 8, 9, 10, 11, 12],
            [13, 14, 15],
        ])
        XCTAssertEqual(SnookerV2.countPages(minimum: 0, maximum: 15, pageSize: 20).map(\.count), [6, 6, 4])
    }

    func testFreeBallOptionsExcludeTheTrueBallOn() {
        let state = SnookerV2State(ballOn: "blue", freeBallAvailable: true)
        XCTAssertEqual(SnookerV2.freeBallOptions(state).map(\.value), [2, 3, 4, 6, 7])
    }

    func testStrokeBuildersSeparateRealRedsFreeBallAndTrueTarget() {
        let redState = SnookerV2State(ballOn: "red")
        XCTAssertEqual(
            SnookerV2.potPayload(
                state: redState,
                actor: 1,
                pottedReds: 2,
                freeBallValue: 7,
                freeBallPotted: true
            ),
            SnookerV2PotPayload(
                actor: 1,
                ballOnValue: 1,
                pottedReds: 2,
                ballOnPotted: false,
                freeBallValue: 7,
                freeBallPotted: true
            )
        )

        let colorState = SnookerV2State(ballOn: "color_choice")
        XCTAssertEqual(
            SnookerV2.potPayload(state: colorState, actor: 2, ballOnValue: 6, ballOnPotted: true).ballOnValue,
            6
        )
        XCTAssertEqual(
            SnookerV2.noScorePayload(state: SnookerV2State(ballOn: "red"), actor: 2),
            SnookerV2NoScorePayload(actor: 2, ballOnValue: 1, freeBallValue: nil)
        )
        XCTAssertEqual(
            SnookerV2.noScorePayload(state: SnookerV2State(ballOn: "yellow"), actor: 2, freeBallValue: 3),
            SnookerV2NoScorePayload(actor: 2, ballOnValue: 2, freeBallValue: 3)
        )
    }

    func testFoulBuilderKeepsOnlyDecisionsValidForTheSelectedResolution() {
        let replay = SnookerV2.foulPayload(
            state: SnookerV2State(ballOn: "black"),
            actor: 1,
            penalty: 7,
            redsRemoved: 1,
            resolution: "offender_replays_original",
            freeBallAwarded: true,
            foulAndMiss: true,
            missSequenceEligible: true,
            cueBallInHand: true
        )
        XCTAssertFalse(replay.freeBallAwarded)
        XCTAssertTrue(replay.missSequenceEligible)
        XCTAssertFalse(replay.cueBallInHand)
        XCTAssertEqual(SnookerV2.minimumPenalty(SnookerV2State(ballOn: "pink")), 6)
    }

    func testPhaseAndTargetCopyAreDerivedFromAuthoritativeState() {
        XCTAssertEqual(SnookerV2.phaseLabel(SnookerV2State(phase: "respotted_black")), "重置黑球决胜")
        XCTAssertEqual(SnookerV2.ballOnLabel(SnookerV2State(ballOn: "color_choice")), "自选彩球")
    }
}
