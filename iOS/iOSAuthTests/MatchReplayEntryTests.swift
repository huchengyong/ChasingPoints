import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/match-replay-entry.test.mjs`。
/// 「result and detail pages wire…」读 `.vue` 源码断言接线，原生端不适用，未搬。
final class MatchReplayEntryTests: XCTestCase {
    func testReplayEntryIsOnlyOfferedToPlayersOfAFinishedMatch() {
        XCTAssertTrue(MatchReplayEntry.canOpenMatchReplay(status: 2, viewerRole: "player1"))
        XCTAssertTrue(MatchReplayEntry.canOpenMatchReplay(status: 2, viewerRole: "player2"))
        XCTAssertFalse(MatchReplayEntry.canOpenMatchReplay(status: 2, viewerRole: "referee"), "裁判没有复盘入口")
        XCTAssertFalse(MatchReplayEntry.canOpenMatchReplay(status: 1, viewerRole: "player1"), "进行中的对局没有复盘")
        XCTAssertFalse(MatchReplayEntry.canOpenMatchReplay(status: 3, viewerRole: "player1"), "已取消的对局没有复盘")
        XCTAssertFalse(MatchReplayEntry.canOpenMatchReplay(status: 2), "观战者没有复盘入口")
        XCTAssertFalse(MatchReplayEntry.canOpenMatchReplay())
    }

    func testHistoryDetailOpenedForTheLoggedInUserCountsAsTheirOwnMatch() {
        XCTAssertTrue(MatchReplayEntry.canOpenMatchReplay(
            status: 2, source: "history", perspectiveUserId: 5, currentUserId: 5
        ))
        // 查看别人的历史对局（视角用户不是自己）没有入口。
        XCTAssertFalse(MatchReplayEntry.canOpenMatchReplay(
            status: 2, source: "history", perspectiveUserId: 6, currentUserId: 5
        ))
        XCTAssertFalse(MatchReplayEntry.canOpenMatchReplay(
            status: 2, source: "history", perspectiveUserId: 0, currentUserId: 0
        ))
        XCTAssertFalse(MatchReplayEntry.canOpenMatchReplay(
            status: 2, source: "", perspectiveUserId: 5, currentUserId: 5
        ))
    }

    func testReplayDestinationCarriesTheMatchId() {
        XCTAssertEqual(MatchReplayEntry.replayDestination(matchId: 88), .replay(matchId: 88))
        // JS 对非法 id（如 'x'）会退化成 match_id=0；Swift 的 Int64 入参在调用前就保证了类型。
    }
}
