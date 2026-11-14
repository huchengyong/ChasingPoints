import Foundation

/// 投诉举报与意见反馈的分类与载荷，迁移自 `uniapp/utils/feedback-ticket.js`。
enum FeedbackCategory: String, CaseIterable, Identifiable {
    case feedback
    case complaint
    case report

    var id: String { rawValue }

    var label: String {
        switch self {
        case .feedback: "意见反馈"
        case .complaint: "投诉"
        case .report: "举报"
        }
    }

    /// 未知值回落到「意见反馈」。
    static func normalized(_ value: String) -> FeedbackCategory {
        FeedbackCategory(rawValue: value.trimmingCharacters(in: .whitespacesAndNewlines)) ?? .feedback
    }
}

struct FeedbackTicketPayload: Encodable, Equatable {
    let source: String
    let category: String
    let content: String
    let contact: String
}

enum FeedbackRules {
    static let maxContentLength = 1000

    static func payload(category: FeedbackCategory, content: String, contact: String) -> FeedbackTicketPayload {
        FeedbackTicketPayload(
            source: "app",
            category: category.rawValue,
            content: content.trimmingCharacters(in: .whitespacesAndNewlines),
            contact: contact.trimmingCharacters(in: .whitespacesAndNewlines)
        )
    }
}
