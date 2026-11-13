import Foundation
import XCTest
@testable import iOS

@MainActor
final class SessionStoreAccountTests: XCTestCase {
    func testSendSMSDefaultsToLoginSceneAndAcceptsDeleteAccountScene() async throws {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore())
        server.stub("POST", "/api/auth/send-sms", .json(AuthFixtures.sendSMSResponse()))

        try await session.sendSMS(phone: AuthFixtures.phone)
        try await session.sendSMS(phone: AuthFixtures.phone, scene: "delete_account")

        let requests = server.requests("/api/auth/send-sms")
        XCTAssertEqual(requests.count, 2)
        XCTAssertEqual(requests[0].bodyJSON["scene"] as? String, "login")
        XCTAssertEqual(requests[1].bodyJSON["scene"] as? String, "delete_account")
        XCTAssertEqual(requests[0].bodyJSON["phone"] as? String, AuthFixtures.phone)
    }

    func testUpdateAvatarSendsCurrentNicknameAndAvatarThenSyncsProfile() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let avatar = "https://cdn.example.com/avatars/1/20260409/a.jpg"
        server.stub(
            "POST",
            "/api/user/profile",
            .json(AuthFixtures.updateProfileResponse(nickname: "测试用户", avatar: avatar))
        )

        try await session.updateAvatar(avatar)

        XCTAssertEqual(session.user?.avatar, avatar)
        XCTAssertEqual(session.user?.nickname, "测试用户")
        let request = try XCTUnwrap(server.requests("/api/user/profile").first)
        XCTAssertEqual(request.method, "POST")
        XCTAssertEqual(request.bearerToken, AuthFixtures.oldAccess)
        XCTAssertEqual(request.bodyJSON.keys.sorted(), ["avatar", "nickname"])
        XCTAssertEqual(request.bodyJSON["nickname"] as? String, "测试用户")
        XCTAssertEqual(request.bodyJSON["avatar"] as? String, avatar)
    }

    func testUpdateAvatarRejectsMismatchedProfileWithoutWritingIt() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub(
            "POST",
            "/api/user/profile",
            .json(["success": true, "message": "更新成功", "user_info": AuthFixtures.userInfo(id: 2)])
        )

        do {
            try await session.updateAvatar("https://cdn.example.com/a.jpg")
            XCTFail("身份不匹配的响应不应发布")
        } catch {
            XCTAssertEqual(error.localizedDescription, "服务器返回的登录信息不完整")
        }

        XCTAssertEqual(session.user?.id, 1)
        XCTAssertEqual(session.user?.avatar, "")
    }

    func testDeleteAccountSendsSmsVerificationAndClearsLocalSession() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)
        await session.restore()
        server.stub("POST", "/api/user/account/delete", .json(["success": true, "message": "账号已注销"]))

        try await session.deleteAccount(smsCode: "123456")

        let request = try XCTUnwrap(server.requests("/api/user/account/delete").first)
        XCTAssertEqual(request.method, "POST")
        XCTAssertEqual(request.bearerToken, AuthFixtures.oldAccess)
        XCTAssertEqual(request.bodyJSON.keys.sorted(), ["confirm_text", "sms_code", "verify_type"])
        XCTAssertEqual(request.bodyJSON["confirm_text"] as? String, "注销账号")
        XCTAssertEqual(request.bodyJSON["verify_type"] as? String, "sms")
        XCTAssertEqual(request.bodyJSON["sms_code"] as? String, "123456")

        XCTAssertNil(session.user)
        XCTAssertNil(store.credentials)
        XCTAssertEqual(store.deleteCount, 1)
        XCTAssertNil(session.sessionExpiredNotice)
        XCTAssertEqual(session.sessionInvalidationCount, 0)
    }

    func testDeleteAccountFailureKeepsSession() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub(
            "POST",
            "/api/user/account/delete",
            .json(["success": false, "message": "注销验证失败"])
        )

        do {
            try await session.deleteAccount(smsCode: "123456")
            XCTFail("注销失败不应清理会话")
        } catch {
            assertAPIError(error, kind: .business, message: "注销验证失败")
        }

        XCTAssertEqual(session.user?.id, 1)
        XCTAssertNotNil(session.identity)
        XCTAssertNil(session.sessionExpiredNotice)
        XCTAssertEqual(session.sessionInvalidationCount, 0)
    }

    func testDeleteAccountLateResponseDoesNotClearNewLogin() async throws {
        let server = MockHTTPServer()
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        let session = makeSessionStore(server, credentialStore: store)
        await session.restore()

        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/user/account/delete") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(["success": true, "message": "账号已注销"])
        }
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse(
            access: "b-access-token",
            refresh: "b-refresh-token",
            user: AuthFixtures.userInfo(id: 2, nickname: "用户B")
        )))

        let deletion = Task { try await session.deleteAccount(smsCode: "123456") }
        await waitUntil { server.requestCount("/api/user/account/delete") == 1 }

        session.logout()
        try await session.login(phone: AuthFixtures.phone, code: "123456")
        XCTAssertEqual(session.user?.id, 2)
        gate.signal()

        do {
            try await deletion.value
            XCTFail("旧身份的注销结果必须被丢弃")
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
}
