import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/match-navigation.test.mjs`。
@MainActor
final class MatchNavigationTests: XCTestCase {
    func testShouldLeavePlayingPageReturnsTrueWhenMatchDetailStatusIsCompleted() {
        XCTAssertTrue(MatchNavigation.shouldLeavePlayingPage(pageMatchId: 42, detailStatus: 2))
    }

    func testShouldLeavePlayingPageReturnsTrueWhenCurrentActiveMatchNoLongerMatchesPageMatch() {
        XCTAssertTrue(MatchNavigation.shouldLeavePlayingPage(pageMatchId: 42, detailStatus: 1, currentMatchId: 99))
    }

    func testShouldLeavePlayingPageReturnsTrueWhenThereIsNoCurrentActiveMatchForThisPageMatch() {
        XCTAssertTrue(MatchNavigation.shouldLeavePlayingPage(pageMatchId: 42, detailStatus: 1, currentMatchId: 0))
    }

    func testShouldLeavePlayingPageReturnsFalseWhenCurrentActiveMatchIsStillThisMatch() {
        XCTAssertFalse(MatchNavigation.shouldLeavePlayingPage(pageMatchId: 42, detailStatus: 1, currentMatchId: 42))
    }

    func testShouldLeavePlayingPageReturnsFalseWhenCurrentActiveMatchIsTemporarilyUnknown() {
        XCTAssertFalse(MatchNavigation.shouldLeavePlayingPage(pageMatchId: 42, detailStatus: 1))
    }

    func testMatchHistoryNavigationRoutesPointToUserCenterAndMatchHistoryPage() {
        XCTAssertEqual(MatchNavigation.historyTab, .userCenter)
        XCTAssertEqual(MatchNavigation.historyPage, .matchHistory)
    }

    func testConsumeResultNavigationGuardOnlyAllowsTheFirstResultPageNavigation() {
        var hasNavigatedToResult = false
        XCTAssertTrue(MatchNavigation.consumeResultNavigationGuard(&hasNavigatedToResult))
        XCTAssertFalse(MatchNavigation.consumeResultNavigationGuard(&hasNavigatedToResult))
    }
}
