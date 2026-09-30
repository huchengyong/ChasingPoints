import Foundation

struct RecordedRequest {
    let method: String
    let path: String
    let bearerToken: String?
    let body: Data

    var bodyJSON: [String: Any] {
        (try? JSONSerialization.jsonObject(with: body)) as? [String: Any] ?? [:]
    }
}

/// 可控 HTTP 测试替身：按方法/路径匹配响应并记录真实发出的请求。
/// 只替换 URLSession 传输层，不触达真实短信接口或开发者凭据。
final class MockHTTPServer: @unchecked Sendable {
    struct Stub {
        var status: Int = 200
        var body: Data = Data()
        var headers: [String: String] = ["Content-Type": "application/json"]
        var error: URLError?

        static func json(_ object: [String: Any], status: Int = 200) -> Stub {
            let data = (try? JSONSerialization.data(withJSONObject: object)) ?? Data()
            return Stub(status: status, body: data)
        }

        static func raw(_ text: String, status: Int, contentType: String = "text/plain; charset=utf-8") -> Stub {
            Stub(status: status, body: Data(text.utf8), headers: ["Content-Type": contentType])
        }

        /// go-zero 默认 JWT 中间件返回的 401 空体。
        static let expiredAccessToken = Stub(status: 401, body: Data(), headers: [:])

        static func failure(_ code: URLError.Code) -> Stub {
            Stub(error: URLError(code))
        }
    }

    private struct Rule {
        let method: String?
        let path: String
        let handler: (RecordedRequest) -> Stub
    }

    private let lock = NSLock()
    private var rules: [Rule] = []
    private var records: [RecordedRequest] = []

    func stub(_ method: String? = nil, _ path: String, handler: @escaping (RecordedRequest) -> Stub) {
        lock.lock()
        rules.append(Rule(method: method, path: path, handler: handler))
        lock.unlock()
    }

    func stub(_ method: String? = nil, _ path: String, _ stub: Stub) {
        self.stub(method, path) { _ in stub }
    }

    func makeSession() -> URLSession {
        MockURLProtocol.server = self
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [MockURLProtocol.self]
        configuration.urlCache = nil
        configuration.requestCachePolicy = .reloadIgnoringLocalCacheData
        return URLSession(configuration: configuration)
    }

    var requests: [RecordedRequest] {
        lock.lock()
        defer { lock.unlock() }
        return records
    }

    func requests(_ path: String) -> [RecordedRequest] {
        requests.filter { $0.path == path }
    }

    func requestCount(_ path: String) -> Int {
        requests(path).count
    }

    fileprivate func handle(_ request: URLRequest) -> Stub {
        let recorded = RecordedRequest(
            method: request.httpMethod ?? "",
            path: request.url?.path ?? "",
            bearerToken: Self.bearerToken(in: request),
            body: Self.readBody(request)
        )
        lock.lock()
        records.append(recorded)
        let rule = rules.last { ($0.method == nil || $0.method == recorded.method) && $0.path == recorded.path }
        lock.unlock()
        guard let rule else {
            return .json(["success": false, "message": "未配置的测试请求 \(recorded.path)"], status: 500)
        }
        return rule.handler(recorded)
    }

    private static func bearerToken(in request: URLRequest) -> String? {
        guard let value = request.value(forHTTPHeaderField: "Authorization"),
              value.hasPrefix("Bearer ") else { return nil }
        return String(value.dropFirst("Bearer ".count))
    }

    /// URLSession 会把 httpBody 转成 httpBodyStream 后才交给 URLProtocol。
    private static func readBody(_ request: URLRequest) -> Data {
        if let body = request.httpBody { return body }
        guard let stream = request.httpBodyStream else { return Data() }
        stream.open()
        defer { stream.close() }
        var data = Data()
        var buffer = [UInt8](repeating: 0, count: 1024)
        while stream.hasBytesAvailable {
            let count = stream.read(&buffer, maxLength: buffer.count)
            if count <= 0 { break }
            data.append(buffer, count: count)
        }
        return data
    }
}

final class MockURLProtocol: URLProtocol {
    nonisolated(unsafe) static var server: MockHTTPServer?

    private let stopped = LockedFlag()

    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

    override func startLoading() {
        guard let server = Self.server else {
            client?.urlProtocol(self, didFailWithError: URLError(.unsupportedURL))
            return
        }
        let request = self.request
        // 异步投递，避免 URLProtocol 的串行加载路径被测试屏障阻塞。
        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            guard let self else { return }
            let stub = server.handle(request)
            guard !self.stopped.value else { return }
            if let error = stub.error {
                self.client?.urlProtocol(self, didFailWithError: error)
                return
            }
            guard let url = request.url,
                  let response = HTTPURLResponse(
                      url: url,
                      statusCode: stub.status,
                      httpVersion: "HTTP/1.1",
                      headerFields: stub.headers
                  ) else {
                self.client?.urlProtocol(self, didFailWithError: URLError(.badServerResponse))
                return
            }
            self.client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            self.client?.urlProtocol(self, didLoad: stub.body)
            self.client?.urlProtocolDidFinishLoading(self)
        }
    }

    override func stopLoading() {
        stopped.set()
    }
}

final class LockedFlag: @unchecked Sendable {
    private let lock = NSLock()
    private var flag = false

    var value: Bool {
        lock.lock()
        defer { lock.unlock() }
        return flag
    }

    func set() {
        lock.lock()
        flag = true
        lock.unlock()
    }
}
