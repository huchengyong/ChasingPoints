import Foundation

enum HTTPMethod: String {
    case get = "GET"
    case post = "POST"
    case put = "PUT"
    case delete = "DELETE"
}

struct APIError: LocalizedError {
    enum Kind: Equatable {
        case network
        case server
        case business
        case unauthorized
        case sessionInvalid
        case invalidResponse
        case canceled
    }

    let message: String
    let statusCode: Int?
    let reason: String?
    let kind: Kind

    init(message: String, statusCode: Int? = nil, reason: String? = nil, kind: Kind? = nil) {
        self.message = message
        self.statusCode = statusCode
        self.reason = reason
        self.kind = kind ?? Self.inferKind(statusCode: statusCode, reason: reason)
    }

    var errorDescription: String? { message }
    var isUnauthorized: Bool { statusCode == 401 }
    var isSessionInvalid: Bool { reason == "SESSION_INVALID" }

    private static func inferKind(statusCode: Int?, reason: String?) -> Kind {
        if reason == "SESSION_INVALID" { return .sessionInvalid }
        if statusCode == 401 { return .unauthorized }
        if let statusCode, statusCode >= 500 { return .server }
        if statusCode != nil { return .business }
        return .network
    }
}

struct APIClient {
    private let baseURL: URL
    private let session: URLSession

    init(baseURL: URL = AppConfiguration.apiBaseURL, session: URLSession = .shared) {
        self.baseURL = baseURL
        self.session = session
    }

    func get<Response: Decodable>(_ path: String, bearerToken: String? = nil) async throws -> Response {
        try await request(path, method: .get, bearerToken: bearerToken)
    }

    func post<Body: Encodable, Response: Decodable>(
        _ path: String,
        body: Body,
        bearerToken: String? = nil
    ) async throws -> Response {
        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase
        return try await request(path, method: .post, body: encoder.encode(body), bearerToken: bearerToken)
    }

    func request<Response: Decodable>(
        _ path: String,
        method: HTTPMethod,
        body: Data? = nil,
        bearerToken: String? = nil
    ) async throws -> Response {
        guard let url = URL(string: path, relativeTo: baseURL)?.absoluteURL else {
            throw APIError(message: "请求地址无效", statusCode: nil, reason: nil)
        }

        var request = URLRequest(url: url, cachePolicy: .reloadIgnoringLocalCacheData, timeoutInterval: 30)
        request.httpMethod = method.rawValue
        request.httpBody = body
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let bearerToken {
            request.setValue("Bearer \(bearerToken)", forHTTPHeaderField: "Authorization")
        }

        let (data, rawResponse): (Data, URLResponse)
        do {
            (data, rawResponse) = try await session.data(for: request)
        } catch let urlError as URLError where urlError.code == .cancelled {
            throw APIError(message: "请求已取消", kind: .canceled)
        } catch {
            throw APIError(message: "网络连接失败，请检查网络后重试", kind: .network)
        }
        guard let response = rawResponse as? HTTPURLResponse else {
            throw APIError(message: "服务器响应无效", kind: .invalidResponse)
        }

        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        let status = try? decoder.decode(ResponseStatus.self, from: data)

        guard (200..<300).contains(response.statusCode) else {
            throw APIError(
                message: Self.failureMessage(status: status, data: data, statusCode: response.statusCode, path: url.path),
                statusCode: response.statusCode,
                reason: status?.reason
            )
        }
        if let status, Self.isBusinessFailure(status) {
            throw APIError(
                message: Self.failureMessage(status: status, data: data, statusCode: response.statusCode, path: url.path),
                statusCode: response.statusCode,
                reason: status.reason
            )
        }

        let payload: Data
        if let object = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any],
           let nested = object["data"] {
            payload = try JSONSerialization.data(withJSONObject: nested, options: .fragmentsAllowed)
        } else {
            payload = data
        }
        do {
            return try decoder.decode(Response.self, from: payload)
        } catch {
            throw APIError(message: "服务器响应格式错误", statusCode: response.statusCode, kind: .invalidResponse)
        }
    }

    private static func isBusinessFailure(_ status: ResponseStatus) -> Bool {
        if let success = status.success { return !success }
        if let code = status.code { return code != 0 && status.accepted == nil }
        return false
    }

    private static func failureMessage(status: ResponseStatus?, data: Data, statusCode: Int, path: String) -> String {
        if statusCode >= 500 { return genericMessage(for: statusCode) }
        if ["/api/auth/send-sms", "/api/auth/login", "/api/auth/refresh-token", "/api/user/info"].contains(path) {
            let message = status?.message ?? status?.msg ?? String(data: data, encoding: .utf8)
            return authenticationMessage(message) ?? genericMessage(for: statusCode)
        }
        if let message = status?.message, !message.isEmpty { return message }
        if let msg = status?.msg, !msg.isEmpty { return msg }
        return genericMessage(for: statusCode)
    }

    /// 仅展示现有认证 logic 和短信 CodeManager 已定义的用户提示。
    private static func authenticationMessage(_ message: String?) -> String? {
        guard let text = message?.trimmingCharacters(in: .whitespacesAndNewlines) else { return nil }
        switch text {
        case "手机号格式不正确", "验证码错误", "验证码已过期或不存在",
             "登录失败，请稍后重试", "注册失败", "系统错误", "系统错误，请稍后重试",
             "请求参数错误", "不支持的验证码场景", "发送短信失败，请稍后重试",
             "refresh token 不能为空", "登录状态已失效", "登录状态已失效，请重新登录":
            return text
        default:
            return text.range(of: #"^请等待[0-9]+秒后再试$"#, options: .regularExpression) != nil ? text : nil
        }
    }

    private static func genericMessage(for statusCode: Int) -> String {
        switch statusCode {
        case 401: "登录状态已失效，请重新登录"
        case 403: "没有权限执行此操作"
        case 404: "请求的资源不存在"
        case 429: "操作过于频繁，请稍后重试"
        case 500...599: "服务暂时不可用，请稍后重试"
        default: "请求失败，请稍后重试"
        }
    }
}

private struct ResponseStatus: Decodable {
    let success: Bool?
    let code: Int?
    let accepted: Bool?
    let message: String?
    let msg: String?
    let reason: String?
}

enum AppConfiguration {
    #if DEBUG
    static let apiBaseURL = URL(string: "https://dev-api.kekemate.cn")!
    static let credentialService = "ChasingPoints.Session.development"
    #else
    static let apiBaseURL = URL(string: "https://api.zhuifen.cn")!
    static let credentialService = "ChasingPoints.Session.production"
    #endif
}
