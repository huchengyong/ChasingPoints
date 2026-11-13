import SwiftUI

/// 会员身份头像：头像 + 段位头像框 + VIP 标识，对应小程序 components/memberAvatar.vue。
struct MemberAvatarView: View {
    @Environment(\.colorScheme) private var colorScheme

    private let avatar: String
    private let userId: Int64
    private let vipLevel: Int
    private let rankLevel: Int
    private let size: CGFloat
    private let reserveFrame: Bool
    private let fitFrame: Bool
    private let showBadge: Bool

    init(
        avatar: String,
        userId: Int64,
        vipLevel: Int = 0,
        rankLevel: Int = 0,
        size: CGFloat = 48,
        reserveFrame: Bool = false,
        fitFrame: Bool = false,
        showBadge: Bool = true
    ) {
        self.avatar = avatar
        self.userId = userId
        self.vipLevel = vipLevel
        self.rankLevel = rankLevel
        self.size = size
        self.reserveFrame = reserveFrame
        self.fitFrame = fitFrame
        self.showBadge = showBadge
    }

    private var identity: MemberIdentity {
        MemberIdentity.resolve(vipLevel: vipLevel, rankLevel: rankLevel)
    }

    private var metrics: MemberAvatarMetrics {
        MemberAvatarMetrics.resolve(
            size: size,
            reserveFrame: reserveFrame,
            fitFrame: fitFrame,
            hasFrame: identity.frameURL != nil
        )
    }

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        let identity = self.identity
        let metrics = self.metrics

        AvatarView(avatar: avatar, userId: userId, size: metrics.avatarSize)
            .overlay {
                if let frameURL = identity.frameURL {
                    // 头像框素材比头像大一圈，不裁剪、允许溢出布局盒。
                    RemoteImage(url: frameURL, contentMode: .fit) {
                        Color.clear
                    }
                    .frame(width: metrics.frameSize, height: metrics.frameSize)
                    .allowsHitTesting(false)
                }
            }
            .overlay(alignment: .bottom) {
                if showBadge && identity.showBadge {
                    Text(identity.badgeText)
                        .font(.system(size: metrics.badgeFontSize, weight: .heavy))
                        .foregroundStyle(badgeTextColor(identity.tier))
                        .padding(.horizontal, metrics.badgePaddingX)
                        .frame(height: metrics.badgeHeight)
                        .background(badgeBackground(identity.tier, palette: palette))
                        .overlay {
                            Capsule().stroke(badgeBorderColor(identity.tier), lineWidth: 1)
                        }
                        .offset(y: metrics.badgeHeight * 0.35)
                        .allowsHitTesting(false)
                }
            }
            .frame(width: metrics.boxSize, height: metrics.boxSize)
            .accessibilityElement(children: .ignore)
            .accessibilityLabel(identity.showBadge ? "VIP\(identity.vipLevel) 会员" : "")
            .accessibilityHidden(!identity.showBadge)
    }

    // 会员档位色，与小程序 .member-avatar-badge--standard/advanced/premium 对应。
    private enum BadgeColor {
        static let standardText = Color(hex: 0xC69200)
        static let standardBorder = Color(hex: 0xE0AE12, opacity: 0.55)
        static let standardTint = Color(hex: 0xE0AE12, opacity: 0.14)

        static let advancedText = Color.white
        static let advancedBorder = Color(hex: 0xBB8400, opacity: 0.6)
        static let advancedStart = Color(hex: 0xD9A617)
        static let advancedEnd = Color(hex: 0xBB8400)

        static let premiumText = Color(hex: 0xF0C542)
        static let premiumBorder = Color(hex: 0xC69200)
        static let premiumStart = Color(hex: 0x3C2B08)
        static let premiumEnd = Color(hex: 0x6B4A07)
    }

    private func badgeTextColor(_ tier: String) -> Color {
        switch tier {
        case "advanced": BadgeColor.advancedText
        case "premium": BadgeColor.premiumText
        default: BadgeColor.standardText
        }
    }

    private func badgeBorderColor(_ tier: String) -> Color {
        switch tier {
        case "advanced": BadgeColor.advancedBorder
        case "premium": BadgeColor.premiumBorder
        default: BadgeColor.standardBorder
        }
    }

    @ViewBuilder
    private func badgeBackground(_ tier: String, palette: AppPalette) -> some View {
        switch tier {
        case "advanced":
            Capsule().fill(
                LinearGradient(
                    colors: [BadgeColor.advancedStart, BadgeColor.advancedEnd],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
        case "premium":
            Capsule().fill(
                LinearGradient(
                    colors: [BadgeColor.premiumStart, BadgeColor.premiumEnd],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
        default:
            Capsule()
                .fill(palette.card)
                .overlay(Capsule().fill(BadgeColor.standardTint))
        }
    }
}
