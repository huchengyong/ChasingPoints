import Foundation
import XCTest
@testable import iOS

@MainActor
final class SessionStoreRequestTests: XCTestCase {
    func testAuthorizedPostEncodesSnakeCaseBodyWithBearer() async throws {
        let server = MockHTTPServer()
        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
        server.stub("POST", "/api/match/action", .json(["success": true, "ok": true]))

        let result: SimpleResult = try await session.authorizedPost(
            "/api/match/action",
            body: MatchActionBody(matchId: 42, scoreDelta: 3)
        )

        XCTAssertTrue(result.ok)
        let request = try XCTUnwrap(server.requests("/api/match/action").first)
        XCTAssertEqual(request.method, "POST")
        XCTAssertEqual(request.bearerToken, AuthFixtures.oldAccess)
        XCTAssertEqual(request.bodyJSON["match_id"] as? Int, 42)
        XCTAssertEqual(request.bodyJSON["score_delta"] as? Int, 3)
        XCTAssertNil(request.bodyJSON["matchId"], "请求体必须是 snake_case")
    }

    func testOptionalAuthRequestWithoutCredentialsOmitsAuthorization() async throws {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        server.stub("GET", "/api/public", .json(["success": true, "ok": true]))

        let result: SimpleResult = try await session.optionalAuthRequest("/api/public")

        XCTAssertTrue(result.ok)
        XCTAssertNil(server.requests("/api/public").first?.bearerToken)
    }

    func testOptionalAuthRequestWithCredentialsRefreshesAndReplaysOnce() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/public") { request in
            request.bearerToken == AuthFixtures.oldAccess
                ? .expiredAccessToken
                : .json(["success": true, "ok": true])
        }
        server.stub("POST", "/api/auth/refresh-token", .json(AuthFixtures.refreshResponse()))
        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )

        let result: SimpleResult = try await session.optionalAuthRequest("/api/public")

        XCTAssertTrue(result.ok)
        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 1)
        XCTAssertEqual(server.requestCount("/api/public"), 2)
        XCTAssertEqual(server.requests("/api/public").last?.bearerToken, AuthFixtures.newAccess)
    }

    func testGuestResultIsRejectedWhenLoginCompletesDuringRequest() async throws {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/public") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(["success": true, "ok": true])
        }
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse()))

        let guest = Task<SimpleResult, Error> {
            try await session.optionalAuthRequest("/api/public")
        }
        await waitUntil { server.requestCount("/api/public") == 1 }
        try await session.login(phone: AuthFixtures.phone, code: "123456")
        gate.signal()

        do {
            _ = try await guest.value
            XCTFail("访客请求进行中完成登录时不得返回访客结果")
        } catch {
            XCTAssertEqual(error.localizedDescription, "登录状态已变更")
        }
    }

    private struct MatchActionBody: Encodable {
        let matchId: Int64
        let scoreDelta: Int
    }

    private struct SimpleResult: Decodable {
        let ok: Bool
    }
}
