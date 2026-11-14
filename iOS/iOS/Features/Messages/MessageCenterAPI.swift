import Foundation

/// 通知接口门面，路径与字段以 `backend/chasing_points.api` 的通知模块为准。
@MainActor
struct MessageCenterAPI {
    let session: SessionStore

    static let pageSize = 20

    func list(page: Int) async throws -> NotificationListResponse {
        let path = APIQuery.path("/api/notification/list", [
            "page": String(page),
            "page_size": String(Self.pageSize),
        ])
        return try await session.authorizedRequest(path)
    }

    func markRead(id: Int64) async throws {
        let _: CommonResponse = try await session.authorizedPost(
            "/api/notification/read",
            body: NotificationIdRequest(notificationId: id)
        )
    }

    func markAllRead() async throws {
        let _: CommonResponse = try await session.authorizedPost(
            "/api/notification/read-all",
            body: EmptyRequest()
        )
    }

    func delete(id: Int64) async throws {
        let _: CommonResponse = try await session.authorizedPost(
            "/api/notification/delete",
            body: NotificationIdRequest(notificationId: id)
        )
    }
}

struct NotificationListResponse: Decodable {
    let success: Bool
    let total: Int64
    let unreadCount: Int
    let list: [NotificationInfo]
}

private struct NotificationIdRequest: Encodable {
    let notificationId: Int64
}

private struct EmptyRequest: Encodable {}

private struct CommonResponse: Decodable {
    let success: Bool
}
