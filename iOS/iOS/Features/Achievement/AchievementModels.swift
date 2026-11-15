import Foundation

// 字段与 `backend/chasing_points.api` 的成就模块一致；标了 optional 的字段用可选类型。

struct AchievementDef: Decodable, Equatable, Identifiable {
    let id: Int64
    let key: String
    let name: String
    let description: String
    let icon: String
    let category: String
    let gameType: Int
    let rewardTitleName: String?
    let threshold: Int
    let progress: Int
    let unlocked: Bool
    let unlockedAt: String?
}

struct HonorWallProfile: Decodable, Equatable {
    let userId: Int64
    let nickname: String
    let avatar: String
}

struct HonorWallSummary: Decodable, Equatable {
    let careerUnlocked: Int
    let careerTotal: Int
    let universalUnlocked: Int
    let universalTotal: Int
    let specialtyGameType: Int
    let specialtyUnlocked: Int
    let specialtyTotal: Int
    let seasonHonors: Int
    let tournamentHonors: Int
}

struct HonorItem: Decodable, Equatable, Identifiable {
    let id: Int64
    let type: String
    let name: String
    let description: String?
    let icon: String?
    let sourceType: String
    let sourceRefId: Int64?
    let sourceRefName: String?
    let rewardTitleName: String?
    let earnedAt: String
}

struct SeasonChallengeInfo: Decodable, Equatable, Identifiable {
    let key: String
    let name: String
    let description: String
    let threshold: Int
    let progress: Int
    let completed: Bool

    var id: String { key }
}

struct HonorWallSeasonInfo: Decodable, Equatable {
    let seasonId: Int64
    let seasonName: String
    let startDate: String
    let endDate: String
    let status: Int
    let gameType: Int
    let challenges: [SeasonChallengeInfo]
}

struct HonorWallHistory: Decodable, Equatable {
    let total: Int64
    var honors: [HonorItem]
    let challengeSeason: HonorWallSeasonInfo?
    let challengeRecords: [SeasonChallengeInfo]
}

struct HonorWallResponse: Decodable, Equatable {
    let success: Bool
    let seasonState: String
    let viewerScope: String
    let profile: HonorWallProfile
    var equippedTitle: TitleInfo?
    let summary: HonorWallSummary
    let recentHonors: [HonorItem]
    let careerAchievements: [AchievementDef]
    let currentSeason: HonorWallSeasonInfo?
    var history: HonorWallHistory
}

struct TitleInfo: Decodable, Equatable, Identifiable {
    let id: Int64
    let titleName: String
    let source: String
    let sourceType: String?
    let sourceRefId: Int64?
    let sourceRefName: String?
    var equipped: Bool
    let grantedAt: String?
    let createdAt: String
}

struct AchievementDetailResponse: Decodable, Equatable {
    let success: Bool
    let achievement: AchievementDef?
}

struct UserTitlesResponse: Decodable, Equatable {
    let success: Bool
    let list: [TitleInfo]
}

struct AchievementCommonResponse: Decodable, Equatable {
    let success: Bool
    let message: String
}
