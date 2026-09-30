import Foundation
import Combine
import Security

struct UserInfo: Decodable {
    let id: Int64
    let phone: String
    let nickname: String
    let avatar: String
    let status: Int
    let createdAt: String
}

struct Credentials: Codable, Equatable {
    let accessToken: String
    let refreshToken: String
}

protocol CredentialStoring {
    func load() -> Credentials?
    func save(_ credentials: Credentials) throws
    func delete()
}

struct KeychainCredentialStore: CredentialStoring {
    let service: String

    init(service: String = AppConfiguration.credentialService) {
        self.service = service
    }

    private var query: [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: "current",
        ]
    }

    func load() -> Credentials? {
        var result: CFTypeRef?
        let status = SecItemCopyMatching(
            query.merging([kSecReturnData as String: true, kSecMatchLimit as String: kSecMatchLimitOne]) { _, new in new } as CFDictionary,
            &result
        )
        guard status == errSecSuccess, let data = result as? Data else { return nil }
        return try? JSONDecoder().decode(Credentials.self, from: data)
    }

    func save(_ credentials: Credentials) throws {
        let data = try JSONEncoder().encode(credentials)
        let addQuery = query.merging([
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
        ]) { _, new in new }
        var status = SecItemAdd(addQuery as CFDictionary, nil)
        if status == errSecDuplicateItem {
            status = SecItemUpdate(query as CFDictionary, [kSecValueData as String: data] as CFDictionary)
        }
        guard status == errSecSuccess else {
            throw APIError(message: "无法安全保存登录状态", kind: .invalidResponse)
        }
    }

    func delete() {
        SecItemDelete(query as CFDictionary)
    }
}

private struct SendSMSRequest: Encodable {
    let phone: String
    let scene = "login"
}

private struct SendSMSResponse: Decodable {
    let message: String
}

private struct LoginRequest: Encodable {
    let phone: String
    let smsCode: String
}

private struct LoginResponse: Decodable {
    let accessToken: String
    let refreshToken: String
    let userInfo: UserInfo?
}

private struct RefreshRequest: Encodable {
    let refreshToken: String
}

private struct RefreshResponse: Decodable {
    let accessToken: String
    let refreshToken: String
}

private struct UserInfoResponse: Decodable {
    let userInfo: UserInfo?
}

private enum SessionError: LocalizedError {
    case signedOut
    case changed
    case invalidResponse

    var errorDescription: String? {
        switch self {
        case .signedOut, .changed: "登录状态已变更"
        case .invalidResponse: "服务器返回的登录信息不完整"
        }
    }
}

@MainActor
final class SessionStore: ObservableObject {
    @Published private(set) var user: UserInfo?
    @Published private(set) var isRestoring = false
    @Published private(set) var restoreError: String?
    /// 被动会话失效提示；每个认证代次最多设置一次，由界面确认后清除。
    @Published private(set) var sessionExpiredNotice: String?

    private(set) var sessionInvalidationCount = 0

    private let api: APIClient
    private let credentialStore: CredentialStoring
    private var credentials: Credentials?
    private var authGeneration = 0
    private var refreshTask: Task<Credentials, Error>?
    private var didAttemptRestore = false

    init(api: APIClient? = nil, credentialStore: CredentialStoring? = nil) {
        self.api = api ?? APIClient()
        self.credentialStore = credentialStore ?? KeychainCredentialStore()
        credentials = self.credentialStore.load()
    }

    func sendSMS(phone: String) async throws {
        let _: SendSMSResponse = try await api.post("/api/auth/send-sms", body: SendSMSRequest(phone: phone))
    }

    func login(phone: String, code: String) async throws {
        let startedGeneration = authGeneration
        let response: LoginResponse = try await api.post(
            "/api/auth/login",
            body: LoginRequest(phone: phone, smsCode: code)
        )
        guard startedGeneration == authGeneration else { throw SessionError.changed }
        guard !response.accessToken.isEmpty, !response.refreshToken.isEmpty,
              let userInfo = response.userInfo, userInfo.status == 1 else {
            throw SessionError.invalidResponse
        }

        // 先持久化再发布会话：写入失败时保持此前的会话状态。
        let next = Credentials(accessToken: response.accessToken, refreshToken: response.refreshToken)
        try credentialStore.save(next)
        authGeneration += 1
        refreshTask?.cancel()
        refreshTask = nil
        credentials = next
        user = userInfo
        restoreError = nil
        sessionExpiredNotice = nil
    }

    func logout() {
        clearSession()
        sessionExpiredNotice = nil
    }

    func acknowledgeSessionExpired() {
        sessionExpiredNotice = nil
    }

    func restore() async {
        guard !didAttemptRestore else { return }
        didAttemptRestore = true
        await loadUser()
    }

    func retryRestore() async {
        await loadUser()
    }

    // 后续页面统一走此入口，401 时刷新令牌并且只重放一次。
    func authorizedRequest<Response: Decodable>(
        _ path: String,
        method: HTTPMethod = .get,
        body: Data? = nil
    ) async throws -> Response {
        guard let original = credentials else { throw SessionError.signedOut }
        let generation = authGeneration

        do {
            let value: Response = try await api.request(
                path, method: method, body: body, bearerToken: original.accessToken
            )
            try ensureCurrent(generation)
            return value
        } catch let error as APIError where error.isUnauthorized {
            try ensureCurrent(generation)
            if error.isSessionInvalid {
                invalidateSession()
                throw error
            }

            do {
                let current: Credentials
                if let latest = credentials, latest.refreshToken != original.refreshToken {
                    current = latest
                } else {
                    current = try await refresh(original, generation: generation)
                }
                try ensureCurrent(generation)
                let value: Response = try await api.request(
                    path, method: method, body: body, bearerToken: current.accessToken
                )
                try ensureCurrent(generation)
                return value
            } catch let retryError as APIError {
                guard generation == authGeneration else { throw SessionError.changed }
                // 刷新明确失效或重放后仍 401：清理当前会话，不再刷新或重放。
                if retryError.isSessionInvalid || retryError.isUnauthorized {
                    invalidateSession()
                }
                throw retryError
            }
        }
    }

    private func ensureCurrent(_ generation: Int) throws {
        guard generation == authGeneration else { throw SessionError.changed }
    }

    private func refresh(_ old: Credentials, generation: Int) async throws -> Credentials {
        if let refreshTask { return try await refreshTask.value }

        let task = Task<Credentials, Error> {
            let response: RefreshResponse = try await api.post(
                "/api/auth/refresh-token",
                body: RefreshRequest(refreshToken: old.refreshToken)
            )
            guard generation == authGeneration,
                  credentials?.refreshToken == old.refreshToken else { throw SessionError.changed }
            guard !response.accessToken.isEmpty, !response.refreshToken.isEmpty else {
                throw SessionError.invalidResponse
            }

            let updated = Credentials(
                accessToken: response.accessToken,
                refreshToken: response.refreshToken
            )
            try credentialStore.save(updated)
            credentials = updated
            return updated
        }
        refreshTask = task
        defer {
            if generation == authGeneration { refreshTask = nil }
        }
        return try await task.value
    }

    private func loadUser() async {
        guard credentials != nil, user == nil, !isRestoring else { return }
        let generation = authGeneration
        isRestoring = true
        restoreError = nil
        defer { isRestoring = false }

        do {
            let response: UserInfoResponse = try await authorizedRequest("/api/user/info")
            guard let userInfo = response.userInfo else { throw SessionError.invalidResponse }
            guard generation == authGeneration else { return }
            user = userInfo
        } catch SessionError.changed {
            return
        } catch {
            // 暂时性故障保留凭据，仅展示可重试的错误；明确失效已由授权入口清理。
            if generation == authGeneration, credentials != nil {
                restoreError = error.localizedDescription
            }
        }
    }

    private func clearSession() {
        authGeneration += 1
        refreshTask?.cancel()
        refreshTask = nil
        credentials = nil
        user = nil
        restoreError = nil
        credentialStore.delete()
    }

    /// 仅用于明确认证失效：清理当前代次，并保证每个代次只提示一次。
    private func invalidateSession() {
        guard user != nil || credentials != nil else { return }
        clearSession()
        sessionInvalidationCount += 1
        sessionExpiredNotice = "登录状态已失效，请重新登录"
    }
}
