import Foundation

/// PK 报表复用交锋概览接口，只读最近 5 场。
@MainActor
struct PKReportAPI {
    static let pageSize = 5

    let session: SessionStore

    func overview(opponentId: Int64, opponentName: String) async throws -> H2HOverviewResponse {
        var params: [String: CustomStringConvertible?] = ["page_size": Self.pageSize]
        if opponentId > 0 {
            params["opponent_id"] = opponentId
        } else if !opponentName.isEmpty {
            params["opponent_name"] = opponentName
        }
        return try await session.authorizedRequest(APIQuery.path("/api/match/h2h/overview", params))
    }
}
