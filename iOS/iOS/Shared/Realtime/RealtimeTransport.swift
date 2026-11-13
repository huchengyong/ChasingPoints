import Foundation

/// WebSocket 连接的最小抽象，便于在测试里注入替身。
protocol RealtimeSocket: AnyObject {
    func send(_ text: String) async throws
    func receive() async throws -> String
    func close()
}

protocol RealtimeTransport {
    func connect(url: URL) async throws -> RealtimeSocket
}

/// 系统实现：URLSessionWebSocketTask。
struct URLSessionRealtimeTransport: RealtimeTransport {
    private let session: URLSession

    init(session: URLSession = .shared) {
        self.session = session
    }

    func connect(url: URL) async throws -> RealtimeSocket {
        let task = session.webSocketTask(with: url)
        task.resume()
        let socket = URLSessionRealtimeSocket(task: task)
        do {
            // URLSessionWebSocketTask 没有 open 回调，用协议级 ping 确认握手成功。
            try await withTaskCancellationHandler {
                try await socket.waitForHandshake()
            } onCancel: {
                Task { @MainActor in socket.close() }
            }
        } catch {
            socket.close()
            throw error
        }
        return socket
    }
}

final class URLSessionRealtimeSocket: RealtimeSocket {
    private let task: URLSessionWebSocketTask

    init(task: URLSessionWebSocketTask) {
        self.task = task
    }

    func waitForHandshake() async throws {
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
            task.sendPing { error in
                if let error {
                    continuation.resume(throwing: error)
                } else {
                    continuation.resume()
                }
            }
        }
    }

    func send(_ text: String) async throws {
        try await task.send(.string(text))
    }

    func receive() async throws -> String {
        switch try await task.receive() {
        case .string(let text):
            return text
        case .data(let data):
            return String(decoding: data, as: UTF8.self)
        @unknown default:
            throw RealtimeError(message: "不支持的实时消息格式")
        }
    }

    func close() {
        task.cancel(with: .normalClosure, reason: nil)
    }
}
