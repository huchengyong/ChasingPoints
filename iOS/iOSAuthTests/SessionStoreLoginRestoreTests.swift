import Foundation
import XCTest
@testable import iOS

@MainActor
final class SessionStoreLoginTests: XCTestCase {
    private var server: MockHTTPServer!
    private var credentialStore: InMemoryCredentialStore!

    override func setUp() {
        super.setUp()
        server = MockHTTPServer()
        credentialStore = InMemoryCredentialStore()
    }

    func testSuccessfulLoginPersistsAndPublishesUser() async throws {
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse()))
        let session = makeSessionStore(server, credentialStore: credentialStore)

        try await session.login(phone: AuthFixtures.phone, code: "123456")

        XCTAssertEqual(session.user?.id, 1)
        XCTAssertEqual(session.user?.phone, AuthFixtures.maskedPhone)
        XCTAssertNil(session.sessionExpiredNotice)
        XCTAssertEqual(
            credentialStore.credentials,
            Credentials(accessToken: "login-access-token", refreshToken: "login-refresh-token")
        )
        XCTAssertEqual(credentialStore.saveCount, 1)

        let request = try XCTUnwrap(server.requests.first)
        XCTAssertEqual(request.method, "POST")
        XCTAssertEqual(request.path, "/api/auth/login")
        XCTAssertEqual(request.bodyJSON["phone"] as? String, AuthFixtures.phone)
        XCTAssertEqual(request.bodyJSON["sms_code"] as? String, "123456")
    }

    func testSendSMSRequestSerialization() async throws {
        server.stub("POST", "/api/auth/send-sms", .json(AuthFixtures.sendSMSResponse()))
        let session = makeSessionStore(server, credentialStore: credentialStore)

        try await session.sendSMS(phone: AuthFixtures.phone)

        let request = try XCTUnwrap(server.requests.first)
        XCTAssertEqual(request.method, "POST")
        XCTAssertEqual(request.bodyJSON["phone"] as? String, AuthFixtures.phone)
        XCTAssertEqual(request.bodyJSON["scene"] as? String, "login")
    }

    func testPlainTextLoginFailureKeepsSessionEmpty() async throws {
        server.stub("POST", "/api/auth/login", .raw("验证码错误\n", status: 400))
        let session = makeSessionStore(server, credentialStore: credentialStore)

        do {
            try await session.login(phone: AuthFixtures.phone, code: "111111")
            XCTFail("应当失败")
        } catch {
            assertAPIError(error, kind: .business, message: "验证码错误", statusCode: 400)
        }

        XCTAssertNil(session.user)
        XCTAssertNil(credentialStore.credentials)
        XCTAssertEqual(credentialStore.saveCount, 0)
    }

    func testIncompleteLoginResponseKeepsPreviousSession() async throws {
        let previous = AuthFixtures.credentials(access: "previous-access", refresh: "previous-refresh")
        credentialStore = InMemoryCredentialStore(credentials: previous)
        server.stub("POST", "/api/auth/login", .json([
            "success": true,
            "access_token": "partial-access",
            "refresh_token": "",
            "expires_in": 1,
            "user_info": AuthFixtures.userInfo(),
        ]))
        let session = makeSessionStore(server, credentialStore: credentialStore)

        do {
            try await session.login(phone: AuthFixtures.phone, code: "123456")
            XCTFail("应当失败")
        } catch {
            XCTAssertEqual(error.localizedDescription, "服务器返回的登录信息不完整")
        }

        XCTAssertNil(session.user)
        XCTAssertEqual(credentialStore.credentials, previous)
        XCTAssertEqual(credentialStore.saveCount, 0)
    }

    func testLoginWithoutUserInfoIsRejected() async throws {
        server.stub("POST", "/api/auth/login", .json([
            "success": true,
            "access_token": "a",
            "refresh_token": "r",
            "expires_in": 1,
            "user_info": NSNull(),
        ]))
        let session = makeSessionStore(server, credentialStore: credentialStore)

        do {
            try await session.login(phone: AuthFixtures.phone, code: "123456")
            XCTFail("应当失败")
        } catch {
            XCTAssertEqual(error.localizedDescription, "服务器返回的登录信息不完整")
        }

        XCTAssertNil(session.user)
        XCTAssertNil(credentialStore.credentials)
    }

    func testLoginWithDisabledUserIsRejected() async throws {
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse(
            user: AuthFixtures.userInfo(status: 0)
        )))
        let session = makeSessionStore(server, credentialStore: credentialStore)

        do {
            try await session.login(phone: AuthFixtures.phone, code: "123456")
            XCTFail("应当失败")
        } catch {
            XCTAssertEqual(error.localizedDescription, "服务器返回的登录信息不完整")
        }

        XCTAssertNil(session.user)
        XCTAssertNil(credentialStore.credentials)
        XCTAssertEqual(credentialStore.saveCount, 0)
    }

    func testCredentialSaveFailureDoesNotPublishHalfSession() async throws {
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse()))
        credentialStore.saveError = APIError(message: "无法安全保存登录状态", kind: .invalidResponse)
        let session = makeSessionStore(server, credentialStore: credentialStore)

        do {
            try await session.login(phone: AuthFixtures.phone, code: "123456")
            XCTFail("应当失败")
        } catch {
            XCTAssertEqual(error.localizedDescription, "无法安全保存登录状态")
        }

        XCTAssertNil(session.user)
        XCTAssertNil(credentialStore.credentials)
        XCTAssertEqual(credentialStore.saveCount, 1)
    }
}

@MainActor
final class SessionStoreRestoreTests: XCTestCase {
    func testRestoreWithoutCredentialsStaysGuestWithoutRequests() async {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore())

        await session.restore()

        XCTAssertNil(session.user)
        XCTAssertNil(session.restoreError)
        XCTAssertFalse(session.isRestoring)
        XCTAssertEqual(server.requests.count, 0)
    }

    func testRestoreVerifiesCredentialsBeforePublishingUser() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        await session.restore()

        XCTAssertEqual(session.user?.id, 1)
        XCTAssertNil(session.restoreError)
        XCTAssertEqual(server.requestCount("/api/user/info"), 1)
        XCTAssertEqual(server.requests.first?.bearerToken, AuthFixtures.oldAccess)
        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 0)
    }

    func testConcurrentRestoreVerifiesOnce() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        async let first: Void = session.restore()
        async let second: Void = session.restore()
        _ = await (first, second)

        XCTAssertEqual(server.requestCount("/api/user/info"), 1)
        XCTAssertEqual(session.user?.id, 1)
    }

    func testRestoreTemporaryFailureKeepsCredentialsAndRetrySucceeds() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info", .raw("内部服务错误", status: 503))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        await session.restore()

        XCTAssertNil(session.user)
        XCTAssertEqual(session.restoreError, "服务暂时不可用，请稍后重试")
        XCTAssertEqual(store.credentials, AuthFixtures.credentials())
        XCTAssertEqual(store.deleteCount, 0)
        XCTAssertEqual(session.sessionInvalidationCount, 0)
        XCTAssertNil(session.sessionExpiredNotice)

        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        await session.retryRestore()

        XCTAssertEqual(session.user?.id, 1)
        XCTAssertNil(session.restoreError)
    }

    func testRestoreOfflineKeepsCredentials() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info", .failure(.notConnectedToInternet))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        await session.restore()

        XCTAssertNil(session.user)
        XCTAssertEqual(session.restoreError, "网络连接失败，请检查网络后重试")
        XCTAssertEqual(store.credentials, AuthFixtures.credentials())
        XCTAssertEqual(store.deleteCount, 0)
        XCTAssertNil(session.sessionExpiredNotice)
    }

    func testRestoreSessionInvalidClearsOnceAndPrompts() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info", .json(AuthFixtures.sessionInvalidPayload(), status: 401))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        await session.restore()

        XCTAssertNil(session.user)
        XCTAssertNil(session.restoreError)
        XCTAssertNil(store.credentials)
        XCTAssertEqual(store.deleteCount, 1)
        XCTAssertEqual(session.sessionInvalidationCount, 1)
        XCTAssertEqual(session.sessionExpiredNotice, "登录状态已失效，请重新登录")
        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 0)
    }
}
