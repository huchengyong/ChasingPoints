import Foundation

// 「我的」页窄解码模型：字段与 backend/chasing_points.api 对齐，只包含页面读取用到的分区。

struct UserStats: Decodable, Equatable {
    let success: Bool
    let totalMatches: Int
    let wins: Int
    let losses: Int
    /// 百分比数值，例如 59.3。
    let winRate: Double
    let maxWinStreak: Int
}

struct ReputationStatus: Decodable, Equatable {
    let success: Bool
    let score: Int
    let status: String
    let statusText: String
    let banUntil: String?
}

struct MemberStatus: Decodable, Equatable {
    let success: Bool
    let isActive: Bool
    let currentTime: String
    let planCode: String?
    let planName: String?
    let memberExpiresAt: String?
    let growthLevel: Int?
    let growthPoints: Int?
    let todayGrowthCount: Int?
    let growthDailyCap: Int?
    let growthFrozen: Bool?
    let nextGrowthLevel: Int?
    let nextGrowthLevelPoints: Int?
    let remainingGrowthPoints: Int?
}

struct FavoriteVenueRewardStatus: Decodable, Equatable {
    let success: Bool
    let enabled: Bool
    let popupEnabled: Bool
    let rewardDays: Int
    let newUserWindowDays: Int
    let status: String
    let memberExpiresAt: String?
    let submittedVenueId: Int64?
    let submittedVenueName: String?
    let rejectReason: String?
}

/// `GET /api/user/overview`：聚合接口，各分区自己的 `success` 由业务判断。
struct UserOverview: Decodable, Equatable {
    let stats: UserStats?
    let reputation: ReputationStatus?
    let memberStatus: MemberStatus?
    let favoriteVenueReward: FavoriteVenueRewardStatus?

    var availableStats: UserStats? { stats?.success == true ? stats : nil }
    var availableReputation: ReputationStatus? { reputation?.success == true ? reputation : nil }
    var availableMemberStatus: MemberStatus? { memberStatus?.success == true ? memberStatus : nil }
    var availableFavoriteVenueReward: FavoriteVenueRewardStatus? {
        favoriteVenueReward?.success == true ? favoriteVenueReward : nil
    }
}

struct RankInfo: Decodable, Equatable {
    let level: Int
    let name: String
    let icon: String
    let rankScore: Int
    let totalWins: Int
    let totalLosses: Int
    let maxStreak: Int
    let nextLevel: Int
    let nextName: String
    let nextScore: Int
    let progress: Int
    /// 排位规则 V2 下的剩余定级赛场次；旧服务端不返回。
    let placementRemaining: Int?
}

/// `GET /api/rank/infos` 归一化后的球种 → 段位映射，只保留支持的球种。
typealias RankInfoMap = [Int: RankInfo]

struct UserRankInfoItem: Decodable, Equatable {
    let gameType: Int
    let rankInfo: RankInfo?
}

struct RankInfosResponse: Decodable, Equatable {
    let rankInfos: [UserRankInfoItem]?

    /// 照搬 `normalizeUserRankInfos`：只保留球种 1–4 且 `rank_info` 非空的项。
    var map: RankInfoMap {
        (rankInfos ?? []).reduce(into: [:]) { result, item in
            guard GameTypeCatalog.isSupported(item.gameType), let rankInfo = item.rankInfo else { return }
            result[item.gameType] = rankInfo
        }
    }
}

/// `GET /api/match/qrcode`。`invite_token` 是凭证，不进模型。
struct MatchQRCode: Decodable, Equatable {
    let qrcodeData: String
    let expiresInSeconds: Int64?
}
