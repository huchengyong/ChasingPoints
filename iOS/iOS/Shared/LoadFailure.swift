import Foundation

/// 读取失败的分类与界面提示，规则对应小程序 utils/async-page-state.js 的 resolveAsyncPageErrorFeedback。
struct LoadFailure: Equatable {
    enum Category: String {
        case network
        case session
        case permission
        case notFound = "not-found"
        case rateLimit = "rate-limit"
        case server
        case business
    }

    /// 原始错误文案。
    let category: Category
    let message: String

    /// 由任意错误得到分类：APIError 按 kind / statusCode 判断，其余错误归为 business。
    init(_ error: Error) {
        message = error.localizedDescription
        guard let apiError = error as? APIError else {
            category = .business
            return
        }
        if apiError.kind == .network {
            category = .network
        } else if apiError.kind == .sessionInvalid || apiError.kind == .unauthorized || apiError.statusCode == 401 {
            category = .session
        } else if apiError.statusCode == 403 {
            category = .permission
        } else if apiError.statusCode == 404 {
            category = .notFound
        } else if apiError.statusCode == 429 {
            category = .rateLimit
        } else if apiError.kind == .server || (apiError.statusCode ?? 0) >= 500 {
            category = .server
        } else {
            category = .business
        }
    }

    init(category: Category, message: String) {
        self.category = category
        self.message = message
    }

    /// 界面提示。resource 是资源名（如「规则内容」），用于「{resource}加载失败」。
    func feedback(resource: String = "内容") -> Feedback {
        let fallback = Feedback(
            title: "\(resource)加载失败",
            description: message.isEmpty ? "请稍后重试" : message,
            actionText: "重试",
            retryable: true
        )

        switch category {
        case .network:
            return Feedback(
                title: "网络连接失败",
                description: "请检查网络后重试",
                actionText: "重试",
                retryable: true
            )
        case .permission:
            return Feedback(
                title: "暂无访问权限",
                description: "你暂时无权查看这部分内容",
                actionText: "返回",
                retryable: false
            )
        case .notFound:
            return Feedback(
                title: "资源不存在或已下线",
                description: "请返回上一页选择其他内容",
                actionText: "返回",
                retryable: false
            )
        case .rateLimit:
            return Feedback(
                title: "请求过于频繁",
                description: "请稍后再试",
                actionText: "重试",
                retryable: true
            )
        case .server:
            return Feedback(
                title: "服务暂不可用",
                description: "请稍后重试",
                actionText: "重试",
                retryable: true
            )
        case .session:
            return Feedback(
                title: "登录状态已失效",
                description: "正在返回登录入口",
                actionText: "返回",
                retryable: false
            )
        case .business:
            return fallback
        }
    }

    struct Feedback: Equatable {
        let title: String
        let description: String
        let actionText: String
        let retryable: Bool
    }
}
