import Foundation
import XCTest
@testable import iOS

/// 添加好友模型：搜索、状态覆盖、发送请求与迟到响应。
@MainActor
final class AddFriendModelTests: XCTestCase {
    private func makeModel(_ server: MockHTTPServer) async -> (AddFriendModel, SessionStore) {
        let session = await makeRestoredAccountSession(server)
        return (AddFriendModel(session: session), session)
    }

    func testEmptyKeywordShowsNoticeWithoutRequest() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)

        model.keyword = "   "
        await model.search()

        XCTAssertEqual(model.notice, "请输入搜索内容")
        XCTAssertFalse(model.hasSearched)
        XCTAssertEqual(server.requestCount("/api/friend/search"), 0)
    }

    func testSearchMapsStatusesAndSendsOnlyKeyword() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        server.stub("GET", "/api/friend/search", .json(FriendFixtures.searchList([
            FriendFixtures.searchItem(userId: 1, nickname: "球友 A", isFriend: true),
            FriendFixtures.searchItem(userId: 2, nickname: "球友 B", hasPendingRequest: true),
            FriendFixtures.searchItem(userId: 3, nickname: "球友 C"),
        ])))

        model.keyword = "  阿杰  "
        await model.search()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.results.map(\.userId), [1, 2, 3])
        XCTAssertEqual(model.status(of: model.results[0]), .friend)
        XCTAssertEqual(model.status(of: model.results[1]), .pending)
        XCTAssertEqual(model.status(of: model.results[2]), .addable)
        XCTAssertEqual(server.requests("/api/friend/search").first?.query, ["keyword": "阿杰"])
    }

    func testEmptySearchResultShowsEmpty() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        server.stub("GET", "/api/friend/search", .json(FriendFixtures.searchList([])))

        model.keyword = "不存在"
        await model.search()

        XCTAssertEqual(model.phase, .empty)
        XCTAssertTrue(model.hasSearched)
    }

    func testSearchFailureCanBeRetried() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        let attempts = LockedCounter()
        server.stub("GET", "/api/friend/search") { _ in
            attempts.next() == 1
                ? .raw("down", status: 503)
                : .json(FriendFixtures.searchList([FriendFixtures.searchItem(userId: 3)]))
        }
        model.keyword = "阿杰"

        await model.search()
        guard case .failed = model.phase else {
            return XCTFail("首次应失败：\(model.phase)")
        }
        XCTAssertNotNil(model.failure)

        await model.retry()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.results.map(\.userId), [3])
    }

    func testSendRequestMarksItemPending() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        server.stub("GET", "/api/friend/search", .json(FriendFixtures.searchList([
            FriendFixtures.searchItem(userId: 3),
        ])))
        server.stub("POST", "/api/friend/request", .json(["success": true]))
        model.keyword = "阿杰"
        await model.search()

        let outcome = await model.sendRequest(to: model.results[0])

        XCTAssertEqual(outcome, .success)
        XCTAssertEqual(model.status(of: model.results[0]), .pending)
        XCTAssertEqual(server.requests("/api/friend/request").first?.bodyJSON["to_user_id"] as? Int, 3)
        XCTAssertFalse(model.isBusy(model.results[0]))
    }

    func testSendRequestFailureSurfacesServerMessage() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        server.stub("GET", "/api/friend/search", .json(FriendFixtures.searchList([
            FriendFixtures.searchItem(userId: 3),
        ])))
        server.stub("POST", "/api/friend/request", .json(["success": false, "message": "对方已是你的好友"]))
        model.keyword = "阿杰"
        await model.search()

        let outcome = await model.sendRequest(to: model.results[0])

        XCTAssertEqual(outcome, .failure("对方已是你的好友"))
        XCTAssertEqual(model.status(of: model.results[0]), .addable)
    }

    func testClearInvalidatesLateSearchResponse() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/friend/search") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(FriendFixtures.searchList([FriendFixtures.searchItem(userId: 3)]))
        }
        model.keyword = "阿杰"

        let search = Task { await model.search() }
        await waitUntil { server.requestCount("/api/friend/search") == 1 }
        model.clear()
        gate.signal()
        await search.value

        XCTAssertTrue(model.results.isEmpty, "清空后迟到的搜索结果不得写回")
        XCTAssertEqual(model.phase, .idle)
    }

    func testGuestSearchFailsWithoutRequest() async {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let model = AddFriendModel(session: session)
        model.keyword = "阿杰"

        await model.search()

        guard case .failed = model.phase else {
            return XCTFail("未登录不应发起搜索：\(model.phase)")
        }
        XCTAssertEqual(server.requestCount("/api/friend/search"), 0)
    }

    func testClearResetsSearchState() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        server.stub("GET", "/api/friend/search", .json(FriendFixtures.searchList([FriendFixtures.searchItem(userId: 3)])))
        model.keyword = "阿杰"
        await model.search()

        model.clear()

        XCTAssertEqual(model.keyword, "")
        XCTAssertFalse(model.hasSearched)
        XCTAssertTrue(model.results.isEmpty)
        XCTAssertEqual(model.phase, .idle)
    }
}
