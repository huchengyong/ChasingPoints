import Foundation
import XCTest
@testable import iOS

@MainActor
final class DeleteAccountModelTests: XCTestCase {
    func testTwoStepFlow() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let model = DeleteAccountModel(session: session)

        model.open()
        XCTAssertEqual(model.step, .warning)
        XCTAssertTrue(model.hasBoundPhone)

        model.continueToVerify()
        XCTAssertEqual(model.step, .verify)
    }

    func testPhoneAndCodeValidationMessages() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let model = DeleteAccountModel(session: session)
        model.open()
        model.continueToVerify()

        model.phone = "123"
        let phoneMessage = await model.sendSMS()
        XCTAssertEqual(phoneMessage, "请输入已绑定的完整手机号")

        let submitMessage = await model.submit()
        XCTAssertEqual(submitMessage, .failure("请填写完整手机号和注销验证码"))

        model.phone = "13800000000"
        model.smsCode = "123"
        let codeMessage = await model.submit()
        XCTAssertEqual(codeMessage, .failure("请填写完整手机号和注销验证码"))

        XCTAssertEqual(server.requestCount("/api/auth/send-sms"), 0)
        XCTAssertEqual(server.requestCount("/api/user/account/delete"), 0)
    }

    func testSendSMSUsesDeleteAccountScene() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/auth/send-sms", .json(AuthFixtures.sendSMSResponse()))
        let model = DeleteAccountModel(session: session)
        model.phone = "13800000000"

        let message = await model.sendSMS()

        XCTAssertEqual(message, "注销验证码已发送")
        let request = server.requests("/api/auth/send-sms").first
        XCTAssertEqual(request?.bodyJSON["phone"] as? String, "13800000000")
        XCTAssertEqual(request?.bodyJSON["scene"] as? String, "delete_account")
    }

    func testSuccessfulDeletionClearsSession() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/user/account/delete", .json(["success": true, "message": "账号已注销"]))
        let model = DeleteAccountModel(session: session)
        model.phone = "13800000000"
        model.smsCode = "123456"

        let result = await model.submit()

        XCTAssertEqual(result, .success)
        XCTAssertNil(session.user)
        XCTAssertNil(session.identity)
        XCTAssertNil(session.sessionExpiredNotice)
    }

    func testFailureStaysOnVerifyStepAndCanRetry() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub(
            "POST",
            "/api/user/account/delete",
            .json(["success": false, "message": "注销验证失败"])
        )
        let model = DeleteAccountModel(session: session)
        model.open()
        model.continueToVerify()
        model.phone = "13800000000"
        model.smsCode = "123456"

        let failure = await model.submit()
        XCTAssertEqual(failure, .failure("注销验证失败"))
        XCTAssertEqual(model.step, .verify)
        XCTAssertNotNil(session.identity)

        server.stub("POST", "/api/user/account/delete", .json(["success": true, "message": "账号已注销"]))
        let success = await model.submit()

        XCTAssertEqual(success, .success)
        XCTAssertNil(session.user)
    }

    func testRepeatedSubmitWhileSavingIsIgnored() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/user/account/delete") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(["success": true, "message": "账号已注销"])
        }
        let model = DeleteAccountModel(session: session)
        model.phone = "13800000000"
        model.smsCode = "123456"

        let first = Task { await model.submit() }
        await waitUntil { server.requestCount("/api/user/account/delete") == 1 }
        let second = await model.submit()
        XCTAssertEqual(second, .ignored)

        gate.signal()
        let result = await first.value

        XCTAssertEqual(result, .success)
        XCTAssertEqual(server.requestCount("/api/user/account/delete"), 1)
    }
}
