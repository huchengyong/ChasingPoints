import Foundation

/// 竞技分析与表现分析的虚构夹具，字段与 `backend/chasing_points.api` 一致；全部为假数据。
enum StatsFixtures {
    static func gameTypeStats(
        gameType: Int,
        name: String = "中式八球",
        totalMatches: Int = 42,
        wins: Int = 26,
        losses: Int = 16,
        winRate: Double = 61.9,
        highestScore: Int = 87
    ) -> [String: Any] {
        [
            "game_type": gameType,
            "game_type_name": name,
            "total_matches": totalMatches,
            "wins": wins,
            "losses": losses,
            "win_rate": winRate,
            "highest_score": highestScore,
        ]
    }

    static func trendPoint(
        matchId: Int64,
        date: String,
        winRate: Double = 50,
        rankScore: Int? = nil,
        result: Int = 1
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "match_id": matchId,
            "date": date,
            "win_rate": winRate,
            "result": result,
        ]
        if let rankScore { payload["rank_score"] = rankScore }
        return payload
    }

    static func highScore(
        matchId: Int64,
        score: Int,
        gameType: Int = 3,
        name: String = "中式八球",
        opponent: String = "阿杰",
        date: String
    ) -> [String: Any] {
        [
            "match_id": matchId,
            "score": score,
            "game_type": gameType,
            "game_type_name": name,
            "opponent_name": opponent,
            "date": date,
        ]
    }

    static func overview(
        success: Bool = true,
        byGameType: [[String: Any]] = [gameTypeStats(gameType: 3)],
        recentTrend: [[String: Any]] = [
            trendPoint(matchId: 1, date: "2026-04-03", result: 1),
            trendPoint(matchId: 2, date: "2026-04-03", result: 1),
            trendPoint(matchId: 3, date: "2026-04-03", result: 2),
            trendPoint(matchId: 4, date: "2026-04-11", result: 2),
        ],
        rankScoreTrend: [[String: Any]] = [
            trendPoint(matchId: 4, date: "2026-04-11", rankScore: 1040, result: 2),
            trendPoint(matchId: 3, date: "2026-04-03", rankScore: 1055, result: 2),
            trendPoint(matchId: 2, date: "2026-04-02", rankScore: 1030, result: 1),
            trendPoint(matchId: 1, date: "2026-03-30", rankScore: 1000, result: 1),
        ],
        singleHighScores: [[String: Any]] = [
            highScore(matchId: 1, score: 87, date: "2026-04-08"),
            highScore(matchId: 2, score: 72, date: "2026-03-30"),
        ],
        duration: [String: Any]? = [
            "average_seconds": 125,
            "fastest_seconds": 40,
            "longest_seconds": 3720,
            "total_matches": 42,
        ],
        opponentStrength: [[String: Any]] = [
            ["rank_range": "高级(2001以上)", "matches": 12, "wins": 9, "win_rate": 75],
            ["rank_range": "中级(1001-2000)", "matches": 25, "wins": 22, "win_rate": 88],
            ["rank_range": "初级(0-1000)", "matches": 5, "wins": 1, "win_rate": 20],
        ],
        availability: [String: Bool]? = [
            "by_game_type": true,
            "recent_trend": true,
            "rank_score_trend": true,
            "single_high_scores": true,
            "duration": true,
            "opponent_strength": true,
        ]
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "success": success,
            "by_game_type": byGameType,
            "recent_trend": recentTrend,
            "rank_score_trend": rankScoreTrend,
            "single_high_scores": singleHighScores,
            "duration": duration.map { $0 as Any } ?? NSNull(),
            "opponent_strength": opponentStrength,
            "availability": availability.map { $0 as Any } ?? NSNull(),
        ]
        if success { payload["competitive_revision"] = 12 }
        return payload
    }

    // MARK: - 表现分析

    static func rating(_ rating: Int, bound: String? = nil) -> [String: Any] {
        var payload: [String: Any] = ["rating": rating]
        if let bound { payload["bound"] = bound }
        return payload
    }

    static func summary(
        wins: Int = 14,
        expectedWins: Double = 11.2,
        winsDiff: Double = 2.8,
        rating: [String: Any]? = rating(1720)
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "wins": wins,
            "expected_wins": expectedWins,
            "wins_diff": winsDiff,
        ]
        payload["rating"] = rating.map { $0 as Any } ?? NSNull()
        return payload
    }

    static func monthPreview(
        month: String = "2026-04",
        sampleCount: Int = 6,
        rating: [String: Any]? = rating(1678)
    ) -> [String: Any] {
        var payload: [String: Any] = ["month": month, "sample_count": sampleCount]
        payload["rating"] = rating.map { $0 as Any } ?? NSNull()
        return payload
    }

    static func matchItem(
        matchId: Int64,
        opponentName: String = "老王",
        beforeScore: Int = 1800,
        winRate: Double = 23.5,
        playedAt: String = "2026-04-09 20:30:00"
    ) -> [String: Any] {
        [
            "match_id": matchId,
            "opponent_name": opponentName,
            "opponent_avatar": "",
            "opponent_before_score": beforeScore,
            "expected_win_rate": winRate,
            "played_at": playedAt,
        ]
    }

    static func monthItem(
        month: String,
        sampleCount: Int,
        rating: [String: Any]?,
        endRankScore: Int,
        incomplete: Bool? = nil
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "month": month,
            "sample_count": sampleCount,
            "rating": rating.map { $0 as Any } ?? NSNull(),
            "end_rank_score": endRankScore,
        ]
        if let incomplete { payload["incomplete"] = incomplete }
        return payload
    }

    static func performance(
        locked: Bool = false,
        gameType: Int = 3,
        sampleCount: Int = 20,
        monthPreview: [String: Any]? = nil,
        summary: [String: Any]? = nil,
        bestWin: [String: Any]? = nil,
        worstLoss: [String: Any]? = nil,
        months: [[String: Any]]? = nil
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "success": true,
            "game_type": gameType,
            "locked": locked,
            "sample_count": sampleCount,
        ]
        payload["month_preview"] = monthPreview.map { $0 as Any } ?? NSNull()
        payload["summary"] = summary.map { $0 as Any } ?? NSNull()
        payload["best_win"] = bestWin.map { $0 as Any } ?? NSNull()
        payload["worst_loss"] = worstLoss.map { $0 as Any } ?? NSNull()
        payload["months"] = months.map { $0 as Any } ?? NSNull()
        return payload
    }

    static func defaultMonths() -> [[String: Any]] {
        [
            monthItem(month: "2025-11", sampleCount: 0, rating: nil, endRankScore: 0),
            monthItem(month: "2025-12", sampleCount: 8, rating: rating(1500), endRankScore: 1480),
            monthItem(month: "2026-01", sampleCount: 6, rating: rating(1700), endRankScore: 1620),
            monthItem(month: "2026-02", sampleCount: 2, rating: nil, endRankScore: 1630),
            monthItem(month: "2026-03", sampleCount: 5, rating: rating(1900, bound: "at_least"), endRankScore: 1700, incomplete: false),
            monthItem(month: "2026-04", sampleCount: 4, rating: rating(1600), endRankScore: 1710),
        ]
    }
}
