import Foundation

/// `GET /api/public/rank/leaderboard-summary` 的窄解码结果，条目复用排行榜模型。
struct LeaderboardSummary: Decodable, Equatable {
    let topThree: [LeaderboardItem]
    let myRanking: LeaderboardItem?
}

/// 首页每个区块独立的读取状态；一个区块失败只影响自己。
enum HomeSectionPhase: Equatable {
    case idle
    case loading
    case loaded
    case failed(LoadFailure)
}

/// 「我的对局」卡，对应小程序 buildOngoingMatchCard。
struct HomeMatchCard: Equatable {
    let id: Int64
    let isReferee: Bool
    let isPending: Bool
    let statusText: String
    let metaText: String
    let durationText: String
    let players: [HomeMatchPlayer]
    let actionLabel: String
}

struct HomeMatchPlayer: Equatable {
    let id: Int64
    let name: String
    let avatar: String
    let score: Int
    let isMe: Bool
}

/// 约球待办卡，对应小程序 resolveChallengeQuickCard。
struct HomeChallengeCard: Equatable {
    let challengeId: Int64
    let matchId: Int64
    let title: String
    let needsAction: Bool
    let subText: String
    let actionLabel: String
}

/// 附近球房空态的操作文案，对应小程序 resolveHomeVenueEmptyAction。
struct HomeVenueEmptyAction: Equatable {
    let text: String
    let desc: String
}

/// 首页附近球房的查询参数；小程序固定 5km、取 3 条。
struct HomeNearbyVenueParams: Equatable {
    let latitude: Double
    let longitude: Double
    let radius: Int
    let limit: Int
}

/// 首页附近球房因定位失败展示的提示。
enum HomeLocationIssue: Equatable {
    case denied
    case unavailable
}
