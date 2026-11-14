import Foundation
import XCTest
@testable import iOS

/// 活动状态与角标规则的回归测试。全部使用 MockHTTPServer，不访问真实后端。
@MainActor
final class ActivityStoreTests: XCTestCase {
    private var now = Date(timeIntervalSince1970: 1_700_000_000)

    private func makeActivity(_ server: MockHTTPServer) async -> (ActivityStore, SessionStore) {
        let session = await makeRestoredAccountSession(server)
        return (ActivityStore(session: session, now: { self.now }), session)
    }

    private func bootstrap(
        match: Any = NSNull(),
        challenge: Any = NSNull(),
        received: Int = 0,
        serverTime: String = "2026-01-01 12:00:00",
        unread: Int = 0,
        pendingFriend: Int = 0,
        season: Any = NSNull(),
        availability: [String: Bool] = ActivityFixtures.allAvailable
    ) -> [String: Any] {
        [
            "success": true,
            "current_match": match,
            "current_challenge": challenge,
            "received_pending_challenge_count": received,
            "server_time": serverTime,
            "unread_count": unread,
            "pending_friend_request_count": pendingFriend,
            "latest_season_rollover": season,
            "availability": availability,
        ]
    }

    func testBootstrapAppliesEveryAvailableSection() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(
            match: ActivityFixtures.currentMatch(),
            challenge: ActivityFixtures.challenge(),
            received: 2,
            unread: 4,
            pendingFriend: 3,
            season: ActivityFixtures.notification(id: 9, type: "season_rollover")
        )))
        let (activity, _) = await makeActivity(server)

        await activity.refresh()

        XCTAssertTrue(activity.isLoaded)
        XCTAssertEqual(activity.currentMatch?.id, 1)
        XCTAssertEqual(activity.currentChallenge?.id, 3)
        XCTAssertEqual(activity.receivedChallengeCount, 2)
        XCTAssertEqual(activity.challengeServerTime, "2026-01-01 12:00:00")
        XCTAssertTrue(activity.challengeAvailable)
        XCTAssertEqual(activity.unreadCount, 4)
        XCTAssertEqual(activity.pendingFriendRequestCount, 3)
        XCTAssertEqual(activity.latestSeasonRollover?.id, 9)
    }

    func testPartialBootstrapKeepsOldValuesAndStaysDirty() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(
            challenge: ActivityFixtures.challenge(),
            received: 5,
            unread: 7,
            pendingFriend: 2
        )))
        let (activity, _) = await makeActivity(server)
        await activity.refresh()
        XCTAssertEqual(activity.currentChallenge?.id, 3)
        XCTAssertEqual(activity.receivedChallengeCount, 5)
        XCTAssertEqual(activity.unreadCount, 7)

        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(
            received: 99,
            unread: 99,
            pendingFriend: 99,
            availability: [
                "current_match": true,
                "current_challenge": false,
                "challenge_received_count": false,
                "unread_count": false,
                "pending_friend_request_count": true,
                "latest_season_rollover": true,
            ]
        )))
        await activity.refresh()

        XCTAssertEqual(activity.currentChallenge?.id, 3, "不可用分区必须保留旧值")
        XCTAssertEqual(activity.receivedChallengeCount, 5)
        XCTAssertEqual(activity.unreadCount, 7)
        XCTAssertFalse(activity.challengeAvailable)
        XCTAssertEqual(activity.pendingFriendRequestCount, 99, "可用分区仍应更新")

        await activity.refreshIfDirty()
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 3, "仍有分区不可用时保持脏，补读会再请求")
    }

    func testOnlyOneChallengeScopeMissingMakesChallengeUnavailable() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(
            challenge: ActivityFixtures.challenge(),
            availability: [
                "current_match": true,
                "current_challenge": true,
                "challenge_received_count": false,
                "unread_count": true,
                "pending_friend_request_count": true,
                "latest_season_rollover": true,
            ]
        )))
        let (activity, _) = await makeActivity(server)

        await activity.refresh()

        XCTAssertFalse(activity.challengeAvailable)
        XCTAssertEqual(activity.currentChallenge?.id, 3)
    }

    func testOverallFailureKeepsStateAndDirty() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(unread: 6, pendingFriend: 1)))
        let (activity, _) = await makeActivity(server)
        await activity.refresh()
        XCTAssertEqual(activity.unreadCount, 6)

        server.stub("GET", "/api/user/bootstrap", .json(["success": false, "message": "服务不可用"], status: 500))
        await activity.refresh()

        XCTAssertEqual(activity.unreadCount, 6, "整体失败不得清空或改写状态")
        XCTAssertEqual(activity.pendingFriendRequestCount, 1)
        XCTAssertTrue(activity.isLoaded)

        await activity.refreshIfDirty()
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 3, "整体失败后保持脏")
    }

    func testFreshnessUsesThirtySecondTtl() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        let (activity, _) = await makeActivity(server)

        now = Date(timeIntervalSince1970: 1_700_000_000)
        await activity.refreshIfNeeded()
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 1)

        now = now.addingTimeInterval(29)
        await activity.refreshIfNeeded()
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 1, "30 秒内沿用缓存")

        now = now.addingTimeInterval(2)
        await activity.refreshIfNeeded()
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 2)
    }

    func testConcurrentRefreshIfNeededSharesOneRequest() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(unread: 1)))
        let (activity, _) = await makeActivity(server)

        async let first: Void = activity.refreshIfNeeded()
        async let second: Void = activity.refreshIfNeeded()
        _ = await (first, second)

        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 1)
        XCTAssertEqual(activity.unreadCount, 1)
    }

    func testMarkDirtyDuringReadSurvivesResponseAndRefreshIfDirtyReloads() async {
        let server = MockHTTPServer()
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/user/bootstrap") { _ in
            gate.wait()
            return .json(self.bootstrap(unread: 2))
        }
        let (activity, _) = await makeActivity(server)

        async let refresh: Void = activity.refresh()
        await waitUntil { server.requestCount("/api/user/bootstrap") == 1 }
        let dirtyBefore = activity.dirtySeq
        activity.markDirty()
        gate.signal()
        await refresh

        XCTAssertEqual(activity.dirtySeq, dirtyBefore + 1)
        XCTAssertEqual(activity.unreadCount, 2)
        let requestsAfterResponse = server.requestCount("/api/user/bootstrap")

        await activity.refreshIfDirty()
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), requestsAfterResponse + 1)
    }

    func testIdentityChangeClearsStateAndDropsLateResponse() async {
        let server = MockHTTPServer()
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/user/bootstrap") { _ in
            gate.wait()
            return .json(self.bootstrap(
                match: ActivityFixtures.currentMatch(),
                unread: 8,
                pendingFriend: 4
            ))
        }
        let (activity, session) = await makeActivity(server)

        async let refresh: Void = activity.refresh()
        await waitUntil { server.requestCount("/api/user/bootstrap") == 1 }
        session.logout()
        activity.syncIdentity()
        gate.signal()
        await refresh

        XCTAssertFalse(activity.isLoaded)
        XCTAssertNil(activity.currentMatch)
        XCTAssertEqual(activity.unreadCount, 0)
        XCTAssertEqual(activity.pendingFriendRequestCount, 0)
        XCTAssertEqual(activity.messageBadge, .none)
    }

    func testCountersNeverGoNegative() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        let (activity, _) = await makeActivity(server)

        activity.setUnreadCount(-5)
        XCTAssertEqual(activity.unreadCount, 0)
        activity.decrementUnread()
        XCTAssertEqual(activity.unreadCount, 0)
        activity.setUnreadCount(2)
        activity.decrementUnread()
        XCTAssertEqual(activity.unreadCount, 1)
        activity.setPendingFriendRequestCount(-3)
        XCTAssertEqual(activity.pendingFriendRequestCount, 0)
    }

    func testUserDataRevisionAndSeasonRolloverClearing() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(
            season: ActivityFixtures.notification(id: 4, type: "season_rollover")
        )))
        let (activity, _) = await makeActivity(server)

        await activity.refresh()
        XCTAssertNotNil(activity.latestSeasonRollover)
        activity.clearLatestSeasonRollover()
        XCTAssertNil(activity.latestSeasonRollover)

        XCTAssertEqual(activity.userDataRevision, 0)
        activity.bumpUserDataRevision()
        activity.bumpUserDataRevision()
        XCTAssertEqual(activity.userDataRevision, 2)
    }

    // MARK: - 消息角标（搬 main-tab-bar.test.mjs）

    func testMessageBadgeNoneWhenSignedOut() async {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let activity = ActivityStore(session: session)

        XCTAssertEqual(activity.messageBadge, .none)
    }

    func testMessageBadgePrefersUnreadCountOverPendingItems() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(received: 2, unread: 3, pendingFriend: 1)))
        let (activity, _) = await makeActivity(server)
        await activity.refresh()

        XCTAssertEqual(activity.messageBadge, .text("3"))
    }

    func testMessageBadgeCapsAt99Plus() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        let (activity, _) = await makeActivity(server)

        activity.setUnreadCount(120)
        XCTAssertEqual(activity.messageBadge, .text("99+"))
        activity.setUnreadCount(99)
        XCTAssertEqual(activity.messageBadge, .text("99"))
        activity.setUnreadCount(7)
        XCTAssertEqual(activity.messageBadge, .text("7"))
    }

    func testMessageBadgeShowsDotForPendingFriendRequestOrChallenge() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(pendingFriend: 1)))
        let (activity, _) = await makeActivity(server)

        XCTAssertEqual(activity.messageBadge, .none)
        activity.setPendingFriendRequestCount(1)
        XCTAssertEqual(activity.messageBadge, .dot)
        activity.setPendingFriendRequestCount(0)
        XCTAssertEqual(activity.messageBadge, .none)

        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(received: 2)))
        await activity.refresh()
        XCTAssertEqual(activity.messageBadge, .dot)
        XCTAssertTrue(activity.challengeAvailable)
    }
}

enum ActivityFixtures {
    static let allAvailable: [String: Bool] = [
        "current_match": true,
        "current_challenge": true,
        "challenge_received_count": true,
        "unread_count": true,
        "pending_friend_request_count": true,
        "latest_season_rollover": true,
    ]

    static func currentMatch(id: Int64 = 1, opponent: String = "对手") -> [String: Any] {
        [
            "id": id,
            "game_type": 3,
            "game_type_name": "中式八球",
            "game_mode": "pvp",
            "player1_id": 1,
            "player1_name": "我",
            "player1_avatar": "",
            "player2_id": 2,
            "player2_name": opponent,
            "player2_avatar": "",
            "opponent_name": opponent,
            "my_score": 0,
            "opponent_score": 0,
            "current_round": 1,
            "server_revision": 1,
            "match_time": "2026-01-01 12:00:00",
            "duration_seconds": 0,
        ]
    }

    static func challenge(id: Int64 = 3) -> [String: Any] {
        [
            "id": id,
            "from_user_id": 2,
            "from_nickname": "甲",
            "from_avatar": "",
            "to_user_id": 1,
            "to_nickname": "我",
            "to_avatar": "",
            "game_type": 3,
            "message": "",
            "status": 0,
            "created_at": "2026-01-01 12:00:00",
        ]
    }

    static func notification(
        id: Int64,
        type: String = "season_rollover",
        title: String = "换季提醒",
        content: String = "新赛季已开始",
        data: String = "",
        isRead: Bool = false
    ) -> [String: Any] {
        [
            "id": id,
            "type": type,
            "title": title,
            "content": content,
            "data": data,
            "is_read": isRead,
            "created_at": "2026-01-01 12:00:00",
        ]
    }
}
