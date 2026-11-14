import Foundation

/// 约球模块的虚构夹具，字段与 `backend/chasing_points.api` 约球段一致。
/// 全部为占位数据，不含真实账号或会话。
enum ChallengeFixtures {
    static func challenge(
        id: Int64,
        status: Int = 0,
        matchId: Int64 = 0,
        fromUserId: Int64 = 18,
        fromNickname: String = "我",
        fromAvatar: String = "",
        fromVipLevel: Int = 0,
        toUserId: Int64 = 29,
        toNickname: String = "球友A",
        toAvatar: String = "",
        toVipLevel: Int = 0,
        gameType: Int = 3,
        matchMode: String = "ranked",
        message: String = "",
        closeReason: String = "",
        waitingUserId: Int64? = nil,
        scheduledDate: String? = "2026-08-25",
        startHour: Int? = 16,
        endHour: Int? = 17,
        expiresAt: String = "2026-08-26 07:00:00",
        createdAt: String = "2026-08-25 12:00:00"
    ) -> [String: Any] {
        var value: [String: Any] = [
            "id": id,
            "match_id": matchId,
            "from_user_id": fromUserId,
            "from_nickname": fromNickname,
            "from_avatar": fromAvatar,
            "from_vip_level": fromVipLevel,
            "to_user_id": toUserId,
            "to_nickname": toNickname,
            "to_avatar": toAvatar,
            "to_vip_level": toVipLevel,
            "game_type": gameType,
            "match_mode": matchMode,
            "message": message,
            "status": status,
            "close_reason": closeReason,
            "expires_at": expiresAt,
            "created_at": createdAt,
        ]
        if let waitingUserId { value["waiting_user_id"] = waitingUserId }
        if let scheduledDate { value["scheduled_date"] = scheduledDate }
        if let startHour { value["start_hour"] = startHour }
        if let endHour { value["end_hour"] = endHour }
        return value
    }

    static func pendingList(_ items: [[String: Any]]) -> [String: Any] {
        ["success": true, "list": items]
    }

    static func historyList(_ items: [[String: Any]], nextBeforeId: Int64 = 0) -> [String: Any] {
        ["success": true, "list": items, "next_before_id": nextBeforeId]
    }

    static func summary(
        serverTime: String = "2026-10-11 12:00:00",
        currentChallenge: [String: Any]? = nil,
        receivedPendingCount: Int = 0
    ) -> [String: Any] {
        [
            "success": true,
            "current_challenge": currentChallenge ?? NSNull(),
            "received_pending_count": receivedPendingCount,
            "server_time": serverTime,
        ]
    }

    static func detail(_ challenge: [String: Any]?) -> [String: Any] {
        ["success": true, "challenge": challenge ?? NSNull()]
    }

    static func sendResponse(challengeId: Int64, message: String = "") -> [String: Any] {
        ["success": true, "challenge_id": challengeId, "message": message]
    }

    static func acceptResponse(challengeId: Int64, matchId: Int64 = 0, message: String = "") -> [String: Any] {
        ["success": true, "challenge_id": challengeId, "match_id": matchId, "message": message]
    }

    static func actionResponse(message: String = "", matchId: Int64 = 0) -> [String: Any] {
        ["success": true, "message": message, "match_id": matchId]
    }

    static func enterResponse(action: String, matchId: Int64 = 0, message: String = "") -> [String: Any] {
        ["success": true, "action": action, "match_id": matchId, "message": message]
    }

    static func businessFailure(_ message: String) -> [String: Any] {
        ["success": false, "message": message]
    }
}
