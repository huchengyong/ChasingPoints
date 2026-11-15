import Foundation
import XCTest
@testable import iOS

/// 消息中心的纯规则、通知跳转与列表模型测试。全部使用 MockHTTPServer，不访问真实后端。
@MainActor
final class MessageCenterTests: XCTestCase {
    // MARK: - 夹具

    private func bootstrap(unread: Int = 0, pendingFriend: Int = 0, received: Int = 0) -> [String: Any] {
        [
            "success": true,
            "current_match": NSNull(),
            "current_challenge": NSNull(),
            "received_pending_challenge_count": received,
            "server_time": "2026-01-01 12:00:00",
            "unread_count": unread,
            "pending_friend_request_count": pendingFriend,
            "latest_season_rollover": NSNull(),
            "availability": ActivityFixtures.allAvailable,
        ]
    }

    private func note(
        _ id: Int64,
        type: String = "system",
        title: String = "通知",
        data: String = "",
        isRead: Bool = false
    ) -> [String: Any] {
        [
            "id": id,
            "type": type,
            "title": title,
            "content": "内容\(id)",
            "data": data,
            "is_read": isRead,
            "created_at": "2026-01-01 12:00:00",
        ]
    }

    private func listJSON(_ list: [[String: Any]], total: Int? = nil, unread: Int? = nil) -> [String: Any] {
        [
            "success": true,
            "total": total ?? list.count,
            "unread_count": unread ?? list.filter { ($0["is_read"] as? Bool) != true }.count,
            "list": list,
        ]
    }

    private func makeModel(_ server: MockHTTPServer, user: [String: Any] = AuthFixtures.userInfo()) async -> (MessageCenterModel, ActivityStore, SessionStore) {
        let session = await makeRestoredAccountSession(server, user: user)
        let activity = ActivityStore(session: session)
        return (MessageCenterModel(session: session, activity: activity), activity, session)
    }

    // MARK: - 何时回源（搬 message-center.test.mjs）

    func testUnchangedIdentityAndScopeReusesLoadedList() {
        XCTAssertEqual(
            MessageCenterRules.resolveNotificationReload(
                identityKey: "7:1",
                loadedIdentityKey: "7:1",
                requestedIdentityKey: "7:1",
                loadedScopeVersion: 3,
                scopeVersion: 3
            ),
            .reuse
        )
    }

    func testBumpedScopeReloadsFirstPage() {
        XCTAssertEqual(
            MessageCenterRules.resolveNotificationReload(
                identityKey: "7:1",
                loadedIdentityKey: "7:1",
                requestedIdentityKey: "7:1",
                loadedScopeVersion: 3,
                scopeVersion: 4
            ),
            .load
        )
    }

    func testFirstVisitLoadsAndInFlightRequestIsAwaited() {
        XCTAssertEqual(
            MessageCenterRules.resolveNotificationReload(
                identityKey: "7:1",
                loadedIdentityKey: "",
                requestedIdentityKey: "",
                loadedScopeVersion: 3,
                scopeVersion: 3
            ),
            .load
        )
        XCTAssertEqual(
            MessageCenterRules.resolveNotificationReload(
                identityKey: "7:1",
                loadedIdentityKey: "",
                requestedIdentityKey: "7:1",
                loadedScopeVersion: 3,
                scopeVersion: 3
            ),
            .wait
        )
    }

    func testDifferentAccountOrGenerationAlwaysLoads() {
        XCTAssertEqual(
            MessageCenterRules.resolveNotificationReload(
                identityKey: "8:1",
                loadedIdentityKey: "7:1",
                requestedIdentityKey: "7:1",
                loadedScopeVersion: 3,
                scopeVersion: 3
            ),
            .load
        )
        XCTAssertEqual(
            MessageCenterRules.resolveNotificationReload(
                identityKey: "7:2",
                loadedIdentityKey: "7:1",
                requestedIdentityKey: "7:1",
                loadedScopeVersion: 3,
                scopeVersion: 3
            ),
            .load
        )
        XCTAssertEqual(
            MessageCenterRules.resolveNotificationReload(
                identityKey: "8:1",
                loadedIdentityKey: "7:1",
                requestedIdentityKey: "8:1",
                loadedScopeVersion: 3,
                scopeVersion: 3
            ),
            .wait,
            "新身份的在途读取应等待"
        )
    }

    func testEntryCountCapsAt99Plus() {
        XCTAssertEqual(MessageCenterRules.formatEntryCount(0), "")
        XCTAssertEqual(MessageCenterRules.formatEntryCount(5), "5")
        XCTAssertEqual(MessageCenterRules.formatEntryCount(99), "99")
        XCTAssertEqual(MessageCenterRules.formatEntryCount(100), "99+")
        XCTAssertEqual(MessageCenterRules.formatEntryCount(-3), "")
    }

    func testEntriesDescribePendingCountsAndRoutes() {
        let entries = MessageCenterRules.buildMessageEntries(friendRequestCount: 2, challengeCount: 0)

        XCTAssertEqual(entries.map(\.id), ["friend-requests", "challenge-invites"])
        XCTAssertEqual(entries.map(\.route), [.friendRequests, .challenges(tab: .received)])
        XCTAssertEqual(entries.map(\.countText), ["2", ""])
        XCTAssertEqual(entries[0].desc, "2 条待处理")
        XCTAssertEqual(entries[1].desc, "暂无待回应的邀请")
    }

    func testEntriesTolerateMissingCounts() {
        let entries = MessageCenterRules.buildMessageEntries()

        XCTAssertEqual(entries.map(\.countText), ["", ""])
        XCTAssertEqual(entries[0].desc, "暂无待处理申请")
    }

    // MARK: - 通知跳转（搬 pages/message/index.vue 的 navigateByNotification）

    func testNotificationTargetsMapToNativeRoutes() {
        func route(_ target: String, extra: [String: Any] = [:]) -> AppRoute? {
            var data = extra
            data["target"] = target
            let raw = String(decoding: (try? JSONSerialization.data(withJSONObject: data)) ?? Data(), as: UTF8.self)
            return MessageCenterRules.notificationRoute(type: "system", data: raw)
        }

        XCTAssertEqual(route("/subPages/match/matchResult", extra: ["match_id": 66]), .matchReport(matchId: 66))
        XCTAssertNil(route("/subPages/match/matchResult"), "缺少 match_id 时不跳转")
        XCTAssertEqual(route("/subPages/match/playing", extra: ["match_id": 7]), .liveMatch(matchId: 7))
        XCTAssertEqual(route("/subPages/social/friendRequests"), .friendRequests)
        XCTAssertEqual(route("/subPages/social/friendList"), .friendList)
        XCTAssertEqual(route("/subPages/social/challenges"), .challenges(tab: .received))
        XCTAssertEqual(route("/subPages/achievement/index?game_type=5"), .achievementWall(gameType: 5))
        XCTAssertEqual(route("/subPages/season/report?game_type=5"), .seasonIndex, "缺少赛季 ID 时退回赛季页")
        XCTAssertEqual(
            route("/subPages/season/report?id=12&game_type=5"),
            .seasonReport(seasonId: 12, gameType: 5)
        )
        XCTAssertEqual(
            route("/subPages/season/report?id=12"),
            .seasonReport(seasonId: 12, gameType: 0)
        )
        XCTAssertEqual(route("/subPages/match/matchResult?from=push", extra: ["match_id": 9]), .matchReport(matchId: 9))
    }

    func testNotificationUrlIsAcceptedAsTarget() {
        let raw = #"{"url":"/subPages/social/friendList"}"#
        XCTAssertEqual(MessageCenterRules.notificationRoute(type: "system", data: raw), .friendList)
    }

    func testChallengeAndFriendRequestFallBackWhenNoTarget() {
        XCTAssertEqual(
            MessageCenterRules.notificationRoute(type: "challenge", data: #"{"challenge_id":12}"#),
            .challengeWaiting(challengeId: 12)
        )
        XCTAssertNil(MessageCenterRules.notificationRoute(type: "challenge", data: ""), "没有 challenge_id 不跳转")
        XCTAssertEqual(MessageCenterRules.notificationRoute(type: "friend_request", data: ""), .friendRequests)
        XCTAssertNil(MessageCenterRules.notificationRoute(type: "system", data: ""))
    }

    func testUnknownTargetAndMalformedDataDoNotNavigate() {
        XCTAssertNil(MessageCenterRules.notificationRoute(type: "system", data: #"{"target":"/subPages/unknown"}"#))
        XCTAssertNil(MessageCenterRules.notificationRoute(type: "friend_request", data: "{not json"))
        XCTAssertEqual(MessageCenterRules.notificationRoute(type: "system", data: "   "), nil)
    }

    // MARK: - 列表模型

    func testFirstShowLoadsOnceAndUsesSharedEntryCounts() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(unread: 1, pendingFriend: 2, received: 1)))
        server.stub("GET", "/api/notification/list", .json(listJSON([note(1), note(2, isRead: true)], unread: 1)))
        let (model, activity, _) = await makeModel(server)

        await model.onAppear()

        XCTAssertEqual(server.requestCount("/api/notification/list"), 1)
        XCTAssertEqual(server.requests("/api/notification/list").first?.query["page"], "1")
        XCTAssertEqual(server.requests("/api/notification/list").first?.query["page_size"], "20")
        XCTAssertEqual(model.items.count, 2)
        XCTAssertEqual(activity.unreadCount, 1)
        XCTAssertEqual(model.entries[0].desc, "2 条待处理")
        XCTAssertEqual(model.entries[1].desc, "1 条待回应")
    }

    func testShowingAgainWithoutChangeDoesNotReload() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list", .json(listJSON([note(1)])))
        let (model, _, _) = await makeModel(server)

        await model.onAppear()
        await model.onAppear()
        await model.onAppear()

        XCTAssertEqual(server.requestCount("/api/notification/list"), 1)
    }

    func testDirtyActivityReloadsFirstPageOnNextShow() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list", .json(listJSON([note(1)])))
        let (model, activity, _) = await makeModel(server)

        await model.onAppear()
        XCTAssertEqual(server.requestCount("/api/notification/list"), 1)

        activity.markDirty()
        await model.onAppear()

        XCTAssertEqual(server.requestCount("/api/notification/list"), 2)
    }

    func testMarkReadUpdatesItemAndDecrementsUnread() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list", .json(listJSON([note(1), note(2, isRead: true)], unread: 1)))
        server.stub("POST", "/api/notification/read", .json(["success": true]))
        let (model, activity, _) = await makeModel(server)
        await model.onAppear()

        await model.markRead(model.items[0])

        XCTAssertTrue(model.isRead(model.items[0]))
        XCTAssertEqual(activity.unreadCount, 0)
        XCTAssertEqual(server.requests("/api/notification/read").first?.bodyJSON["notification_id"] as? Int, 1)

        await model.markRead(model.items[0])
        XCTAssertEqual(server.requestCount("/api/notification/read"), 1, "已读条目不再发写请求")
    }

    func testFailedMarkReadKeepsItemUnread() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list", .json(listJSON([note(1)], unread: 1)))
        server.stub("POST", "/api/notification/read", .json(["success": false, "message": "操作失败"], status: 500))
        let (model, activity, _) = await makeModel(server)
        await model.onAppear()

        await model.markRead(model.items[0])

        XCTAssertFalse(model.isRead(model.items[0]))
        XCTAssertEqual(activity.unreadCount, 1)
    }

    func testMarkAllReadClearsUnreadAndReturnsTrue() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list", .json(listJSON([note(1), note(2)], unread: 2)))
        server.stub("POST", "/api/notification/read-all", .json(["success": true]))
        let (model, activity, _) = await makeModel(server)
        await model.onAppear()

        let success = await model.markAllRead()

        XCTAssertTrue(success)
        XCTAssertEqual(activity.unreadCount, 0)
        XCTAssertTrue(model.items.allSatisfy { model.isRead($0) })
    }

    func testDeleteUnreadItemHidesItAndDecrementsUnread() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list", .json(listJSON([note(1), note(2, isRead: true)], unread: 1)))
        server.stub("POST", "/api/notification/delete", .json(["success": true]))
        let (model, activity, _) = await makeModel(server)
        await model.onAppear()

        let deletedUnread = await model.delete(model.items[0])
        XCTAssertTrue(deletedUnread)
        XCTAssertEqual(model.items.map(\.id), [2])
        XCTAssertEqual(activity.unreadCount, 0)

        let deletedRead = await model.delete(model.items[0])
        XCTAssertTrue(deletedRead)
        XCTAssertEqual(activity.unreadCount, 0, "已读条目删除不再减少未读数")
    }

    func testFailedDeleteKeepsItem() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list", .json(listJSON([note(1)])))
        server.stub("POST", "/api/notification/delete", .json(["success": false, "message": "操作失败"], status: 500))
        let (model, _, _) = await makeModel(server)
        await model.onAppear()

        let deleted = await model.delete(model.items[0])
        XCTAssertFalse(deleted)
        XCTAssertEqual(model.items.count, 1)
    }

    func testFailedFirstLoadIsRetryableAndRecoversOnNextShow() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list", .json(["success": false, "message": "服务不可用"], status: 500))
        let (model, _, _) = await makeModel(server)

        await model.onAppear()
        guard case .failed = model.phase else {
            return XCTFail("首次加载失败应显示失败状态")
        }

        server.stub("GET", "/api/notification/list", .json(listJSON([note(1)])))
        await model.onAppear()

        XCTAssertEqual(server.requestCount("/api/notification/list"), 2, "失败后再次显示应重新回源")
        XCTAssertEqual(model.items.count, 1)
    }

    func testGuestModelNeverRequests() async {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let activity = ActivityStore(session: session)
        let model = MessageCenterModel(session: session, activity: activity)

        await model.onAppear()

        XCTAssertEqual(model.phase, .idle)
        XCTAssertTrue(model.items.isEmpty)
        XCTAssertEqual(server.requestCount("/api/notification/list"), 0)
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 0)
    }

    func testPullDownRefreshReloadsListAndForcesActivity() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list", .json(listJSON([note(1)])))
        let (model, _, _) = await makeModel(server)
        await model.onAppear()
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 1)

        await model.refresh()

        XCTAssertEqual(server.requestCount("/api/notification/list"), 2)
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 2, "下拉刷新强制重读共享活动快照")
    }

    func testRefreshFailureKeepsListWithBanner() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list", .json(listJSON([note(1)])))
        let (model, _, _) = await makeModel(server)
        await model.onAppear()

        server.stub("GET", "/api/notification/list", .json(["success": false, "message": "网络不可用"], status: 500))
        await model.refresh()

        XCTAssertEqual(model.items.count, 1, "刷新失败保留已加载列表")
        XCTAssertNotNil(model.refreshError)
        XCTAssertEqual(model.phase, .loaded)
    }

    func testEmptyListShowsEmptyPhase() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list", .json(listJSON([])))
        let (model, _, _) = await makeModel(server)

        await model.onAppear()

        XCTAssertEqual(model.phase, .empty)
        XCTAssertFalse(model.hasMore)
    }

    func testLoadMoreAppendsSecondPage() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list") { request in
            let page = Int(request.query["page"] ?? "1") ?? 1
            let notes = (0..<20).map { self.note(Int64(page * 100 + $0)) }
            return .json(self.listJSON(notes, total: 40))
        }
        let (model, _, _) = await makeModel(server)
        await model.onAppear()

        XCTAssertTrue(model.hasMore)
        await model.loadMore()

        XCTAssertEqual(model.items.count, 40)
        XCTAssertFalse(model.hasMore)
    }

    func testSwitchingAccountClearsListAndDropsLateResponse() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        let gate = DispatchSemaphore(value: 0)
        let firstResponse = listJSON([note(1, title: "旧账号的通知")], unread: 5)
        server.stub("GET", "/api/notification/list") { _ in
            gate.wait()
            return .json(firstResponse)
        }
        let (model, activity, session) = await makeModel(server)

        async let firstShow: Void = model.onAppear()
        await waitUntil { server.requestCount("/api/notification/list") == 1 }

        session.logout()
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse(user: AuthFixtures.userInfo(id: 8))))
        try await session.login(phone: AuthFixtures.phone, code: "123456")
        server.stub("GET", "/api/notification/list", .json(listJSON([note(9, title: "新账号的通知")], unread: 1)))

        await model.identityChanged()
        gate.signal()
        await firstShow

        XCTAssertEqual(model.items.map(\.title), ["新账号的通知"])
        XCTAssertEqual(activity.unreadCount, 1, "旧账号的未读计数不得写入新会话")
    }

    func testLogoutClearsLoadedList() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list", .json(listJSON([note(1)])))
        let (model, _, session) = await makeModel(server)
        await model.onAppear()
        XCTAssertEqual(model.items.count, 1)

        session.logout()
        await model.identityChanged()

        XCTAssertTrue(model.items.isEmpty)
        XCTAssertEqual(model.phase, .idle)
    }
}
