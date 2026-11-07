import Foundation
import XCTest
@testable import iOS

@MainActor
final class SessionStoreIsolationTests: XCTestCase {
    func testLogoutDuringRefreshDiscardsLateSuccess() async throws {
        let server = MockHTTPServer()
        let refreshGate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/user/info") { request in
            request.bearerToken == AuthFixtures.oldAccess
                ? .expiredAccessToken
                : .json(AuthFixtures.userInfoResponse())
        }
        server.stub("POST", "/api/auth/refresh-token") { _ in
            _ = refreshGate.wait(timeout: .now() + 5)
            return .json(AuthFixtures.refreshResponse())
        }
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        let request = Task { try await session.authorizedRequest("/api/user/info") as TestUserInfoResponse }
        await waitUntil { server.requestCount("/api/auth/refresh-token") == 1 }

        session.logout()
        XCTAssertNil(store.credentials)
        refreshGate.signal()

        do {
            _ = try await request.value
            XCTFail("退出后旧结果必须被丢弃")
        } catch {
            XCTAssertEqual(error.localizedDescription, "登录状态已变更")
        }

        XCTAssertNil(session.user)
        XCTAssertNil(session.sessionExpiredNotice)
        XCTAssertEqual(session.sessionInvalidationCount, 0)
        XCTAssertNil(store.credentials)
        XCTAssertEqual(store.saveCount, 0)
    }

    func testOldSessionInvalidAfterNewLoginKeepsNewSession() async throws {
        let server = MockHTTPServer()
        let userInfoGate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/user/info") { _ in
            _ = userInfoGate.wait(timeout: .now() + 5)
            return .json(AuthFixtures.sessionInvalidPayload(), status: 401)
        }
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse(
            access: "b-access-token",
            refresh: "b-refresh-token",
            user: AuthFixtures.userInfo(id: 2, nickname: "用户B")
        )))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        let oldRequest = Task { try await session.authorizedRequest("/api/user/info") as TestUserInfoResponse }
        await waitUntil { server.requestCount("/api/user/info") == 1 }

        session.logout()
        try await session.login(phone: AuthFixtures.phone, code: "123456")
        XCTAssertEqual(session.user?.id, 2)

        userInfoGate.signal()
        do {
            _ = try await oldRequest.value
            XCTFail("旧会话结果必须被丢弃")
        } catch {
            XCTAssertEqual(error.localizedDescription, "登录状态已变更")
        }

        XCTAssertEqual(session.user?.id, 2)
        XCTAssertEqual(
            store.credentials,
            Credentials(accessToken: "b-access-token", refreshToken: "b-refresh-token")
        )
        XCTAssertNil(session.sessionExpiredNotice)
        XCTAssertEqual(session.sessionInvalidationCount, 0)
    }

    func testLogoutDuringLoginDiscardsLateSuccess() async throws {
        let server = MockHTTPServer()
        let loginGate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/auth/login") { _ in
            _ = loginGate.wait(timeout: .now() + 5)
            return .json(AuthFixtures.loginResponse())
        }
        let store = InMemoryCredentialStore()
        let session = makeSessionStore(server, credentialStore: store)

        let login = Task { try await session.login(phone: AuthFixtures.phone, code: "123456") }
        await waitUntil { server.requestCount("/api/auth/login") == 1 }

        session.logout()
        loginGate.signal()

        do {
            try await login.value
            XCTFail("退出后旧登录结果必须被丢弃")
        } catch {
            XCTAssertEqual(error.localizedDescription, "登录状态已变更")
        }

        XCTAssertNil(session.user)
        XCTAssertNil(store.credentials)
        XCTAssertEqual(store.saveCount, 0)
    }

    func testActiveLogoutClearsLocallyWithoutPassiveNotice() async throws {
        let server = MockHTTPServer()
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse()))
        let store = InMemoryCredentialStore()
        let session = makeSessionStore(server, credentialStore: store)
        try await session.login(phone: AuthFixtures.phone, code: "123456")

        session.logout()

        XCTAssertNil(session.user)
        XCTAssertNil(store.credentials)
        XCTAssertEqual(store.deleteCount, 1)
        XCTAssertEqual(session.sessionInvalidationCount, 0)
        XCTAssertNil(session.sessionExpiredNotice)
    }
}
