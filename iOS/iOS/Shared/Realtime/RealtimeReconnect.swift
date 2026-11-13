import Foundation

/// 实时连接失败分类：与 `uniapp/utils/websocket-reconnect.js` 的返回值逐字对应。
enum WebSocketFailureCategory: String, Equatable {
    case sessionInvalid = "session-invalid"
    case permissionDenied = "permission-denied"
    case recoverable = "recoverable"
}

/// 失败信息的原生形态，替代 JS 里结构不固定的 error 对象。
struct RealtimeFailure {
    enum Kind: String {
        case session
        case forbidden
        case network
    }

    var category: Kind?
    var statusCode: Int?
    var reason: String?

    init(category: Kind? = nil, statusCode: Int? = nil, reason: String? = nil) {
        self.category = category
        self.statusCode = statusCode
        self.reason = reason
    }

    init(error: Error) {
        if let apiError = error as? APIError {
            category = apiError.statusCode == 403 ? .forbidden : nil
            if apiError.kind == .sessionInvalid { category = .session }
            statusCode = apiError.statusCode
            reason = apiError.reason
            return
        }
        if error is URLError {
            category = .network
        }
    }
}

struct ReconnectDecision: Equatable {
    enum Reason: String {
        case manualDisconnect = "manual-disconnect"
        case background
        case offline
        case sessionInvalid = "session-invalid"
        case permissionDenied = "permission-denied"
        case targetInactive = "target-inactive"
        case recoverable
    }

    let allow: Bool
    let reason: Reason
}

enum RealtimeReconnect {
    static let defaultBaseDelayMs: Double = 3000
    static let defaultMaxDelayMs: Double = 60000
    static let defaultJitterRatio: Double = 0.2

    static func classifyWebSocketFailure(_ failure: RealtimeFailure? = nil) -> WebSocketFailureCategory {
        let statusCode = failure?.statusCode ?? 0
        let category = failure?.category?.rawValue.lowercased() ?? ""
        let reason = (failure?.reason ?? "").uppercased()

        if category == "session" || statusCode == 401 || reason == "SESSION_INVALID" {
            return .sessionInvalid
        }
        if category == "forbidden" || statusCode == 403 || reason == "FORBIDDEN" {
            return .permissionDenied
        }
        return .recoverable
    }

    /// 指数退避 + 抖动，单位毫秒。随机数可注入，便于测试复现。
    static func calculateReconnectDelay(
        attempt: Int,
        baseDelayMs: Double = defaultBaseDelayMs,
        maxDelayMs: Double = defaultMaxDelayMs,
        jitterRatio: Double = defaultJitterRatio,
        random: () -> Double = { Double.random(in: 0..<1) }
    ) -> Int {
        let safeAttempt = max(1, attempt)
        let safeBase = max(0, baseDelayMs == 0 ? defaultBaseDelayMs : baseDelayMs)
        let safeMax = max(safeBase, maxDelayMs == 0 ? defaultMaxDelayMs : maxDelayMs)
        let capped = min(safeMax, safeBase * pow(2, Double(safeAttempt - 1)))
        let safeJitter = max(0, min(1, jitterRatio))
        let jitter = capped * safeJitter * (random() * 2 - 1)
        return Int(max(0, min(safeMax, (capped + jitter).rounded())))
    }

    static func shouldReconnectWebSocket(
        manualDisconnect: Bool = false,
        foreground: Bool = true,
        online: Bool = true,
        sessionInvalid: Bool = false,
        permissionDenied: Bool = false,
        targetActive: Bool = true,
        error: RealtimeFailure? = nil
    ) -> ReconnectDecision {
        let failure = classifyWebSocketFailure(error)
        if manualDisconnect { return ReconnectDecision(allow: false, reason: .manualDisconnect) }
        if !foreground { return ReconnectDecision(allow: false, reason: .background) }
        if !online { return ReconnectDecision(allow: false, reason: .offline) }
        if sessionInvalid || failure == .sessionInvalid {
            return ReconnectDecision(allow: false, reason: .sessionInvalid)
        }
        if permissionDenied || failure == .permissionDenied {
            return ReconnectDecision(allow: false, reason: .permissionDenied)
        }
        if !targetActive { return ReconnectDecision(allow: false, reason: .targetInactive) }
        return ReconnectDecision(allow: true, reason: .recoverable)
    }

    static func nextConnectionGeneration(_ currentGeneration: Int = 0) -> Int {
        max(0, currentGeneration) + 1
    }

    static func isCurrentConnectionGeneration(
        _ currentGeneration: Int,
        _ callbackGeneration: Int
    ) -> Bool {
        currentGeneration == callbackGeneration
    }
}
