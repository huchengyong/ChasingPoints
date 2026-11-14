import XCTest
@testable import iOS

/// 投诉举报表单：校验、登录/访客分流与提交结果。
@MainActor
final class FeedbackModelTests: XCTestCase {
    func testEmptyContentIsRejectedWithoutRequest() async {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore())
        let model = FeedbackModel(session: session)

        let outcome = await model.submit()

        XCTAssertEqual(outcome, .validation("请输入投诉举报或反馈内容"))
        XCTAssertEqual(model.validationMessage, "请输入投诉举报或反馈内容")
        XCTAssertEqual(server.requestCount("/api/feedback/create"), 0)
        XCTAssertEqual(server.requestCount("/api/user/feedback/create"), 0)
    }

    func testSignedInSubmissionUsesUserEndpointAndTrimsPayload() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/user/feedback/create", .json(["success": true, "ticket_id": 12]))
        let model = FeedbackModel(session: session)
        model.category = .complaint
        model.content = "  对处理结果有异议  "
        model.contact = "  13800000000  "

        let outcome = await model.submit()

        XCTAssertEqual(outcome, .success)
        XCTAssertEqual(model.content, "")
        XCTAssertEqual(model.contact, "")
        XCTAssertEqual(model.category, .feedback)
        XCTAssertEqual(server.requestCount("/api/user/feedback/create"), 1)
        let request = server.requests("/api/user/feedback/create").first
        XCTAssertEqual(request?.bodyJSON["source"] as? String, "app")
        XCTAssertEqual(request?.bodyJSON["category"] as? String, "complaint")
        XCTAssertEqual(request?.bodyJSON["content"] as? String, "对处理结果有异议")
        XCTAssertEqual(request?.bodyJSON["contact"] as? String, "13800000000")
        XCTAssertNotNil(request?.bearerToken)
    }

    func testGuestSubmissionUsesPublicEndpointWithoutToken() async {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore())
        server.stub("POST", "/api/feedback/create", .json(["success": true]))
        let model = FeedbackModel(session: session)
        model.content = "举报虚假球馆信息"

        let outcome = await model.submit()

        XCTAssertEqual(outcome, .success)
        XCTAssertEqual(server.requestCount("/api/feedback/create"), 1)
        XCTAssertNil(server.requests("/api/feedback/create").first?.bearerToken)
    }

    func testBusinessFailureReturnsServerMessage() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/user/feedback/create", .json(["success": false, "message": "内容涉及违规"]))
        let model = FeedbackModel(session: session)
        model.content = "测试内容"

        let outcome = await model.submit()

        XCTAssertEqual(outcome, .failure("内容涉及违规"))
        XCTAssertFalse(model.isSubmitting)
    }

    func testFailureWithoutServerMessageUsesClientMessage() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/user/feedback/create", .json(["success": false]))
        let model = FeedbackModel(session: session)
        model.content = "测试内容"

        let outcome = await model.submit()

        XCTAssertEqual(outcome, .failure("请求失败，请稍后重试"))
    }

    func testContentIsTruncatedToLimit() {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore())
        let model = FeedbackModel(session: session)

        model.content = String(repeating: "球", count: FeedbackRules.maxContentLength + 20)

        XCTAssertEqual(model.content.count, FeedbackRules.maxContentLength)
    }
}
