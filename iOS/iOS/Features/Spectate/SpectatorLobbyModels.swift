import Foundation

/// 公开观赛列表项，字段与 backend/chasing_points.api 的 PublicMatchListItem 对齐。
struct SpectatorMatch: Decodable, Identifiable, Equatable {
    let id: Int64
    let gameType: Int
    let gameTypeName: String
    let matchMode: String?
    let status: Int
    let player1Id: Int64
    let player1Name: String
    let player1Avatar: String
    let player2Id: Int64
    let player2Name: String
    let player2Avatar: String
    let player1Score: Int
    let player2Score: Int
    let durationSeconds: Int64
    let endTime: String?
}

struct SpectatorMatchPage: Equatable {
    let items: [SpectatorMatch]
    let total: Int
}

struct SpectatorPlayer: Equatable {
    enum Outcome: String {
        case winner
        case loser
    }

    let id: Int64
    let name: String
    let avatar: String
    let score: Int
    let isMe: Bool
    let outcome: Outcome?
}

/// 列表卡片的展示模型；“我”的标记按当前访客身份组装。
struct SpectatorMatchCard: Identifiable, Equatable {
    let id: Int64
    let rawStatus: Int
    let isLive: Bool
    let isMine: Bool
    let metaText: String
    let statusText: String
    let players: [SpectatorPlayer]
}
