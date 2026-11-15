import Foundation

/// 观赛大厅读取入口；大厅公开可看，好友维度需要登录后由服务端按当前用户过滤。
@MainActor
struct SpectatorLobbyAPI {
    let session: SessionStore

    func matches(
        scope: String,
        status: Int,
        gameType: Int,
        page: Int
    ) async throws -> SpectatorMatchPage {
        let params = SpectatorLobbyRules.listParams(
            scope: scope,
            status: status,
            gameType: gameType,
            page: page,
            pageSize: SpectatorLobbyRules.pageSize
        )
        var query: [String: CustomStringConvertible?] = [:]
        for (key, value) in params {
            query[key] = value
        }
        let response: SpectatorMatchListResponse = try await session.optionalAuthRequest(
            APIQuery.path("/api/public/matches", query)
        )
        return SpectatorMatchPage(items: response.list, total: response.total)
    }
}

private struct SpectatorMatchListResponse: Decodable {
    let total: Int
    let list: [SpectatorMatch]
}
