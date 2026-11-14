import Foundation

/// 成就模块的读取与写入门面，统一经 `SessionStore` 的认证入口。
@MainActor
struct AchievementAPI {
    let session: SessionStore

    func honorWall(
        userId: Int64 = 0,
        gameType: Int,
        historySeasonId: Int64,
        historyPage: Int,
        historyPageSize: Int = 20
    ) async throws -> HonorWallResponse {
        let path = APIQuery.path("/api/achievement/honor-wall", [
            "user_id": userId > 0 ? userId : nil,
            "game_type": gameType,
            "history_season_id": historySeasonId > 0 ? historySeasonId : nil,
            "history_page": historyPage,
            "history_page_size": historyPageSize,
        ])
        return try await session.authorizedRequest(path)
    }

    func detail(achievementId: Int64) async throws -> AchievementDetailResponse {
        let path = APIQuery.path("/api/achievement/detail", ["achievement_id": achievementId])
        return try await session.authorizedRequest(path)
    }

    func titles() async throws -> UserTitlesResponse {
        try await session.authorizedRequest("/api/achievement/titles")
    }

    func equipTitle(titleId: Int64, equip: Bool) async throws {
        let _: AchievementCommonResponse = try await session.authorizedPost(
            "/api/achievement/title/equip",
            body: EquipTitleRequest(titleId: titleId, equip: equip)
        )
    }

    /// 荣誉墙换季通知：进入页面只沿用小程序阅读过的最近一条未读换季通知。
    func latestSeasonRolloverNotifications() async throws -> [NotificationInfo] {
        let path = APIQuery.path("/api/notification/list", [
            "page": 1,
            "page_size": 10,
            "type": "season_rollover",
        ])
        let response: NotificationListResponse = try await session.authorizedRequest(path)
        return response.list
    }

    func markNotificationRead(notificationId: Int64) async throws {
        let _: AchievementCommonResponse = try await session.authorizedPost(
            "/api/notification/read",
            body: NotificationIdRequest(notificationId: notificationId)
        )
    }
}

private struct EquipTitleRequest: Encodable {
    let titleId: Int64
    let equip: Bool
}

private struct NotificationIdRequest: Encodable {
    let notificationId: Int64
}
