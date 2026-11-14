import Foundation

/// VIP 中心文案规则，迁移小程序 `utils/member-center.js` 的非支付部分。
/// 支付、下单与订单轮询不进原生端；App 内购买接入前由 `AppFlags.canShowPaidMembershipEntry` 关闭入口。
struct MemberCenterSummary: Equatable {
    let statusText: String
    let title: String
    let description: String
    let primaryActionText: String
}

struct MemberEntryCard: Equatable {
    let visible: Bool
    let eyebrow: String
    let statusText: String
    let title: String
    let description: String
    let actionText: String
    let priceText: String
}

struct MemberTierPreview: Equatable, Identifiable {
    let tier: String
    let label: String
    let levelText: String
    let sampleLevel: Int
    let badgeText: String
    let current: Bool
    let reached: Bool

    var id: String { tier }
}

struct MemberIdentityPreview: Equatable {
    let active: Bool
    let frozen: Bool
    let level: Int
    let identity: MemberIdentity
    let hint: String
    let tiers: [MemberTierPreview]
}

/// 套餐只保留卡片选择规则；原生端当前不展示订阅区块。
struct MemberPlan: Equatable {
    let planCode: String
    let planName: String
}

struct MemberPlanCard: Equatable, Identifiable {
    let planCode: String
    let planName: String
    let selected: Bool

    var id: String { planCode }
}

enum MemberCenterRules {
    private static let tiers: [(tier: String, label: String, levelText: String, minLevel: Int, maxLevel: Int)] = [
        ("standard", "标准", "Lv1–Lv2", 1, 2),
        ("advanced", "进阶", "Lv3–Lv4", 3, 4),
        ("premium", "尊享", "Lv5", 5, 5),
    ]

    static func resolveEntryCard(
        _ memberStatus: MemberStatus?,
        now: Date,
        complianceMode: Bool
    ) -> MemberEntryCard {
        let expiresAtText = trimmed(memberStatus?.memberExpiresAt)
        let isActive = isActive(memberStatus, now: now)

        if complianceMode {
            if isActive {
                return MemberEntryCard(
                    visible: true,
                    eyebrow: "VIP权益",
                    statusText: "VIP权益中",
                    title: "VIP权益已生效",
                    description: "查看当前权益明细和有效期。",
                    actionText: "查看权益",
                    priceText: ""
                )
            }
            if !expiresAtText.isEmpty {
                return MemberEntryCard(
                    visible: true,
                    eyebrow: "VIP权益",
                    statusText: "已结束",
                    title: "获赠VIP已结束",
                    description: "上次获赠VIP有效期到 \(expiresAtText)，新的权益到账后会继续展示。",
                    actionText: "查看权益",
                    priceText: ""
                )
            }
            return MemberEntryCard(
                visible: false, eyebrow: "", statusText: "", title: "", description: "", actionText: "", priceText: ""
            )
        }

        if isActive {
            return MemberEntryCard(
                visible: true,
                eyebrow: "VIP中心",
                statusText: "VIP中",
                title: "VIP有效期至 \(expiresAtText)",
                description: "当前VIP已生效，可随时续费顺延时长。",
                actionText: "立即续费",
                priceText: "月卡 ¥19"
            )
        }
        if !expiresAtText.isEmpty {
            return MemberEntryCard(
                visible: true,
                eyebrow: "VIP中心",
                statusText: "已到期",
                title: "VIP已到期",
                description: "上次VIP有效期到 \(expiresAtText)，现在可以继续续费。",
                actionText: "立即续费",
                priceText: "月卡 ¥19"
            )
        }
        return MemberEntryCard(
            visible: true,
            eyebrow: "VIP中心",
            statusText: "未开通",
            title: "月卡VIP",
            description: "开通后即可在这里查看VIP状态，并随时继续订阅。",
            actionText: "立即开通",
            priceText: "月卡 ¥19"
        )
    }

    static func resolveSummary(
        _ memberStatus: MemberStatus?,
        now: Date,
        complianceMode: Bool
    ) -> MemberCenterSummary {
        let expiresAtText = trimmed(memberStatus?.memberExpiresAt)
        let isActive = isActive(memberStatus, now: now)
        let growth = MemberGrowthCard.resolve(memberStatus, now: now)
        let tierLabel = tierLabel(level: growth.level)

        if complianceMode {
            if isActive {
                return MemberCenterSummary(
                    statusText: "VIP权益中",
                    title: "获赠VIP权益已生效",
                    description: "当前有效期至 \(expiresAtText)，新的系统奖励或活动奖励也会继续同步到这里。",
                    primaryActionText: "查看权益"
                )
            }
            if !expiresAtText.isEmpty {
                return MemberCenterSummary(
                    statusText: "已结束",
                    title: "获赠VIP权益已结束",
                    description: "上次获赠VIP有效期到 \(expiresAtText)，如后续继续发放，会在这里展示最新状态。",
                    primaryActionText: "查看权益"
                )
            }
            return MemberCenterSummary(
                statusText: "暂无权益",
                title: "暂未获得VIP权益",
                description: "活动奖励到账后，无需订阅或支付，会自动更新这里的状态。",
                primaryActionText: "查看权益"
            )
        }

        if isActive {
            let growthStatus = "Lv\(growth.level)，今日已计入 \(growth.todayGrowthCount)/\(growth.dailyCap)"
            return MemberCenterSummary(
                statusText: "VIP中",
                title: "月卡VIP生效中",
                description: "当前有效期至 \(expiresAtText)，VIP成长 \(growthStatus)，展示「\(tierLabel)」外观。",
                primaryActionText: "立即续费"
            )
        }
        if !expiresAtText.isEmpty {
            let hasGrowthProgress = growth.growthPoints > 0 || growth.level > 1 || memberStatus?.growthFrozen == true
            let description = hasGrowthProgress
                ? "上次VIP有效期到 \(expiresAtText)，当前成长 Lv\(growth.level) 已冻结，续开后恢复继续成长和「\(tierLabel)」外观。"
                : "上次VIP有效期到 \(expiresAtText)，续费后会从当前时间重新开始计算。"
            return MemberCenterSummary(
                statusText: "已到期",
                title: "VIP已到期",
                description: description,
                primaryActionText: "立即续费"
            )
        }
        return MemberCenterSummary(
            statusText: "未开通",
            title: "开通月卡VIP",
            description: "目前先开放月卡订阅，开通后完成真实对局可累计VIP成长，成长等级解锁更高阶外观。",
            primaryActionText: "立即开通"
        )
    }

    static func resolveIdentityPreview(
        _ memberStatus: MemberStatus?,
        now: Date,
        rankLevel: Int,
        complianceMode: Bool
    ) -> MemberIdentityPreview {
        let expiresAtText = trimmed(memberStatus?.memberExpiresAt)
        let isActive = isActive(memberStatus, now: now)
        let growth = MemberGrowthCard.resolve(memberStatus, now: now)
        let level = growth.level
        let frozen = !isActive && !expiresAtText.isEmpty
        let identity = MemberIdentity.resolve(vipLevel: isActive ? level : 0, rankLevel: rankLevel)
        let tierLabel = tierLabel(level: level)

        let hint: String
        if isActive {
            hint = "当前展示「\(identity.tierLabel)」外观：别人会在排行榜、对局和好友列表里看到你的头像框与 \(identity.badgeText)。"
        } else if frozen {
            hint = "VIP已到期，外观已隐藏；\(complianceMode ? "权益恢复" : "续开")后会恢复「\(tierLabel)」外观。"
        } else {
            hint = complianceMode
                ? "获得VIP权益后，别人会在排行榜、对局和好友列表里看到你的头像框与 VIP 标识。"
                : "开通VIP后，别人会在排行榜、对局和好友列表里看到你的头像框与 VIP 标识。"
        }

        return MemberIdentityPreview(
            active: isActive,
            frozen: frozen,
            level: level,
            identity: identity,
            hint: hint,
            tiers: tiers.map {
                MemberTierPreview(
                    tier: $0.tier,
                    label: $0.label,
                    levelText: $0.levelText,
                    sampleLevel: $0.minLevel,
                    badgeText: "VIP\($0.minLevel)",
                    current: isActive && $0.tier == identity.tier,
                    reached: level >= $0.minLevel
                )
            }
        )
    }

    static func resolvePlanCards(_ plans: [MemberPlan], selectedPlanCode: String) -> [MemberPlanCard] {
        plans.map {
            MemberPlanCard(planCode: $0.planCode, planName: $0.planName, selected: $0.planCode == selectedPlanCode)
        }
    }

    static func formatPrice(priceFen: Int = 1900) -> String {
        String(format: "¥%.2f", Double(priceFen) / 100)
    }

    private static func isActive(_ memberStatus: MemberStatus?, now: Date) -> Bool {
        guard memberStatus?.isActive == true, let expiresAt = ServerTime.utc8(memberStatus?.memberExpiresAt) else {
            return false
        }
        return expiresAt > now
    }

    private static func tierLabel(level: Int) -> String {
        tiers.first { level >= $0.minLevel && level <= $0.maxLevel }?.label ?? tiers[tiers.count - 1].label
    }

    private static func trimmed(_ value: String?) -> String {
        (value ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
