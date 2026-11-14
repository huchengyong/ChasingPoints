import Foundation
import XCTest
@testable import iOS

/// 用户实时通道与应用生命周期、活动状态的接线测试；全部使用测试替身，不连真实 WebSocket。
@MainActor
final class AppRealtimeTests: XCTestCase {
    private func bootstrap(unread: Int = 0, pendingFriend: Int = 0) -> [String: Any] {
        [
            "success": true,
            "current_match": NSNull(),
            "current_challenge": NSNull(),
            "received_pending_challenge_count": 0,
            "server_time": "2026-01-01 12:00:00",
            "unread_count": unread,
            "pending_friend_request_count": pendingFriend,
            "latest_season_rollover": NSNull(),
            "availability": ActivityFixtures.allAvailable,
        ]
    }

    private struct Harness {
        let realtime: AppRealtime
        let user: UserRealtime
        let activity: ActivityStore
        let session: SessionStore
        let transport: FakeRealtimeTransport
        let clock: TestRealtimeClock

        var socket: FakeRealtimeSocket? { transport.sockets.last }
    }

    private func makeHarness(_ server: MockHTTPServer) async -> Harness {
        let session = await makeRestoredAccountSession(server)
        let activity = ActivityStore(session: session)
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let user = UserRealtime(
            session: session,
            transport: transport,
            wsBaseURL: URL(string: "wss://ws.example.test")!,
            random: { 0.5 },
            sleep: { try await clock.sleep($0) },
            ticketIssuer: { _ in "test-ticket" }
        )
        let realtime = AppRealtime(session: session, activity: activity, user: user)
        // 正式启动路径在身份变化时先同步活动状态，测试保持一致，避免首次标脏被身份重置覆盖。
        activity.syncIdentity()
        return Harness(
            realtime: realtime,
            user: user,
            activity: activity,
            session: session,
            transport: transport,
            clock: clock
        )
    }

    private func emit(_ type: String, _ data: [String: Any]? = nil, to socket: FakeRealtimeSocket) {
        var object: [String: Any] = ["type": type]
        if let data { object["data"] = data }
        let payload = (try? JSONSerialization.data(withJSONObject: object)) ?? Data()
        socket.emit(String(decoding: payload, as: UTF8.self))
    }

    // MARK: - 身份与连接

    func testIdentityConnectsOnceAndLogoutDisconnects() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        let harness = await makeHarness(server)

        await harness.realtime.handleIdentityChange()
        XCTAssertTrue(harness.user.isConnected)
        XCTAssertEqual(harness.transport.sockets.count, 1)
        XCTAssertEqual(harness.transport.urls.first?.path, "/api/user/ws")

        await harness.realtime.handleIdentityChange()
        XCTAssertEqual(harness.transport.sockets.count, 1, "同一身份已连接时不得重复建连")

        harness.session.logout()
        await harness.realtime.handleIdentityChange()
        XCTAssertFalse(harness.user.isConnected)
        XCTAssertEqual(harness.transport.sockets[0].closeCount, 1)
    }

    func testAccountSwitchClosesOldSocketAndReconnects() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        let harness = await makeHarness(server)

        await harness.realtime.handleIdentityChange()
        let oldSocket = try XCTUnwrap(harness.transport.sockets.first)

        harness.session.logout()
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse(user: AuthFixtures.userInfo(id: 8))))
        try await harness.session.login(phone: AuthFixtures.phone, code: "123456")
        await harness.realtime.handleIdentityChange()

        XCTAssertEqual(harness.transport.sockets.count, 2)
        XCTAssertEqual(oldSocket.closeCount, 1, "切号必须断开旧连接")
        XCTAssertTrue(harness.user.isConnected)
    }

    func testBackgroundDisconnectsAndForegroundReconnects() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        let harness = await makeHarness(server)
        await harness.realtime.handleIdentityChange()

        harness.realtime.setForeground(false)
        XCTAssertEqual(harness.transport.sockets[0].closeCount, 1)
        XCTAssertFalse(harness.user.isConnected)

        harness.realtime.setForeground(true)
        await waitUntil { harness.transport.sockets.count == 2 && harness.user.isConnected }
    }

    func testOfflineSkipsConnectAndRecoveryConnects() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        let harness = await makeHarness(server)

        harness.realtime.setNetworkOnline(false)
        await harness.realtime.handleIdentityChange()
        XCTAssertTrue(harness.transport.sockets.isEmpty, "离线时不得尝试建连")

        harness.realtime.setNetworkOnline(true)
        await waitUntil { harness.transport.sockets.count == 1 && harness.user.isConnected }
    }

    // MARK: - 实时消息

    func testNotificationUpdateAppliesUnreadCountAndReloadsActivity() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(unread: 5)))
        let harness = await makeHarness(server)
        await harness.realtime.handleIdentityChange()

        emit("notification_update", ["unread_count": 5, "category": "system"], to: try XCTUnwrap(harness.socket))

        await waitUntil { server.requestCount("/api/user/bootstrap") == 1 }
        XCTAssertEqual(harness.activity.unreadCount, 5)
        XCTAssertEqual(harness.activity.dirtySeq, 1)
        XCTAssertTrue(harness.activity.isLoaded)
    }

    func testUserDataUpdatedAppliesPendingCountAndBumpsRevision() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        let harness = await makeHarness(server)
        await harness.realtime.handleIdentityChange()

        emit(
            "user_data_updated",
            ["scopes": ["friend"], "pending_friend_request_count": 3, "competitive_revision": 7],
            to: try XCTUnwrap(harness.socket)
        )

        await waitUntil { harness.activity.userDataRevision == 1 }
        XCTAssertEqual(harness.activity.pendingFriendRequestCount, 3)
        XCTAssertEqual(harness.activity.dirtySeq, 1)
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 0, "活动数据变更只标脏，不立刻强制回源")
    }

    func testRankInfoUpdatedAlsoBumpsRevision() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        let harness = await makeHarness(server)
        await harness.realtime.handleIdentityChange()

        emit("rank_info_updated", ["scopes": ["rank"]], to: try XCTUnwrap(harness.socket))

        await waitUntil { harness.activity.userDataRevision == 1 }
    }

    func testSessionInvalidRefreshesActivityOnlyOnce() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(unread: 2)))
        let harness = await makeHarness(server)
        await harness.realtime.handleIdentityChange()
        let socket = try XCTUnwrap(harness.socket)

        emit("session_invalid", ["reason": "SESSION_INVALID"], to: socket)
        await waitUntil { server.requestCount("/api/user/bootstrap") == 1 }
        XCTAssertTrue(harness.activity.isLoaded)
        XCTAssertEqual(harness.activity.unreadCount, 2)

        emit("session_invalid", ["reason": "SESSION_INVALID"], to: socket)
        try? await Task.sleep(nanoseconds: 150_000_000)
        XCTAssertEqual(server.requestCount("/api/user/bootstrap"), 1, "同一段连接只通知一次会话失效")
    }

    // MARK: - 网络监听

    func testNetworkMonitorStartsOnlyOnce() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        let harness = await makeHarness(server)
        let monitor = FakeNetworkMonitor()

        harness.realtime.startNetworkMonitoring(monitor)
        harness.realtime.startNetworkMonitoring(monitor)

        XCTAssertEqual(monitor.startCount, 1, "根视图重建不得重复启动监听")
    }

    func testNetworkStatusChangePausesAndResumesConnection() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        let harness = await makeHarness(server)
        await harness.realtime.handleIdentityChange()
        let monitor = FakeNetworkMonitor()
        harness.realtime.startNetworkMonitoring(monitor)

        monitor.emit(false)
        XCTAssertFalse(harness.user.isConnected)
        XCTAssertEqual(harness.transport.sockets[0].closeCount, 1)

        monitor.emit(true)
        await waitUntil { harness.transport.sockets.count == 2 && harness.user.isConnected }
    }
}

/// 网络状态监听替身：由测试手动触发在线 / 离线。
@MainActor
private final class FakeNetworkMonitor: NetworkStatusMonitor {
    var onStatusChange: ((Bool) -> Void)?
    private(set) var startCount = 0

    func start() {
        startCount += 1
    }

    func emit(_ online: Bool) {
        onStatusChange?(online)
    }
}
