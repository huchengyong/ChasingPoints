import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/pool-match-format.test.mjs`（读取 `.vue` 源码的一条不搬）。
@MainActor
final class PoolMatchFormatTests: XCTestCase {
    func testPoolMatchFormatNormalizesLegacyFreeAndRaceToTargets() {
        XCTAssertEqual(
            PoolMatchFormat.normalize(),
            PoolMatchFormatState(format: PoolMatchFormat.free, targetWins: 0)
        )
        XCTAssertEqual(
            PoolMatchFormat.normalize(PoolMatchFormatInput(matchFormat: "legacy")),
            PoolMatchFormatState(format: PoolMatchFormat.legacy, targetWins: 0)
        )
        XCTAssertEqual(
            PoolMatchFormat.normalize(PoolMatchFormatInput(matchFormat: "race_to", targetWins: 65)),
            PoolMatchFormatState(format: PoolMatchFormat.raceTo, targetWins: 65)
        )
        XCTAssertEqual(
            PoolMatchFormat.normalize(PoolMatchFormatInput(matchFormat: "race_to", targetWins: 99)),
            PoolMatchFormatState(format: PoolMatchFormat.raceTo, targetWins: 65)
        )
        XCTAssertEqual(PoolMatchFormat.targetOptions.count, 65)
    }

    func testPoolMatchFormatCopyDescribesFreeDrawAndRaceToLimits() {
        XCTAssertEqual(PoolMatchFormat.label("free"), "自由局数")
        XCTAssertEqual(PoolMatchFormat.label("race_to", targetWins: 65), "抢65局制")
        XCTAssertEqual(PoolMatchFormat.hint("free"), "不预设胜局，最多进行129局，允许平局")
        XCTAssertEqual(PoolMatchFormat.hint("race_to", targetWins: 65), "先达到65胜，最多可能进行129局")
    }

    func testPoolFinishAndContinuationRulesMirrorServerLimits() {
        XCTAssertFalse(PoolMatchFormat.canFinishFreePoolMatch(format: "free", myScore: 0, opponentScore: 0))
        XCTAssertTrue(PoolMatchFormat.canFinishFreePoolMatch(format: "free", myScore: 1, opponentScore: 0))
        XCTAssertTrue(PoolMatchFormat.canFinishFreePoolMatch(format: "free", myScore: 64, opponentScore: 64))
        XCTAssertTrue(PoolMatchFormat.canContinuePoolMatch(format: "free", myScore: 64, opponentScore: 64))
        XCTAssertFalse(PoolMatchFormat.canContinuePoolMatch(format: "free", myScore: 65, opponentScore: 64))
        XCTAssertTrue(PoolMatchFormat.canContinuePoolMatch(
            format: "race_to", targetWins: 65, myScore: 64, opponentScore: 64
        ))
        XCTAssertFalse(PoolMatchFormat.canContinuePoolMatch(
            format: "race_to", targetWins: 65, myScore: 65, opponentScore: 64
        ))
    }

    func testPoolMatchFormatAppliesOnlyToChineseEightAndAmericanNineBall() {
        XCTAssertTrue(PoolMatchFormat.isPoolMatchFormatGameType(3))
        XCTAssertTrue(PoolMatchFormat.isPoolMatchFormatGameType(4))
        XCTAssertFalse(PoolMatchFormat.isPoolMatchFormatGameType(2))
    }
}
