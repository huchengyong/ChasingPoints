import Foundation

/// 好友模块的虚构夹具，字段与 `backend/chasing_points.api` 好友段一致。
/// 全部为假数据：昵称、头像与时间都是占位，不包含真实账号。
enum FriendFixtures {
    static func friend(
        id: Int64,
        userId: Int64,
        nickname: String = "球友",
        avatar: String = "",
        rankLevel: Int = 0,
        rankName: String = "",
        vipLevel: Int = 0,
        createdAt: String = "2026-10-01 12:00:00"
    ) -> [String: Any] {
        [
            "id": id,
            "user_id": userId,
            "nickname": nickname,
            "avatar": avatar,
            "rank_level": rankLevel,
            "rank_name": rankName,
            "vip_level": vipLevel,
            "created_at": createdAt,
        ]
    }

    static func friendList(_ items: [[String: Any]], total: Int? = nil) -> [String: Any] {
        ["success": true, "total": total ?? items.count, "list": items]
    }

    static func request(
        id: Int64,
        fromUserId: Int64,
        nickname: String = "球友",
        avatar: String = "",
        message: String = "",
        status: Int = 0,
        createdAt: String = "2026-10-08 12:00:00"
    ) -> [String: Any] {
        [
            "id": id,
            "from_user_id": fromUserId,
            "nickname": nickname,
            "avatar": avatar,
            "message": message,
            "status": status,
            "created_at": createdAt,
        ]
    }

    static func requestList(_ items: [[String: Any]], total: Int? = nil) -> [String: Any] {
        ["success": true, "total": total ?? items.count, "list": items]
    }

    static func blacklistItem(
        userId: Int64,
        nickname: String = "球友",
        avatar: String = "",
        createdAt: String = "2026-10-08 12:00:00"
    ) -> [String: Any] {
        [
            "user_id": userId,
            "nickname": nickname,
            "avatar": avatar,
            "created_at": createdAt,
        ]
    }

    static func blacklistList(_ items: [[String: Any]], total: Int? = nil) -> [String: Any] {
        ["success": true, "total": total ?? items.count, "list": items]
    }

    static func searchItem(
        userId: Int64,
        nickname: String = "球友",
        avatar: String = "",
        rankName: String = "",
        isFriend: Bool = false,
        hasPendingRequest: Bool = false
    ) -> [String: Any] {
        [
            "user_id": userId,
            "nickname": nickname,
            "avatar": avatar,
            "rank_name": rankName,
            "is_friend": isFriend,
            "has_pending_request": hasPendingRequest,
        ]
    }

    static func searchList(_ items: [[String: Any]]) -> [String: Any] {
        ["success": true, "list": items]
    }

    static func bootstrap(pendingFriend: Int = 0) -> [String: Any] {
        [
            "success": true,
            "current_match": NSNull(),
            "current_challenge": NSNull(),
            "received_pending_challenge_count": 0,
            "server_time": "2026-10-11 12:00:00",
            "unread_count": 0,
            "pending_friend_request_count": pendingFriend,
            "latest_season_rollover": NSNull(),
            "availability": ActivityFixtures.allAvailable,
        ]
    }
}
