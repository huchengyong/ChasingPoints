import Foundation

/// 信誉明细、过往对手与交锋记录的虚构夹具，字段与 `backend/chasing_points.api` 及 logic 的实际响应一致。
/// 全部为假数据：昵称、头像与时间为占位，不包含真实账号或对局。
enum SubPageFixtures {
    static let matchTime = "2026-10-09T20:30:00+08:00"

    // MARK: - 信誉明细

    static func reputationLog(
        id: Int64,
        changeScore: Int = -2,
        reasonText: String = "异常比赛扣分",
        matchId: Int64? = 9
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "id": id,
            "change_type": "deduct",
            "change_type_text": "扣分",
            "reason_code": "abnormal_match",
            "reason_text": reasonText,
            "change_score": changeScore,
            "before_score": 100,
            "after_score": 100 + changeScore,
            "created_at": "2026-10-09 20:30:00",
        ]
        if let matchId { payload["match_id"] = matchId }
        return payload
    }

    static func reputationLogs(items: [[String: Any]], total: Int) -> [String: Any] {
        ["success": true, "total": total, "list": items]
    }

    // MARK: - 过往对手

    static func opponentItem(
        id: Int64,
        name: String = "阿杰",
        totalMatches: Int = 0,
        wins: Int = 7,
        losses: Int = 5,
        winRate: Double = 58.33,
        lastMatchAt: String = "2026-10-08 18:30:00"
    ) -> [String: Any] {
        [
            "id": id,
            "name": name,
            "avatar": "",
            "total_matches": totalMatches,
            "wins": wins,
            "losses": losses,
            "win_rate": winRate,
            "last_match_at": lastMatchAt,
        ]
    }

    static func opponentList(
        items: [[String: Any]],
        totalOpponents: Int = 4,
        totalWins: Int = 14,
        total: Int? = nil
    ) -> [String: Any] {
        [
            "success": true,
            "hidden": false,
            "total_opponents": totalOpponents,
            "total_matches": totalWins,
            "total_wins": totalWins,
            "total": total ?? items.count,
            "list": items,
        ]
    }

    // MARK: - 交锋记录

    static func h2hStats(
        totalMatches: Int = 8,
        myWins: Int = 5,
        opponentWins: Int = 3,
        avgScoreDiff: Double = 1.8,
        maxWinStreak: Int = 3
    ) -> [String: Any] {
        [
            "total_matches": totalMatches,
            "my_wins": myWins,
            "opponent_wins": opponentWins,
            "win_rate": 62.5,
            "avg_score_diff": avgScoreDiff,
            "max_win_streak": maxWinStreak,
        ]
    }

    static func h2hOpponent(id: Int64 = 18, name: String = "阿杰") -> [String: Any] {
        ["id": id, "name": name, "avatar": ""]
    }

    static func h2hOverview(
        hidden: Bool = false,
        success: Bool = true,
        message: String? = nil,
        opponent: Any? = nil,
        stats: Any? = nil,
        items: [[String: Any]] = [],
        total: Int? = nil,
        hasMore: Bool = false,
        availability: [String: Bool]? = nil
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "success": success,
            "hidden": hidden,
            "opponent": hidden ? NSNull() : (opponent ?? h2hOpponent()),
            "stats": hidden ? NSNull() : (stats ?? h2hStats()),
            "total": total ?? items.count,
            "list": hidden ? NSNull() : items,
            "has_more": hasMore,
            "competitive_revision": 12,
            "availability": availability ?? ["opponent": true, "stats": true, "history": true],
        ]
        if let message { payload["message"] = message }
        return payload
    }

    static func h2hHistory(
        items: [[String: Any]],
        total: Int? = nil,
        hidden: Bool = false,
        success: Bool = true,
        message: String? = nil
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "success": success,
            "hidden": hidden,
            "total": total ?? items.count,
            "list": items,
        ]
        if let message { payload["message"] = message }
        return payload
    }
}
