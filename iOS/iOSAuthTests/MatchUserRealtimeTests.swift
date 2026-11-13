import Foundation
import XCTest
@testable import iOS

@MainActor
final class MatchUserRealtimeTests: XCTestCase {
    private func queryValue(_ name: String, _ url: URL) -> String? {
        URLComponents(url: url, resolvingAgainstBaseURL: false)?
            .queryItems?
            .first(where: { $0.name == name })?
            .value
    }

    private func makeMatch(
        session: SessionStore,
        transport: FakeRealtimeTransport,
        clock: TestRealtimeClock,
        ticketIssuer: RealtimeTicketIssuer? = nil
    ) -> MatchRealtime {
        MatchRealtime(
            session: session,
            transport: transport,
            wsBaseURL: URL(string: "wss://ws.example.test")!,
            random: { 0.5 },
            sleep: { try await clock.sleep($0) },
            ticketIssuer: ticketIssuer
        )
    }

    private func makeUser(
        session: SessionStore,
        transport: FakeRealtimeTransport,
        clock: TestRealtimeClock
    ) -> UserRealtime {
        UserRealtime(
            session: session,
            transport: transport,
            wsBaseURL: URL(string: "wss://ws.example.test")!,
            random: { 0.5 },
            sleep: { try await clock.sleep($0) }
        )
    }

    // MARK: - 对局通道

    func testMatchChannelConnectsWithTicketAndRequestsSyncOnConnectAndReuse() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let session = makeSessionStore(MockHTTPServer(), credentialStore: InMemoryCredentialStore())
        let match = makeMatch(session: session, transport: transport, clock: clock)

        try await match.connect(matchId: 42, ticketIssuer: { _ in "match-ticket-42" })

        let url = try XCTUnwrap(transport.urls.first)
        XCTAssertEqual(url.path, "/api/match/ws")
        XCTAssertEqual(queryValue("match_id", url), "42")
        XCTAssertEqual(queryValue("ticket", url), "match-ticket-42")
        XCTAssertNil(queryValue("token", url), "不得把长期 token 放进查询串")
        XCTAssertEqual(match.matchId, 42)
        let socket = try XCTUnwrap(transport.sockets.first)
        await waitUntil { socket.sentTexts.contains { $0.contains("\"type\":\"sync\"") } }

        try await match.connect(matchId: 42, ticketIssuer: { _ in "match-ticket-42" })
        XCTAssertEqual(transport.sockets.count, 1, "同目标必须复用连接")
        await waitUntil { socket.sentTexts.filter { $0.contains("\"sync\"") }.count == 2 }
    }

    func testMatchChannelGuestSpectateConnectsWithoutTicket() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let session = makeSessionStore(MockHTTPServer(), credentialStore: InMemoryCredentialStore())
        let match = makeMatch(session: session, transport: transport, clock: clock)

        try await match.connect(matchId: 7, allowAnonymous: true)

        let url = try XCTUnwrap(transport.urls.first)
        XCTAssertEqual(queryValue("match_id", url), "7")
        XCTAssertNil(queryValue("ticket", url), "访客观赛不应伪造票据")
        XCTAssertTrue(match.allowAnonymous)
    }

    func testMatchChannelIssuesTicketThroughSession() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let server = MockHTTPServer()
        server.stub("POST", "/api/match/ws-ticket", .json(["success": true, "ticket": "server-ticket"]))
        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
        let match = makeMatch(session: session, transport: transport, clock: clock)

        try await match.connect(matchId: 42)

        let request = try XCTUnwrap(server.requests("/api/match/ws-ticket").first)
        XCTAssertEqual(request.bodyJSON["match_id"] as? Int, 42)
        XCTAssertEqual(request.bearerToken, AuthFixtures.oldAccess)
        XCTAssertEqual(queryValue("ticket", try XCTUnwrap(transport.urls.first)), "server-ticket")
    }

    // MARK: - 用户通道

    func testUserChannelIssuesTicketThroughSession() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let server = MockHTTPServer()
        server.stub("POST", "/api/user/ws-ticket", .json(["success": true, "ticket": "user-ticket"]))
        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
        let user = makeUser(session: session, transport: transport, clock: clock)

        try await user.connect(authGeneration: 1)

        let url = try XCTUnwrap(transport.urls.first)
        XCTAssertEqual(url.path, "/api/user/ws")
        XCTAssertEqual(queryValue("ticket", url), "user-ticket")
        XCTAssertEqual(server.requests("/api/user/ws-ticket").first?.bearerToken, AuthFixtures.oldAccess)
        XCTAssertTrue(user.isConnected)
    }

    func testUserChannelRotatesConnectionWhenAuthGenerationChanges() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let server = MockHTTPServer()
        server.stub("POST", "/api/user/ws-ticket", .json(["success": true, "ticket": "user-ticket"]))
        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
        let user = makeUser(session: session, transport: transport, clock: clock)

        try await user.connect(authGeneration: 1)
        let firstSocket = try XCTUnwrap(transport.sockets.first)

        try await user.connect(authGeneration: 2)

        XCTAssertEqual(firstSocket.closeCount, 1, "切号必须断开旧连接")
        XCTAssertEqual(transport.sockets.count, 2)
        XCTAssertEqual(user.authGeneration, 2)
    }

    func testUserChannelRequestsFreshTicketOnReconnect() async throws {
        let transport = FakeRealtimeTransport()
        let clock = TestRealtimeClock()
        let server = MockHTTPServer()
        let ticketCounter = LockedCounter()
        server.stub("POST", "/api/user/ws-ticket") { _ in
            .json(["success": true, "ticket": "ticket-\(ticketCounter.next())"])
        }
        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
        let user = makeUser(session: session, transport: transport, clock: clock)

        try await user.connect(authGeneration: 1)
        XCTAssertEqual(queryValue("ticket", try XCTUnwrap(transport.urls.first)), "ticket-1")

        let firstSocket = try XCTUnwrap(transport.sockets.first)
        firstSocket.emitClose()
        await waitUntil { user.reconnectAttempts == 1 }
        clock.releaseAll()
        await waitUntil { transport.sockets.count == 2 }

        XCTAssertEqual(queryValue("ticket", transport.urls[1]), "ticket-2", "重连必须换新票据")
        XCTAssertEqual(server.requestCount("/api/user/ws-ticket"), 2)
    }
}
