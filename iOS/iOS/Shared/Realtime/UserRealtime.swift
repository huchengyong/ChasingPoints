import Foundation

private struct UserWSTicketRequest: Encodable {}

/// `UserWebSocket`：用户级通道，只带一次性 `ticket`。
@MainActor
final class UserRealtime: RealtimeConnection {
    private(set) var authGeneration: Int?

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
        self.ticketIssuer = ticketIssuer ?? { [session] _ in
            let response: WSTicketResponse = try await session.authorizedPost(
                "/api/user/ws-ticket",
                body: UserWSTicketRequest()
            )
            return response.ticket
        }
    }

    /// 认证代次变化（切号）时先断开旧连接，再按新代次重连。
    func connect(authGeneration requestedGeneration: Int? = nil) async throws {
        if let requestedGeneration {
            if let current = authGeneration, current != requestedGeneration {
                disconnect()
            }
            authGeneration = requestedGeneration
        }
        try await connectTarget(RealtimeTarget(key: "user"))
    }

    override func socketPath(for target: RealtimeTarget) -> String {
        "/api/user/ws"
    }

    override func queryItems(for target: RealtimeTarget, ticket: String?) -> [URLQueryItem] {
        guard let ticket else { return [] }
        return [URLQueryItem(name: "ticket", value: ticket)]
    }

    override func disconnect() {
        super.disconnect()
        authGeneration = nil
    }
}
