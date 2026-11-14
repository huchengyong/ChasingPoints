import Foundation

/// 好友模块接口门面，全部经由 `SessionStore.authorizedRequest` / `authorizedPost`。
@MainActor
struct FriendAPI {
    static let pageSize = 20
    static let homepagePageSize = 100

    let session: SessionStore

    func list(page: Int) async throws -> FriendListResponse {
        try await session.authorizedRequest(pagePath("/api/friend/list", page: page))
    }

    func requests(page: Int) async throws -> FriendRequestListResponse {
        try await session.authorizedRequest(pagePath("/api/friend/requests", page: page))
    }

    func blacklist(page: Int) async throws -> FriendBlacklistResponse {
        try await session.authorizedRequest(pagePath("/api/friend/blacklist/list", page: page))
    }

    func search(keyword: String) async throws -> SearchUserResponse {
        let path = APIQuery.path("/api/friend/search", ["keyword": keyword])
        return try await session.authorizedRequest(path)
    }

    func sendRequest(toUserId: Int64) async throws -> FriendCommonResponse {
        try await session.authorizedPost(
            "/api/friend/request",
            body: SendFriendRequestBody(toUserId: toUserId, message: "")
        )
    }

    func accept(requestId: Int64) async throws -> FriendCommonResponse {
        try await session.authorizedPost(
            "/api/friend/accept",
            body: FriendRequestActionBody(requestId: requestId)
        )
    }

    func reject(requestId: Int64) async throws -> FriendCommonResponse {
        try await session.authorizedPost(
            "/api/friend/reject",
            body: FriendRequestActionBody(requestId: requestId)
        )
    }

    func delete(friendUserId: Int64) async throws -> FriendCommonResponse {
        try await session.authorizedPost(
            "/api/friend/delete",
            body: FriendUserIdBody(friendUserId: friendUserId)
        )
    }

    func addToBlacklist(friendUserId: Int64) async throws -> FriendCommonResponse {
        try await session.authorizedPost(
            "/api/friend/blacklist",
            body: FriendUserIdBody(friendUserId: friendUserId)
        )
    }

    func removeFromBlacklist(friendUserId: Int64) async throws -> FriendCommonResponse {
        try await session.authorizedPost(
            "/api/friend/unblacklist",
            body: FriendUserIdBody(friendUserId: friendUserId)
        )
    }

    /// 好友主页直接复用目标模式的对手列表：一次拿到总场次、总胜场和战绩列表。
    func homepage(userId: Int64) async throws -> FriendHomepageData {
        guard userId > 0 else {
            throw APIError(message: "好友信息异常，请返回好友列表后重试。", statusCode: 404, reason: nil)
        }
        let path = APIQuery.path(
            "/api/opponent/list",
            OpponentRecordRules.requestParams(
                page: 1,
                pageSize: Self.homepagePageSize,
                keyword: "",
                targetUserId: userId
            )
        )
        let response: OpponentListResponse = try await session.authorizedRequest(path)
        let list = response.list ?? []
        let hidden = response.hidden || response.message == FriendHomepageRules.hiddenTitle
        let totalMatches = response.totalMatches > 0
            ? response.totalMatches
            : list.reduce(0) { $0 + $1.totalMatches }
        return FriendHomepageData(
            totalMatches: totalMatches,
            totalWins: response.totalWins,
            lastMatchAt: list.first?.lastMatchAt ?? "",
            hidden: hidden,
            opponents: hidden ? [] : list
        )
    }

    private func pagePath(_ path: String, page: Int) -> String {
        APIQuery.path(path, ["page": page, "page_size": Self.pageSize])
    }
}

private struct FriendUserIdBody: Encodable {
    let friendUserId: Int64
}

private struct FriendRequestActionBody: Encodable {
    let requestId: Int64
}

private struct SendFriendRequestBody: Encodable {
    let toUserId: Int64
    let message: String
}
