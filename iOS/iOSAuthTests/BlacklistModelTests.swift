import Foundation
import XCTest
@testable import iOS

/// 黑名单模型：分页读取、移出后的本地删除与身份隔离。
@MainActor
final class BlacklistModelTests: XCTestCase {
    private func makeModel(_ server: MockHTTPServer) async -> (BlacklistModel, SessionStore) {
        let session = await makeRestoredAccountSession(server)
        return (BlacklistModel(session: session), session)
    }

    func testFirstShowLoadsBlacklist() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        server.stub("GET", "/api/friend/blacklist/list", .json(FriendFixtures.blacklistList([
            FriendFixtures.blacklistItem(userId: 66, nickname: "球友 A"),
            FriendFixtures.blacklistItem(userId: 67, nickname: "球友 B"),
        ])))

        await model.onAppear()

        XCTAssertEqual(model.items.map(\.friendUserId), [66, 67])
        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(server.requests("/api/friend/blacklist/list").first?.query["page_size"], "20")
    }

    func testRemoveSuccessDropsItem() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        server.stub("GET", "/api/friend/blacklist/list", .json(FriendFixtures.blacklistList([
            FriendFixtures.blacklistItem(userId: 66),
        ])))
        server.stub("POST", "/api/friend/unblacklist", .json(["success": true]))
        await model.onAppear()

        let outcome = await model.remove(model.items[0])

        XCTAssertEqual(outcome, .success)
        XCTAssertTrue(model.items.isEmpty)
        XCTAssertEqual(model.phase, .empty)
        XCTAssertEqual(server.requests("/api/friend/unblacklist").first?.bodyJSON["friend_user_id"] as? Int, 66)
        XCTAssertTrue(model.busyIDs.isEmpty)
    }

    func testRemoveFailureKeepsItem() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        server.stub("GET", "/api/friend/blacklist/list", .json(FriendFixtures.blacklistList([
            FriendFixtures.blacklistItem(userId: 66),
        ])))
        server.stub("POST", "/api/friend/unblacklist", .json(["success": false, "message": "操作失败"]))
        await model.onAppear()

        let outcome = await model.remove(model.items[0])

        XCTAssertEqual(outcome, .failure("操作失败"))
        XCTAssertEqual(model.items.count, 1)
    }

    func testEmptyListShowsEmptyPhase() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        server.stub("GET", "/api/friend/blacklist/list", .json(FriendFixtures.blacklistList([])))

        await model.onAppear()

        XCTAssertEqual(model.phase, .empty)
        XCTAssertFalse(model.hasMore)
    }

    func testFailureCanBeRetried() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        let attempts = LockedCounter()
        server.stub("GET", "/api/friend/blacklist/list") { _ in
            attempts.next() == 1
                ? .raw("down", status: 503)
                : .json(FriendFixtures.blacklistList([FriendFixtures.blacklistItem(userId: 66)]))
        }

        await model.onAppear()
        guard case .failed = model.phase else {
            return XCTFail("首次应失败：\(model.phase)")
        }

        await model.retry()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.items.count, 1)
    }

    func testLoadMoreAppendsNextPage() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        server.stub("GET", "/api/friend/blacklist/list") { request in
            let page = Int(request.query["page"] ?? "") ?? 1
            let items = page == 1
                ? (1...20).map { FriendFixtures.blacklistItem(userId: Int64($0)) }
                : (21...25).map { FriendFixtures.blacklistItem(userId: Int64($0)) }
            return .json(FriendFixtures.blacklistList(items, total: 25))
        }
        await model.onAppear()
        XCTAssertTrue(model.hasMore)

        await model.loadMore()

        XCTAssertEqual(model.items.count, 25)
        XCTAssertFalse(model.hasMore)
    }

    func testSwitchingAccountDropsLateResponse() async {
        let server = MockHTTPServer()
        let (model, session) = await makeModel(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/friend/blacklist/list") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(FriendFixtures.blacklistList([FriendFixtures.blacklistItem(userId: 66)]))
        }

        let load = Task { await model.onAppear() }
        await waitUntil { server.requestCount("/api/friend/blacklist/list") == 1 }
        session.logout()
        await model.identityChanged()
        gate.signal()
        await load.value

        XCTAssertTrue(model.items.isEmpty, "退出后迟到的黑名单不得写入")
        XCTAssertEqual(model.phase, .idle)
    }

    func testGuestModelNeverRequests() async {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let model = BlacklistModel(session: session)

        await model.onAppear()

        XCTAssertEqual(model.phase, .idle)
        XCTAssertEqual(server.requestCount("/api/friend/blacklist/list"), 0)
    }
}
