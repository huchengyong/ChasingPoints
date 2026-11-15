import Foundation
import XCTest
@testable import iOS

/// 约球列表模型：分栏过滤、历史游标分页、写操作与身份隔离。
@MainActor
final class ChallengesModelTests: XCTestCase {
    private func bootstrap() -> [String: Any] {
        [
            "success": true,
            "current_match": NSNull(),
            "current_challenge": NSNull(),
            "received_pending_challenge_count": 0,
            "server_time": "2026-08-25 12:00:00",
            "unread_count": 0,
            "pending_friend_request_count": 0,
            "latest_season_rollover": NSNull(),
            "availability": ActivityFixtures.allAvailable,
        ]
    }

    private func makeModel(
        _ server: MockHTTPServer,
        tab: ChallengeTab = .received
    ) async -> (ChallengesModel, ActivityStore, SessionStore) {
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 12:00:00")))
        let session = await makeRestoredAccountSession(server)
        let activity = ActivityStore(session: session)
        return (ChallengesModel(session: session, activity: activity, initialTab: tab), activity, session)
    }

    private func received(id: Int64, fromUserId: Int64 = 18) -> [String: Any] {
        ChallengeFixtures.challenge(id: id, fromUserId: fromUserId, toUserId: 1)
    }

    private func sent(id: Int64, status: Int = 0) -> [String: Any] {
        ChallengeFixtures.challenge(id: id, status: status, fromUserId: 1, toUserId: 29)
    }

    func testFirstAppearLoadsPendingAndReadsServerTime() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([
            received(id: 1),
            sent(id: 2),
        ])))

        await model.onAppear()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.items.map(\.id), [1])
        XCTAssertEqual(model.receivedCount, 1)
        XCTAssertEqual(model.serverTime, "2026-08-25 12:00:00")
        XCTAssertEqual(server.requestCount("/api/challenge/pending"), 1)
        XCTAssertEqual(server.requestCount("/api/challenge/history"), 0, "收到分栏不应读取历史")
    }

    func testTabsFilterActiveItemsAndLoadHistoryOnDemand() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([
            received(id: 1),
            sent(id: 2),
            sent(id: 3, status: 1),
            ChallengeFixtures.challenge(id: 4, status: 6, matchId: 88, fromUserId: 1, toUserId: 29),
            ChallengeFixtures.challenge(id: 5, status: 1, fromUserId: 18, toUserId: 1),
            sent(id: 6, status: 2),
        ])))
        server.stub("GET", "/api/challenge/history", .json(ChallengeFixtures.historyList([
            ChallengeFixtures.challenge(id: 7, status: 2, fromUserId: 1, toUserId: 29),
        ])))

        await model.onAppear()
        XCTAssertEqual(model.items.map(\.id), [1])

        await model.switchTab(.sent)
        XCTAssertEqual(model.items.map(\.id), [2, 3, 4, 5])

        await model.switchTab(.history)
        XCTAssertEqual(model.items.map(\.id), [7])
        XCTAssertEqual(server.requestCount("/api/challenge/history"), 1)

        await model.switchTab(.received)
        await model.switchTab(.history)
        XCTAssertEqual(server.requestCount("/api/challenge/history"), 1, "历史已读过且未标脏时不重复读取")
    }

    func testHistoryFirstLoadFailureShowsFailureInsteadOfEmpty() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server, tab: .history)
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([])))
        server.stub("GET", "/api/challenge/history", .raw("down", status: 503))

        await model.onAppear()

        guard case .failed = model.phase else {
            return XCTFail("历史首读失败应进入失败态：\(model.phase)")
        }
        XCTAssertTrue(model.items.isEmpty)
    }

    func testHistoryRefreshFailureKeepsListAndCursor() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server, tab: .history)
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([])))
        let attempts = LockedCounter()
        server.stub("GET", "/api/challenge/history") { request in
            attempts.next() == 1
                ? .json(ChallengeFixtures.historyList([self.sent(id: 9, status: 2)], nextBeforeId: 9))
                : .raw("down", status: 503)
        }

        await model.onAppear()
        XCTAssertEqual(model.items.count, 1)
        XCTAssertTrue(model.hasMoreHistory)

        await model.retry()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.items.count, 1)
        XCTAssertTrue(model.hasMoreHistory, "刷新失败不能丢弃已有游标")
        XCTAssertNotNil(model.refreshError)
    }

    func testLoadMoreHistoryAppendsNextPageWithBeforeId() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server, tab: .history)
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([])))
        server.stub("GET", "/api/challenge/history") { request in
            if request.query["before_id"] == "11" {
                return .json(ChallengeFixtures.historyList([self.sent(id: 5, status: 2)]))
            }
            return .json(ChallengeFixtures.historyList(
                [self.sent(id: 12, status: 2), self.sent(id: 11, status: 2)],
                nextBeforeId: 11
            ))
        }

        await model.onAppear()
        XCTAssertTrue(model.hasMoreHistory)

        await model.loadMoreHistory()

        XCTAssertEqual(model.items.map(\.id), [12, 11, 5])
        XCTAssertFalse(model.hasMoreHistory)
        XCTAssertEqual(server.requests("/api/challenge/history").map { $0.query["before_id"] }, [nil, "11"])
    }

    func testLoadMoreHistoryFailureKeepsCursorAndShowsError() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server, tab: .history)
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([])))
        let attempts = LockedCounter()
        server.stub("GET", "/api/challenge/history") { _ in
            attempts.next() == 1
                ? .json(ChallengeFixtures.historyList(
                    [self.sent(id: 12, status: 2), self.sent(id: 11, status: 2)],
                    nextBeforeId: 11
                ))
                : .raw("down", status: 503)
        }
        await model.onAppear()

        await model.loadMoreHistory()

        XCTAssertEqual(model.items.count, 2)
        XCTAssertTrue(model.hasMoreHistory)
        XCTAssertNotNil(model.historyLoadMoreError)
        XCTAssertFalse(model.isLoadingMoreHistory)
    }

    func testAcceptNavigatesToWaitingAndMarksActivityDirty() async {
        let server = MockHTTPServer()
        let (model, activity, _) = await makeModel(server)
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([received(id: 7)])))
        server.stub("POST", "/api/challenge/accept", .json(ChallengeFixtures.acceptResponse(challengeId: 7)))
        await model.onAppear()
        let dirtyBefore = activity.dirtySeq
        let item = try! XCTUnwrap(model.items.first)

        await model.accept(item)

        XCTAssertEqual(model.navigation, .waiting(7))
        XCTAssertEqual(model.feedback, "已接受，双方进入后正式开局")
        XCTAssertGreaterThan(activity.dirtySeq, dirtyBefore)
        XCTAssertNil(model.prompt)
    }

    func testAcceptNeedConfirmCloseOwnPromptResendsWithFlag() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([received(id: 7)])))
        let attempts = LockedCounter()
        server.stub("POST", "/api/challenge/accept") { _ in
            attempts.next() == 1
                ? .json(ChallengeFixtures.businessFailure("接受后会自动关闭已发出的邀请", needConfirmCloseOwn: true))
                : .json(ChallengeFixtures.acceptResponse(challengeId: 7))
        }
        await model.onAppear()
        let item = try! XCTUnwrap(model.items.first)

        await model.accept(item)
        XCTAssertEqual(model.prompt, .closeOwn(item))

        await model.accept(item, confirmCloseOwn: true)

        XCTAssertEqual(server.requestCount("/api/challenge/accept"), 2)
        XCTAssertEqual(server.requests("/api/challenge/accept").last?.bodyJSON["confirm_close_own_pending"] as? Bool, true)
        XCTAssertEqual(model.navigation, .waiting(7))
    }

    func testAcceptTypeConflictOpensPrompt() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([received(id: 7)])))
        server.stub("POST", "/api/challenge/accept", .json(ChallengeFixtures.businessFailure(
            "你们发起的约球类型不同，不能直接匹配",
            typeConflict: true
        )))
        await model.onAppear()
        let item = try! XCTUnwrap(model.items.first)

        await model.accept(item)

        XCTAssertEqual(model.prompt, .typeConflict(message: "你们发起的约球类型不同，不能直接匹配"))
        XCTAssertNil(model.navigation)
    }

    func testRejectAndCancelFeedBackAndMarkDirty() async {
        let server = MockHTTPServer()
        let (model, activity, _) = await makeModel(server)
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([received(id: 7), sent(id: 8)])))
        server.stub("POST", "/api/challenge/reject", .json(ChallengeFixtures.actionResponse()))
        server.stub("POST", "/api/challenge/cancel", .json(ChallengeFixtures.actionResponse()))
        await model.onAppear()
        let dirtyBefore = activity.dirtySeq
        let incoming = try! XCTUnwrap(model.items.first)

        await model.reject(incoming)
        XCTAssertEqual(model.feedback, "已拒绝")

        await model.cancel(sentInfo(id: 8))
        XCTAssertEqual(model.feedback, "约球已取消")
        XCTAssertGreaterThan(activity.dirtySeq, dirtyBefore)
    }

    func testRematchBuildsComposeTargetFromOpponentAndMode() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        let item = ChallengeInfo.decode(ChallengeFixtures.challenge(
            id: 9,
            status: 4,
            fromUserId: 1,
            toUserId: 29,
            toNickname: "球友B",
            toAvatar: "avatar-b",
            gameType: 2,
            matchMode: "practice"
        ))

        model.rematch(item)

        XCTAssertEqual(model.navigation, .compose(ChallengeComposeTarget(
            userId: 29,
            nickname: "球友B",
            avatar: "avatar-b",
            gameType: 2,
            matchMode: "practice"
        )))
    }

    func testSignOutClearsListsAndStopsRequests() async {
        let server = MockHTTPServer()
        let (model, _, session) = await makeModel(server)
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([received(id: 7)])))
        await model.onAppear()
        XCTAssertEqual(model.items.count, 1)
        let requestsBefore = server.requestCount("/api/challenge/pending")

        session.logout()
        await model.onAppear()

        XCTAssertTrue(model.items.isEmpty)
        XCTAssertEqual(model.receivedCount, 0)
        XCTAssertEqual(model.phase, .empty)
        XCTAssertEqual(server.requestCount("/api/challenge/pending"), requestsBefore)
    }

    func testWriteDuringHistoryLoadWinsOverStaleResponse() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server, tab: .history)
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([received(id: 1)])))
        server.stub("POST", "/api/challenge/accept", .json(ChallengeFixtures.acceptResponse(challengeId: 1)))
        let gate = DispatchSemaphore(value: 0)
        let attempts = LockedCounter()
        server.stub("GET", "/api/challenge/history") { _ in
            if attempts.next() == 1 {
                _ = gate.wait(timeout: .now() + 5)
                return .json(ChallengeFixtures.historyList([self.sent(id: 9, status: 2)]))
            }
            return .json(ChallengeFixtures.historyList([self.sent(id: 10, status: 2)]))
        }

        let switching = Task { await model.switchTab(.history) }
        await waitUntil { server.requestCount("/api/challenge/history") == 1 }
        await model.accept(try! XCTUnwrap(receivedInfo(id: 1)))
        gate.signal()
        await switching.value

        XCTAssertEqual(model.items.map(\.id), [10], "读取期间发生的写入后，旧响应不得覆盖新数据")
    }

    // MARK: - 请求取消

    func testCanceledHistoryLoadDoesNotShowFailureAndCanReload() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server, tab: .history)
        server.stub("GET", "/api/challenge/history", .failure(.cancelled))

        await model.switchTab(.history)

        XCTAssertNil(model.historyFailure, "取消不是失败")
        XCTAssertNil(model.historyRefreshError, "取消不写刷新提示")
        XCTAssertFalse(model.historyLoading, "取消后必须复位加载标记")
        XCTAssertEqual(model.phase, .empty)

        server.stub("GET", "/api/challenge/history", .json(ChallengeFixtures.historyList([
            sent(id: 11, status: 2),
        ])))
        await model.switchTab(.history)

        XCTAssertNil(model.historyFailure)
        XCTAssertEqual(model.items.map(\.id), [11], "取消后重新读取应正常加载")
        XCTAssertEqual(model.phase, .loaded)
    }

    func testCanceledActiveLoadDoesNotShowFailureAndCanReload() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)
        server.stub("GET", "/api/challenge/pending", .failure(.cancelled))

        await model.onAppear()

        XCTAssertNil(model.activeFailure, "取消不是失败")
        XCTAssertNil(model.activeRefreshError, "取消不写刷新提示")
        XCTAssertFalse(model.activeLoading, "取消后必须复位加载标记")
        XCTAssertEqual(model.phase, .empty)

        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([
            received(id: 1),
        ])))
        await model.retry()

        XCTAssertNil(model.activeFailure)
        XCTAssertEqual(model.items.map(\.id), [1], "取消后重新读取应正常加载")
        XCTAssertEqual(model.phase, .loaded)
    }

    // MARK: - 辅助

    private func receivedInfo(id: Int64) -> ChallengeInfo? {
        ChallengeInfo.decode(received(id: id))
    }

    private func sentInfo(id: Int64) -> ChallengeInfo {
        ChallengeInfo.decode(sent(id: id))
    }
}

private extension ChallengeInfo {
    /// 测试里直接复用后端字段名解码，避免手写第二套模型。
    static func decode(_ value: [String: Any]) -> ChallengeInfo {
        let data = try! JSONSerialization.data(withJSONObject: value)
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        return try! decoder.decode(ChallengeInfo.self, from: data)
    }
}
