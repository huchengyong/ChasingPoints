import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/snooker-match-format.test.mjs`（读取 `.vue` 源码的两条不搬）。
@MainActor
final class SnookerMatchFormatTests: XCTestCase {
    func testSnookerFormatNormalizesFreeAndClampsRaceToTargets() {
        XCTAssertEqual(
            SnookerMatchFormat.normalize(),
            SnookerMatchFormatState(format: SnookerMatchFormat.free, targetWins: 0)
        )
        XCTAssertEqual(
            SnookerMatchFormat.normalize(SnookerMatchFormatInput(snookerFormat: "race_to", snookerTargetWins: 10)),
            SnookerMatchFormatState(format: SnookerMatchFormat.raceTo, targetWins: 10)
        )
        XCTAssertEqual(
            SnookerMatchFormat.normalize(SnookerMatchFormatInput(snookerFormat: "race_to", snookerTargetWins: 99)),
            SnookerMatchFormatState(format: SnookerMatchFormat.raceTo, targetWins: 25)
        )
        XCTAssertEqual(SnookerMatchFormat.targetOptions.count, 25)
    }

    func testSnookerFormatCopyDescribesFreeAndRaceToLimits() {
        XCTAssertEqual(SnookerMatchFormat.label("free"), "自由局数")
        XCTAssertEqual(SnookerMatchFormat.label("race_to", targetWins: 10), "抢10局制")
        XCTAssertEqual(SnookerMatchFormat.hint("free"), "不预设胜局，最多进行49局")
        XCTAssertEqual(SnookerMatchFormat.hint("race_to", targetWins: 10), "先达到10胜，最多可能进行19局")
    }

    func testFreeFormatCanEndOnlyAfterACompletedFrameAndStopsAt49Frames() {
        XCTAssertTrue(SnookerMatchFormat.canFinishFreeSnookerMatch(
            format: "free", currentFrameStarted: false, myScore: 1, opponentScore: 1
        ))
        XCTAssertFalse(SnookerMatchFormat.canFinishFreeSnookerMatch(
            format: "free", currentFrameStarted: true, myScore: 1, opponentScore: 0
        ))
        XCTAssertFalse(SnookerMatchFormat.canFinishFreeSnookerMatch(
            format: "free", currentFrameStarted: false, myScore: 0, opponentScore: 0
        ))
        XCTAssertTrue(SnookerMatchFormat.canStartNextSnookerFrame(
            format: "free", myScore: 24, opponentScore: 24
        ))
        XCTAssertFalse(SnookerMatchFormat.canStartNextSnookerFrame(
            format: "free", myScore: 25, opponentScore: 24
        ))
    }

    func testRaceToFormatStopsWhenEitherPlayerReachesTheTarget() {
        XCTAssertTrue(SnookerMatchFormat.canStartNextSnookerFrame(
            format: "race_to", targetWins: 10, myScore: 9, opponentScore: 9
        ))
        XCTAssertFalse(SnookerMatchFormat.canStartNextSnookerFrame(
            format: "race_to", targetWins: 10, myScore: 10, opponentScore: 9
        ))
    }
}
