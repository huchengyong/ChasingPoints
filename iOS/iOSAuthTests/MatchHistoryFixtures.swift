import Foundation

/// 对局记录与战报接口的虚构夹具，字段与 `backend/chasing_points.api` 及 logic 的实际响应一致。
/// 全部为假数据：头像为空、昵称为占位、时间为固定值，不包含真实账号或对局。
enum MatchHistoryFixtures {
    static let matchTime = "2026-10-09T20:30:00+08:00"

    static func participantItem(id: Int64, result: Int = 1, myScore: Int = 3, opponentScore: Int = 1) -> [String: Any] {
        [
            "id": id,
            "opponent_id": 2,
            "game_type": 3,
            "game_type_name": "中式八球",
            "match_mode": "ranked",
            "visibility": "public",
            "finish_state": "completed",
            "opponent_name": "对手\(id)",
            "opponent_avatar": "",
            "my_score": myScore,
            "opponent_score": opponentScore,
            "result": result,
            "match_time": matchTime,
        ]
    }

    /// `GET /api/match/list`：`list` 始终为数组，即使为空。
    static func participantPage(items: [[String: Any]], total: Int) -> [String: Any] {
        ["success": true, "total": total, "list": items]
    }

    static func refereeItem(id: Int64, status: Int = 2) -> [String: Any] {
        [
            "id": id,
            "player1_id": 11,
            "player1_name": "选手甲\(id)",
            "player1_avatar": "",
            "player2_id": 12,
            "player2_name": "选手乙\(id)",
            "player2_avatar": "",
            "player1_score": 4,
            "player2_score": 2,
            "game_type": 2,
            "game_type_name": "九球追分",
            "match_mode": "ranked",
            "visibility": "public",
            "status": status,
            "status_text": status == 3 ? "已取消" : "已完成",
            "referee_joined_at": matchTime,
            "end_time": matchTime,
            "referee_duration_seconds": 1260,
            "completed_by_user_id": 11,
            "completion_source": "referee",
        ]
    }

    static func refereePage(items: [[String: Any]], total: Int) -> [String: Any] {
        ["success": true, "total": total, "list": items]
    }

    /// `GET /api/match/detail`：选手视角。`snooker_*` 等实时计分字段故意保留，验证窄解码会忽略它们。
    static func playerDetail(
        id: Int64 = 101,
        status: Int = 2,
        myScore: Int = 3,
        opponentScore: Int = 1,
        highlights: [[String: Any]]? = nil,
        stats: [[String: Any]]? = nil
    ) -> [String: Any] {
        [
            "success": true,
            "match": [
                "id": id,
                "player1_id": 1,
                "player2_id": 2,
                "opponent_id": 2,
                "game_type": 3,
                "match_mode": "ranked",
                "status": status,
                "server_revision": 9,
                "is_player1": true,
                "viewer_role": "player1",
                "referee_bound": true,
                "referee_user_id": 11,
                "referee_name": "裁判小王",
                "referee_avatar": "",
                "referee_joined_at": matchTime,
                "referee_duration_seconds": 1260,
                "completed_by_user_id": 11,
                "completion_source": "referee",
                "snooker_ball_on": "",
                "my_score": myScore,
                "opponent_score": opponentScore,
                "my_name": "测试用户",
                "opponent_name": "对手101",
                "my_avatar": "",
                "opponent_avatar": "",
                "my_win_rate": 62.5,
                "opponent_win_rate": 37.5,
                "my_max_score": 27,
                "opponent_max_score": 14,
                "my_rank_change": 12,
                "opponent_rank_change": -12,
                "my_rank_details": [
                    ["label": "基础分", "value": 10],
                    ["label": "高光排位分", "value": 2],
                ],
                "summary_highlights": highlights ?? [],
                "summary_stats": stats ?? [],
                "achievements": [
                    "break_clear": 0,
                    "continue_clear": 0,
                    "golden_break": 0,
                    "nine_on_break": 0,
                    "break_50": 0,
                    "break_100": 0,
                    "break_147": 0,
                ],
                "created_at": matchTime,
            ],
        ]
    }

    /// `GET /api/match/detail`：裁判视角，不含选手排位数据。
    static func refereeDetail(id: Int64 = 201) -> [String: Any] {
        [
            "success": true,
            "match": [
                "id": id,
                "player1_id": 11,
                "player2_id": 12,
                "game_type": 2,
                "status": 2,
                "viewer_role": "referee",
                "is_player1": false,
                "referee_bound": true,
                "referee_user_id": 1,
                "referee_name": "测试用户",
                "referee_avatar": "",
                "referee_joined_at": matchTime,
                "referee_duration_seconds": 1260,
                "completed_by_user_id": 1,
                "completion_source": "referee",
                "my_score": 4,
                "opponent_score": 2,
                "my_name": "选手甲",
                "opponent_name": "选手乙",
                "my_avatar": "",
                "opponent_avatar": "",
                "my_win_rate": 0,
                "opponent_win_rate": 0,
                "my_max_score": 0,
                "opponent_max_score": 0,
                "my_rank_change": 0,
                "opponent_rank_change": 0,
                "my_rank_details": NSNull(),
                "summary_highlights": NSNull(),
                "summary_stats": NSNull(),
                "achievements": [
                    "break_clear": 0,
                    "continue_clear": 0,
                    "golden_break": 0,
                    "nine_on_break": 0,
                    "break_50": 0,
                    "break_100": 0,
                    "break_147": 0,
                ],
                "created_at": matchTime,
            ],
        ]
    }

    /// `GET /api/match/reward-summary`：`ready` 或 `pending`，失败时服务端返回 `success: false`。
    static func rewardSummary(status: String = "ready", list: [[String: Any]] = []) -> [String: Any] {
        ["success": true, "status": status, "list": list]
    }

    static func rewardItem(id: Int64 = 9, title: String? = "夜战王") -> [String: Any] {
        [
            "achievement_id": id,
            "achievement_key": "night_owl",
            "achievement_name": "夜猫子",
            "description": "夜间完成一场对局",
            "icon": "",
            "reward_title_id": title == nil ? 0 : 3,
            "reward_title_name": title ?? "",
            "unlocked_at": matchTime,
        ]
    }

    /// 业务失败：HTTP 200 + `success: false`，不含 message。
    static let businessFailure: [String: Any] = ["success": false]
}
