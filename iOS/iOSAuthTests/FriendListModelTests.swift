import Foundation
import XCTest
@testable import iOS

/// 好友列表模型：分页、删除/拉黑后的本地覆盖与角标同步。
@MainActor
final class FriendListModelTests: XCTestCase {
    private func makeModel(_ server: MockHTTPServer, pendingFriend: Int = 0) async -> (FriendListModel, ActivityStore, SessionStore) {
        server.stub("GET", "/api/user/bootstrap", .json(FriendFixtures.bootstrap(pendingFriend: pendingFriend)))
        let session = await makeRestoredAccountSession(server)
        let activity = ActivityStore(session: session)
        return (FriendListModel(session: session, activity: activity), activity, session)
    }

    func testFirstShowLoadsOnceAndReadsPendingBadge() async {
        let server = MockHTTPServer()
        let (model, activity, _) = await makeModel(server, pendingFriend: 3)
        server.stub("GET", "/api/friend/list", .json(FriendFixtures.friendList([
            FriendFixtures.friend(id: 1, userId: 18, nickname: "球友 A"),
            FriendFixtures.friend(id: 2, userId: 19, nickname: "球友 B"),
        ])))

        await model.onAppear()
        await model.onAppear()

        XCTAssertEqual(model.items.map(\.friendUserId), [18, 19])
        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.pendingCount, 3)
        XCTAssertEqual(activity.pendingFriendRequestCount, 3)
        XCTAssertEqual(server.requestCount("/api/friend/list"), 1)
        XCTAssertEqual(server.requests("/api/friend/list").first?.query["page_size"], "20")
    }

    func testDeleteSuccessRemovesFriendAndMarksDirty() async {
        let server = MockHTTPServer()
        let (model, activity, _) = await makeModel(server)
        server.stub("GET", "/api/friend/list", .json(FriendFixtures.friendList([
            FriendFixtures.friend(id: 1, userId: 18),
            FriendFixtures.friend(id: 2, userId: 19),
        ])))
        server.stub("POST", "/api/friend/delete", .json(["success": true]))
        await model.onAppear()
        let dirtyBefore = activity.dirtySeq

        let removed = await model.delete(model.items[0])

        XCTAssertTrue(removed)
        XCTAssertEqual(model.items.map(\.friendUserId), [19])
        XCTAssertEqual(server.requests("/api/friend/delete").first?.bodyJSON["friend_user_id"] as? Int, 18)
        XCTAssertGreaterThan(activity.dirtySeq, dirtyBefore)
    }

    func testDeleteFailureKeepsFriend() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/friend/list", .json(FriendFixtures.friendList([
            FriendFixtures.friend(id: 1, userId: 18),
        ])))
        server.stub("POST", "/api/friend/delete", .json(["success": false, "message": "删除失败"]))
        await model.onAppear()

        let removed = await model.delete(model.items[0])

        XCTAssertFalse(removed)
        XCTAssertEqual(model.items.count, 1)
    }

    func testBlacklistSuccessRemovesFriend() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/friend/list", .json(FriendFixtures.friendList([
            FriendFixtures.friend(id: 1, userId: 18),
        ])))
        server.stub("POST", "/api/friend/blacklist", .json(["success": true]))
        await model.onAppear()

        let removed = await model.addToBlacklist(model.items[0])

        XCTAssertTrue(removed)
        XCTAssertTrue(model.items.isEmpty)
        XCTAssertEqual(model.phase, .empty)
    }

    func testRemovedFriendsNoLongerCountTowardsHasMore() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/friend/list", .json(FriendFixtures.friendList([
            FriendFixtures.friend(id: 1, userId: 18),
            FriendFixtures.friend(id: 2, userId: 19),
            FriendFixtures.friend(id: 3, userId: 20),
        ], total: 3)))
        server.stub("POST", "/api/friend/delete", .json(["success": true]))
        await model.onAppear()
        XCTAssertFalse(model.hasMore)

        _ = await model.delete(model.items[0])

        XCTAssertEqual(model.items.count, 2)
        XCTAssertFalse(model.hasMore, "删除后 total 同步递减，不应残留「加载更多」")
    }

    func testEmptyListShowsEmptyPhase() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/friend/list", .json(FriendFixtures.friendList([])))

        await model.onAppear()

        XCTAssertEqual(model.phase, .empty)
        XCTAssertNil(model.failure)
    }

    func testFailureCanBeRetried() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        let attempts = LockedCounter()
        server.stub("GET", "/api/friend/list") { _ in
            attempts.next() == 1
                ? .raw("down", status: 503)
                : .json(FriendFixtures.friendList([FriendFixtures.friend(id: 1, userId: 18)]))
        }

        await model.onAppear()
        guard case .failed = model.phase else {
            return XCTFail("首次应失败：\(model.phase)")
        }

        await model.retry()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.items.map(\.friendUserId), [18])
    }

    func testLoadMoreAppendsNextPage() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/friend/list") { request in
            let page = Int(request.query["page"] ?? "") ?? 1
            let ids = page == 1 ? Array(1...20) : Array(21...25)
            var items: [[String: Any]] = []
            for id in ids {
                items.append(FriendFixtures.friend(id: Int64(id), userId: Int64(id + 100)))
            }
            return .json(FriendFixtures.friendList(items, total: 25))
        }
        await model.onAppear()
        XCTAssertEqual(model.items.count, 20)
        XCTAssertTrue(model.hasMore)

        await model.loadMore()

        XCTAssertEqual(model.items.count, 25)
        XCTAssertFalse(model.hasMore)
        XCTAssertEqual(server.requests("/api/friend/list").map { $0.query["page"] }, ["1", "2"])
    }

    func testPullDownRefreshReloadsListAndForcesBadge() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/friend/list", .json(FriendFixtures.friendList([
            FriendFixtures.friend(id: 1, userId: 18),
        ])))
        await model.onAppear()
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 1)

        await model.refresh()

        XCTAssertEqual(server.requestCount("/api/friend/list"), 2)
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 2)
    }

    func testGuestModelNeverRequests() async {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let activity = ActivityStore(session: session)
        let model = FriendListModel(session: session, activity: activity)

        await model.onAppear()

        XCTAssertEqual(model.phase, .idle)
        XCTAssertTrue(model.items.isEmpty)
        XCTAssertEqual(server.requestCount("/api/friend/list"), 0)
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 0)
    }

    func testSwitchingAccountDropsLateResponse() async {
        let server = MockHTTPServer()
        let (model, _, session) = await makeModel(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/friend/list") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(FriendFixtures.friendList([FriendFixtures.friend(id: 1, userId: 18, nickname: "旧账号好友")]))
        }

        let load = Task { await model.onAppear() }
        await waitUntil { server.requestCount("/api/friend/list") == 1 }
        session.logout()
        await model.identityChanged()
        gate.signal()
        await load.value

        XCTAssertTrue(model.items.isEmpty, "退出后迟到的列表不得写入")
        XCTAssertEqual(model.phase, .idle)
    }
}
