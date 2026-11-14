import Foundation

/// `CreateFeedbackTicketResp`。
struct CreateFeedbackTicketResponse: Decodable, Equatable {
    let success: Bool
    let message: String?
    let ticketId: Int64?
}

/// 反馈提交入口：已登录走用户接口，访客走公开接口。
@MainActor
struct FeedbackAPI {
    let session: SessionStore

    func create(_ payload: FeedbackTicketPayload) async throws -> CreateFeedbackTicketResponse {
        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase
        let body = try encoder.encode(payload)
        if session.identity != nil {
            return try await session.authorizedRequest("/api/user/feedback/create", method: .post, body: body)
        }
        return try await session.optionalAuthRequest("/api/feedback/create", method: .post, body: body)
    }
}
