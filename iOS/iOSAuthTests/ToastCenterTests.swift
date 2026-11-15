import XCTest
@testable import iOS

@MainActor
final class ToastCenterTests: XCTestCase {
    func testToastDismissesAfterDuration() async {
        let center = ToastCenter(duration: 0.05)

        center.show("已保存")

        XCTAssertEqual(center.message, "已保存")
        await waitUntil(timeout: 1) { center.message == nil }
    }

    func testBlankMessageIsIgnored() {
        let center = ToastCenter(duration: 0.05)

        center.show("   ")
        XCTAssertNil(center.message)

        center.show("有内容")
        center.show("\n\t")
        XCTAssertEqual(center.message, "有内容")
    }

    func testNewToastReplacesOldAndOldTimerCannotClearIt() async throws {
        let center = ToastCenter(duration: 0.15)

        center.show("第一条")
        try await Task.sleep(nanoseconds: 100_000_000)
        center.show("第二条")

        // 第一条的原定时间已过，但它不能清掉后一条。
        try await Task.sleep(nanoseconds: 100_000_000)
        XCTAssertEqual(center.message, "第二条")

        await waitUntil(timeout: 1) { center.message == nil }
    }

    func testDismissClearsImmediately() {
        let center = ToastCenter(duration: 10)

        center.show("稍后消失")
        center.dismiss()

        XCTAssertNil(center.message)
    }
}
