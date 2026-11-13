import XCTest
@testable import iOS

@MainActor
final class AppFormatTests: XCTestCase {
    func testDateTimeReadsISOAndSpaceSeparatedTimestamps() {
        XCTAssertEqual(AppFormat.dateTime("2026-11-10T20:28:53+08:00"), "2026年11月10日 20:28")
        XCTAssertEqual(AppFormat.dateTime("2026-11-10 20:28:53"), "2026年11月10日 20:28")
    }

    func testDateTimeReturnsEmptyForUnparseableInput() {
        XCTAssertEqual(AppFormat.dateTime("not-a-date"), "")
        XCTAssertEqual(AppFormat.dateTime(nil), "")
    }

    func testRelativeTimeReadsSpaceSeparatedAndISOTimestamps() {
        let now = AppFormat.parseDateTime("2026-11-10T22:28:53+08:00")!

        XCTAssertEqual(AppFormat.relativeTime("2026-11-10 20:28:53", now: now), "2小时前")
        XCTAssertEqual(
            AppFormat.relativeTime("2026-11-10T22:25:53+08:00", now: now),
            "3分钟前"
        )
    }

    func testRelativeTimeReturnsEmptyForUnparseableInput() {
        XCTAssertEqual(AppFormat.relativeTime("not-a-date"), "")
        XCTAssertEqual(AppFormat.relativeTime(""), "")
    }

    func testMonthKeyUsesBeijingCalendarInsteadOfUTC() {
        XCTAssertEqual(AppFormat.monthKey("2026-03-31T16:30:00Z"), "2026-04")
    }

    func testDateFormatDefaultsToChineseDate() {
        XCTAssertEqual(AppFormat.date("2026-10-09T20:30:00+08:00"), "2026年10月09日")
        XCTAssertEqual(AppFormat.monthDay("2026-10-09T20:30:00+08:00"), "10月9日")
    }
}
