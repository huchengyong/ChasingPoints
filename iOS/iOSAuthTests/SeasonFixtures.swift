import Foundation

/// 赛季接口的虚构夹具，字段与 `backend/chasing_points.api` 的赛季模块一致。
enum SeasonFixtures {
    static func season(
        id: Int64 = 2,
        name: String = "S3",
        startDate: String = "2026-07-01",
        endDate: String = "2026-09-30",
        startAt: String = "2026-07-01T00:00:00+08:00",
        endAtExclusive: String = "2026-10-01T00:00:00+08:00",
        status: Int = 1,
        qualifyMinMatches: Int = 0,
        qualifyMinOpponents: Int = 0
    ) -> [String: Any] {
        [
            "id": id,
            "name": name,
            "start_date": startDate,
            "end_date": endDate,
            "start_at": startAt,
            "end_at_exclusive": endAtExclusive,
            "status": status,
            "rank_reset_ratio": 1,
            "qualify_min_matches": qualifyMinMatches,
            "qualify_min_opponents": qualifyMinOpponents,
        ]
    }

    static func record(
        seasonId: Int64 = 2,
        seasonName: String = "S3",
        startRankScore: Int = 1250,
        endRankScore: Int = 1310,
        peakRankScore: Int = 1320,
        matchesPlayed: Int = 8,
        wins: Int = 6,
        finalRank: Int = 3,
        distinctOpponents: Int = 5,
        qualified: Bool = true
    ) -> [String: Any] {
        [
            "season_id": seasonId,
            "season_name": seasonName,
            "start_rank_score": startRankScore,
            "end_rank_score": endRankScore,
            "peak_rank_score": peakRankScore,
            "matches_played": matchesPlayed,
            "wins": wins,
            "final_rank": finalRank,
            "distinct_opponents": distinctOpponents,
            "qualified": qualified,
        ]
    }

    static func leaderboardItem(
        rank: Int,
        userId: Int64,
        nickname: String = "球友",
        avatar: String = "",
        rankScore: Int = 1400,
        peakScore: Int = 1450,
        matchesPlayed: Int = 10,
        wins: Int = 7,
        qualified: Bool = true
    ) -> [String: Any] {
        [
            "rank": rank,
            "user_id": userId,
            "nickname": nickname,
            "avatar": avatar,
            "rank_score": rankScore,
            "peak_score": peakScore,
            "matches_played": matchesPlayed,
            "wins": wins,
            "qualified": qualified,
        ]
    }

    static func current(
        seasonState: String = "active",
        season: [String: Any]? = season()
    ) -> [String: Any] {
        [
            "success": true,
            "season_state": seasonState,
            "season": season ?? NSNull(),
        ]
    }

    static func leaderboard(
        total: Int64 = 1,
        list: [[String: Any]] = [leaderboardItem(rank: 1, userId: 7)]
    ) -> [String: Any] {
        [
            "success": true,
            "season": season(),
            "total": total,
            "list": list,
        ]
    }

    static func overview(
        seasonState: String = "active",
        season: [String: Any]? = season(),
        record: [String: Any]? = record(),
        myRank: Int = 3,
        leaderboard: [[String: Any]] = [leaderboardItem(rank: 1, userId: 7)],
        availability: [String: Bool] = [:]
    ) -> [String: Any] {
        [
            "success": true,
            "season_state": seasonState,
            "season": season ?? NSNull(),
            "record": record ?? NSNull(),
            "my_rank": myRank,
            "leaderboard": leaderboard,
            "competitive_revision": 1,
            "availability": availability,
        ]
    }

    static func report(
        record: [String: Any]? = record(),
        rankTrend: [Int] = [1250, 1280, 1310],
        winRateByType: [String: Int] = ["1": 2, "3": 6],
        topAchievements: [[String: Any]] = []
    ) -> [String: Any] {
        [
            "success": true,
            "report": [
                "record": record ?? NSNull(),
                "rank_trend": rankTrend,
                "win_rate_by_type": winRateByType,
                "top_achievements": topAchievements,
            ],
        ]
    }

    static let emptyReport: [String: Any] = ["success": true, "report": NSNull()]

    static func historyItem(
        season: [String: Any]? = season(id: 2, name: "S2", status: 2),
        record: [String: Any]? = record(seasonId: 2, seasonName: "S2", finalRank: 3)
    ) -> [String: Any] {
        [
            "season": season ?? NSNull(),
            "record": record ?? NSNull(),
        ]
    }

    static func history(
        total: Int64 = 1,
        list: [[String: Any]] = [historyItem()]
    ) -> [String: Any] {
        [
            "success": true,
            "total": total,
            "list": list,
        ]
    }

    static let achievement: [String: Any] = [
        "id": 5,
        "key": "season_wins_10",
        "name": "赛季十胜",
        "description": "单赛季赢下 10 场",
        "icon": "",
        "category": "season",
        "game_type": 3,
        "threshold": 10,
        "progress": 10,
        "unlocked": true,
        "unlocked_at": "2026-09-20 12:00:00",
    ]
}
