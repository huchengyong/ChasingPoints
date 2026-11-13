import Foundation

/// 常玩球馆奖励的紧凑任务卡。
struct FavoriteVenueRewardTaskCard: Equatable {
    let visible: Bool
    let title: String
    let description: String
    let actionText: String
    let statusText: String
}

/// 迁移小程序 `favorite-venue-reward.js` 里「我的」页用到的规则。
enum FavoriteVenueReward {
    private static let snoozeDuration: TimeInterval = 3 * 24 * 60 * 60

    /// 奖励时长文案：整月用「N 个月VIP」，否则「N 天VIP」。
    static func formatDuration(rewardDays: Int) -> String {
        if rewardDays % 30 == 0 {
            return "\(rewardDays / 30) 个月VIP"
        }
        return "\(rewardDays) 天VIP"
    }

    /// 活动开启且本人尚未获得奖励时，才能对外承诺审核通过后发放VIP。
    static func isClaimable(_ status: FavoriteVenueRewardStatus?) -> Bool {
        status?.enabled == true && status?.status != "reward_granted"
    }

    /// `status` 为 nil 等同 JS 传入 `{}`（`reward_days` 缺省或为 0 按 30）。
    static func taskCard(_ status: FavoriteVenueRewardStatus?) -> FavoriteVenueRewardTaskCard {
        let rewardText = formatDuration(rewardDays: rewardDays(status))
        let claimable = isClaimable(status)
        let rawStatus = trimmed(status?.status)
        let state = rawStatus.isEmpty ? "not_started" : rawStatus
        let submittedVenueName = trimmed(status?.submittedVenueName)
        let rejectReason = trimmed(status?.rejectReason)

        if status?.enabled != true && state == "not_started" {
            return FavoriteVenueRewardTaskCard(
                visible: false, title: "", description: "", actionText: "", statusText: ""
            )
        }

        if state == "pending_review" {
            let reviewOutcome = claimable ? "通过后会自动发放VIP。" : "审核通过后会展示给附近球友。"
            let description = submittedVenueName.isEmpty
                ? "球馆信息审核中，\(reviewOutcome)"
                : "\(submittedVenueName) 正在审核中，\(reviewOutcome)"
            return FavoriteVenueRewardTaskCard(
                visible: true,
                title: "常玩球馆审核中",
                description: description,
                actionText: "",
                statusText: "审核中"
            )
        }

        if state == "reward_granted" {
            return FavoriteVenueRewardTaskCard(
                visible: false, title: "", description: "", actionText: "", statusText: ""
            )
        }

        if state == "rejected" {
            return FavoriteVenueRewardTaskCard(
                visible: true,
                title: claimable ? "补充信息后可重新领取VIP" : "补充信息后可重新提交",
                description: rejectReason.isEmpty ? "本次提交未通过，补充完整后可重新提交。" : "本次未通过：\(rejectReason)",
                actionText: "重新提交",
                statusText: "未通过"
            )
        }

        return FavoriteVenueRewardTaskCard(
            visible: true,
            title: "补充常玩球馆，送 \(rewardText)",
            description: "首次有效补充并审核通过后，VIP会自动到账。",
            actionText: "去领取",
            statusText: "常玩球馆奖励"
        )
    }

    static func floatSnoozeKey(userId: Int64, status: String) -> String {
        "favorite-venue-reward-float-snoozed-until:\(userId):\(status.isEmpty ? "not_started" : status)"
    }

    static func snoozeUntil(from now: Date) -> Date {
        now.addingTimeInterval(snoozeDuration)
    }

    static func isSnoozed(until: Date?, now: Date) -> Bool {
        guard let until else { return false }
        return now < until
    }

    /// JS `reward_days || 30`：缺省与 0 都按 30 天。
    static func rewardDays(_ status: FavoriteVenueRewardStatus?) -> Int {
        let days = status?.rewardDays ?? 0
        return days == 0 ? 30 : days
    }

    private static func trimmed(_ value: String?) -> String {
        (value ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
