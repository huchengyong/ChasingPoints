import Foundation

/// 票据签发方：返回 nil 表示该目标允许匿名连接、无需票据。
typealias RealtimeTicketIssuer = @MainActor (RealtimeTarget) async throws -> String?

/// 可注入的睡眠，测试用可控时钟替换，避免真实等待。
typealias RealtimeSleep = (TimeInterval) async throws -> Void

struct RealtimeTarget: Equatable {
    let key: String
    var matchId: Int64?
    var allowAnonymous: Bool

    init(key: String, matchId: Int64? = nil, allowAnonymous: Bool = false) {
        self.key = key
        self.matchId = matchId
        self.allowAnonymous = allowAnonymous
    }
}

struct RealtimeConnectOptions {
    var force = false
    var reconnect = false
    var ticketIssuer: RealtimeTicketIssuer?
}

struct RealtimeError: LocalizedError {
    let message: String
    var errorDescription: String? { message }
}

/// `uniapp/utils/websocket.js` 基类 `ManagedWebSocket` 的原生实现。
@MainActor
class RealtimeConnection {
    enum State: String {
        case idle
        case connecting
        case open
        case reconnecting
        case closed
        case error
    }

    static let heartbeatInterval: TimeInterval = 30

    private(set) var state: State = .idle
    private(set) var reconnectAttempts = 0
    private(set) var isConnecting = false
    private(set) var manualDisconnect = false
    private(set) var connectionGeneration = 0
    private(set) var target: RealtimeTarget?
    private(set) var targetActive = false
    private(set) var foreground = true
    private(set) var online = true
    private(set) var sessionInvalid = false
    private(set) var permissionDenied = false

    var ticketIssuer: RealtimeTicketIssuer?
    var sessionInvalidHandler: ((Data?) -> Void)?

    private let transport: RealtimeTransport
    private let wsBaseURL: URL
    private let baseReconnectDelayMs: Double
    private let random: () -> Double
    private let sleep: RealtimeSleep

    private var socket: RealtimeSocket?
    private var pendingConnect: Task<Void, Error>?
    private var heartbeatTask: Task<Void, Never>?
    private var reconnectTask: Task<Void, Never>?
    private var sessionInvalidNotified = false
    private var messageHandlers: [String: [UUID: (RealtimeMessage) -> Void]] = [:]

    init(
        transport: RealtimeTransport? = nil,
        wsBaseURL: URL? = nil,
        baseReconnectDelayMs: Double? = nil,
        random: @escaping () -> Double = { Double.random(in: 0..<1) },
        sleep: @escaping RealtimeSleep = { try await Task.sleep(for: .seconds($0)) }
    ) {
        self.transport = transport ?? URLSessionRealtimeTransport()
        self.wsBaseURL = wsBaseURL ?? AppConfiguration.wsBaseURL
        self.baseReconnectDelayMs = baseReconnectDelayMs ?? RealtimeReconnect.defaultBaseDelayMs
        self.random = random
        self.sleep = sleep
    }

    var isConnected: Bool {
        state == .open && socket != nil
    }

    // MARK: - 连接

    /// 同一目标已连接时复用并触发 `onReuse`；连接中则共享同一个在途连接。
    func connectTarget(_ target: RealtimeTarget, options: RealtimeConnectOptions? = nil) async throws {
        let options = options ?? RealtimeConnectOptions()
        let sameTarget = isSameTarget(target)

        if socket != nil, state == .open, sameTarget {
            await onReuse()
            return
        }
        if isConnecting, sameTarget, let pendingConnect {
            try await pendingConnect.value
            return
        }
        if isConnecting || socket != nil {
            cancelConnection(clearTarget: false, state: .closed)
        }

        if !sameTarget || options.force {
            sessionInvalid = false
            permissionDenied = false
            sessionInvalidNotified = false
        }
        if !options.reconnect {
            manualDisconnect = false
        }

        self.target = target
        targetActive = true
        let decision = reconnectDecision()
        guard decision.allow else {
            state = .closed
            throw RealtimeError(message: "实时连接暂不可恢复：" + decision.reason.rawValue)
        }

        clearReconnectTimer()
        stopHeartbeat()
        isConnecting = true
        state = reconnectAttempts > 0 ? .reconnecting : .connecting
        connectionGeneration = RealtimeReconnect.nextConnectionGeneration(connectionGeneration)
        let generation = connectionGeneration

        let task = Task { [weak self] in
            guard let self else { return }
            try await self.openConnection(target: target, generation: generation, issuer: options.ticketIssuer)
        }
        pendingConnect = task
        try await task.value
    }

    private func openConnection(
        target: RealtimeTarget,
        generation: Int,
        issuer: RealtimeTicketIssuer?
    ) async throws {
        do {
            let ticket = try await resolveTicket(target: target, issuer: issuer)
            guard isCurrentGeneration(generation), !manualDisconnect else {
                throw RealtimeError(message: "实时连接已替换")
            }

            let url = try makeSocketURL(target: target, ticket: ticket)
            let newSocket = try await transport.connect(url: url)
            guard isCurrentGeneration(generation), !manualDisconnect else {
                newSocket.close()
                throw RealtimeError(message: "实时连接已替换")
            }

            socket = newSocket
            isConnecting = false
            state = .open
            reconnectAttempts = 0
            startHeartbeat()
            await onConnected()
            startReceiveLoop(socket: newSocket, generation: generation)
            if isCurrentGeneration(generation) {
                pendingConnect = nil
            }
        } catch {
            if isCurrentGeneration(generation) {
                pendingConnect = nil
                handleConnectionFailure(error, generation: generation)
            }
            throw error
        }
    }

    private func resolveTicket(target: RealtimeTarget, issuer: RealtimeTicketIssuer?) async throws -> String? {
        guard let issuer = issuer ?? ticketIssuer else {
            throw RealtimeError(message: "实时连接票据服务未初始化")
        }
        let raw = try await issuer(target)
        let ticket = raw?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if ticket.isEmpty {
            if target.allowAnonymous { return nil }
            throw RealtimeError(message: "实时连接票据无效")
        }
        return ticket
    }

    func socketPath(for target: RealtimeTarget) -> String { "" }

    func queryItems(for target: RealtimeTarget, ticket: String?) -> [URLQueryItem] { [] }

    private func makeSocketURL(target: RealtimeTarget, ticket: String?) throws -> URL {
        guard var components = URLComponents(url: wsBaseURL, resolvingAgainstBaseURL: false) else {
            throw RealtimeError(message: "实时连接地址无效")
        }
        components.path = (components.path) + socketPath(for: target)
        components.queryItems = queryItems(for: target, ticket: ticket)
        guard let url = components.url else {
            throw RealtimeError(message: "实时连接地址无效")
        }
        return url
    }

    // MARK: - 收消息

    private func startReceiveLoop(socket: RealtimeSocket, generation: Int) {
        Task { [weak self] in
            while true {
                let text: String
                do {
                    text = try await socket.receive()
                } catch {
                    self?.handleSocketClosed(socket: socket, generation: generation)
                    return
                }
                guard let self, !Task.isCancelled else { return }
                guard self.isCurrentSocket(socket, generation: generation) else { return }
                self.handleMessage(text)
            }
        }
    }

    private func handleMessage(_ text: String) {
        guard let message = RealtimeMessage(text: text) else { return }

        if message.type == WSMessageType.sessionInvalid,
           let payload = message.payload,
           let signal = try? JSONDecoder().decode(SessionInvalidSignal.self, from: payload),
           signal.reason == "SESSION_INVALID" {
            sessionInvalid = true
            clearReconnectTimer()
            stopHeartbeat()
            state = .closed
            notifySessionInvalid(payload: payload)
        }

        if let handlers = messageHandlers[message.type] {
            for handler in Array(handlers.values) { handler(message) }
        }
        if let handlers = messageHandlers[WSMessageType.any] {
            for handler in Array(handlers.values) { handler(message) }
        }
    }

    @discardableResult
    func on(_ type: String, _ handler: @escaping (RealtimeMessage) -> Void) -> UUID {
        let id = UUID()
        messageHandlers[type, default: [:]][id] = handler
        return id
    }

    func off(_ type: String, _ id: UUID) {
        messageHandlers[type]?[id] = nil
    }

    func off(_ id: UUID) {
        for type in Array(messageHandlers.keys) {
            messageHandlers[type]?[id] = nil
        }
    }

    // MARK: - 发消息

    /// 未连接时静默丢弃，与 JS 行为一致。
    func send(_ message: [String: Any]) async {
        guard let socket, state == .open else { return }
        guard let data = try? JSONSerialization.data(withJSONObject: message),
              let text = String(data: data, encoding: .utf8) else { return }
        try? await socket.send(text)
    }

    func send(type: String) async {
        await send(["type": type])
    }

    // MARK: - 心跳

    private func startHeartbeat() {
        stopHeartbeat()
        guard let socket else { return }
        let generation = connectionGeneration
        heartbeatTask = Task { [weak self] in
            guard let self else { return }
            while !Task.isCancelled {
                do {
                    try await self.sleep(Self.heartbeatInterval)
                } catch {
                    return
                }
                guard !Task.isCancelled else { return }
                guard self.isCurrentSocket(socket, generation: generation), self.state == .open else { return }
                await self.send(type: WSMessageType.ping)
            }
        }
    }

    private func stopHeartbeat() {
        heartbeatTask?.cancel()
        heartbeatTask = nil
    }

    // MARK: - 失败与重连

    private func handleConnectionFailure(_ error: Error, generation: Int) {
        guard isCurrentGeneration(generation) else { return }
        stopHeartbeat()
        isConnecting = false
        socket = nil
        state = .error

        let failure = RealtimeFailure(error: error)
        let category = RealtimeReconnect.classifyWebSocketFailure(failure)
        sessionInvalid = category == .sessionInvalid
        permissionDenied = category == .permissionDenied
        if category == .sessionInvalid {
            notifySessionInvalid(payload: nil)
        }
        scheduleReconnect(failure)
    }

    private func handleSocketClosed(socket: RealtimeSocket, generation: Int) {
        guard isCurrentSocket(socket, generation: generation) else { return }
        stopHeartbeat()
        isConnecting = false
        self.socket = nil
        state = .closed
        scheduleReconnect()
    }

    private func scheduleReconnect(_ failure: RealtimeFailure? = nil) {
        let decision = reconnectDecision(error: failure)
        guard decision.allow, reconnectTask == nil, !isConnecting, let target else { return }

        let generation = connectionGeneration
        reconnectAttempts += 1
        state = .reconnecting
        let delay = RealtimeReconnect.calculateReconnectDelay(
            attempt: reconnectAttempts,
            baseDelayMs: baseReconnectDelayMs,
            random: random
        )

        reconnectTask = Task { [weak self] in
            guard let self else { return }
            do {
                try await self.sleep(Double(delay) / 1000)
            } catch {
                return
            }
            guard !Task.isCancelled else { return }
            self.reconnectTask = nil
            guard self.isCurrentGeneration(generation), self.target == target else { return }
            guard self.reconnectDecision().allow else { return }
            try? await self.connectTarget(target, options: RealtimeConnectOptions(reconnect: true))
        }
    }

    private func clearReconnectTimer() {
        reconnectTask?.cancel()
        reconnectTask = nil
    }

    private func notifySessionInvalid(payload: Data?) {
        guard !sessionInvalidNotified else { return }
        sessionInvalidNotified = true
        sessionInvalidHandler?(payload)
    }

    func reconnectDecision(error: RealtimeFailure? = nil) -> ReconnectDecision {
        RealtimeReconnect.shouldReconnectWebSocket(
            manualDisconnect: manualDisconnect,
            foreground: foreground,
            online: online,
            sessionInvalid: sessionInvalid,
            permissionDenied: permissionDenied,
            targetActive: targetActive && target != nil,
            error: error
        )
    }

    private func isSameTarget(_ target: RealtimeTarget) -> Bool {
        self.target?.key == target.key
    }

    private func isCurrentGeneration(_ generation: Int) -> Bool {
        RealtimeReconnect.isCurrentConnectionGeneration(connectionGeneration, generation)
    }

    private func isCurrentSocket(_ socket: RealtimeSocket, generation: Int) -> Bool {
        self.socket === socket && isCurrentGeneration(generation)
    }

    // MARK: - 生命周期

    func setForeground(_ foreground: Bool) {
        guard self.foreground != foreground else { return }
        self.foreground = foreground
        if foreground {
            resumeConnection()
        } else {
            pauseConnection()
        }
    }

    func setNetworkOnline(_ online: Bool) {
        guard self.online != online else { return }
        self.online = online
        if online {
            resumeConnection()
        } else {
            pauseConnection()
        }
    }

    func setTargetActive(_ active: Bool) {
        guard targetActive != active else { return }
        targetActive = active
        if active {
            resumeConnection()
        } else {
            pauseConnection()
        }
    }

    func pauseConnection() {
        cancelConnection(clearTarget: false, state: .closed)
    }

    func resumeConnection() {
        guard let target, reconnectDecision().allow else { return }
        Task { [weak self] in
            guard let self else { return }
            try? await self.connectTarget(target, options: RealtimeConnectOptions(reconnect: true))
        }
    }

    func disconnect() {
        manualDisconnect = true
        targetActive = false
        reconnectAttempts = 0
        sessionInvalid = false
        permissionDenied = false
        sessionInvalidNotified = false
        cancelConnection(clearTarget: true, state: .idle)
    }

    private func cancelConnection(clearTarget: Bool, state: State) {
        let activeSocket = socket
        // 代次递增后，旧 socket 的迟到回调与消息一律失效。
        connectionGeneration = RealtimeReconnect.nextConnectionGeneration(connectionGeneration)
        socket = nil
        pendingConnect?.cancel()
        pendingConnect = nil
        clearReconnectTimer()
        stopHeartbeat()
        isConnecting = false
        self.state = state
        if clearTarget {
            target = nil
            targetActive = false
        }
        activeSocket?.close()
    }

    // MARK: - 子类扩展点

    func onConnected() async {}

    func onReuse() async {}
}

private struct SessionInvalidSignal: Decodable {
    let reason: String?
}

/// `POST /api/match/ws-ticket`、`POST /api/user/ws-ticket` 的响应。
struct WSTicketResponse: Decodable {
    let success: Bool?
    let ticket: String
}
