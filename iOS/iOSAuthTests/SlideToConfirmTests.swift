import XCTest
@testable import iOS

/// 对应 `uniapp/tests/slide-to-confirm.test.mjs` 的进度与松手判定；键盘用例在原生端没有对应交互。
final class SlideToConfirmTests: XCTestCase {
    func testProgressClampsToUnitRange() {
        XCTAssertEqual(SlideToConfirm.progress(deltaX: -5, maxTrack: 200), 0)
        XCTAssertEqual(SlideToConfirm.progress(deltaX: 0, maxTrack: 200), 0)
        XCTAssertEqual(SlideToConfirm.progress(deltaX: 100, maxTrack: 200), 0.5)
        XCTAssertEqual(SlideToConfirm.progress(deltaX: 300, maxTrack: 200), 1)
        XCTAssertEqual(SlideToConfirm.progress(deltaX: 100, maxTrack: 0), 0)
    }

    func testOnlyReleaseAtTrackEndCommits() {
        XCTAssertFalse(SlideToConfirm.isComplete(deltaX: 100, maxTrack: 200))
        XCTAssertTrue(SlideToConfirm.isComplete(deltaX: 200, maxTrack: 200))
        XCTAssertTrue(SlideToConfirm.shouldCommitOnRelease(deltaX: 198, maxTrack: 200))
        XCTAssertFalse(SlideToConfirm.shouldCommitOnRelease(deltaX: 195, maxTrack: 200))
        XCTAssertLessThanOrEqual(SlideToConfirm.completeRatio, 1)
    }
}
