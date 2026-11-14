import Foundation

// 好友模块的窄解码模型，字段与 `backend/chasing_points.api` 好友段对齐。

struct FriendInfo: Decodable, Identifiable, Equatable {
    let id: Int64
    let userId: Int64
    let nickname: String
    let avatar: String
    let rankLevel: Int
    let rankName: String
    let vipLevel: Int
    let createdAt: String

    /// 好友的稳定用户 ID：后端 `FriendInfo` 同时给出关系记录 id 与 user_id。
    var friendUserId: Int64 { userId > 0 ? userId : id }
}

struct FriendListResponse: Decodable {
    let success: Bool
    let total: Int64
    let list: [FriendInfo]?
}

struct FriendRequestInfo: Decodable, Identifiable, Equatable {
    let id: Int64
    let fromUserId: Int64
    let nickname: String
    let avatar: String
    let message: String
    let status: Int
    let createdAt: String

    /// 申请人 ID 只在 `from_user_id`，拉黑时按同一规范取值。
    var applicantUserId: Int64 { fromUserId }
}

struct FriendRequestListResponse: Decodable {
    let success: Bool
    let total: Int64
    let list: [FriendRequestInfo]?
}

struct FriendBlacklistItem: Decodable, Identifiable, Equatable {
    let userId: Int64
    let nickname: String
    let avatar: String
    let createdAt: String

    /// 后端黑名单项只有 `user_id`，与好友列表归一成同一个字段。
    var friendUserId: Int64 { userId }
    var id: Int64 { userId }
}

struct FriendBlacklistResponse: Decodable {
    let success: Bool
    let total: Int64
    let list: [FriendBlacklistItem]?
}

struct SearchUserItem: Decodable, Identifiable, Equatable {
    let userId: Int64
    let nickname: String
    let avatar: String
    let rankName: String
    let isFriend: Bool
    let hasPendingRequest: Bool

    var id: Int64 { userId }
}

struct SearchUserResponse: Decodable {
    let success: Bool
    let list: [SearchUserItem]?
}

struct FriendCommonResponse: Decodable {
    let success: Bool
    let message: String?
}

/// 搜索结果里的按钮状态，对应小程序 `normalizeSearchResultItem` 的 status。
enum FriendSearchStatus: Equatable {
    case addable
    case pending
    case friend
}

/// 好友主页的身份信息来自好友列表，接口不返回目标用户自己的资料。
struct FriendHomepageTarget: Hashable {
    let userId: Int64
    let name: String
    let avatar: String
    let rankName: String
    let vipLevel: Int
    let rankLevel: Int
}

/// 好友主页一次性读取的战绩数据。
struct FriendHomepageData: Equatable {
    let totalMatches: Int
    let totalWins: Int
    let lastMatchAt: String
    let hidden: Bool
    let opponents: [OpponentRecord]
}
