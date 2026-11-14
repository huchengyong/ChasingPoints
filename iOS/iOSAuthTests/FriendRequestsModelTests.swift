import Foundation
import XCTest
@testable import iOS

/// 好友申请模型：接受/拒绝/拉黑的本地覆盖、角标递减与身份隔离。
@MainActor
final class FriendRequestsModelTests: XCTestCase {
    private func makeModel(_ server: MockHTTPServer) async -> (FriendRequestsModel, ActivityStore, SessionStore) {
        server.stub("GET", "/api/user/bootstrap", .json(FriendFixtures.bootstrap(pendingFriend: 0)))
        let session = await makeRestoredAccountSession(server)
        let activity = ActivityStore(session: session)
        return (FriendRequestsModel(session: session, activity: activity), activity, session)
    }

    func testFirstShowLoadsRequestsAndSyncsPendingCount() async {
        let server = MockHTTPServer()
        let (model, activity, _) = await makeModel(server)
        server.stub("GET", "/api/friend/requests", .json(FriendFixtures.requestList([
            FriendFixtures.request(id: 9, fromUserId: 52, nickname: "球友 A"),
            FriendFixtures.request(id: 10, fromUserId: 53, nickname: "球友 B", message: "一起打球"),
        ])))

        await model.onAppear()

        XCTAssertEqual(model.items.map(\.id), [9, 10])
        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(activity.pendingFriendRequestCount, 2)
        XCTAssertEqual(model.status(of: model.items[0]), 0)
        XCTAssertEqual(server.requests("/api/friend/requests").first?.query["page"], "1")
    }

    func testAcceptOverridesStatusAndDecrementsPending() async {
        let server = MockHTTPServer()
        let (model, activity, _) = await makeModel(server)
        server.stub("GET", "/api/friend/requests", .json(FriendFixtures.requestList([
            FriendFixtures.request(id: 9, fromUserId: 52),
            FriendFixtures.request(id: 10, fromUserId: 53),
        ])))
        server.stub("POST", "/api/friend/accept", .json(["success": true]))
        await model.onAppear()
        let dirtyBefore = activity.dirtySeq

        let outcome = await model.accept(model.items[0])

        XCTAssertEqual(outcome, .success)
        XCTAssertEqual(model.status(of: model.items[0]), 1)
        XCTAssertEqual(activity.pendingFriendRequestCount, 1)
        XCTAssertGreaterThan(activity.dirtySeq, dirtyBefore)
        XCTAssertEqual(server.requests("/api/friend/accept").first?.bodyJSON["request_id"] as? Int, 9)
    }

    func testRejectOverridesStatusAndDecrementsPending() async {
        let server = MockHTTPServer()
        let (model, activity, _) = await makeModel(server)
        server.stub("GET", "/api/friend/requests", .json(FriendFixtures.requestList([
            FriendFixtures.request(id: 9, fromUserId: 52),
        ])))
        server.stub("POST", "/api/friend/reject", .json(["success": true]))
        await model.onAppear()

        let outcome = await model.reject(model.items[0])

        XCTAssertEqual(outcome, .success)
        XCTAssertEqual(model.status(of: model.items[0]), 2)
        XCTAssertEqual(activity.pendingFriendRequestCount, 0)
        XCTAssertEqual(server.requests("/api/friend/reject").first?.bodyJSON["request_id"] as? Int, 9)
    }

    func testRejectAndBlacklistRemovesRequest() async {
        let server = MockHTTPServer()
        let (model, activity, _) = await makeModel(server)
        server.stub("GET", "/api/friend/requests", .json(FriendFixtures.requestList([
            FriendFixtures.request(id: 9, fromUserId: 52),
            FriendFixtures.request(id: 10, fromUserId: 53),
        ])))
        server.stub("POST", "/api/friend/blacklist", .json(["success": true]))
        await model.onAppear()

        let outcome = await model.rejectAndBlacklist(model.items[0])

        XCTAssertEqual(outcome, .success)
        XCTAssertEqual(model.items.map(\.id), [10])
        XCTAssertEqual(activity.pendingFriendRequestCount, 1)
        XCTAssertEqual(server.requests("/api/friend/blacklist").first?.bodyJSON["friend_user_id"] as? Int, 52)
    }

    func testFailedActionKeepsStatusAndCount() async {
        let server = MockHTTPServer()
        let (model, activity, _) = await makeModel(server)
        server.stub("GET", "/api/friend/requests", .json(FriendFixtures.requestList([
            FriendFixtures.request(id: 9, fromUserId: 52),
        ])))
        server.stub("POST", "/api/friend/accept", .json(["success": false, "message": "操作失败"]))
        await model.onAppear()

        let outcome = await model.accept(model.items[0])

        XCTAssertEqual(outcome, .failure("操作失败"))
        XCTAssertEqual(model.status(of: model.items[0]), 0)
        XCTAssertEqual(activity.pendingFriendRequestCount, 1)
    }

    func testSecondActionWhileFirstIsInFlightIsRejected() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/friend/requests", .json(FriendFixtures.requestList([
            FriendFixtures.request(id: 9, fromUserId: 52),
        ])))
        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/friend/accept") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(["success": true])
        }
        await model.onAppear()
        let request = model.items[0]

        let first = Task { await model.accept(request) }
        await waitUntil { model.isBusy(request) }
        let second = await model.accept(request)
        gate.signal()
        let firstOutcome = await first.value

        XCTAssertEqual(firstOutcome, .success)
        XCTAssertEqual(second, .failure("操作失败"))
        XCTAssertEqual(server.requestCount("/api/friend/accept"), 1, "同一张卡片同时只允许一个操作")
        XCTAssertFalse(model.isBusy(request))
    }

    func testEmptyListShowsEmptyPhase() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/friend/requests", .json(FriendFixtures.requestList([])))

        await model.onAppear()

        XCTAssertEqual(model.phase, .empty)
        XCTAssertFalse(model.hasMore)
    }

    func testFailureCanBeRetried() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        let attempts = LockedCounter()
        server.stub("GET", "/api/friend/requests") { _ in
            attempts.next() == 1
                ? .raw("down", status: 503)
                : .json(FriendFixtures.requestList([FriendFixtures.request(id: 9, fromUserId: 52)]))
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
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/friend/requests") { request in
            let page = Int(request.query["page"] ?? "") ?? 1
            let ids = page == 1 ? Array(1...20) : Array(21...25)
            var items: [[String: Any]] = []
            for id in ids {
                items.append(FriendFixtures.request(id: Int64(id), fromUserId: Int64(id + 100)))
            }
            return .json(FriendFixtures.requestList(items, total: 25))
        }
        await model.onAppear()
        XCTAssertTrue(model.hasMore)

        await model.loadMore()

        XCTAssertEqual(model.items.count, 25)
        XCTAssertFalse(model.hasMore)
    }

    func testSwitchingAccountClearsOverlayAndDropsLateResponse() async {
        let server = MockHTTPServer()
        let (model, _, session) = await makeModel(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/friend/requests") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(FriendFixtures.requestList([FriendFixtures.request(id: 9, fromUserId: 52)]))
        }

        let load = Task { await model.onAppear() }
        await waitUntil { server.requestCount("/api/friend/requests") == 1 }
        session.logout()
        await model.identityChanged()
        gate.signal()
        await load.value

        XCTAssertTrue(model.items.isEmpty, "退出后迟到的申请不得写入")
        XCTAssertTrue(model.statusOverrides.isEmpty)
        XCTAssertTrue(model.removedIDs.isEmpty)
    }

    func testGuestModelNeverRequests() async {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let activity = ActivityStore(session: session)
        let model = FriendRequestsModel(session: session, activity: activity)

        await model.onAppear()

        XCTAssertEqual(model.phase, .idle)
        XCTAssertEqual(server.requestCount("/api/friend/requests"), 0)
    }
}
