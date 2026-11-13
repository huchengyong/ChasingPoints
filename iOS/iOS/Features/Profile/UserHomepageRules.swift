import Foundation

struct CoreMetric: Equatable, Identifiable {
    /// total / wins / win-rate / max-streak。
    let key: String
    let label: String
    /// 加载中为 "—"。
    let value: String

    var id: String { key }
}

struct RankCardView: Equatable {
    let ranked: Bool
    let name: String
    let iconURL: URL?
    let score: Int
    /// 0...100。
    let progress: Int
    let hint: String
}

struct RankGameTypeView: Equatable {
    let primaryGameType: Int
    let primaryLabel: String
    /// 不足两个时为空。
    let tabs: [GameTypeOption]
    let needsPick: Bool
}

struct ReputationChip: Equatable {
    let text: String
    let restricted: Bool
}

struct MemberEntry: Equatable {
    /// none / active / expired。
    let state: String
    let text: String
}

struct CompactRewardEntry: Equatable {
    let visible: Bool
    let floatable: Bool
    let inline: Bool
    let state: String
    let title: String
    let description: String
    let statusText: String
    /// "favorite-venue" 或 ""。
    let action: String
    let actionText: String
}

/// 「我的」页的展示规则，逐条对应小程序 `utils/user-homepage.js`。
enum UserHomepageRules {
    private static let defaultRankIcon = "/static/images/ranks/rank_bronze.png"
    private static let loadingValue = "—"

    /// 生涯数据统计全部球种；段位分随球种切换，只放在段位卡里。
    static func coreMetrics(stats: UserStats?, isLoading: Bool) -> [CoreMetric] {
        let winRate = stats?.winRate ?? 0
        return [
            CoreMetric(
                key: "total", label: "场次",
                value: isLoading ? loadingValue : String(stats?.totalMatches ?? 0)
            ),
            CoreMetric(
                key: "wins", label: "胜场",
                value: isLoading ? loadingValue : String(stats?.wins ?? 0)
            ),
            CoreMetric(
                key: "win-rate", label: "胜率",
                value: isLoading ? loadingValue : "\(jsNumber(winRate))%"
            ),
            CoreMetric(
                key: "max-streak", label: "最高连胜",
                value: isLoading ? loadingValue : String(stats?.maxWinStreak ?? 0)
            ),
        ]
    }

    /// 段位卡：`next_score` 是下一段位的起始分；6 级（王者）为最高段位。
    static func rankCard(_ rankInfo: RankInfo?) -> RankCardView {
        guard let rankInfo else {
            return RankCardView(
                ranked: false,
                name: "未定级",
                iconURL: CDNAsset.url(for: defaultRankIcon),
                score: 0,
                progress: 0,
                hint: "完成首场排位赛后开始定级"
            )
        }

        let isMaxLevel = rankInfo.level >= 6
        let score = rankInfo.rankScore
        let nextScore = rankInfo.nextScore
        let progress = isMaxLevel ? 100 : min(100, max(0, rankInfo.progress))
        var hint = "晋级进度 \(progress)%"
        if isMaxLevel {
            hint = "已达到最高段位"
        } else if nextScore > 0 {
            let nextName = trimmed(rankInfo.nextName).isEmpty ? "下一段位" : trimmed(rankInfo.nextName)
            hint = "距\(nextName)还差 \(max(0, nextScore - score)) 分"
        }

        return RankCardView(
            ranked: true,
            name: trimmed(rankInfo.name).isEmpty ? "未定级" : trimmed(rankInfo.name),
            iconURL: CDNAsset.url(for: trimmed(rankInfo.icon).isEmpty ? defaultRankIcon : trimmed(rankInfo.icon)),
            score: score,
            progress: progress,
            hint: hint
        )
    }

    /// 段位卡球种：优先用户设的常用球种；没设时取排位场次最多的球种，同场次比排位分，再按球种固定顺序。
    /// 只在常用球种与打过排位的球种之间切换，不足两个时不显示切换。
    static func rankGameTypeView(preferredGameType: Int, rankInfoMap: RankInfoMap) -> RankGameTypeView {
        func rankedMatches(_ option: GameTypeOption) -> Int {
            guard let info = rankInfoMap[option.value] else { return 0 }
            return info.totalWins + info.totalLosses
        }

        let played = GameTypeCatalog.ordered.filter { rankedMatches($0) > 0 }
        let preferred = GameTypeCatalog.ordered.first { $0.value == preferredGameType }
        var mostPlayed: GameTypeOption?
        for item in played {
            guard let best = mostPlayed else {
                mostPlayed = item
                continue
            }
            let matchDiff = rankedMatches(item) - rankedMatches(best)
            if matchDiff != 0 {
                mostPlayed = matchDiff > 0 ? item : best
            } else if (rankInfoMap[item.value]?.rankScore ?? 0) > (rankInfoMap[best.value]?.rankScore ?? 0) {
                mostPlayed = item
            }
        }

        let primary = preferred ?? mostPlayed ?? GameTypeCatalog.ordered[0]
        let tabs = GameTypeCatalog.ordered.filter { $0 == primary || played.contains($0) }
        return RankGameTypeView(
            primaryGameType: primary.value,
            primaryLabel: primary.label,
            tabs: tabs.count > 1 ? tabs : [],
            needsPick: preferred == nil && played.isEmpty
        )
    }

    static func reputationChip(_ reputation: ReputationStatus?, isLoading: Bool) -> ReputationChip {
        let statusText = trimmed(reputation?.statusText)
        guard !statusText.isEmpty else {
            return ReputationChip(text: isLoading ? "信誉同步中" : "信誉暂不可用", restricted: false)
        }
        let restricted = reputation?.status == "restricted"
        return ReputationChip(text: restricted ? statusText : "信誉\(statusText)", restricted: restricted)
    }

    /// 本人「我的」页的会员身份：会员有效且读到真实段位才显示头像框，段位加载中不沿用旧数据。
    static func selfMemberIdentity(
        isLoggedIn: Bool,
        memberStatus: MemberStatus?,
        rankInfo: RankInfo?,
        rankLoading: Bool,
        now: Date
    ) -> MemberIdentity {
        guard isLoggedIn else { return MemberIdentity.resolve(vipLevel: 0, rankLevel: 0) }

        let expiresAt = ServerTime.flexible(memberStatus?.memberExpiresAt)
        let active = memberStatus?.isActive == true && expiresAt.map { $0 > now } == true
        let rankLevel = rankLoading ? 0 : (rankInfo?.level ?? 0)
        return MemberIdentity.resolve(
            vipLevel: active ? MemberGrowthCard.resolve(memberStatus, now: now).level : 0,
            rankLevel: rankLevel
        )
    }

    /// 会员中心入口的右侧状态：从未开通时不提示，避免把入口做成推销位。
    static func memberEntry(_ memberStatus: MemberStatus?, now: Date) -> MemberEntry {
        let expiresAtText = trimmed(memberStatus?.memberExpiresAt)
        guard !expiresAtText.isEmpty else { return MemberEntry(state: "none", text: "") }

        let expiresAt = ServerTime.flexible(expiresAtText)
        let isActive = memberStatus?.isActive == true && (expiresAt.map { $0 > now } ?? true)
        return isActive
            ? MemberEntry(state: "active", text: "\(String(expiresAtText.prefix(10))) 到期")
            : MemberEntry(state: "expired", text: "已到期")
    }

    static func compactRewardEntry(_ rewardStatus: FavoriteVenueRewardStatus?) -> CompactRewardEntry {
        let card = FavoriteVenueReward.taskCard(rewardStatus)
        let rawStatus = trimmed(rewardStatus?.status)
        let state = rawStatus.isEmpty ? "not_started" : rawStatus
        let floatable = card.visible && (state == "not_started" || state == "rejected")
        let inline = card.visible && state == "pending_review"

        var title = card.title
        var description = card.description
        if state == "not_started" {
            title = "免费VIP赠送"
            description = "绑定你常去的球馆，审核通过后可免费领取 "
                + FavoriteVenueReward.formatDuration(rewardDays: FavoriteVenueReward.rewardDays(rewardStatus))
        }

        return CompactRewardEntry(
            visible: card.visible,
            floatable: floatable,
            inline: inline,
            state: state,
            title: title,
            description: description,
            statusText: card.statusText,
            action: floatable ? "favorite-venue" : "",
            actionText: floatable ? card.actionText : ""
        )
    }

    /// 资料头部等级标签：对应页面里的 `profileLevelText`。
    static func profileLevelText(
        memberStatus: MemberStatus?,
        identity: MemberIdentity,
        isLoading: Bool,
        now: Date
    ) -> String {
        if memberStatus == nil && isLoading { return "等级同步中" }
        if identity.showBadge { return "\(identity.badgeText) · \(identity.tierLabel)" }
        return MemberGrowthCard.resolve(memberStatus, now: now).levelLabel
    }

    /// JS 数字转字符串：整数不带小数点（50 → "50"），小数保留有效数字（66.7 → "66.7"）。
    private static func jsNumber(_ value: Double) -> String {
        guard value.isFinite else { return "0" }
        if value.rounded() == value, abs(value) < 1e15 {
            return String(Int64(value))
        }
        return String(value)
    }

    private static func trimmed(_ value: String?) -> String {
        (value ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
