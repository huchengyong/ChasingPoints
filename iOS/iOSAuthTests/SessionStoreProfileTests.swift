import Foundation
import XCTest
@testable import iOS

@MainActor
final class SessionStoreProfileTests: XCTestCase {
    func testNicknameValidationBoundariesUseUnicodeScalars() {
        XCTAssertEqual(SessionStore.normalizedNickname("  小明  "), "小明")
        XCTAssertNil(SessionStore.nicknameValidationMessage("阿福"))
        XCTAssertNil(SessionStore.nicknameValidationMessage("十二个字符长度刚刚好合适"))
        XCTAssertNil(SessionStore.nicknameValidationMessage("👍👍"))
        XCTAssertNil(SessionStore.nicknameValidationMessage("👨‍👩‍👧"))

        XCTAssertEqual(SessionStore.nicknameValidationMessage(""), "请输入昵称")
        XCTAssertEqual(SessionStore.nicknameValidationMessage("   "), "请输入昵称")
        XCTAssertEqual(SessionStore.nicknameValidationMessage("甲"), "昵称长度需要在2-12个字符之间")
        XCTAssertEqual(SessionStore.nicknameValidationMessage("十二个字符长度刚刚好合适啊"), "昵称长度需要在2-12个字符之间")
        // 组合序列：String.count 为 1，但 Unicode scalar 为 2，与服务端 rune 计数一致。
        XCTAssertNil(SessionStore.nicknameValidationMessage("e\u{301}"))
    }

    func testUpdateNicknameSendsOnlyTrimmedNicknameAndSyncsProfile() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("POST", "/api/user/profile", .json(AuthFixtures.updateProfileResponse(nickname: "新昵称")))

        try await session.updateNickname("  新昵称  ")

        XCTAssertEqual(session.user?.nickname, "新昵称")
        XCTAssertEqual(session.user?.id, 1)
        let updates = server.requests("/api/user/profile")
        XCTAssertEqual(updates.count, 1)
        XCTAssertEqual(updates.first?.method, "POST")
        XCTAssertEqual(updates.first?.bearerToken, AuthFixtures.oldAccess)
        XCTAssertEqual(updates.first?.bodyJSON.keys.sorted(), ["nickname"])
        XCTAssertEqual(updates.first?.bodyJSON["nickname"] as? String, "新昵称")
        // 只读恢复之外不能再有用户资料 GET。
        XCTAssertEqual(server.requestCount("/api/user/info"), 1)
    }

    func testUpdateNicknameRejectsInvalidInputWithoutRequest() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)

        for invalid in ["", "   ", "甲", "十二个字符长度刚刚好合适啊"] {
            do {
                try await session.updateNickname(invalid)
                XCTFail("非法昵称不应提交：\(invalid)")
            } catch {
                let trimmed = invalid.trimmingCharacters(in: .whitespacesAndNewlines)
                XCTAssertEqual(
                    error.localizedDescription,
                    trimmed.isEmpty ? "请输入昵称" : "昵称长度需要在2-12个字符之间"
                )
            }
        }

        XCTAssertEqual(server.requestCount("/api/user/profile"), 0)
        XCTAssertEqual(session.user?.nickname, "测试用户")
    }

    func testUpdateNicknameBusinessFailureKeepsCurrentProfile() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("POST", "/api/user/profile", .json(AuthFixtures.updateProfileFailure()))

        do {
            try await session.updateNickname("新昵称")
            XCTFail("业务失败不应发布成功")
        } catch {
            assertAPIError(error, kind: .business, message: "昵称长度需要在2-12个字符之间")
        }

        XCTAssertEqual(session.user?.nickname, "测试用户")
    }

    func testUpdateNicknameNetworkFailureKeepsCurrentProfile() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("POST", "/api/user/profile", .failure(.notConnectedToInternet))

        do {
            try await session.updateNickname("新昵称")
            XCTFail("网络失败不应发布成功")
        } catch {
            assertAPIError(error, kind: .network, message: "网络连接失败，请检查网络后重试")
        }

        XCTAssertEqual(session.user?.nickname, "测试用户")
    }

    func testUpdateNicknameIncompleteOrMismatchedResponseIsRejected() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)

        let invalidPayloads: [[String: Any]] = [
            ["success": true, "message": "更新成功", "user_info": NSNull()],
            ["success": true, "message": "更新成功", "user_info": AuthFixtures.userInfo(id: 2, nickname: "用户B")],
            ["success": true, "message": "更新成功", "user_info": AuthFixtures.userInfo(status: 0)],
        ]
        for payload in invalidPayloads {
            server.stub("POST", "/api/user/profile", .json(payload))
            do {
                try await session.updateNickname("新昵称")
                XCTFail("不完整或身份不匹配的响不应发布成功")
            } catch {
                XCTAssertEqual(error.localizedDescription, "服务器返回的登录信息不完整")
            }
        }

        XCTAssertEqual(session.user?.nickname, "测试用户")
        XCTAssertEqual(session.user?.id, 1)
    }

    func testUpdateNicknameBeforeProfileReadyIsRejected() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info", .raw("内部服务错误", status: 503))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)
        await session.restore()

        do {
            try await session.updateNickname("新昵称")
            XCTFail("资料就绪前不应允许编辑")
        } catch {
            XCTAssertEqual(error.localizedDescription, "登录状态已变更")
        }

        XCTAssertEqual(server.requestCount("/api/user/profile"), 0)
    }

    func testUpdateNicknameReplaysAfterRefreshWithSameBody() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("POST", "/api/user/profile") { request in
            request.bearerToken == AuthFixtures.oldAccess
                ? .expiredAccessToken
                : .json(AuthFixtures.updateProfileResponse(nickname: "新昵称"))
        }
        server.stub("POST", "/api/auth/refresh-token", .json(AuthFixtures.refreshResponse()))

        try await session.updateNickname("新昵称")

        XCTAssertEqual(server.requestCount("/api/auth/refresh-token"), 1)
        XCTAssertEqual(server.requestCount("/api/user/profile"), 2)
        XCTAssertEqual(server.requests("/api/user/profile").last?.bearerToken, AuthFixtures.newAccess)
        XCTAssertEqual(server.requests("/api/user/profile").last?.bodyJSON["nickname"] as? String, "新昵称")
        XCTAssertEqual(session.user?.nickname, "新昵称")
    }

    func testUpdateNicknameLateSuccessAfterLogoutIsDiscarded() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/user/profile") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(AuthFixtures.updateProfileResponse(nickname: "新昵称"))
        }

        let update = Task { try await session.updateNickname("新昵称") }
        await waitUntil { server.requestCount("/api/user/profile") == 1 }

        session.logout()
        gate.signal()

        do {
            try await update.value
            XCTFail("退出后旧保存结果必须被丢弃")
        } catch {
            XCTAssertEqual(error.localizedDescription, "登录状态已变更")
        }

        XCTAssertNil(session.user)
        XCTAssertNil(session.sessionExpiredNotice)
        XCTAssertEqual(session.sessionInvalidationCount, 0)
    }

    func testUpdateNicknameLateFailureAfterSwitchDoesNotAffectNewUser() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/user/profile") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .raw("service unavailable", status: 503)
        }
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse(
            access: "b-access-token",
            refresh: "b-refresh-token",
            user: AuthFixtures.userInfo(id: 2, nickname: "用户B")
        )))

        let update = Task { try await session.updateNickname("新昵称") }
        await waitUntil { server.requestCount("/api/user/profile") == 1 }

        try await session.login(phone: AuthFixtures.phone, code: "123456")
        XCTAssertEqual(session.user?.id, 2)
        gate.signal()

        do {
            try await update.value
            XCTFail("切号后旧保存失败必须被丢弃")
        } catch {
            XCTAssertEqual(error.localizedDescription, "登录状态已变更")
        }

        XCTAssertEqual(session.user?.id, 2)
        XCTAssertEqual(session.user?.nickname, "用户B")
        XCTAssertNil(session.sessionExpiredNotice)
        XCTAssertEqual(session.sessionInvalidationCount, 0)
    }

    private func makeRestoredSession(_ server: MockHTTPServer) async -> SessionStore {
        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)
        await session.restore()
        return session
    }
}
