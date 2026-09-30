import Foundation
import XCTest
@testable import iOS

@MainActor
final class APIClientTests: XCTestCase {
    private var server: MockHTTPServer!
    private var client: APIClient!

    private struct LoginPayload: Decodable {
        let accessToken: String
        let refreshToken: String
        let userInfo: UserInfo?
    }

    private struct MessagePayload: Decodable {
        let message: String
    }

    private var loginBody: [String: String] {
        ["phone": AuthFixtures.phone, "sms_code": "123456"]
    }

    override func setUp() {
        super.setUp()
        server = MockHTTPServer()
        client = APIClient(baseURL: URL(string: "https://example.test")!, session: server.makeSession())
    }

    func testLoginResponseParsesTopLevelFields() async throws {
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse()))
        let payload: LoginPayload = try await client.post("/api/auth/login", body: loginBody)

        XCTAssertEqual(payload.accessToken, "login-access-token")
        XCTAssertEqual(payload.refreshToken, "login-refresh-token")
        XCTAssertEqual(payload.userInfo?.id, 1)
        XCTAssertEqual(payload.userInfo?.phone, AuthFixtures.maskedPhone)
        XCTAssertEqual(server.requests.first?.bodyJSON["sms_code"] as? String, "123456")
    }

    func testUserInfoResponseParsesTopLevelFields() async throws {
        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        let payload: TestUserInfoResponse = try await client.get("/api/user/info", bearerToken: AuthFixtures.oldAccess)

        XCTAssertEqual(payload.userInfo?.nickname, "测试用户")
        XCTAssertEqual(server.requests.first?.bearerToken, AuthFixtures.oldAccess)
    }

    func testHTTP200BusinessFailureThrowsBusinessError() async throws {
        server.stub("POST", "/api/auth/send-sms", .json(AuthFixtures.sendSMSResponse(
            success: false,
            message: "请等待42秒后再试"
        )))

        do {
            let _: MessagePayload = try await client.post("/api/auth/send-sms", body: ["phone": AuthFixtures.phone])
            XCTFail("HTTP 200 + success:false 不应被视为成功")
        } catch {
            assertAPIError(error, kind: .business, message: "请等待42秒后再试", statusCode: 200)
        }
    }

    func testPlainText400ErrorMessageIsPreserved() async throws {
        // 后端 login logic 返回 fmt.Errorf，go-zero 默认以 HTTP 400 纯文本写出。
        server.stub("POST", "/api/auth/login", .raw("验证码错误\n", status: 400))

        do {
            let _: LoginPayload = try await client.post("/api/auth/login", body: loginBody)
            XCTFail("应当抛出业务错误")
        } catch {
            assertAPIError(error, kind: .business, message: "验证码错误", statusCode: 400)
        }
    }

    func testExpiredCode400MessageIsPreserved() async throws {
        server.stub("POST", "/api/auth/login", .raw("验证码已过期或不存在\n", status: 400))

        do {
            let _: LoginPayload = try await client.post("/api/auth/login", body: loginBody)
            XCTFail("过期验证码应当抛出业务错误")
        } catch {
            assertAPIError(error, kind: .business, message: "验证码已过期或不存在", statusCode: 400)
        }
    }

    func testRedis400ErrorDoesNotLeakBody() async throws {
        server.stub("POST", "/api/auth/login", .raw(
            "dial tcp 127.0.0.1:6379: connect: connection refused\n",
            status: 400
        ))

        do {
            let _: LoginPayload = try await client.post("/api/auth/login", body: loginBody)
            XCTFail("服务内部错误应当抛出")
        } catch {
            assertAPIError(error, kind: .business, message: "请求失败，请稍后重试", statusCode: 400)
        }
    }

    func testHTTP200RedisBusinessErrorDoesNotLeakBody() async throws {
        server.stub("POST", "/api/auth/send-sms", .json(AuthFixtures.sendSMSResponse(
            success: false,
            message: "dial tcp 127.0.0.1:6379: connect: connection refused"
        )))

        do {
            let _: MessagePayload = try await client.post("/api/auth/send-sms", body: ["phone": AuthFixtures.phone])
            XCTFail("服务内部错误应当抛出")
        } catch {
            assertAPIError(error, kind: .business, message: "请求失败，请稍后重试", statusCode: 200)
        }
    }

    func testHTML400DoesNotLeakBody() async throws {
        server.stub("POST", "/api/auth/login", .raw("<html><body>Bad Gateway</body></html>", status: 400))

        do {
            let _: LoginPayload = try await client.post("/api/auth/login", body: loginBody)
            XCTFail("应当抛出业务错误")
        } catch {
            let apiError = try XCTUnwrap(error as? APIError)
            XCTAssertEqual(apiError.kind, .business)
            XCTAssertEqual(apiError.message, "请求失败，请稍后重试")
            XCTAssertFalse(apiError.message.contains("<"))
        }
    }

    func testServerErrorUsesGenericMessage() async throws {
        server.stub("GET", "/api/user/info", .json(
            ["message": "panic: runtime error: index out of range"],
            status: 500
        ))

        do {
            let _: TestUserInfoResponse = try await client.get("/api/user/info")
            XCTFail("应当抛出服务错误")
        } catch {
            let apiError = try XCTUnwrap(error as? APIError)
            XCTAssertEqual(apiError.kind, .server)
            XCTAssertEqual(apiError.message, "服务暂时不可用，请稍后重试")
            XCTAssertFalse(apiError.message.contains("panic"))
        }
    }

    func testEmpty401IsOrdinaryUnauthorized() async throws {
        server.stub("GET", "/api/user/info", .expiredAccessToken)

        do {
            let _: TestUserInfoResponse = try await client.get("/api/user/info")
            XCTFail("应当抛出 401")
        } catch {
            let apiError = try XCTUnwrap(error as? APIError)
            XCTAssertEqual(apiError.kind, .unauthorized)
            XCTAssertTrue(apiError.isUnauthorized)
            XCTAssertFalse(apiError.isSessionInvalid)
            XCTAssertEqual(apiError.message, "登录状态已失效，请重新登录")
            XCTAssertNil(apiError.reason)
        }
    }

    func testSessionInvalid401PreservesReason() async throws {
        server.stub("GET", "/api/user/info", .json(AuthFixtures.sessionInvalidPayload(), status: 401))

        do {
            let _: TestUserInfoResponse = try await client.get("/api/user/info")
            XCTFail("应当抛出 SESSION_INVALID")
        } catch {
            let apiError = try XCTUnwrap(error as? APIError)
            XCTAssertEqual(apiError.kind, .sessionInvalid)
            XCTAssertTrue(apiError.isSessionInvalid)
            XCTAssertEqual(apiError.reason, "SESSION_INVALID")
            XCTAssertEqual(apiError.message, "登录状态已失效，请重新登录")
        }
    }

    func testHTTP200SessionInvalidPreservesReason() async throws {
        server.stub("POST", "/api/auth/refresh-token", .json(AuthFixtures.sessionInvalidPayload()))

        do {
            let _: MessagePayload = try await client.post("/api/auth/refresh-token", body: ["refresh_token": "x"])
            XCTFail("应当抛出 SESSION_INVALID")
        } catch {
            assertAPIError(error, kind: .sessionInvalid, message: "登录状态已失效，请重新登录", statusCode: 200)
        }
    }

    func testUnknownInternalBodyIsInvalidResponse() async throws {
        server.stub("POST", "/api/auth/login", .json(["result": "unexpected"]))

        do {
            let _: LoginPayload = try await client.post("/api/auth/login", body: loginBody)
            XCTFail("应当抛出解析错误")
        } catch {
            assertAPIError(error, kind: .invalidResponse, message: "服务器响应格式错误", statusCode: 200)
        }
    }

    func testOfflineIsNetworkError() async throws {
        server.stub("GET", "/api/user/info", .failure(.notConnectedToInternet))

        do {
            let _: TestUserInfoResponse = try await client.get("/api/user/info")
            XCTFail("应当抛出网络错误")
        } catch {
            let apiError = try XCTUnwrap(error as? APIError)
            XCTAssertEqual(apiError.kind, .network)
            XCTAssertNil(apiError.statusCode)
            XCTAssertFalse(apiError.isUnauthorized)
            XCTAssertFalse(apiError.isSessionInvalid)
        }
    }

    func testTimeoutIsNetworkError() async throws {
        server.stub("GET", "/api/user/info", .failure(.timedOut))

        do {
            let _: TestUserInfoResponse = try await client.get("/api/user/info")
            XCTFail("应当超时")
        } catch {
            assertAPIError(error, kind: .network, message: "网络连接失败，请检查网络后重试")
        }
    }

    func testDebugConfigurationUsesDevelopmentEnvironment() {
        #if DEBUG
        XCTAssertEqual(AppConfiguration.apiBaseURL.absoluteString, "https://dev-api.kekemate.cn")
        XCTAssertEqual(AppConfiguration.credentialService, "ChasingPoints.Session.development")
        #else
        XCTAssertEqual(AppConfiguration.apiBaseURL.absoluteString, "https://api.zhuifen.cn")
        XCTAssertEqual(AppConfiguration.credentialService, "ChasingPoints.Session.production")
        #endif
    }
}
