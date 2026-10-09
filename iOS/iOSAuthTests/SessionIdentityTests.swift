import Foundation
import XCTest
@testable import iOS

@MainActor
final class SessionIdentityTests: XCTestCase {
    func testIdentityIsNilUntilProfileIsVerified() {
        let server = MockHTTPServer()
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)

        XCTAssertNil(session.identity, "仅有本地凭据、尚未验证资料时不得暴露身份")
    }

    func testIdentityIsStableAcrossNicknameUpdate() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        let before = session.identity
        XCTAssertEqual(before?.userId, 1)

        server.stub("POST", "/api/user/profile", .json(AuthFixtures.updateProfileResponse(nickname: "新昵称")))
        try await session.updateNickname("新昵称")

        XCTAssertEqual(session.identity, before, "改昵称不改变账号身份，列表不应被清空")
    }

    func testIdentityIsNilAfterLogout() async {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)

        session.logout()

        XCTAssertNil(session.identity)
    }

    func testReloginAsSameUserChangesIdentity() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        let first = session.identity
        session.logout()

        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse()))
        try await session.login(phone: AuthFixtures.phone, code: "123456")

        XCTAssertEqual(session.identity?.userId, first?.userId)
        XCTAssertNotEqual(session.identity, first, "同一用户重新登录也必须视为新的认证代次")
    }

    private func makeRestoredSession(_ server: MockHTTPServer) async -> SessionStore {
        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)
        await session.restore()
        return session
    }
}
