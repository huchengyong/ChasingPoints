import Foundation
@testable import iOS

/// 赛讯接口与规则的虚构夹具，字段与 `backend/chasing_points.api` 一致。
/// 全部为假数据：赛事、球员、城市与封面地址均为占位，不包含真实账号或生产资源。
enum EventNewsFixtures {
    /// 把测试里的 ISO 8601 时间文本转成固定时刻。
    static func time(_ iso: String) -> Date {
        guard let date = ISO8601DateFormatter().date(from: iso) else {
            fatalError("测试时间格式错误：\(iso)")
        }
        return date
    }

    // MARK: - 模型

    static func eventNewsModel(
        id: Int64 = 1,
        title: String = "",
        sourceType: String = "",
        sourceName: String = "",
        sourceUrl: String = "",
        coverImage: String = "",
        summary: String = "",
        content: String = "",
        description: String = "",
        tournamentId: Int64 = 0,
        tournamentName: String = "",
        gameType: Int = 0,
        country: String = "",
        city: String = "",
        venue: String = "",
        startDate: String = "",
        endDate: String = "",
        startTime: String? = nil,
        endTime: String? = nil,
        status: Int = 0,
        currentRoundText: String = "",
        latestResultText: String = "",
        matchCount: Int = 0,
        sortTime: String = "",
        published: Bool = true,
        publishedAt: String = "",
        createdAt: String = "",
        updatedAt: String = ""
    ) -> EventNewsInfo {
        EventNewsInfo(
            id: id,
            title: title,
            sourceType: sourceType,
            sourceName: sourceName,
            sourceUrl: sourceUrl,
            coverImage: coverImage,
            summary: summary,
            content: content,
            description: description,
            tournamentId: tournamentId,
            tournamentName: tournamentName,
            gameType: gameType,
            country: country,
            city: city,
            venue: venue,
            startDate: startDate,
            endDate: endDate,
            startTime: startTime,
            endTime: endTime,
            status: status,
            currentRoundText: currentRoundText,
            latestResultText: latestResultText,
            matchCount: matchCount,
            sortTime: sortTime,
            published: published,
            publishedAt: publishedAt,
            createdAt: createdAt,
            updatedAt: updatedAt
        )
    }

    static func matchModel(
        id: Int64 = 1,
        eventId: Int64 = 0,
        tournamentId: Int64 = 0,
        sourceType: String = "",
        sourceMatchId: String = "",
        roundName: String = "",
        roundOrder: Int = 0,
        matchOrder: Int = 0,
        startTime: String = "",
        status: Int = 0,
        bestOf: Int = 0,
        homePlayerId: Int64 = 0,
        homePlayerName: String = "",
        homePlayerNameZh: String? = nil,
        homePlayerFirstName: String? = nil,
        homePlayerLastName: String? = nil,
        homePlayerFlagEmoji: String? = nil,
        homePlayerCountryCode: String? = nil,
        homePlayerAvatar: String? = nil,
        awayPlayerId: Int64 = 0,
        awayPlayerName: String = "",
        awayPlayerNameZh: String? = nil,
        awayPlayerFirstName: String? = nil,
        awayPlayerLastName: String? = nil,
        awayPlayerFlagEmoji: String? = nil,
        awayPlayerCountryCode: String? = nil,
        awayPlayerAvatar: String? = nil,
        homeScore: Int = 0,
        awayScore: Int = 0,
        winnerSide: Int = 0,
        isPlaceholder: Bool = false,
        createdAt: String = "",
        updatedAt: String = ""
    ) -> EventNewsMatchInfo {
        EventNewsMatchInfo(
            id: id,
            eventId: eventId,
            tournamentId: tournamentId,
            sourceType: sourceType,
            sourceMatchId: sourceMatchId,
            roundName: roundName,
            roundOrder: roundOrder,
            matchOrder: matchOrder,
            startTime: startTime,
            status: status,
            bestOf: bestOf,
            homePlayerId: homePlayerId,
            homePlayerName: homePlayerName,
            homePlayerNameZh: homePlayerNameZh,
            homePlayerFirstName: homePlayerFirstName,
            homePlayerLastName: homePlayerLastName,
            homePlayerFlagEmoji: homePlayerFlagEmoji,
            homePlayerCountryCode: homePlayerCountryCode,
            homePlayerAvatar: homePlayerAvatar,
            awayPlayerId: awayPlayerId,
            awayPlayerName: awayPlayerName,
            awayPlayerNameZh: awayPlayerNameZh,
            awayPlayerFirstName: awayPlayerFirstName,
            awayPlayerLastName: awayPlayerLastName,
            awayPlayerFlagEmoji: awayPlayerFlagEmoji,
            awayPlayerCountryCode: awayPlayerCountryCode,
            awayPlayerAvatar: awayPlayerAvatar,
            homeScore: homeScore,
            awayScore: awayScore,
            winnerSide: winnerSide,
            isPlaceholder: isPlaceholder,
            createdAt: createdAt,
            updatedAt: updatedAt
        )
    }

    // MARK: - 接口响应（snake_case）

    static func eventNewsJSON(
        id: Int64 = 18,
        title: String = "赛事详情",
        sourceType: String = "official",
        sourceName: String = "WST",
        tournamentName: String = "",
        gameType: Int = 1,
        status: Int = 0,
        startDate: String = "2026-03-30",
        endDate: String = "2026-04-05",
        startTime: String? = nil,
        endTime: String? = nil,
        coverImage: String = "",
        summary: String = "",
        currentRoundText: String = "",
        latestResultText: String = "",
        matchCount: Int = 0
    ) -> [String: Any] {
        var json: [String: Any] = [
            "id": id,
            "title": title,
            "source_type": sourceType,
            "source_name": sourceName,
            "source_url": "",
            "cover_image": coverImage,
            "summary": summary,
            "content": "",
            "description": "",
            "tournament_id": 3,
            "tournament_name": tournamentName,
            "game_type": gameType,
            "country": "",
            "city": "",
            "venue": "",
            "start_date": startDate,
            "end_date": endDate,
            "status": status,
            "current_round_text": currentRoundText,
            "latest_result_text": latestResultText,
            "match_count": matchCount,
            "sort_time": startDate,
            "published": true,
            "published_at": "",
            "created_at": "",
            "updated_at": "",
        ]
        if let startTime { json["start_time"] = startTime }
        if let endTime { json["end_time"] = endTime }
        return json
    }

    static func matchJSON(
        id: Int64 = 88,
        roundName: String = "Final",
        roundOrder: Int = 70,
        status: Int = 0,
        homePlayerName: String = "Yuan Sijun",
        awayPlayerName: String = "Wu Yize",
        homeScore: Int = 0,
        awayScore: Int = 0,
        winnerSide: Int = 0
    ) -> [String: Any] {
        [
            "id": id,
            "event_id": 18,
            "tournament_id": 3,
            "source_type": "official",
            "source_match_id": "",
            "round_name": roundName,
            "round_order": roundOrder,
            "match_order": 1,
            "start_time": "2026-04-05 20:00:00",
            "status": status,
            "best_of": 19,
            "home_player_id": 1,
            "home_player_name": homePlayerName,
            "away_player_id": 2,
            "away_player_name": awayPlayerName,
            "home_score": homeScore,
            "away_score": awayScore,
            "winner_side": winnerSide,
            "is_placeholder": false,
            "created_at": "",
            "updated_at": "",
        ]
    }

    static func tournamentJSON(id: Int64 = 3, name: String = "赛事实体") -> [String: Any] {
        [
            "id": id,
            "creator_id": 0,
            "creator_name": "",
            "name": name,
            "description": "",
            "cover_image": "",
            "game_type": 1,
            "format": 1,
            "max_players": 16,
            "current_players": 8,
            "status": 0,
            "country": "China",
            "city": "Shenzhen",
            "venue_name": "",
            "start_date": "2026-03-30",
            "end_date": "2026-04-05",
            "created_at": "",
        ]
    }

    static func listResponse(
        total: Int = 1,
        list: [[String: Any]]? = nil
    ) -> [String: Any] {
        ["success": true, "total": total, "list": list ?? [eventNewsJSON()]]
    }

    static func viewResponse(
        eventNews: [String: Any]? = nil,
        tournament: [String: Any]? = nil,
        matches: [[String: Any]]? = nil
    ) -> [String: Any] {
        [
            "success": true,
            "event_news": eventNews ?? eventNewsJSON(),
            "tournament": tournament ?? tournamentJSON(),
            "matches": matches ?? [matchJSON()],
        ]
    }
}
