import Foundation

struct LinkEntryPreviewRequest: Encodable, Equatable {
    let inviteCode: String
}

/// `PublicMatchInvitePreviewInfo`：访客在兑换前只能看到邀请人的公开资料。
struct LinkEntryPreviewInfo: Decodable, Equatable {
    let inviterName: String
    let inviterAvatar: String
    let expiresAt: String
}

/// `PublicMatchInvitePreviewResp`。
struct LinkEntryPreviewResponse: Decodable, Equatable {
    let success: Bool
    let message: String?
    let preview: LinkEntryPreviewInfo?
}

/// 公开预览接口；访客也能调用，登录态下仍走统一的凭证与失效规则。
@MainActor
struct LinkEntryAPI {
    let session: SessionStore

    func preview(_ inviteCode: String) async throws -> LinkEntryPreviewResponse {
        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase
        let body = try encoder.encode(
            LinkEntryPreviewRequest(inviteCode: inviteCode.trimmingCharacters(in: .whitespacesAndNewlines))
        )
        return try await session.optionalAuthRequest(
            "/api/public/match-invite/preview",
            method: .post,
            body: body
        )
    }
}
