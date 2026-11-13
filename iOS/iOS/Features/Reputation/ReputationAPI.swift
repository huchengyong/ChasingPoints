import Foundation

/// 信誉明细的读取入口，全部经由 `SessionStore.authorizedRequest`。
@MainActor
struct ReputationAPI {
    static let pageSize = 20

    let session: SessionStore

    /// 页面只用到概览里的信誉分区；分区不可用时与小程序一样按业务失败处理。
    func status() async throws -> ReputationStatus {
        let overview: UserOverview = try await session.authorizedRequest("/api/user/overview")
        guard let reputation = overview.availableReputation else {
            throw APIError(message: "信誉状态暂时不可用", kind: .business)
        }
        return reputation
    }

    func logsPage(page: Int) async throws -> (items: [ReputationLog], total: Int) {
        let path = APIQuery.path(
            "/api/user/reputation/logs",
            ["page": page, "page_size": Self.pageSize]
        )
        let response: ReputationLogsResponse = try await session.authorizedRequest(path)
        return (response.list ?? [], response.total)
    }
}
