import XCTest
@testable import iOS

final class RealtimeReconnectTests: XCTestCase {
    /// 搬运自 websocket-reconnect.test.mjs：指数退避 + 抖动，60 秒封顶。
    func testReconnectDelayGrowsExponentiallyWithJitterAndCapsAtSixtySeconds() {
        XCTAssertEqual(RealtimeReconnect.calculateReconnectDelay(attempt: 1, random: { 0.5 }), 3000)
        XCTAssertEqual(RealtimeReconnect.calculateReconnectDelay(attempt: 2, random: { 0.5 }), 6000)
        XCTAssertEqual(RealtimeReconnect.calculateReconnectDelay(attempt: 6, random: { 0.5 }), 60000)
        XCTAssertEqual(RealtimeReconnect.calculateReconnectDelay(attempt: 6, random: { 1 }), 60000)
        XCTAssertEqual(RealtimeReconnect.calculateReconnectDelay(attempt: 3, random: { 0 }), 9600)
        XCTAssertEqual(RealtimeReconnect.calculateReconnectDelay(attempt: 3, random: { 1 }), 14400)
    }

    /// 只有生命周期与认证终止条件会阻止重连。
    func testReconnectPolicyStopsOnlyForLifecycleAndAuthTerminalConditions() {
        XCTAssertEqual(RealtimeReconnect.shouldReconnectWebSocket(), ReconnectDecision(allow: true, reason: .recoverable))
        XCTAssertEqual(
            RealtimeReconnect.shouldReconnectWebSocket(manualDisconnect: true),
            ReconnectDecision(allow: false, reason: .manualDisconnect)
        )
        XCTAssertEqual(
            RealtimeReconnect.shouldReconnectWebSocket(foreground: false),
            ReconnectDecision(allow: false, reason: .background)
        )
        XCTAssertEqual(
            RealtimeReconnect.shouldReconnectWebSocket(online: false),
            ReconnectDecision(allow: false, reason: .offline)
        )
        XCTAssertEqual(
            RealtimeReconnect.shouldReconnectWebSocket(sessionInvalid: true),
            ReconnectDecision(allow: false, reason: .sessionInvalid)
        )
        XCTAssertEqual(
            RealtimeReconnect.shouldReconnectWebSocket(permissionDenied: true),
            ReconnectDecision(allow: false, reason: .permissionDenied)
        )
        XCTAssertEqual(
            RealtimeReconnect.shouldReconnectWebSocket(targetActive: false),
            ReconnectDecision(allow: false, reason: .targetInactive)
        )
    }

    /// 票据错误仍保留会话与权限的终止边界。
    func testTicketErrorsPreserveSessionAndPermissionTerminalBoundaries() {
        XCTAssertEqual(
            RealtimeReconnect.classifyWebSocketFailure(RealtimeFailure(category: .session, statusCode: 401)),
            .sessionInvalid
        )
        XCTAssertEqual(
            RealtimeReconnect.classifyWebSocketFailure(RealtimeFailure(category: .forbidden, statusCode: 403)),
            .permissionDenied
        )
        XCTAssertEqual(
            RealtimeReconnect.classifyWebSocketFailure(RealtimeFailure(category: .network)),
            .recoverable
        )
        XCTAssertEqual(
            RealtimeReconnect.classifyWebSocketFailure(RealtimeFailure(statusCode: 503)),
            .recoverable
        )
        XCTAssertEqual(
            RealtimeReconnect.shouldReconnectWebSocket(
                error: RealtimeFailure(reason: "SESSION_INVALID")
            ),
            ReconnectDecision(allow: false, reason: .sessionInvalid)
        )
    }

    func testConnectionGenerationRejectsObsoleteCallbacks() {
        let first = RealtimeReconnect.nextConnectionGeneration()
        let second = RealtimeReconnect.nextConnectionGeneration(first)
        XCTAssertFalse(RealtimeReconnect.isCurrentConnectionGeneration(second, first))
        XCTAssertTrue(RealtimeReconnect.isCurrentConnectionGeneration(second, second))
    }

    func testAPIErrorMapsToFailureCategory() {
        let unauthorized = RealtimeFailure(error: APIError(message: "x", statusCode: 401))
        XCTAssertEqual(RealtimeReconnect.classifyWebSocketFailure(unauthorized), .sessionInvalid)

        let forbidden = RealtimeFailure(error: APIError(message: "x", statusCode: 403))
        XCTAssertEqual(RealtimeReconnect.classifyWebSocketFailure(forbidden), .permissionDenied)

        let network = RealtimeFailure(error: APIError(message: "x", kind: .network))
        XCTAssertEqual(RealtimeReconnect.classifyWebSocketFailure(network), .recoverable)
    }
}
