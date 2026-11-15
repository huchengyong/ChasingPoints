import XCTest
@testable import iOS

/// 迁移自 `uniapp/tests/feedback-ticket.test.mjs`。
final class FeedbackRulesTests: XCTestCase {
    func testCategoryNormalizationFallsBackToFeedback() {
        XCTAssertEqual(FeedbackCategory.normalized("complaint"), .complaint)
        XCTAssertEqual(FeedbackCategory.normalized("report"), .report)
        XCTAssertEqual(FeedbackCategory.normalized("unknown"), .feedback)
        XCTAssertEqual(FeedbackCategory.normalized(""), .feedback)
    }

    func testPayloadTrimsContentAndContactForAppSubmissions() {
        let payload = FeedbackRules.payload(
            category: .complaint,
            content: "  对处理结果有异议  ",
            contact: "  13800000000  "
        )

        XCTAssertEqual(payload, FeedbackTicketPayload(
            source: "app",
            category: "complaint",
            content: "对处理结果有异议",
            contact: "13800000000"
        ))
    }

    func testCategoryOptionsCoverComplaintsAndReports() {
        XCTAssertEqual(FeedbackCategory.allCases.map(\.rawValue), ["feedback", "complaint", "report"])
        XCTAssertTrue(FeedbackCategory.allCases.contains { $0.label.contains("投诉") })
        XCTAssertTrue(FeedbackCategory.allCases.contains { $0.label.contains("举报") })
    }
}
