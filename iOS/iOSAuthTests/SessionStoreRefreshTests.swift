import Foundation
import XCTest
@testable import iOS

@MainActor
final class SessionStoreRefreshTests: XCTestCase {
    func testExpiredAccessTokenRefreshesAndReplaysOnce() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info") { request in
            request.bearerToken == AuthFixtures.oldAccess
                ? .expiredAccessToken
                : .json(AuthFixtures.userInfoResponse())
        }
        server.stub("POST", "/api/auth/refresh-token", .json(AuthFixtures.refreshResponse()))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        let response: TestUserInfoResponse = try await session.authorizedRequest("/api/user/info")

        XCTAssertEqual(response.userInfo?.id, 1)
        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 1)
        XCTAssertEqual(server.requestCount("/api/user/info"), 2)
        XCTAssertEqual(server.requests("/api/user/info").last?.bearerToken, AuthFixtures.newAccess)
        XCTAssertEqual(
            store.credentials,
            Credentials(accessToken: AuthFixtures.newAccess, refreshToken: AuthFixtures.newRefresh)
        )
        XCTAssertEqual(session.sessionInvalidationCount, 0)
        XCTAssertNil(session.sessionExpiredNotice)

        let refreshBody = try XCTUnwrap(server.requests("/api/auth/refresh-token").first).bodyJSON
        XCTAssertEqual(refreshBody["refresh_token"] as? String, AuthFixtures.oldRefresh)
    }

    func testLateOldUnauthorizedReusesRefreshedCredentialsWithoutSecondRefresh() async throws {
        let server = MockHTTPServer()
        let lateGate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/user/info") { request in
            request.bearerToken == AuthFixtures.oldAccess
                ? .expiredAccessToken
                : .json(AuthFixtures.userInfoResponse())
        }
        server.stub("GET", "/api/match/ongoing") { request in
            guard request.bearerToken == AuthFixtures.oldAccess else { return .json(["success": true]) }
            _ = lateGate.wait(timeout: .now() + 5)
            return .expiredAccessToken
        }
        server.stub("POST", "/api/auth/refresh-token", .json(AuthFixtures.refreshResponse()))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        async let late: [String: Bool] = session.authorizedRequest("/api/match/ongoing")
        await waitUntil { server.requestCount("/api/match/ongoing") == 1 }

        let first: TestUserInfoResponse = try await session.authorizedRequest("/api/user/info")
        XCTAssertEqual(first.userInfo?.id, 1)
        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 1)

        lateGate.signal()
        let lateResponse = try await late

        XCTAssertEqual(lateResponse["success"], true)
        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 1)
        XCTAssertEqual(server.requestCount("/api/user/info"), 2)
        XCTAssertEqual(server.requestCount("/api/match/ongoing"), 2)
        XCTAssertEqual(server.requests("/api/match/ongoing").last?.bearerToken, AuthFixtures.newAccess)
    }

    func testConcurrentUnauthorizedShareSingleRefreshAndReplayOnceEach() async throws {
        let server = MockHTTPServer()
        let refreshGate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/user/info") { request in
            request.bearerToken == AuthFixtures.oldAccess
                ? .expiredAccessToken
                : .json(AuthFixtures.userInfoResponse())
        }
        server.stub("GET", "/api/match/ongoing") { request in
            request.bearerToken == AuthFixtures.oldAccess
                ? .expiredAccessToken
                : .json(["success": true])
        }
        server.stub("POST", "/api/auth/refresh-token") { _ in
            _ = refreshGate.wait(timeout: .now() + 5)
            return .json(AuthFixtures.refreshResponse())
        }
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        async let first: TestUserInfoResponse = session.authorizedRequest("/api/user/info")
        async let second: [String: Bool] = session.authorizedRequest("/api/match/ongoing")
        await waitUntil {
            server.requestCount("/api/user/info") == 1 && server.requestCount("/api/match/ongoing") == 1
        }
        refreshGate.signal()
        let (firstResponse, secondResponse) = try await (first, second)

        XCTAssertEqual(firstResponse.userInfo?.id, 1)
        XCTAssertEqual(secondResponse["success"], true)
        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 1)
        XCTAssertEqual(server.requestCount("/api/user/info"), 2)
        XCTAssertEqual(server.requestCount("/api/match/ongoing"), 2)
    }

    func testSharedRefreshTemporaryFailureKeepsCredentialsAndRetriesLater() async throws {
        let server = MockHTTPServer()
        let pairGate = DispatchSemaphore(value: 0)
        let refreshGate = DispatchSemaphore(value: 0)
        let refreshAttempts = LockedCounter()
        let originalUserInfoRequests = LockedCounter()
        server.stub("GET", "/api/user/info") { request in
            guard request.bearerToken == AuthFixtures.oldAccess else {
                return .json(AuthFixtures.userInfoResponse())
            }
            // 仅第一批原请求需要等待配对；刷新失败后的恢复请求不再阻塞。
            if originalUserInfoRequests.next() == 1 {
                _ = pairGate.wait(timeout: .now() + 5)
            }
            return .expiredAccessToken
        }
        server.stub("GET", "/api/match/ongoing") { request in
            guard request.bearerToken == AuthFixtures.oldAccess else { return .json(["success": true]) }
            pairGate.signal()
            return .expiredAccessToken
        }
        server.stub("POST", "/api/auth/refresh-token") { _ in
            if refreshAttempts.next() == 1 {
                _ = refreshGate.wait(timeout: .now() + 5)
                return .raw("service unavailable", status: 503)
            }
            return .json(AuthFixtures.refreshResponse())
        }
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        async let first: TestUserInfoResponse = session.authorizedRequest("/api/user/info")
        async let second: [String: Bool] = session.authorizedRequest("/api/match/ongoing")
        await waitUntil { server.requestCount("/api/auth/refresh-token") == 1 }
        refreshGate.signal()

        do {
            _ = try await first
            XCTFail("共享刷新 503 时不应成功")
        } catch {
            assertAPIError(error, kind: .server, message: "服务暂时不可用，请稍后重试", statusCode: 503)
        }
        do {
            _ = try await second
            XCTFail("共享刷新 503 时不应成功")
        } catch {
            assertAPIError(error, kind: .server)
        }

        // 暂时性故障：保留凭据，零清理、零重新登录提示。
        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 1)
        XCTAssertEqual(store.credentials, AuthFixtures.credentials())
        XCTAssertEqual(store.deleteCount, 0)
        XCTAssertEqual(session.sessionInvalidationCount, 0)
        XCTAssertNil(session.sessionExpiredNotice)
        XCTAssertNil(session.user)

        // 服务恢复后，新请求重新发起刷新并在成功后重放一次。
        let recovered: TestUserInfoResponse = try await session.authorizedRequest("/api/user/info")

        XCTAssertEqual(recovered.userInfo?.id, 1)
        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 2)
        XCTAssertEqual(
            store.credentials,
            Credentials(accessToken: AuthFixtures.newAccess, refreshToken: AuthFixtures.newRefresh)
        )
    }

    func testSharedRefreshDefinitiveFailureClearsOnce() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info") { request in
            request.bearerToken == AuthFixtures.oldAccess
                ? .expiredAccessToken
                : .json(AuthFixtures.userInfoResponse())
        }
        server.stub("GET", "/api/match/ongoing") { request in
            request.bearerToken == AuthFixtures.oldAccess
                ? .expiredAccessToken
                : .json(["success": true])
        }
        server.stub("POST", "/api/auth/refresh-token", .json(AuthFixtures.sessionInvalidPayload()))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        async let first: TestUserInfoResponse = session.authorizedRequest("/api/user/info")
        async let second: [String: Bool] = session.authorizedRequest("/api/match/ongoing")
        _ = try? await first
        _ = try? await second

        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 1)
        XCTAssertEqual(server.requestCount("/api/user/info"), 1)
        XCTAssertEqual(server.requestCount("/api/match/ongoing"), 1)
        XCTAssertEqual(session.sessionInvalidationCount, 1)
        XCTAssertEqual(session.sessionExpiredNotice, "登录状态已失效，请重新登录")
        XCTAssertNil(store.credentials)
        XCTAssertEqual(store.deleteCount, 1)
    }

    func testReplayUnauthorizedClearsSessionWithoutSecondRefresh() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info", .expiredAccessToken)
        server.stub("POST", "/api/auth/refresh-token", .json(AuthFixtures.refreshResponse()))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        do {
            let _: TestUserInfoResponse = try await session.authorizedRequest("/api/user/info")
            XCTFail("重放仍 401 时不应成功")
        } catch {
            assertAPIError(error, kind: .unauthorized, statusCode: 401)
        }

        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 1)
        XCTAssertEqual(server.requestCount("/api/user/info"), 2)
        XCTAssertEqual(session.sessionInvalidationCount, 1)
        XCTAssertEqual(session.sessionExpiredNotice, "登录状态已失效，请重新登录")
        XCTAssertNil(store.credentials)
    }

    func testSessionInvalidSkipsRefreshAndClears() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info", .json(AuthFixtures.sessionInvalidPayload(), status: 401))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        do {
            let _: TestUserInfoResponse = try await session.authorizedRequest("/api/user/info")
            XCTFail("SESSION_INVALID 时不应成功")
        } catch {
            assertAPIError(error, kind: .sessionInvalid, statusCode: 401)
        }

        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 0)
        XCTAssertEqual(session.sessionInvalidationCount, 1)
        XCTAssertEqual(session.sessionExpiredNotice, "登录状态已失效，请重新登录")
        XCTAssertNil(store.credentials)
        XCTAssertEqual(store.deleteCount, 1)
    }

    func testConcurrentSessionInvalidClearsAndPromptsOnce() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info", .json(AuthFixtures.sessionInvalidPayload(), status: 401))
        server.stub("GET", "/api/match/ongoing", .json(AuthFixtures.sessionInvalidPayload(), status: 401))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        async let first: TestUserInfoResponse = session.authorizedRequest("/api/user/info")
        async let second: [String: Bool] = session.authorizedRequest("/api/match/ongoing")
        _ = try? await first
        _ = try? await second

        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 0)
        XCTAssertEqual(session.sessionInvalidationCount, 1)
        XCTAssertEqual(session.sessionExpiredNotice, "登录状态已失效，请重新登录")
        XCTAssertNil(store.credentials)
        XCTAssertEqual(store.deleteCount, 1)
    }
}
