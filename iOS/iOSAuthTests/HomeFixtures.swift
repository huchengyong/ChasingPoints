import Foundation
@testable import iOS

/// 首页接口与活动状态的虚构夹具，字段与服务端响应一致。
enum HomeFixtures {
    static let now = EventNewsFixtures.time("2026-04-01T21:00:00+08:00")
    static let leaderboardPath = "/api/public/rank/leaderboard-summary"
    static let eventNewsPath = "/api/event-news/list"
    static let matchesPath = "/api/public/matches"
    static let nearbyPath = "/api/venue/nearby"
    static let bootstrapPath = "/api/user/bootstrap"

    static func leaderboardSummary(
        topThree: [[String: Any]] = [],
        myRanking: [String: Any]? = nil
    ) -> MockHTTPServer.Stub {
        var payload: [String: Any] = [
            "success": true,
            "total": topThree.count,
            "top_three": topThree,
        ]
        if let myRanking {
            payload["my_ranking"] = myRanking
        }
        return .json(payload)
    }

    static func currentMatch(
        id: Int64 = 1,
        viewerRole: String = "player1",
        matchMode: String = "ranked",
        finishState: String? = nil,
        durationSeconds: Int64 = 600,
        myScore: Int = 3,
        opponentScore: Int = 1,
        player1Name: String = "我",
        player2Name: String = "对手",
        opponentName: String = "对手",
        gameTypeName: String = "中式八球",
        challengeId: Int64? = nil
    ) -> [String: Any] {
        var json = ActivityFixtures.currentMatch(id: id, opponent: opponentName)
        json["viewer_role"] = viewerRole
        json["game_type_name"] = gameTypeName
        json["match_mode"] = matchMode
        json["duration_seconds"] = durationSeconds
        json["my_score"] = myScore
        json["opponent_score"] = opponentScore
        json["player1_name"] = player1Name
        json["player2_name"] = player2Name
        if let finishState { json["finish_state"] = finishState }
        if let challengeId { json["challenge_id"] = challengeId }
        return json
    }

    static func challenge(
        id: Int64 = 3,
        fromUserId: Int64 = 2,
        fromNickname: String = "甲",
        toUserId: Int64 = 1,
        toNickname: String = "我",
        status: Int = 0,
        matchId: Int64? = nil,
        waitingUserId: Int64? = nil,
        scheduledDate: String? = "2026-04-01",
        startHour: Int? = 16,
        endHour: Int? = 17,
        matchMode: String = "ranked",
        gameType: Int = 3
    ) -> [String: Any] {
        var json = ActivityFixtures.challenge(id: id)
        json["from_user_id"] = fromUserId
        json["from_nickname"] = fromNickname
        json["to_user_id"] = toUserId
        json["to_nickname"] = toNickname
        json["status"] = status
        json["game_type"] = gameType
        json["match_mode"] = matchMode
        if let matchId { json["match_id"] = matchId }
        if let waitingUserId { json["waiting_user_id"] = waitingUserId }
        if let scheduledDate { json["scheduled_date"] = scheduledDate }
        if let startHour { json["start_hour"] = startHour }
        if let endHour { json["end_hour"] = endHour }
        return json
    }

    static func bootstrap(
        currentMatch: [String: Any]? = nil,
        challenge: [String: Any]? = nil,
        challengeAvailable: Bool = true
    ) -> [String: Any] {
        var availability = ActivityFixtures.allAvailable
        if !challengeAvailable {
            availability["current_challenge"] = false
            availability["challenge_received_count"] = false
        }
        return [
            "success": true,
            "current_match": currentMatch ?? NSNull(),
            "current_challenge": challenge ?? NSNull(),
            "received_pending_challenge_count": 0,
            "server_time": "2026-04-01 12:00:00",
            "unread_count": 0,
            "pending_friend_request_count": 0,
            "latest_season_rollover": NSNull(),
            "availability": availability,
        ]
    }

    static func decode<T: Decodable>(_ json: [String: Any], as type: T.Type) -> T {
        let data = (try? JSONSerialization.data(withJSONObject: json)) ?? Data()
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        return (try? decoder.decode(type, from: data))!
    }

    static func currentMatchModel(_ json: [String: Any]) -> CurrentMatchInfo {
        decode(json, as: CurrentMatchInfo.self)
    }

    static func challengeModel(_ json: [String: Any]) -> ChallengeInfo {
        decode(json, as: ChallengeInfo.self)
    }
}
