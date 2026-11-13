import Foundation

/// 过往对手的读取入口，全部经由 `SessionStore.authorizedRequest`。
@MainActor
struct OpponentAPI {
    static let pageSize = 20

    let session: SessionStore

    func page(page: Int, keyword: String, targetUserId: Int64) async throws -> OpponentListResponse {
        let path = APIQuery.path(
            "/api/opponent/list",
            OpponentRecordRules.requestParams(
                page: page,
                pageSize: Self.pageSize,
                keyword: keyword,
                targetUserId: targetUserId
            )
        )
        return try await session.authorizedRequest(path)
    }
}
