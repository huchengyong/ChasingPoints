import XCTest
@testable import iOS

@MainActor
final class AppNavigatorTests: XCTestCase {
    func testSelectScanWhileSignedInOnlyPresentsScanner() {
        let navigator = AppNavigator()

        navigator.select(.scan, isSignedIn: true)

        XCTAssertEqual(navigator.selectedTab, .home)
        XCTAssertTrue(navigator.isScanPresented)
    }

    func testSelectScanWhileSignedOutOpensLoginInMeTab() {
        let navigator = AppNavigator()

        navigator.select(.scan, isSignedIn: false)

        XCTAssertEqual(navigator.selectedTab, .me)
        XCTAssertEqual(navigator.path(for: .me).count, 1)
        XCTAssertFalse(navigator.isScanPresented)
    }

    func testOpenPushesOntoCurrentTabOnly() {
        let navigator = AppNavigator()
        navigator.select(.news, isSignedIn: true)

        navigator.open(.settings)

        XCTAssertEqual(navigator.selectedTab, .news)
        XCTAssertEqual(navigator.path(for: .news).count, 1)
        XCTAssertEqual(navigator.path(for: .home).count, 0)
        XCTAssertEqual(navigator.path(for: .me).count, 0)
    }

    func testOpenInTabSwitchesTabClearsItsStackAndPushes() {
        let navigator = AppNavigator()
        navigator.select(.home, isSignedIn: true)
        navigator.open(.settings)
        navigator.open(.feedback)
        XCTAssertEqual(navigator.path(for: .home).count, 2)

        navigator.open(.ranking, in: .me)

        XCTAssertEqual(navigator.selectedTab, .me)
        XCTAssertEqual(navigator.path(for: .me).count, 1)

        navigator.open(.settings, in: .home)
        XCTAssertEqual(navigator.selectedTab, .home)
        XCTAssertEqual(navigator.path(for: .home).count, 1, "跨 Tab 打开应清掉该 Tab 原有栈")
    }

    func testSelectingSelectedTabPopsToRoot() {
        let navigator = AppNavigator()
        navigator.select(.messages, isSignedIn: true)
        navigator.open(.challenges(showHistory: false))
        XCTAssertEqual(navigator.path(for: .messages).count, 1)

        navigator.select(.messages, isSignedIn: true)

        XCTAssertEqual(navigator.selectedTab, .messages)
        XCTAssertEqual(navigator.path(for: .messages).count, 0)
    }

    func testResetClearsAllStacksAndScan() {
        let navigator = AppNavigator()
        navigator.open(.settings)
        navigator.select(.me, isSignedIn: true)
        navigator.open(.matchHistory)
        navigator.requestScan(isSignedIn: true)

        navigator.reset()

        for tab in [MainTab.home, .news, .messages, .me] {
            XCTAssertEqual(navigator.path(for: tab).count, 0)
        }
        XCTAssertFalse(navigator.isScanPresented)
        XCTAssertEqual(navigator.selectedTab, .me)
    }

    func testOpenLoginKeepsSingleLoginEntry() {
        let navigator = AppNavigator()

        navigator.openLogin()
        navigator.openLogin()
        navigator.openLogin()

        XCTAssertEqual(navigator.selectedTab, .me)
        XCTAssertEqual(navigator.path(for: .me).count, 1)
    }
}
