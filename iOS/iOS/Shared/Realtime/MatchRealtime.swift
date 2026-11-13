import Foundation

private struct MatchWSTicketRequest: Encodable {
    let matchId: Int64
}

/// `MatchWebSocket`：按 `matchId` 连接，查询参数 `match_id` + 一次性 `ticket`。
@MainActor
final class MatchRealtime: RealtimeConnection {
    private(set) var matchId: Int64?
    private(set) var allowAnonymous = false

    init(
        session: SessionStore,
        onSessionInvalid: ((Data?) -> Void)? = nil,
        transport: RealtimeTransport? = nil,
        wsBaseURL: URL? = nil,
        baseReconnectDelayMs: Double? = nil,
        random: @escaping () -> Double = { Double.random(in: 0..<1) },
        sleep: @escaping RealtimeSleep = { try await Task.sleep(for: .seconds($0)) },
        ticketIssuer: RealtimeTicketIssuer? = nil
    ) {
        super.init(
            transport: transport,
            wsBaseURL: wsBaseURL,
            baseReconnectDelayMs: baseReconnectDelayMs,
            random: random,
            sleep: sleep
        )
        sessionInvalidHandler = onSessionInvalid
        // shortcut: 访客判定用已加载的用户资料；未登录观赛时用无票据连接公开对局。
        self.ticketIssuer = ticketIssuer ?? { [session] target in
            if target.allowAnonymous, session.identity == nil { return nil }
            let response: WSTicketResponse = try await session.authorizedPost(
                "/api/match/ws-ticket",
                body: MatchWSTicketRequest(matchId: target.matchId ?? 0)
            )
            return response.ticket
        }
    }

    func connect(
        matchId: Int64,
        allowAnonymous: Bool = false,
        force: Bool = false,
        ticketIssuer: RealtimeTicketIssuer? = nil
    ) async throws {
        self.matchId = matchId
        self.allowAnonymous = allowAnonymous
        try await connectTarget(
            RealtimeTarget(
                key: "match:" + String(matchId) + ":" + String(allowAnonymous),
                matchId: matchId,
                allowAnonymous: allowAnonymous
            ),
            options: RealtimeConnectOptions(force: force, ticketIssuer: ticketIssuer)
        )
    }

    override func socketPath(for target: RealtimeTarget) -> String {
        "/api/match/ws"
    }

    override func queryItems(for target: RealtimeTarget, ticket: String?) -> [URLQueryItem] {
        var items = [URLQueryItem(name: "match_id", value: String(target.matchId ?? 0))]
        if let ticket {
            items.append(URLQueryItem(name: "ticket", value: ticket))
        }
        return items
    }

    override func onConnected() async {
        await requestSync()
    }

    override func onReuse() async {
        await requestSync()
    }

    /// 页面回到前台或复用连接后主动要一次快照。
    func requestSync() async {
        await send(type: WSMessageType.sync)
    }

    override func disconnect() {
        super.disconnect()
        matchId = nil
        allowAnonymous = false
    }
}
