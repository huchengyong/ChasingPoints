import Foundation
@testable import iOS

/// 排行榜与段位说明的虚构夹具，字段与 `backend/chasing_points.api` 一致。
enum RankingFixtures {
    // MARK: - 排行榜

    static func itemJSON(
        rank: Int,
        userId: Int64,
        nickname: String = "球手",
        avatar: String = "",
        rankLevel: Int = 3,
        rankName: String = "黄金球手",
        rankScore: Int = 1280,
        winRate: Int = 56,
        vipLevel: Int? = 0
    ) -> [String: Any] {
        var item: [String: Any] = [
            "rank": rank,
            "user_id": userId,
            "nickname": nickname,
            "avatar": avatar,
            "rank_level": rankLevel,
            "rank_name": rankName,
            "rank_score": rankScore,
            "win_rate": winRate,
        ]
        if let vipLevel {
            item["vip_level"] = vipLevel
        }
        return item
    }

    static func item(rank: Int, userId: Int64, nickname: String = "球手") -> LeaderboardItem {
        LeaderboardItem(
            rank: rank,
            userId: userId,
            nickname: nickname,
            avatar: "",
            rankLevel: 3,
            rankName: "黄金球手",
            rankScore: 1280,
            winRate: 56,
            vipLevel: 0
        )
    }

    static func leaderboard(
        topThree: [[String: Any]] = [],
        list: [[String: Any]] = [],
        total: Int,
        myRanking: [String: Any]? = nil
    ) -> MockHTTPServer.Stub {
        var payload: [String: Any] = [
            "success": true,
            "total": total,
            "top_three": topThree,
            "list": list,
        ]
        if let myRanking {
            payload["my_ranking"] = myRanking
        }
        return .json(payload)
    }

    /// 第 4 名起的榜单：`ranks` 为要生成的真实名次。
    static func listItems(_ ranks: ClosedRange<Int>) -> [[String: Any]] {
        ranks.map { rank in
            itemJSON(rank: rank, userId: Int64(1000 + rank), nickname: "球手\(rank)")
        }
    }

    // MARK: - 段位

    static func rankItemJSON(
        level: Int,
        name: String,
        icon: String,
        minScore: Int
    ) -> [String: Any] {
        [
            "game_type": 0,
            "level": level,
            "name": name,
            "icon": icon,
            "min_score": minScore,
        ]
    }

    static let rankItemListJSON: [[String: Any]] = [
        rankItemJSON(level: 1, name: "青铜球手", icon: "/static/images/ranks/rank_bronze.png", minScore: 0),
        rankItemJSON(level: 2, name: "白银球手", icon: "/static/images/ranks/rank_silver.png", minScore: 500),
        rankItemJSON(level: 3, name: "黄金球手", icon: "/static/images/ranks/rank_gold.png", minScore: 1000),
        rankItemJSON(level: 4, name: "铂金大师", icon: "/static/images/ranks/rank_platinum.png", minScore: 1500),
        rankItemJSON(level: 5, name: "钻石", icon: "/static/images/ranks/rank_diamond.png", minScore: 2000),
        rankItemJSON(level: 6, name: "王者", icon: "/static/images/ranks/rank_king.png", minScore: 2500),
    ]

    /// 与小程序的六级段位列表一致。
    static let rankList: [RankItem] = [
        RankItem(level: 1, name: "青铜球手", icon: "/static/images/ranks/rank_bronze.png", minScore: 0),
        RankItem(level: 2, name: "白银球手", icon: "/static/images/ranks/rank_silver.png", minScore: 500),
        RankItem(level: 3, name: "黄金球手", icon: "/static/images/ranks/rank_gold.png", minScore: 1000),
        RankItem(level: 4, name: "铂金大师", icon: "/static/images/ranks/rank_platinum.png", minScore: 1500),
        RankItem(level: 5, name: "钻石", icon: "/static/images/ranks/rank_diamond.png", minScore: 2000),
        RankItem(level: 6, name: "王者", icon: "/static/images/ranks/rank_king.png", minScore: 2500),
    ]

    static func rankInfoJSON(
        level: Int,
        name: String,
        icon: String = "/static/images/ranks/rank_gold.png",
        rankScore: Int,
        totalWins: Int = 10,
        totalLosses: Int = 5,
        maxStreak: Int = 4,
        nextLevel: Int,
        nextName: String,
        nextScore: Int,
        progress: Int,
        placementRemaining: Int? = nil
    ) -> [String: Any] {
        var info: [String: Any] = [
            "level": level,
            "name": name,
            "icon": icon,
            "rank_score": rankScore,
            "total_wins": totalWins,
            "total_losses": totalLosses,
            "max_streak": maxStreak,
            "next_level": nextLevel,
            "next_name": nextName,
            "next_score": nextScore,
            "progress": progress,
        ]
        if let placementRemaining {
            info["placement_remaining"] = placementRemaining
        }
        return info
    }

    static func rankInfo(
        level: Int,
        name: String,
        icon: String = "/static/images/ranks/rank_gold.png",
        rankScore: Int,
        nextLevel: Int = 4,
        nextName: String = "铂金大师",
        nextScore: Int = 1500,
        progress: Int = 56,
        placementRemaining: Int? = nil
    ) -> RankInfo {
        RankInfo(
            level: level,
            name: name,
            icon: icon,
            rankScore: rankScore,
            totalWins: 10,
            totalLosses: 5,
            maxStreak: 4,
            nextLevel: nextLevel,
            nextName: nextName,
            nextScore: nextScore,
            progress: progress,
            placementRemaining: placementRemaining
        )
    }

    static func goldRankInfo(placementRemaining: Int? = nil) -> RankInfo {
        rankInfo(level: 3, name: "黄金球手", rankScore: 1280, placementRemaining: placementRemaining)
    }

    static func kingRankInfo() -> RankInfo {
        rankInfo(
            level: 6,
            name: "王者",
            icon: "/static/images/ranks/rank_king.png",
            rankScore: 2780,
            nextLevel: 6,
            nextName: "王者",
            nextScore: 2500,
            progress: 100
        )
    }

    static func rankInfos(_ items: [(gameType: Int, info: [String: Any]?)]) -> MockHTTPServer.Stub {
        .json([
            "success": true,
            "rank_infos": items.map { item -> [String: Any] in
                [
                    "game_type": item.gameType,
                    "rank_info": item.info ?? NSNull(),
                ]
            },
        ])
    }

    static func configs(
        rulesVersion: Int,
        rulesEffectiveAt: String = "",
        list: [[String: Any]] = rankItemListJSON
    ) -> MockHTTPServer.Stub {
        .json([
            "success": true,
            "version": "1",
            "list": list,
            "rules_version": rulesVersion,
            "rules_effective_at": rulesEffectiveAt,
        ])
    }

    // MARK: - 会话

    @MainActor
    static func makeGuestSession(_ server: MockHTTPServer) -> SessionStore {
        makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
    }

    @MainActor
    static func makeSignedInSession(_ server: MockHTTPServer) async -> SessionStore {
        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
        await session.restore()
        return session
    }
}
