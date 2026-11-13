import Foundation

/// 赛讯列表与详情的读取入口。访客可读，有凭据时沿用会话的刷新与失效规则。
@MainActor
struct EventNewsAPI {
    let session: SessionStore

    func list(
        filter: EventNewsFilter,
        page: Int,
        pageSize: Int
    ) async throws -> (items: [EventNewsInfo], total: Int) {
        let path = APIQuery.path(
            "/api/event-news/list",
            SaiXunFilter.buildEventNewsListParams(filter, page: page, pageSize: pageSize)
        )
        let response: EventNewsListResponse = try await session.optionalAuthRequest(path)
        return (response.list, response.total)
    }

    func view(eventId: Int64) async throws -> EventNewsView {
        let path = APIQuery.path("/api/event-news/view", ["event_id": eventId])
        return try await session.optionalAuthRequest(path)
    }
}

private struct EventNewsListResponse: Decodable {
    let total: Int
    let list: [EventNewsInfo]
}
