import Foundation

/// 「我的」页的读取入口，全部经由 `SessionStore.authorizedRequest`。
@MainActor
struct ProfileAPI {
    let session: SessionStore

    func overview() async throws -> UserOverview {
        try await session.authorizedRequest("/api/user/overview")
    }

    func rankInfos() async throws -> RankInfoMap {
        let response: RankInfosResponse = try await session.authorizedRequest("/api/rank/infos")
        return response.map
    }

    func matchQRCode() async throws -> MatchQRCode {
        let response: MatchQRCode = try await session.authorizedRequest("/api/match/qrcode")
        guard !response.qrcodeData.isEmpty else {
            throw APIError(message: "二维码加载失败", kind: .invalidResponse)
        }
        return response
    }
}
