import Foundation
import XCTest
@testable import iOS

@MainActor
final class RealtimeConnectionTests: XCTestCase {
    private let target = RealtimeTarget(key: "match:1:false", matchId: 1)

    private func makeConnection(
        _ transport: FakeRealtimeTransport,
        _ clock: TestRealtimeClock
    ) -> TestRealtimeConnection {
        let connection = TestRealtimeConnection(
            transport: transport,
            wsBaseURL: URL(string: "wss://ws.example.test")!,
            random: { 0.5 },
            sleep: { try await clock.sleep($0) }
        )
        // 自动重连与恢复连接不经过 connect 的显式注入，依赖通道上的默认签发方。
        connection.ticketIssuer = { _ in "ticket" }
        return connection
    }

    private func connect(
        _ connection: TestRealtimeConnection,
        ticket: String? = "ticket",
        target: RealtimeTarget? = nil
    ) async throws {
        try await connection.connectTarget(
            target ?? self.target,
            options: RealtimeConnectOptions(ticketIssuer: { _ in ticket })
        )
    }

    // MARK: - 复用与代次

    func testSameTargetReusesOpenConnectionWithoutConnectingAgain() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let connection = makeConnection(transport, clock)

        try await connect(connection)
        try await connect(connection)

        XCTAssertEqual(transport.urls.count, 1)
        XCTAssertEqual(transport.sockets.count, 1)
        XCTAssertEqual(connection.reuseCount, 1)
    }

    func testInFlightConnectIsSharedBySameTarget() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let gate = FakeConnectGate()
        transport.gate = { await gate.wait() }
        let connection = makeConnection(transport, clock)

        let issuer: RealtimeTicketIssuer = { _ in "ticket" }
        let first = Task {
            try await connection.connectTarget(self.target, options: RealtimeConnectOptions(ticketIssuer: issuer))
        }
        await waitUntil { transport.urls.count == 1 }

        let second = Task {
            try await connection.connectTarget(self.target, options: RealtimeConnectOptions(ticketIssuer: issuer))
        }
        for _ in 0..<3 { await Task.yield() }
        XCTAssertEqual(transport.urls.count, 1, "同目标在途连接必须共享")

        gate.release()
        try await first.value
        try await second.value
        XCTAssertEqual(transport.urls.count, 1)
        XCTAssertEqual(transport.sockets.count, 1)
    }

    func testSwitchingTargetIgnoresLateMessagesAndCloseFromOldSocket() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let connection = makeConnection(transport, clock)

        try await connect(connection)
        let oldSocket = try XCTUnwrap(transport.sockets.first)
        var dispatched = 0
        connection.on(WSMessageType.any) { _ in dispatched += 1 }

        try await connect(
            connection,
            target: RealtimeTarget(key: "match:2:false", matchId: 2)
        )
        XCTAssertEqual(oldSocket.closeCount, 1)

        oldSocket.emit("{\"type\":\"score_update\"}")
        oldSocket.emitClose()
        for _ in 0..<5 { await Task.yield() }

        XCTAssertEqual(dispatched, 0, "旧连接的迟到消息必须被忽略")
        XCTAssertEqual(connection.state, .open)
        XCTAssertEqual(connection.reconnectAttempts, 0)
        XCTAssertEqual(transport.sockets.count, 2)
    }

    // MARK: - 票据失败

    func testTicketSessionInvalidStopsReconnectAndNotifiesOnce() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let connection = makeConnection(transport, clock)
        var notifications = 0
        connection.sessionInvalidHandler = { _ in notifications += 1 }

        do {
            try await connection.connectTarget(
                target,
                options: RealtimeConnectOptions(ticketIssuer: { _ in
                    throw APIError(message: "session invalid", statusCode: 401)
                })
            )
            XCTFail("会话失效必须抛出")
        } catch {}

        XCTAssertTrue(connection.sessionInvalid)
        XCTAssertEqual(connection.state, .error)
        for _ in 0..<5 { await Task.yield() }
        XCTAssertEqual(clock.pendingCount, 0, "会话失效后不得安排重连")
        XCTAssertEqual(notifications, 1)
        XCTAssertTrue(transport.urls.isEmpty)
    }

    func testTicketPermissionDeniedStopsReconnect() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let connection = makeConnection(transport, clock)

        do {
            try await connection.connectTarget(
                target,
                options: RealtimeConnectOptions(ticketIssuer: { _ in
                    throw APIError(message: "permission denied", statusCode: 403)
                })
            )
            XCTFail("403 必须抛出")
        } catch {}

        XCTAssertTrue(connection.permissionDenied)
        XCTAssertFalse(connection.sessionInvalid)
        XCTAssertEqual(connection.state, .error)
        for _ in 0..<5 { await Task.yield() }
        XCTAssertEqual(clock.pendingCount, 0, "权限拒绝后不得安排重连")
        XCTAssertTrue(transport.urls.isEmpty)
    }

    func testRecoverableTicketFailureSchedulesBackoffReconnect() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let connection = makeConnection(transport, clock)
        var attempts = 0
        let issuer: RealtimeTicketIssuer = { _ in
            attempts += 1
            if attempts == 1 {
                throw URLError(.networkConnectionLost)
            }
            return "ticket-2"
        }
        connection.ticketIssuer = issuer

        do {
            try await connection.connectTarget(target)
            XCTFail("首次票据失败必须抛出")
        } catch {}

        XCTAssertFalse(connection.sessionInvalid)
        XCTAssertFalse(connection.permissionDenied)
        XCTAssertEqual(connection.reconnectAttempts, 1)
        await waitUntil { clock.pendingCount == 1 }
        XCTAssertEqual(clock.requestedDelays, [3])

        clock.releaseAll()
        await waitUntil { connection.isConnected }
        XCTAssertEqual(attempts, 2)
        XCTAssertEqual(connection.reconnectAttempts, 0, "连接成功后必须重置退避")
        XCTAssertEqual(transport.urls.count, 1)
    }

    func testReconnectBackoffGrowsAndKeepsRetryingBeyondFiveFailures() async throws {
        let transport = FakeRealtimeTransport()
        transport.connectError = URLError(.networkConnectionLost)
        let clock = TestRealtimeClock()
        let connection = makeConnection(transport, clock)

        do {
            try await connect(connection)
            XCTFail("首次连接必须失败")
        } catch {}
        await waitUntil { clock.pendingCount == 1 }
        XCTAssertEqual(clock.requestedDelays, [3])

        // 连续 5 次重连失败后仍继续重试，不设次数上限。
        for expectedAttempt in 2...6 {
            clock.releaseAll()
            await waitUntil { connection.reconnectAttempts == expectedAttempt && clock.pendingCount == 1 }
        }
        XCTAssertEqual(clock.requestedDelays, [3, 6, 12, 24, 48, 60])

        transport.connectError = nil
        clock.releaseAll()
        await waitUntil { connection.isConnected }
        XCTAssertEqual(connection.reconnectAttempts, 0)
    }

    // MARK: - 心跳

    func testHeartbeatSendsPingAtEachInterval() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let connection = makeConnection(transport, clock)

        try await connect(connection)
        await waitUntil { clock.pendingCount == 1 }
        XCTAssertEqual(clock.requestedDelays, [30])
        let socket = try XCTUnwrap(transport.sockets.first)
        XCTAssertTrue(socket.sentTexts.isEmpty)

        clock.releaseAll()
        await waitUntil { socket.sentTexts.contains { $0.contains("\"type\":\"ping\"") } }

        await waitUntil { clock.pendingCount == 1 }
        clock.releaseAll()
        await waitUntil { socket.sentTexts.filter { $0.contains("\"ping\"") }.count == 2 }
    }

    // MARK: - 生命周期

    func testBackgroundOfflineAndInactiveTargetPauseThenResume() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let connection = makeConnection(transport, clock)
        try await connect(connection)

        connection.setForeground(false)
        XCTAssertEqual(connection.state, .closed)
        XCTAssertEqual(transport.sockets[0].closeCount, 1)
        XCTAssertEqual(connection.reconnectAttempts, 0, "后台不得重连")

        connection.setForeground(true)
        await waitUntil { transport.sockets.count == 2 }

        connection.setNetworkOnline(false)
        XCTAssertEqual(transport.sockets[1].closeCount, 1)
        XCTAssertEqual(connection.reconnectAttempts, 0, "离线不得重连")

        connection.setNetworkOnline(true)
        await waitUntil { transport.sockets.count == 3 }

        connection.setTargetActive(false)
        XCTAssertEqual(transport.sockets[2].closeCount, 1)
        XCTAssertEqual(connection.reconnectAttempts, 0, "目标不活跃不得重连")

        connection.setTargetActive(true)
        await waitUntil { transport.sockets.count == 4 }
        XCTAssertTrue(connection.isConnected)
    }

    func testDisconnectStopsReconnectAndDispatch() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let connection = makeConnection(transport, clock)
        try await connect(connection)

        let socket = try XCTUnwrap(transport.sockets.first)
        var dispatched = 0
        connection.on(WSMessageType.any) { _ in dispatched += 1 }

        connection.disconnect()
        XCTAssertEqual(socket.closeCount, 1)
        XCTAssertEqual(connection.state, .idle)
        XCTAssertNil(connection.target)
        XCTAssertTrue(connection.manualDisconnect)

        socket.emit("{\"type\":\"score_update\"}")
        socket.emitClose()
        for _ in 0..<5 { await Task.yield() }

        XCTAssertEqual(dispatched, 0)
        XCTAssertEqual(transport.sockets.count, 1)
        XCTAssertEqual(connection.reconnectAttempts, 0)
        XCTAssertEqual(clock.pendingCount, 0)
    }

    func testServerSessionInvalidMessageNotifiesOnceAndStopsReconnect() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let connection = makeConnection(transport, clock)
        var notifications = 0
        connection.sessionInvalidHandler = { _ in notifications += 1 }

        try await connect(connection)
        let socket = try XCTUnwrap(transport.sockets.first)
        let text = "{\"type\":\"session_invalid\",\"data\":{\"reason\":\"SESSION_INVALID\"}}"
        socket.emit(text)
        socket.emit(text)
        await waitUntil { notifications == 1 }
        for _ in 0..<5 { await Task.yield() }

        XCTAssertEqual(notifications, 1, "会话失效回调只触发一次")
        XCTAssertTrue(connection.sessionInvalid)
        XCTAssertEqual(connection.state, .closed)

        socket.emitClose()
        for _ in 0..<5 { await Task.yield() }
        XCTAssertEqual(connection.reconnectAttempts, 0, "会话失效后不得重连")
        XCTAssertEqual(transport.sockets.count, 1)
    }

    // MARK: - 地址与分发

    func testTicketIsPercentEncodedInSocketURL() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let connection = makeConnection(transport, clock)
        let ticket = "tk+1/2&x=y 中"

        try await connect(connection, ticket: ticket)

        let url = try XCTUnwrap(transport.urls.first)
        XCTAssertEqual(url.scheme, "wss")
        XCTAssertEqual(url.host, "ws.example.test")
        XCTAssertEqual(url.path, "/api/test/ws")
        let components = try XCTUnwrap(URLComponents(url: url, resolvingAgainstBaseURL: false))
        XCTAssertEqual(components.queryItems?.first(where: { $0.name == "ticket" })?.value, ticket)
    }

    func testHandlersReceiveEnvelopeAndIgnoreUnparseableMessages() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let connection = makeConnection(transport, clock)
        try await connect(connection)

        var scores: [RealtimeMessage] = []
        var all: [RealtimeMessage] = []
        let token = connection.on(WSMessageType.scoreUpdate) { scores.append($0) }
        connection.on(WSMessageType.any) { all.append($0) }

        let socket = try XCTUnwrap(transport.sockets.first)
        socket.emit("not-json")
        socket.emit("{\"type\":\"score_update\",\"data\":{\"my_score\":3}}")
        await waitUntil { scores.count == 1 }

        XCTAssertEqual(all.count, 1)
        XCTAssertEqual(scores[0].type, WSMessageType.scoreUpdate)
        XCTAssertEqual(
            try JSONDecoder().decode([String: Int].self, from: XCTUnwrap(scores[0].payload)),
            ["my_score": 3]
        )

        connection.off(token)
        socket.emit("{\"type\":\"score_update\",\"data\":{\"my_score\":4}}")
        await waitUntil { all.count == 2 }
        XCTAssertEqual(scores.count, 1)
    }
}

final class RealtimeMessageTests: XCTestCase {
    func testEnvelopeKeepsPayloadAsRawData() throws {
        let message = try XCTUnwrap(
            RealtimeMessage(text: "{\"type\":\"sync\",\"data\":{\"match_id\":9}}")
        )
        XCTAssertEqual(message.type, "sync")
        let payload = try XCTUnwrap(message.payload)
        XCTAssertEqual(try JSONDecoder().decode([String: Int].self, from: payload), ["match_id": 9])
    }

    func testNullPayloadAndInvalidJSON() throws {
        XCTAssertNil(try XCTUnwrap(RealtimeMessage(text: "{\"type\":\"pong\",\"data\":null}")).payload)
        XCTAssertNil(RealtimeMessage(text: "not-json"))
        XCTAssertNil(RealtimeMessage(text: "{\"data\":{}}"), "缺少 type 的消息无法分发")
    }

    func testMessageTypeConstantsMatchServerContract() {
        XCTAssertEqual(WSMessageType.scoreUpdate, "score_update")
        XCTAssertEqual(WSMessageType.roundEnd, "round_end")
        XCTAssertEqual(WSMessageType.matchFinishTimeout, "match_finish_timeout")
        XCTAssertEqual(WSMessageType.sessionInvalid, "session_invalid")
        XCTAssertEqual(WSMessageType.ping, "ping")
        XCTAssertEqual(WSMessageType.pong, "pong")
        XCTAssertEqual(WSMessageType.any, "*")
    }
}
