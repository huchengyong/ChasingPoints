import Foundation
import XCTest
@testable import iOS

@MainActor
final class LoadFailureTests: XCTestCase {
    func testCategoryFromAPIErrorKindAndStatusCode() {
        XCTAssertEqual(LoadFailure(APIError(message: "断网", kind: .network)).category, .network)
        XCTAssertEqual(LoadFailure(APIError(message: "失效", kind: .sessionInvalid)).category, .session)
        XCTAssertEqual(LoadFailure(APIError(message: "未登录", kind: .unauthorized)).category, .session)
        XCTAssertEqual(LoadFailure(APIError(message: "401", statusCode: 401)).category, .session)
        XCTAssertEqual(LoadFailure(APIError(message: "403", statusCode: 403)).category, .permission)
        XCTAssertEqual(LoadFailure(message: "404", statusCode: 404).category, .notFound)
        XCTAssertEqual(LoadFailure(APIError(message: "429", statusCode: 429)).category, .rateLimit)
        XCTAssertEqual(LoadFailure(APIError(message: "503", statusCode: 503)).category, .server)
        XCTAssertEqual(LoadFailure(APIError(message: "挂了", kind: .server)).category, .server)
        XCTAssertEqual(LoadFailure(APIError(message: "业务", statusCode: 400)).category, .business)
        XCTAssertEqual(LoadFailure(APIError(message: "格式", kind: .invalidResponse)).category, .business)
        XCTAssertEqual(LoadFailure(NSError(domain: "test", code: 1)).category, .business)
    }

    func testFeedbackMatchesAsyncPageErrorFeedback() {
        XCTAssertEqual(
            LoadFailure(category: .network, message: "断网").feedback(),
            .init(title: "网络连接失败", description: "请检查网络后重试", actionText: "重试", retryable: true)
        )
        XCTAssertEqual(
            LoadFailure(category: .permission, message: "403").feedback(),
            .init(title: "暂无访问权限", description: "你暂时无权查看这部分内容", actionText: "返回", retryable: false)
        )
        XCTAssertEqual(
            LoadFailure(category: .notFound, message: "404").feedback(),
            .init(title: "资源不存在或已下线", description: "请返回上一页选择其他内容", actionText: "返回", retryable: false)
        )
        XCTAssertEqual(
            LoadFailure(category: .rateLimit, message: "429").feedback(),
            .init(title: "请求过于频繁", description: "请稍后再试", actionText: "重试", retryable: true)
        )
        XCTAssertEqual(
            LoadFailure(category: .server, message: "503").feedback(),
            .init(title: "服务暂不可用", description: "请稍后重试", actionText: "重试", retryable: true)
        )
        XCTAssertEqual(
            LoadFailure(category: .session, message: "401").feedback(),
            .init(title: "登录状态已失效", description: "正在返回登录入口", actionText: "返回", retryable: false)
        )
        XCTAssertEqual(
            LoadFailure(category: .business, message: "原始文案").feedback(),
            .init(title: "内容加载失败", description: "原始文案", actionText: "重试", retryable: true)
        )
    }

    func testBusinessFeedbackUsesResourceAndFallbackDescription() {
        XCTAssertEqual(
            LoadFailure(category: .business, message: "原始文案").feedback(resource: "规则内容"),
            .init(title: "规则内容加载失败", description: "原始文案", actionText: "重试", retryable: true)
        )
        XCTAssertEqual(
            LoadFailure(category: .business, message: "").feedback(resource: "规则搜索"),
            .init(title: "规则搜索加载失败", description: "请稍后重试", actionText: "重试", retryable: true)
        )
    }
}

private extension LoadFailure {
    init(message: String, statusCode: Int) {
        self.init(APIError(message: message, statusCode: statusCode))
    }
}
