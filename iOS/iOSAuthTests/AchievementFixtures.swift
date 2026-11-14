import Foundation

/// 荣誉墙、成就详情与称号的虚构夹具，字段与 `backend/chasing_points.api` 一致。
/// 全部为假数据：昵称、头像与时间为占位，不包含真实账号或对局。
enum AchievementFixtures {
    static let earnedAt = "2026-09-01 12:00:00"

    // MARK: - 基础条目

    static func achievement(
        id: Int64,
        key: String = "achievement",
        name: String = "成就",
        description: String = "完成目标即可永久点亮",
        icon: String = "/static/images/achievements/season_matches_20.png",
        category: String = "wins",
        gameType: Int = 0,
        rewardTitleName: String? = nil,
        threshold: Int = 10,
        progress: Int = 0,
        unlocked: Bool = false,
        unlockedAt: String? = nil
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "id": id,
            "key": key,
            "name": name,
            "description": description,
            "icon": icon,
            "category": category,
            "game_type": gameType,
            "threshold": threshold,
            "progress": progress,
            "unlocked": unlocked,
        ]
        if let rewardTitleName { payload["reward_title_name"] = rewardTitleName }
        if let unlockedAt { payload["unlocked_at"] = unlockedAt }
        return payload
    }

    static func honorItem(
        id: Int64,
        name: String = "十胜起步",
        description: String = "完成 10 场有效对局",
        icon: String = "/static/images/achievements/season_matches_20.png",
        sourceType: String = "achievement",
        sourceRefId: Int64 = 0,
        sourceRefName: String = "",
        rewardTitleName: String? = nil,
        earnedAt: String = AchievementFixtures.earnedAt
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "id": id,
            "type": "achievement",
            "name": name,
            "description": description,
            "icon": icon,
            "source_type": sourceType,
            "earned_at": earnedAt,
        ]
        if sourceRefId > 0 { payload["source_ref_id"] = sourceRefId }
        if !sourceRefName.isEmpty { payload["source_ref_name"] = sourceRefName }
        if let rewardTitleName { payload["reward_title_name"] = rewardTitleName }
        return payload
    }

    static func titleInfo(
        id: Int64,
        titleName: String = "胜场新星",
        source: String = "achievement",
        sourceType: String? = "achievement",
        sourceRefId: Int64 = 0,
        sourceRefName: String = "",
        equipped: Bool = false
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "id": id,
            "title_name": titleName,
            "source": source,
            "equipped": equipped,
            "created_at": "2026-08-01 09:00:00",
        ]
        if let sourceType { payload["source_type"] = sourceType }
        if sourceRefId > 0 { payload["source_ref_id"] = sourceRefId }
        if !sourceRefName.isEmpty { payload["source_ref_name"] = sourceRefName }
        return payload
    }

    static func challenge(
        key: String,
        name: String = "赛季对局",
        description: String = "本赛季完成有效对局",
        threshold: Int = 20,
        progress: Int = 12,
        completed: Bool = false
    ) -> [String: Any] {
        [
            "key": key,
            "name": name,
            "description": description,
            "threshold": threshold,
            "progress": progress,
            "completed": completed,
        ]
    }

    static func season(
        seasonId: Int64 = 3,
        seasonName: String = "S3",
        gameType: Int = 3,
        challenges: [[String: Any]] = [
            challenge(key: "season_matches_20"),
            challenge(key: "season_wins_10", name: "赛季胜场", threshold: 10, progress: 10, completed: true),
            challenge(key: "season_tournament_finish_1", name: "赛季赛事", threshold: 1, progress: 0),
        ]
    ) -> [String: Any] {
        [
            "season_id": seasonId,
            "season_name": seasonName,
            "start_date": "2026-07-01 00:00:00",
            "end_date": "2026-09-30 23:59:59",
            "status": 1,
            "game_type": gameType,
            "challenges": challenges,
        ]
    }

    static func summary(
        universalUnlocked: Int = 3,
        universalTotal: Int = 9,
        specialtyUnlocked: Int = 2,
        specialtyTotal: Int = 4,
        seasonHonors: Int = 1,
        tournamentHonors: Int = 1
    ) -> [String: Any] {
        [
            "career_unlocked": universalUnlocked + specialtyUnlocked,
            "career_total": universalTotal + specialtyTotal,
            "universal_unlocked": universalUnlocked,
            "universal_total": universalTotal,
            "specialty_game_type": 3,
            "specialty_unlocked": specialtyUnlocked,
            "specialty_total": specialtyTotal,
            "season_honors": seasonHonors,
            "tournament_honors": tournamentHonors,
        ]
    }

    static let defaultAchievements: [[String: Any]] = [
        achievement(
            id: 1,
            key: "wins_10",
            name: "十胜起步",
            description: "累计赢得 10 场有效对局",
            category: "wins",
            threshold: 10,
            progress: 8
        ),
        achievement(
            id: 2,
            key: "match_5",
            name: "五场起步",
            description: "完成 5 场有效对局",
            category: "match",
            threshold: 5,
            progress: 5,
            unlocked: true,
            unlockedAt: "2026-08-12 10:00:00"
        ),
        achievement(
            id: 3,
            key: "streak_3",
            name: "三连胜",
            description: "取得一次三连胜",
            category: "streak",
            threshold: 3,
            progress: 1
        ),
        achievement(
            id: 4,
            key: "special_break_50",
            name: "单杆 50+",
            description: "单杆得分达到 50",
            icon: "",
            category: "special",
            gameType: 3,
            threshold: 1,
            progress: 0
        ),
        achievement(
            id: 5,
            key: "special_break_30",
            name: "单杆 30+",
            description: "单杆得分达到 30",
            icon: "",
            category: "special",
            gameType: 3,
            threshold: 1,
            progress: 1,
            unlocked: true,
            unlockedAt: "2026-09-02 21:30:00"
        ),
    ]

    static let defaultHistory: [String: Any] = [
        "total": 2,
        "honors": [
            honorItem(
                id: 10,
                name: "S3 冠军",
                description: "S3 赛季排位第一",
                sourceType: "season",
                sourceRefId: 3,
                sourceRefName: "S3"
            ),
            honorItem(
                id: 11,
                name: "城市杯四强",
                description: "城市杯进入四强",
                sourceType: "tournament",
                sourceRefId: 9,
                sourceRefName: "城市杯"
            ),
        ],
        "challenge_season": season(),
        "challenge_records": [
            challenge(key: "season_matches_20"),
            challenge(key: "season_wins_10", name: "赛季胜场", threshold: 10, progress: 10, completed: true),
        ],
    ]

    // MARK: - 荣誉墙

    static func honorWall(
        success: Bool = true,
        viewerScope: String = "self",
        seasonState: String = "active",
        userId: Int64 = 1,
        nickname: String = "测试用户",
        equippedTitle: Any? = titleInfo(
            id: 2,
            titleName: "S3 冠军",
            source: "season",
            sourceType: "season",
            sourceRefId: 3,
            sourceRefName: "S3",
            equipped: true
        ),
        summary: [String: Any] = summary(),
        recentHonors: [[String: Any]] = [
            honorItem(id: 20, name: "十胜起步", rewardTitleName: "胜场新星"),
        ],
        careerAchievements: [[String: Any]] = defaultAchievements,
        currentSeason: Any? = season(),
        history: [String: Any] = defaultHistory
    ) -> [String: Any] {
        [
            "success": success,
            "season_state": seasonState,
            "viewer_scope": viewerScope,
            "profile": [
                "user_id": userId,
                "nickname": nickname,
                "avatar": "",
            ],
            "equipped_title": equippedTitle ?? NSNull(),
            "summary": summary,
            "recent_honors": recentHonors,
            "career_achievements": careerAchievements,
            "current_season": currentSeason ?? NSNull(),
            "history": history,
        ]
    }

    static func emptyHonorWall() -> [String: Any] {
        honorWall(
            seasonState: "not_started",
            equippedTitle: nil,
            summary: summary(
                universalUnlocked: 0,
                universalTotal: 0,
                specialtyUnlocked: 0,
                specialtyTotal: 0,
                seasonHonors: 0,
                tournamentHonors: 0
            ),
            recentHonors: [],
            careerAchievements: [],
            currentSeason: nil,
            history: [
                "total": 0,
                "honors": [],
                "challenge_season": NSNull(),
                "challenge_records": [],
            ]
        )
    }

    static func friendHonorWall() -> [String: Any] {
        honorWall(
            viewerScope: "friend",
            nickname: "好友甲",
            equippedTitle: nil,
            recentHonors: [honorItem(id: 30, name: "五场起步", sourceType: "achievement")],
            careerAchievements: [
                achievement(
                    id: 2,
                    key: "match_5",
                    name: "五场起步",
                    description: "完成 5 场有效对局",
                    category: "match",
                    threshold: 5,
                    progress: 5,
                    unlocked: true,
                    unlockedAt: "2026-08-12 10:00:00"
                ),
            ],
            currentSeason: nil,
            history: [
                "total": 1,
                "honors": [
                    honorItem(
                        id: 10,
                        name: "S3 冠军",
                        sourceType: "season",
                        sourceRefId: 3,
                        sourceRefName: "S3"
                    ),
                ],
                "challenge_season": NSNull(),
                "challenge_records": [],
            ]
        )
    }

    // MARK: - 成就详情

    static func detail(_ achievement: [String: Any]) -> [String: Any] {
        ["success": true, "achievement": achievement]
    }

    static func missingDetail() -> [String: Any] {
        ["success": false, "achievement": NSNull()]
    }

    static func unlockedDetail() -> [String: Any] {
        detail(achievement(
            id: 7,
            key: "wins_10",
            name: "十胜起步",
            description: "累计赢得 10 场有效对局",
            category: "wins",
            rewardTitleName: "胜场新星",
            threshold: 10,
            progress: 10,
            unlocked: true,
            unlockedAt: "2026-08-12 10:00:00"
        ))
    }

    static func lockedDetail() -> [String: Any] {
        detail(achievement(
            id: 8,
            key: "streak_3",
            name: "三连胜",
            description: "取得一次三连胜",
            icon: "",
            category: "streak",
            threshold: 3,
            progress: 1
        ))
    }

    // MARK: - 称号与通知

    static func titles(_ list: [[String: Any]]) -> [String: Any] {
        ["success": true, "list": list]
    }

    static let defaultTitles: [[String: Any]] = [
        titleInfo(id: 2, titleName: "S3 冠军", source: "season", sourceType: "season", sourceRefId: 3, sourceRefName: "S3", equipped: true),
        titleInfo(id: 3, titleName: "城市杯四强", source: "tournament", sourceType: "tournament", sourceRefName: "城市杯"),
    ]

    static func common(message: String = "操作成功") -> [String: Any] {
        ["success": true, "message": message]
    }

    static func notification(
        id: Int64,
        type: String = "season_rollover",
        isRead: Bool = false,
        createdAt: String = "2026-07-03 10:00:00",
        data: String = "{\"from_season_name\":\"S3\",\"to_season_id\":0,\"intermission\":true}"
    ) -> [String: Any] {
        [
            "id": id,
            "type": type,
            "title": "赛季结算",
            "content": "S3 已结算",
            "data": data,
            "is_read": isRead,
            "created_at": createdAt,
        ]
    }

    static func notificationList(_ list: [[String: Any]], unreadCount: Int = 1) -> [String: Any] {
        [
            "success": true,
            "total": list.count,
            "unread_count": unreadCount,
            "list": list,
        ]
    }
}
