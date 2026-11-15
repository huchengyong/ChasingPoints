import CoreGraphics
import Foundation

enum DefaultAvatar {
    /// userId 为正整数时取 ((userId - 1) % 15) + 1 号球，否则 1 号。返回 asset 名，如 "DefaultAvatar07"。
    static func assetName(userId: Int64) -> String {
        let ball = userId > 0 ? Int((userId - 1) % 15) + 1 : 1
        return String(format: "DefaultAvatar%02d", ball)
    }
}

/// 会员身份的展示规则：VIP 成长等级决定标识档位，该球种的当前段位决定头像框。
/// 身份只是外观，与排位分、对局权限无关。
struct MemberIdentity: Equatable {
    let showBadge: Bool
    /// 0 表示非会员。
    let vipLevel: Int
    /// "standard" / "advanced" / "premium" / ""。
    let tier: String
    /// 标准 / 进阶 / 尊享 / ""。
    let tierLabel: String
    /// "VIP3" / ""。
    let badgeText: String
    /// 段位头像框（CDN），非会员或没有段位时为 nil。
    let frameURL: URL?

    private static let maxVipLevel = 5
    private static let tiers: [(tier: String, label: String, levels: ClosedRange<Int>)] = [
        ("standard", "标准", 1...2),
        ("advanced", "进阶", 3...4),
        ("premium", "尊享", 5...5),
    ]
    private static let rankFramePaths: [Int: String] = [
        1: "/static/images/avatar-frames/frame_bronze.png",
        2: "/static/images/avatar-frames/frame_silver.png",
        3: "/static/images/avatar-frames/frame_gold.png",
        4: "/static/images/avatar-frames/frame_platinum.png",
        5: "/static/images/avatar-frames/frame_diamond.png",
        6: "/static/images/avatar-frames/frame_king.png",
    ]

    static func resolve(vipLevel: Int, rankLevel: Int) -> MemberIdentity {
        // 接口只返回有效会员的 1–5；缺失、0、非法值都按非会员处理，超出上限按最高档。
        let level = vipLevel < 1 ? 0 : min(vipLevel, maxVipLevel)
        guard let tier = tiers.first(where: { $0.levels.contains(level) }) else {
            return MemberIdentity(
                showBadge: false, vipLevel: 0, tier: "", tierLabel: "", badgeText: "", frameURL: nil
            )
        }
        return MemberIdentity(
            showBadge: true,
            vipLevel: level,
            tier: tier.tier,
            tierLabel: tier.label,
            badgeText: "VIP\(level)",
            frameURL: CDNAsset.url(for: rankFramePaths[rankLevel] ?? "")
        )
    }
}

struct MemberAvatarMetrics: Equatable {
    let avatarSize: CGFloat
    let boxSize: CGFloat
    let frameSize: CGFloat
    let badgeFontSize: CGFloat
    let badgeHeight: CGFloat
    let badgePaddingX: CGFloat

    /// 头像框素材的翼饰横向几乎铺满整图，所以框比头像大一圈。
    private static let frameSizeRatio: CGFloat = 156 / 96
    private static let frameBoxRatio: CGFloat = 124 / 96
    private static let minBadgeFontSize: CGFloat = 9
    private static let maxBadgeFontSize: CGFloat = 13

    /// size 为头像直径（point）。比例与小程序一致；标识字号下限 9、上限 13（point）。
    ///
    /// - 默认：布局盒就是头像本身，框图向外溢出（用于四周有空白的位置）。
    /// - reserveFrame：布局盒按头像框预留，会员与非会员占位一致（用于列表行）。
    /// - fitFrame：size 是含头像框的总占位。有头像框时头像缩小到框内，框图不再向外溢出（用于领奖台这类空间固定的位置）。
    static func resolve(
        size: CGFloat,
        reserveFrame: Bool = false,
        fitFrame: Bool = false,
        hasFrame: Bool = false
    ) -> MemberAvatarMetrics {
        let requested = max(0, size).rounded()
        let fitted = fitFrame && hasFrame
        let avatarSize = fitted ? (requested / frameSizeRatio).rounded() : requested
        let badgeFontSize = min(maxBadgeFontSize, max(minBadgeFontSize, (avatarSize * 0.22).rounded()))
        return MemberAvatarMetrics(
            avatarSize: avatarSize,
            boxSize: fitted ? requested : (reserveFrame ? (avatarSize * frameBoxRatio).rounded() : avatarSize),
            frameSize: fitted ? requested : (avatarSize * frameSizeRatio).rounded(),
            badgeFontSize: badgeFontSize,
            badgeHeight: (badgeFontSize * 1.5).rounded(),
            badgePaddingX: (badgeFontSize * 0.45).rounded()
        )
    }
}
