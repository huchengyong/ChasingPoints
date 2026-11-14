import Combine
import Foundation

/// 帮助、投诉与举报表单，对应小程序 `subPages/help/feedback.vue`。
@MainActor
final class FeedbackModel: ObservableObject {
    enum Outcome: Equatable {
        case success
        case validation(String)
        case failure(String)
        /// 身份或页面已变化，迟到的结果不提示。
        case discarded
    }

    @Published var category: FeedbackCategory = .feedback
    @Published var content = "" {
        didSet {
            if content.count > FeedbackRules.maxContentLength {
                content = String(content.prefix(FeedbackRules.maxContentLength))
            }
            validationMessage = nil
        }
    }
    @Published var contact = ""
    @Published private(set) var isSubmitting = false
    @Published private(set) var validationMessage: String?

    private let session: SessionStore
    private let api: FeedbackAPI

    init(session: SessionStore, api: FeedbackAPI? = nil) {
        self.session = session
        self.api = api ?? FeedbackAPI(session: session)
    }

    var canSubmit: Bool { !isSubmitting }

    func submit() async -> Outcome {
        guard !isSubmitting else { return .discarded }
        let payload = FeedbackRules.payload(category: category, content: content, contact: contact)
        guard !payload.content.isEmpty else {
            validationMessage = "请输入投诉举报或反馈内容"
            return .validation("请输入投诉举报或反馈内容")
        }

        let owner = session.identity
        isSubmitting = true
        do {
            let response = try await api.create(payload)
            guard session.identity == owner else { return .discarded }
            isSubmitting = false
            guard response.success else {
                let message = response.message ?? ""
                return .failure(message.isEmpty ? "提交失败，请稍后重试" : message)
            }
            reset()
            return .success
        } catch {
            guard session.identity == owner else { return .discarded }
            isSubmitting = false
            let message = (error as? APIError)?.message ?? ""
            return .failure(message.isEmpty ? "提交失败，请稍后重试" : message)
        }
    }

    private func reset() {
        content = ""
        contact = ""
        category = .feedback
        validationMessage = nil
    }
}
