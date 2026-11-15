import Foundation
@testable import iOS

/// 「我的」页接口与规则的虚构夹具，字段与 `backend/chasing_points.api` 及 logic 的实际响应一致。
/// 全部为假数据：昵称、球馆与二维码正文均为占位，不包含真实账号或邀请凭证。
enum ProfileFixtures {
    static let memberExpiresAt = "2026-08-01 12:00:00"

    /// 把测试里的 ISO 8601 时间文本转成固定时刻。
    static func time(_ iso: String) -> Date {
        guard let date = ISO8601DateFormatter().date(from: iso) else {
            fatalError("测试时间格式错误：\(iso)")
        }
        return date
    }

    // MARK: - 接口响应（snake_case）

    static func userStats(
        success: Bool = true,
        totalMatches: Int = 86,
        wins: Int = 51,
        losses: Int = 35,
        winRate: Double = 59.3,
        maxWinStreak: Int = 7
    ) -> [String: Any] {
        [
            "success": success,
            "total_matches": totalMatches,
            "wins": wins,
            "losses": losses,
            "win_rate": winRate,
            "max_win_streak": maxWinStreak,
        ]
    }

    static func reputation(
        success: Bool = true,
        score: Int = 100,
        status: String = "good",
        statusText: String = "良好",
        banUntil: String? = nil
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "success": success,
            "score": score,
            "status": status,
            "status_text": statusText,
        ]
        if let banUntil { payload["ban_until"] = banUntil }
        return payload
    }

    static func memberStatusJSON(
        success: Bool = true,
        isActive: Bool = true,
        currentTime: String = "2026-07-20 12:00:00",
        planCode: String? = "monthly",
        planName: String? = "月卡",
        memberExpiresAt: String? = memberExpiresAt,
        growthLevel: Int? = nil,
        growthPoints: Int? = nil,
        todayGrowthCount: Int? = nil,
        growthDailyCap: Int? = nil,
        growthFrozen: Bool? = nil,
        nextGrowthLevel: Int? = nil,
        nextGrowthLevelPoints: Int? = nil,
        remainingGrowthPoints: Int? = nil
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "success": success,
            "is_active": isActive,
            "current_time": currentTime,
        ]
        if let planCode { payload["plan_code"] = planCode }
        if let planName { payload["plan_name"] = planName }
        if let memberExpiresAt { payload["member_expires_at"] = memberExpiresAt }
        if let growthLevel { payload["growth_level"] = growthLevel }
        if let growthPoints { payload["growth_points"] = growthPoints }
        if let todayGrowthCount { payload["today_growth_count"] = todayGrowthCount }
        if let growthDailyCap { payload["growth_daily_cap"] = growthDailyCap }
        if let growthFrozen { payload["growth_frozen"] = growthFrozen }
        if let nextGrowthLevel { payload["next_growth_level"] = nextGrowthLevel }
        if let nextGrowthLevelPoints { payload["next_growth_level_points"] = nextGrowthLevelPoints }
        if let remainingGrowthPoints { payload["remaining_growth_points"] = remainingGrowthPoints }
        return payload
    }

    static func favoriteVenueRewardJSON(
        success: Bool = true,
        enabled: Bool = true,
        popupEnabled: Bool = true,
        rewardDays: Int = 30,
        newUserWindowDays: Int = 7,
        status: String = "not_started",
        memberExpiresAt: String? = nil,
        submittedVenueId: Int64? = nil,
        submittedVenueName: String? = nil,
        rejectReason: String? = nil
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "success": success,
            "enabled": enabled,
            "popup_enabled": popupEnabled,
            "reward_days": rewardDays,
            "new_user_window_days": newUserWindowDays,
            "status": status,
        ]
        if let memberExpiresAt { payload["member_expires_at"] = memberExpiresAt }
        if let submittedVenueId { payload["submitted_venue_id"] = submittedVenueId }
        if let submittedVenueName { payload["submitted_venue_name"] = submittedVenueName }
        if let rejectReason { payload["reject_reason"] = rejectReason }
        return payload
    }

    /// 不传的分区按服务端失败时的形态留空。
    static func overview(
        stats: Any = NSNull(),
        reputation: Any = NSNull(),
        memberStatus: Any = NSNull(),
        favoriteVenueReward: Any = NSNull()
    ) -> [String: Any] {
        [
            "success": true,
            "stats": stats,
            "reputation": reputation,
            "member_status": memberStatus,
            "favorite_venue_reward": favoriteVenueReward,
        ]
    }

    static func rankInfoJSON(
        level: Int = 3,
        name: String = "黄金",
        icon: String = "/static/images/ranks/rank_gold.png",
        rankScore: Int = 1000,
        totalWins: Int = 0,
        totalLosses: Int = 0,
        maxStreak: Int = 0,
        nextLevel: Int = 4,
        nextName: String = "铂金",
        nextScore: Int = 1500,
        progress: Int = 0,
        placementRemaining: Int? = nil
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "level": level,
            "name": name,
            "icon": icon,
            "rank_score": rankScore,
            "total_wins": totalWins,
            "total_losses": totalLosses,
            "max_streak": maxStreak,
            "next_level": nextLevel,
            "next_name": nextName,
            "next_score": nextScore,
            "progress": progress,
        ]
        if let placementRemaining { payload["placement_remaining"] = placementRemaining }
        return payload
    }

    static func rankInfos(_ items: [[String: Any]]) -> [String: Any] {
        ["success": true, "rank_infos": items]
    }

    static func rankInfoItem(gameType: Int, rankInfo: Any?) -> [String: Any] {
        ["game_type": gameType, "rank_info": rankInfo ?? NSNull()]
    }

    static func matchQRCode(
        qrcodeData: String = "https://zhuifen.cn/invite?code=ABC123",
        expiresInSeconds: Int64? = 300
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "success": true,
            "qrcode_data": qrcodeData,
            "invite_token": "fixture-invite-token",
        ]
        if let expiresInSeconds { payload["expires_in_seconds"] = expiresInSeconds }
        return payload
    }

    // MARK: - 规则测试用的模型

    static func statsModel(
        totalMatches: Int = 0,
        wins: Int = 0,
        losses: Int = 0,
        winRate: Double = 0,
        maxWinStreak: Int = 0
    ) -> UserStats {
        UserStats(
            success: true,
            totalMatches: totalMatches,
            wins: wins,
            losses: losses,
            winRate: winRate,
            maxWinStreak: maxWinStreak
        )
    }

    static func reputationModel(
        status: String = "good",
        statusText: String = "良好"
    ) -> ReputationStatus {
        ReputationStatus(success: true, score: 100, status: status, statusText: statusText, banUntil: nil)
    }

    static func memberStatusModel(
        isActive: Bool = false,
        memberExpiresAt: String? = ProfileFixtures.memberExpiresAt,
        growthLevel: Int? = nil,
        growthPoints: Int? = nil,
        todayGrowthCount: Int? = nil,
        growthDailyCap: Int? = nil,
        growthFrozen: Bool? = nil,
        nextGrowthLevel: Int? = nil,
        nextGrowthLevelPoints: Int? = nil,
        remainingGrowthPoints: Int? = nil
    ) -> MemberStatus {
        MemberStatus(
            success: true,
            isActive: isActive,
            currentTime: "2026-07-20 12:00:00",
            planCode: nil,
            planName: nil,
            memberExpiresAt: memberExpiresAt,
            growthLevel: growthLevel,
            growthPoints: growthPoints,
            todayGrowthCount: todayGrowthCount,
            growthDailyCap: growthDailyCap,
            growthFrozen: growthFrozen,
            nextGrowthLevel: nextGrowthLevel,
            nextGrowthLevelPoints: nextGrowthLevelPoints,
            remainingGrowthPoints: remainingGrowthPoints
        )
    }

    static func favoriteVenueRewardModel(
        enabled: Bool = true,
        rewardDays: Int = 30,
        status: String = "not_started",
        submittedVenueName: String? = nil,
        rejectReason: String? = nil
    ) -> FavoriteVenueRewardStatus {
        FavoriteVenueRewardStatus(
            success: true,
            enabled: enabled,
            popupEnabled: true,
            rewardDays: rewardDays,
            newUserWindowDays: 7,
            status: status,
            memberExpiresAt: nil,
            submittedVenueId: nil,
            submittedVenueName: submittedVenueName,
            rejectReason: rejectReason
        )
    }

    static func rankInfoModel(
        level: Int = 1,
        name: String = "青铜",
        icon: String = "",
        rankScore: Int = 1000,
        totalWins: Int = 0,
        totalLosses: Int = 0,
        maxStreak: Int = 0,
        nextLevel: Int = 2,
        nextName: String = "",
        nextScore: Int = 0,
        progress: Int = 0,
        placementRemaining: Int? = nil
    ) -> RankInfo {
        RankInfo(
            level: level,
            name: name,
            icon: icon,
            rankScore: rankScore,
            totalWins: totalWins,
            totalLosses: totalLosses,
            maxStreak: maxStreak,
            nextLevel: nextLevel,
            nextName: nextName,
            nextScore: nextScore,
            progress: progress,
            placementRemaining: placementRemaining
        )
    }
}
