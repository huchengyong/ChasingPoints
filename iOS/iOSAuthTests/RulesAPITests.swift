import Foundation
import XCTest
@testable import iOS

@MainActor
final class RulesAPITests: XCTestCase {
    func testContentSendsCategoryAndContentTypeAsGuest() async throws {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", "/api/rules/content", .json([
            "success": true,
            "list": [itemJSON(id: 1, title: "开球", content: "正文", sortOrder: 1)],
        ]))

        let items = try await RulesAPI(session: session).content(category: "nine_ball", type: .foul)

        XCTAssertEqual(items, [RuleContentItem(id: 1, title: "开球", content: "正文", sortOrder: 1)])
        let request = server.requests("/api/rules/content").first
        XCTAssertEqual(request?.query["category"], "nine_ball")
        XCTAssertEqual(request?.query["content_type"], "foul")
        XCTAssertNil(request?.bearerToken, "访客请求不带 Authorization")
    }

    func testGlossaryReadsListAsGuest() async throws {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", "/api/rules/glossary", .json([
            "success": true,
            "list": [
                ["id": 2, "title": "定杆", "content": "Stun", "category": "snooker"],
            ],
        ]))

        let items = try await RulesAPI(session: session).glossary()

        XCTAssertEqual(items, [GlossaryItem(id: 2, title: "定杆", content: "Stun", category: "snooker")])
        let request = server.requests("/api/rules/glossary").first
        XCTAssertEqual(request?.path, "/api/rules/glossary")
        XCTAssertNil(request?.bearerToken, "访客请求不带 Authorization")
    }

    func testSearchEncodesChineseKeywordAsGuest() async throws {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", "/api/rules/search", .json([
            "success": true,
            "list": [itemJSON(id: 3, title: "九球 规则", content: "正文", sortOrder: 1)],
        ]))

        let items = try await RulesAPI(session: session).search(keyword: "九球 规则")

        XCTAssertEqual(items.map(\.id), [3])
        let request = server.requests("/api/rules/search").first
        XCTAssertEqual(request?.query["keyword"], "九球 规则")
        XCTAssertNil(request?.bearerToken, "访客请求不带 Authorization")
    }

    func testBusinessFailureThrows() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", "/api/rules/search", .json(["success": false, "message": "搜索规则失败"]))

        do {
            _ = try await RulesAPI(session: session).search(keyword: "开球")
            XCTFail("success=false 应当抛错")
        } catch {
            assertAPIError(error, kind: .business, message: "搜索规则失败")
        }
    }

    func testBlankKeywordSkipsRequest() async throws {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)

        let items = try await RulesAPI(session: session).search(keyword: "   ")

        XCTAssertTrue(items.isEmpty)
        XCTAssertEqual(server.requestCount("/api/rules/search"), 0)
    }

    private func makeGuestSession(_ server: MockHTTPServer) -> SessionStore {
        makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
    }
}

private func itemJSON(id: Int64, title: String, content: String, sortOrder: Int) -> [String: Any] {
    ["id": id, "title": title, "content": content, "sort_order": sortOrder]
}
