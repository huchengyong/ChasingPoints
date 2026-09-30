import Foundation
import XCTest
@testable import iOS

/// 使用独立 Keychain service 的实际读写测试，不触碰开发者日常会话。
@MainActor
final class KeychainSessionTests: XCTestCase {
    private var service = ""
    private var store: KeychainCredentialStore!

    override func setUp() {
        super.setUp()
        service = "ChasingPoints.Session.tests.\(UUID().uuidString)"
        store = KeychainCredentialStore(service: service)
    }

    override func tearDown() {
        store.delete()
        super.tearDown()
    }

    func testKeychainRoundTripAndDelete() throws {
        let credentials = AuthFixtures.credentials()
        XCTAssertNil(store.load())

        try store.save(credentials)
        XCTAssertEqual(store.load(), credentials)

        let updated = Credentials(accessToken: "updated-access", refreshToken: "updated-refresh")
        try store.save(updated)
        XCTAssertEqual(store.load(), updated)

        store.delete()
        XCTAssertNil(store.load())
    }

    func testNewSessionStoreRestoresSessionFromKeychain() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        try store.save(AuthFixtures.credentials())

        let session = makeSessionStore(server, credentialStore: KeychainCredentialStore(service: service))
        await session.restore()

        XCTAssertEqual(session.user?.id, 1)
        XCTAssertEqual(server.requestCount("/api/user/info"), 1)
    }

    func testRefreshPersistsCredentialsForNextLaunch() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info") { request in
            request.bearerToken == AuthFixtures.oldAccess
                ? .expiredAccessToken
                : .json(AuthFixtures.userInfoResponse())
        }
        server.stub("POST", "/api/auth/refresh-token", .json(AuthFixtures.refreshResponse()))
        try store.save(AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: KeychainCredentialStore(service: service))

        let _: TestUserInfoResponse = try await session.authorizedRequest("/api/user/info")

        XCTAssertEqual(
            store.load(),
            Credentials(accessToken: AuthFixtures.newAccess, refreshToken: AuthFixtures.newRefresh)
        )

        // 模拟杀进程重启：新 SessionStore 从 Keychain 读取刷新后的凭据并恢复。
        let restarted = MockHTTPServer()
        restarted.stub("GET", "/api/user/info") { request in
            XCTAssertEqual(request.bearerToken, AuthFixtures.newAccess)
            return .json(AuthFixtures.userInfoResponse())
        }
        let nextSession = makeSessionStore(restarted, credentialStore: KeychainCredentialStore(service: service))
        await nextSession.restore()

        XCTAssertEqual(nextSession.user?.id, 1)
        XCTAssertEqual(restarted.requestCount("/api/auth/refresh-token"), 0)
    }

    func testLogoutDeletesKeychainSoRestartStaysGuest() async throws {
        let server = MockHTTPServer()
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse()))
        let session = makeSessionStore(server, credentialStore: store)

        try await session.login(phone: AuthFixtures.phone, code: "123456")
        XCTAssertNotNil(store.load())

        session.logout()
        XCTAssertNil(store.load())

        let restarted = MockHTTPServer()
        let nextSession = makeSessionStore(restarted, credentialStore: KeychainCredentialStore(service: service))
        await nextSession.restore()

        XCTAssertNil(nextSession.user)
        XCTAssertEqual(restarted.requests.count, 0)
    }
}
