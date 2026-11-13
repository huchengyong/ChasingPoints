import Foundation

// 赛讯 Tab 与赛事详情的窄解码模型：字段与 backend/chasing_points.api 对齐。
// `.api` 里标了 `optional` 的字段用可选类型，其余字段按非可选解码；解码统一走 convertFromSnakeCase。

struct EventNewsInfo: Decodable, Equatable, Identifiable {
    let id: Int64
    let title: String
    let sourceType: String
    let sourceName: String
    let sourceUrl: String
    let coverImage: String
    let summary: String
    let content: String
    let description: String
    let tournamentId: Int64
    let tournamentName: String
    let gameType: Int
    let country: String
    let city: String
    let venue: String
    let startDate: String
    let endDate: String
    let startTime: String?
    let endTime: String?
    /// 0 即将开始 / 1 进行中 / 2 已结束 / 3 已取消。
    let status: Int
    let currentRoundText: String
    let latestResultText: String
    let matchCount: Int
    let sortTime: String
    let published: Bool
    let publishedAt: String
    let createdAt: String
    let updatedAt: String
}

struct EventNewsMatchInfo: Decodable, Equatable, Identifiable {
    let id: Int64
    let eventId: Int64
    let tournamentId: Int64
    let sourceType: String
    let sourceMatchId: String
    let roundName: String
    let roundOrder: Int
    let matchOrder: Int
    let startTime: String
    let status: Int
    let bestOf: Int
    let homePlayerId: Int64
    let homePlayerName: String
    /// 管理后台维护的中文名，未维护时为空。
    let homePlayerNameZh: String?
    let homePlayerFirstName: String?
    let homePlayerLastName: String?
    let homePlayerFlagEmoji: String?
    let homePlayerCountryCode: String?
    let homePlayerAvatar: String?
    let awayPlayerId: Int64
    let awayPlayerName: String
    let awayPlayerNameZh: String?
    let awayPlayerFirstName: String?
    let awayPlayerLastName: String?
    let awayPlayerFlagEmoji: String?
    let awayPlayerCountryCode: String?
    let awayPlayerAvatar: String?
    let homeScore: Int
    let awayScore: Int
    let winnerSide: Int
    let isPlaceholder: Bool
    let createdAt: String
    let updatedAt: String
}

struct TournamentInfo: Decodable, Equatable, Identifiable {
    let id: Int64
    let creatorId: Int64
    let creatorName: String
    let name: String
    let description: String
    let coverImage: String
    let gameType: Int
    let format: Int
    let maxPlayers: Int
    let currentPlayers: Int
    let status: Int
    let country: String
    let city: String
    let venueName: String
    let startDate: String
    let endDate: String
    let startTime: String?
    let endTime: String?
    let createdAt: String
}

/// `GET /api/event-news/view` 的三段内容。
struct EventNewsView: Decodable, Equatable {
    let eventNews: EventNewsInfo?
    let tournament: TournamentInfo?
    let matches: [EventNewsMatchInfo]

    /// 对应小程序 `pickEventNewsViewPayload`：缺少 `event_news` 时整体视为无内容。
    /// 响应把内容包在顶层 `data` 里的兼容逻辑已由 `APIClient` 统一解开，这里不再重复。
    var pickedPayload: EventNewsView? {
        eventNews == nil ? nil : self
    }
}
