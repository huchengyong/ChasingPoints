import Foundation

enum HTTPMethod: String {
    case get = "GET"
    case post = "POST"
    case put = "PUT"
    case delete = "DELETE"
}

struct APIError: LocalizedError {
    let message: String
    let statusCode: Int?
    let reason: String?

    var errorDescription: String? { message }
    var isUnauthorized: Bool { statusCode == 401 }
    var isSessionInvalid: Bool { reason == "SESSION_INVALID" }
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

        let (data, rawResponse) = try await session.data(for: request)
        guard let response = rawResponse as? HTTPURLResponse else {
            throw APIError(message: "服务器响应无效", statusCode: nil, reason: nil)
        }

        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        let status = try? decoder.decode(ResponseStatus.self, from: data)
        let message = status?.message ?? status?.msg ?? "请求失败"

        guard (200..<300).contains(response.statusCode) else {
            throw APIError(message: message, statusCode: response.statusCode, reason: status?.reason)
        }
        if let status, status.success != true && status.code != 0 && status.accepted == nil {
            throw APIError(message: message, statusCode: response.statusCode, reason: status.reason)
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
            throw APIError(message: "服务器响应格式错误", statusCode: response.statusCode, reason: nil)
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
