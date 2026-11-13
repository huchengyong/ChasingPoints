import Foundation

/// 交锋记录的读取入口，全部经由 `SessionStore.authorizedRequest`。
@MainActor
struct H2HAPI {
    static let pageSize = 20

    let session: SessionStore

    /// 首屏概览：统计与当页历史一次返回，月份区间跟随当前选择的月份。
    func overview(
        targetUserId: Int64,
        opponentId: Int64,
        opponentName: String,
        monthKey: String
    ) async throws -> H2HOverviewResponse {
        let path = APIQuery.path(
            "/api/match/h2h/overview",
            params(
                page: 1,
                targetUserId: targetUserId,
                opponentId: opponentId,
                opponentName: opponentName,
                monthKey: monthKey
            )
        )
        return try await session.authorizedRequest(path)
    }

    func history(
        page: Int,
        targetUserId: Int64,
        opponentId: Int64,
        opponentName: String,
        monthKey: String
    ) async throws -> H2HHistoryResponse {
        let path = APIQuery.path(
            "/api/match/h2h/history",
            params(
                page: page,
                targetUserId: targetUserId,
                opponentId: opponentId,
                opponentName: opponentName,
                monthKey: monthKey
            )
        )
        return try await session.authorizedRequest(path)
    }

    /// 与小程序 `buildH2HParams` 一致：概览也带 `page`，服务端忽略该参数。
    private func params(
        page: Int,
        targetUserId: Int64,
        opponentId: Int64,
        opponentName: String,
        monthKey: String
    ) -> [String: CustomStringConvertible?] {
        let range = H2HRules.monthDateRange(monthKey)
        return H2HRules.historyParams(
            targetUserId: targetUserId,
            opponentId: opponentId,
            opponentName: opponentName,
            page: page,
            pageSize: Self.pageSize,
            startDate: range?.startDate ?? "",
            endDate: range?.endDate ?? ""
        )
    }
}
