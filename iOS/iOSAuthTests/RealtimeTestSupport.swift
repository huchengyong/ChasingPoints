import Foundation
import XCTest
@testable import iOS

/// 可控时钟：记录每次请求的等待时长，由测试决定何时放行，不真的等待。
@MainActor
final class TestRealtimeClock {
    private var waiters: [UUID: CheckedContinuation<Void, Error>] = [:]
    private(set) var requestedDelays: [TimeInterval] = []

    var pendingCount: Int { waiters.count }

    func sleep(_ seconds: TimeInterval) async throws {
        requestedDelays.append(seconds)
        let id = UUID()
        try await withTaskCancellationHandler {
            try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
                waiters[id] = continuation
            }
        } onCancel: {
            Task { @MainActor in self.cancel(id) }
        }
    }

    private func cancel(_ id: UUID) {
        waiters.removeValue(forKey: id)?.resume(throwing: CancellationError())
    }

    func releaseAll() {
        let pending = waiters
        waiters.removeAll()
        for continuation in pending.values { continuation.resume() }
    }
}

final class FakeRealtimeSocket: RealtimeSocket {
    private var pendingReceive: CheckedContinuation<String, Error>?
    private var buffered: [Result<String, Error>] = []
    private(set) var sentTexts: [String] = []
    private(set) var closeCount = 0
    private var closed = false

    func send(_ text: String) async throws {
        sentTexts.append(text)
    }

    func receive() async throws -> String {
        if !buffered.isEmpty {
            return try buffered.removeFirst().get()
        }
        if closed { throw URLError(.cancelled) }
        return try await withCheckedThrowingContinuation { continuation in
            pendingReceive = continuation
        }
    }

    func close() {
        closeCount += 1
        closed = true
        pendingReceive?.resume(throwing: URLError(.cancelled))
        pendingReceive = nil
    }

    func emit(_ text: String) {
        if let pendingReceive {
            self.pendingReceive = nil
            pendingReceive.resume(returning: text)
        } else {
            buffered.append(.success(text))
        }
    }

    /// 模拟服务端断开。
    func emitClose() {
        closed = true
        if let pendingReceive {
            self.pendingReceive = nil
            pendingReceive.resume(throwing: URLError(.networkConnectionLost))
        }
    }
}

final class FakeRealtimeTransport: RealtimeTransport {
    private(set) var urls: [URL] = []
    private(set) var sockets: [FakeRealtimeSocket] = []
    var connectError: Error?
    var gate: (() async -> Void)?

    func connect(url: URL) async throws -> RealtimeSocket {
        urls.append(url)
        if let gate { await gate() }
        if let connectError { throw connectError }
        let socket = FakeRealtimeSocket()
        sockets.append(socket)
        return socket
    }
}

/// 让首个连接停在握手中，用于验证同目标的在途连接被共享。
@MainActor
final class FakeConnectGate {
    private var waiters: [CheckedContinuation<Void, Never>] = []

    func wait() async {
        await withCheckedContinuation { waiters.append($0) }
    }

    func release() {
        let pending = waiters
        waiters.removeAll()
        for continuation in pending { continuation.resume() }
    }
}

/// 只验证基类行为的简单通道。
@MainActor
final class TestRealtimeConnection: RealtimeConnection {
    var path = "/api/test/ws"
    var extraQueryItems: [URLQueryItem] = []
    private(set) var connectedCount = 0
    private(set) var reuseCount = 0

    override func socketPath(for target: RealtimeTarget) -> String {
        path
    }

    override func queryItems(for target: RealtimeTarget, ticket: String?) -> [URLQueryItem] {
        guard let ticket else { return extraQueryItems }
        return extraQueryItems + [URLQueryItem(name: "ticket", value: ticket)]
    }

    override func onConnected() async {
        connectedCount += 1
    }

    override func onReuse() async {
        reuseCount += 1
    }
}
