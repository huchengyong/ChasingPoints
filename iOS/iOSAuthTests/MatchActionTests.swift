import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/match-action.test.mjs`。
@MainActor
final class MatchActionTests: XCTestCase {
    func testBuildMatchActionPayloadAppendsClientActionIdAndBaseRevisionToWritePayloads() {
        let payload = MatchAction.buildPayload(
            [
                "match_id": .int(18),
                "actor": .int(2),
                "score": .int(7),
            ],
            baseRevision: 12,
            createActionId: { "action-123" }
        )

        XCTAssertEqual(payload, [
            "match_id": .int(18),
            "actor": .int(2),
            "score": .int(7),
            "client_action_id": .string("action-123"),
            "base_revision": .int(12),
        ])
    }

    func testBuildMatchActionPayloadFallsBackToRevision0WhenClientHasNotSyncedYet() {
        let payload = MatchAction.buildPayload(
            ["match_id": .int(19)],
            baseRevision: nil,
            createActionId: { "action-456" }
        )

        XCTAssertEqual(payload["client_action_id"], .string("action-456"))
        XCTAssertEqual(payload["base_revision"], .int(0))
    }

    /// JS 测试没有覆盖 `createMatchActionId`，但它的格式与唯一性影响服务端去重。
    func testCreateActionIdKeepsTheExpectedShapeAndStaysUnique() {
        let first = MatchAction.createActionId(now: Date(timeIntervalSince1970: 1_800_000_000))
        let second = MatchAction.createActionId(now: Date(timeIntervalSince1970: 1_800_000_000))
        XCTAssertTrue(first.hasPrefix("ma_"))
        XCTAssertEqual(first.split(separator: "_").count, 4)
        XCTAssertNotEqual(first, second)
    }
}
