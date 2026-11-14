import SwiftUI

/// 战绩海报，版式、文案与配色复刻 `posterGenerator.js` 的 `drawMatchPoster`。
/// 分享者是有效会员时用 `MEMBER_POSTER_THEMES` 配色与头像行，否则保持标准模板。
struct ShareResultPoster: View {
    let payload: SharePosterPayload
    let qrImage: UIImage?
    let assets: SharePosterAssets

    init(payload: SharePosterPayload, qrImage: UIImage?, assets: SharePosterAssets = .empty) {
        self.payload = payload
        self.qrImage = qrImage
        self.assets = assets
    }

    private var theme: SharePosterTheme? {
        payload.isMember ? SharePosterTheme.resolve(payload.myIdentity.tier) : nil
    }

    /// 会员模板在头部多出的高度，卡片下方所有内容整体下移。
    private var dy: CGFloat { payload.headerExtra }

    private var backgroundColors: [Color] {
        guard let theme else { return [Color(hex: 0x0F172A), Color(hex: 0x111827)] }
        return [Color(hex: theme.backgroundTop), Color(hex: theme.backgroundBottom)]
    }

    var body: some View {
        ZStack(alignment: .topLeading) {
            LinearGradient(colors: backgroundColors, startPoint: .topLeading, endPoint: .bottomTrailing)

            Circle()
                .fill(theme.map { Color(hex: $0.glowPrimary, opacity: $0.glowPrimaryOpacity) }
                    ?? Color(hex: 0x3B82F6, opacity: 0.12))
                .frame(width: 360, height: 360)
                .position(x: 640, y: 110)
            Circle()
                .fill(theme.map { Color(hex: $0.glowSecondary, opacity: $0.glowSecondaryOpacity) }
                    ?? Color(hex: 0x10B981, opacity: 0.10))
                .frame(width: 280, height: 280)
                .position(x: 80, y: 220)

            centered(
                ShareResultRules.headerTitle,
                size: 30,
                color: theme.map { Color(hex: $0.title) } ?? Color(hex: 0xCBD5E1),
                y: 84
            )
            centered(
                payload.gameTypeName,
                size: 24,
                color: theme.map { Color(hex: $0.subtitle) } ?? Color(hex: 0x94A3B8),
                y: 126
            )

            RoundedRectangle(cornerRadius: 32)
                .fill(theme.map { Color(hex: $0.card, opacity: $0.cardOpacity) }
                    ?? Color(hex: 0x0F172A, opacity: 0.72))
                .frame(width: 670, height: 300 + dy)
                .position(x: 375, y: 320 + dy / 2)

            ZStack {
                RoundedRectangle(cornerRadius: 24).fill(Color.white.opacity(0.08))
                Text(payload.resultText)
                    .font(.system(size: 24))
                    .foregroundStyle(resultColor)
            }
            .frame(width: 132, height: 48)
            .position(x: 126, y: 218)

            ZStack {
                RoundedRectangle(cornerRadius: 24)
                    .fill(theme.map { Color(hex: $0.chipBackground, opacity: $0.chipBackgroundOpacity) }
                        ?? Color(hex: 0x3B82F6, opacity: 0.14))
                Text(payload.rankChangeLabel)
                    .font(.system(size: 24))
                    .foregroundStyle(theme.map { Color(hex: $0.chipText) } ?? Color(hex: 0x93C5FD))
            }
            .frame(width: 112, height: 48)
            .position(x: 614, y: 218)

            memberHeader

            centered(payload.myName, size: 28, color: Color(hex: 0xF8FAFC), x: 156, y: 306 + dy)
            centered(payload.opponentName, size: 28, color: Color(hex: 0xF8FAFC), x: 594, y: 306 + dy)

            centered(
                ShareResultRules.scoreText(payload.myScore),
                size: 86,
                color: Color(hex: 0xF8FAFC),
                x: 172,
                y: 392 + dy
            )
            centered(":", size: 42, color: Color(hex: 0x64748B), x: 375, y: 382 + dy)
            centered(
                ShareResultRules.scoreText(payload.opponentScore),
                size: 86,
                color: Color(hex: 0xF8FAFC),
                x: 578,
                y: 392 + dy
            )

            centered(ShareResultRules.myScoreLabel, size: 22, color: Color(hex: 0x94A3B8), x: 100, y: 438 + dy)
            centered(ShareResultRules.opponentScoreLabel, size: 22, color: Color(hex: 0x94A3B8), x: 514, y: 438 + dy)

            Path { path in
                path.move(to: CGPoint(x: 70, y: 438 + dy))
                path.addLine(to: CGPoint(x: 680, y: 438 + dy))
            }
            .stroke(Color.white.opacity(0.08), lineWidth: 1)

            baselineText(
                ShareResultRules.matchTimeText(payload.matchTime),
                size: 22,
                color: Color(hex: 0xCBD5E1),
                width: 590,
                baseline: 510 + dy
            )

            baselineText(
                ShareResultRules.summaryNote,
                size: 22,
                color: Color(hex: 0xCBD5E1),
                width: 590,
                alignment: .trailing,
                baseline: 510 + dy
            )

            sectionHeader(
                ShareResultRules.highlightsTitle,
                subtitle: ShareResultRules.highlightsSubtitle(gameTypeName: payload.gameTypeName),
                y: 572 + dy
            )
            highlightCards

            sectionHeader(ShareResultRules.statsTitle, subtitle: ShareResultRules.statsSubtitle, y: payload.statsTop)
            statCards

            qrBlock
            centered(
                ShareResultRules.brand,
                size: 30,
                color: theme.map { Color(hex: $0.brand) } ?? Color(hex: 0x60A5FA),
                y: payload.height - 92
            )
            centered(
                ShareResultRules.footnote,
                size: 20,
                color: theme.map { Color(hex: $0.footnote) } ?? Color(hex: 0x64748B),
                y: payload.height - 54
            )
        }
        .frame(width: 750, height: payload.height, alignment: .topLeading)
        .clipped()
    }

    /// 会员模板：分享者头像 + 段位头像框 + VIP 标识；对手是会员时同样标出 VIP 标识。
    @ViewBuilder
    private var memberHeader: some View {
        if payload.isMember {
            avatarView(assets.myAvatar, name: payload.myName)
                .position(x: 156, y: 322)
            if let frame = assets.frame {
                let size = (ShareResultRules.memberAvatarRadius * 2 * 156 / 96).rounded()
                Image(uiImage: frame)
                    .resizable()
                    .scaledToFit()
                    .frame(width: size, height: size)
                    .position(x: 156, y: 322)
            }
            badgeView(
                SharePosterBadge(text: payload.myIdentity.badgeText, tier: payload.myIdentity.tier),
                scale: 1
            )
            .position(x: 156, y: 372)
            avatarView(assets.opponentAvatar, name: payload.opponentName)
                .position(x: 594, y: 322)
            if let badge = payload.opponentBadge {
                badgeView(badge, scale: 1)
                    .position(x: 594, y: 372)
            }
        } else if let badge = payload.opponentBadge {
            badgeView(badge, scale: 0.85)
                .position(x: 594, y: 264)
        }
    }

    /// 头像读不到时画名字首字的占位圆，海报仍然完整。
    private func avatarView(_ image: UIImage?, name: String) -> some View {
        ZStack {
            if let image {
                Image(uiImage: image)
                    .resizable()
                    .scaledToFill()
            } else {
                Circle().fill(Color.white.opacity(0.14))
                Text(String(name.prefix(1)))
                    .font(.system(size: 44))
                    .foregroundStyle(Color(hex: 0xF8FAFC))
            }
        }
        .frame(width: ShareResultRules.memberAvatarRadius * 2, height: ShareResultRules.memberAvatarRadius * 2)
        .clipShape(Circle())
    }

    private var resultColor: Color {
        switch payload.resultText {
        case "胜利": Color(hex: 0x22C55E)
        case "失利": Color(hex: 0xF97316)
        default: Color(hex: 0xCBD5E1)
        }
    }

    @ViewBuilder
    private var highlightCards: some View {
        if payload.highlights.isEmpty {
            RoundedRectangle(cornerRadius: 24)
                .fill(Color.white.opacity(0.05))
                .frame(width: 630, height: 146)
                .position(x: 375, y: 701 + dy)
            Text(ShareResultRules.emptyHighlights)
                .font(.system(size: 24))
                .foregroundStyle(Color(hex: 0x94A3B8))
                .position(x: 375, y: 710 + dy)
        } else {
            ForEach(Array(payload.highlights.enumerated()), id: \.offset) { index, card in
                SharePosterCardView(card: card)
                    .position(
                        x: 60 + CGFloat(index % 2) * (305 + 20) + 152.5,
                        y: 628 + dy + CGFloat(index / 2) * 170 + 73
                    )
            }
        }
    }

    @ViewBuilder
    private var statCards: some View {
        ForEach(Array(payload.stats.enumerated()), id: \.offset) { index, card in
            SharePosterCardView(card: card)
                .position(
                    x: 60 + CGFloat(index % 2) * (305 + 20) + 152.5,
                    y: payload.statsTop + 56 + CGFloat(index / 2) * 170 + 73
                )
        }
    }

    @ViewBuilder
    private var qrBlock: some View {
        if let qrImage {
            Rectangle()
                .fill(.white)
                .frame(width: 120, height: 120)
                .position(x: 120, y: payload.height - 156)
            Image(uiImage: qrImage)
                .interpolation(.none)
                .resizable()
                .scaledToFit()
                .frame(width: 108, height: 108)
                .position(x: 120, y: payload.height - 156)

            baselineText(
                payload.qrCaption,
                size: 22,
                color: Color(hex: 0xCBD5E1),
                width: 480,
                x: 444,
                baseline: payload.height - 176
            )
            baselineText(
                ShareResultRules.qrHint,
                size: 18,
                color: Color(hex: 0x64748B),
                width: 480,
                x: 444,
                baseline: payload.height - 140
            )
        }
    }

    @ViewBuilder
    private func sectionHeader(_ title: String, subtitle: String, y: CGFloat) -> some View {
        baselineText(title, size: 30, color: Color(hex: 0xF8FAFC), baseline: y)
        baselineText(subtitle, size: 20, color: Color(hex: 0x94A3B8), baseline: y + 34)
    }

    /// `posterGenerator.js` 的 y 都是文字基线；`.position` 对准的是行框中心。
    private func baselineText(
        _ text: String,
        size: CGFloat,
        color: Color,
        width: CGFloat = 630,
        alignment: Alignment = .leading,
        x: CGFloat = 375,
        baseline: CGFloat
    ) -> some View {
        let font = UIFont.systemFont(ofSize: size)
        return Text(text)
            .font(.system(size: size))
            .foregroundStyle(color)
            .frame(width: width, alignment: alignment)
            .position(x: x, y: baseline - font.ascender + font.lineHeight / 2)
    }

    private func centered(
        _ text: String,
        size: CGFloat,
        color: Color,
        x: CGFloat = 375,
        y: CGFloat
    ) -> some View {
        let font = UIFont.systemFont(ofSize: size)
        return Text(text)
            .font(.system(size: size))
            .foregroundStyle(color)
            .position(x: x, y: y - font.ascender + font.lineHeight / 2)
    }

    private func badgeView(_ badge: SharePosterBadge, scale: CGFloat) -> some View {
        let style = SharePosterBadgeStyle.resolve(badge.tier)
        return Text(badge.text)
            .font(.system(size: (22 * scale).rounded(), weight: .semibold))
            .foregroundStyle(style.text)
            .frame(width: (88 * scale).rounded(), height: (34 * scale).rounded())
            .background {
                if let gradient = style.gradient {
                    LinearGradient(colors: gradient, startPoint: .topLeading, endPoint: .bottomTrailing)
                } else {
                    style.fill
                }
            }
            .clipShape(Capsule())
            .overlay {
                Capsule().stroke(style.stroke, lineWidth: 1)
            }
    }

    /// 渲染到相册 / 分享用的位图；scale 2 按 750pt 画布输出 1500px 宽。
    @MainActor
    static func render(
        _ payload: SharePosterPayload,
        qrImage: UIImage?,
        assets: SharePosterAssets = .empty,
        scale: CGFloat = 2
    ) -> UIImage? {
        let renderer = ImageRenderer(
            content: ShareResultPoster(payload: payload, qrImage: qrImage, assets: assets)
        )
        renderer.scale = scale
        return renderer.uiImage
    }
}

/// 会员模板配色，逐项照搬 `posterGenerator.js` 的 `MEMBER_POSTER_THEMES`。
struct SharePosterTheme: Equatable {
    let backgroundTop: UInt32
    let backgroundBottom: UInt32
    let card: UInt32
    let cardOpacity: Double
    let glowPrimary: UInt32
    let glowPrimaryOpacity: Double
    let glowSecondary: UInt32
    let glowSecondaryOpacity: Double
    let title: UInt32
    let subtitle: UInt32
    let chipBackground: UInt32
    let chipBackgroundOpacity: Double
    let chipText: UInt32
    let brand: UInt32
    let footnote: UInt32

    static func resolve(_ tier: String?) -> SharePosterTheme {
        switch tier {
        case "advanced":
            SharePosterTheme(
                backgroundTop: 0x2E2209,
                backgroundBottom: 0x141109,
                card: 0x241A08,
                cardOpacity: 0.78,
                glowPrimary: 0xF0C542,
                glowPrimaryOpacity: 0.20,
                glowSecondary: 0xE0AE12,
                glowSecondaryOpacity: 0.10,
                title: 0xF5CF5A,
                subtitle: 0xE2D3A5,
                chipBackground: 0xF0C542,
                chipBackgroundOpacity: 0.20,
                chipText: 0xF5CF5A,
                brand: 0xF5CF5A,
                footnote: 0xA89A74
            )
        case "premium":
            SharePosterTheme(
                backgroundTop: 0x3C2B08,
                backgroundBottom: 0x8F6507,
                card: 0x140E04,
                cardOpacity: 0.72,
                glowPrimary: 0xFDE9A6,
                glowPrimaryOpacity: 0.18,
                glowSecondary: 0xFDE9A6,
                glowSecondaryOpacity: 0.10,
                title: 0xFDE9A6,
                subtitle: 0xF0DEB0,
                chipBackground: 0xFDE9A6,
                chipBackgroundOpacity: 0.20,
                chipText: 0xFDE9A6,
                brand: 0xFDE9A6,
                footnote: 0xD9C48A
            )
        default:
            SharePosterTheme(
                backgroundTop: 0x1E180D,
                backgroundBottom: 0x141109,
                card: 0x1E180D,
                cardOpacity: 0.78,
                glowPrimary: 0xE0AE12,
                glowPrimaryOpacity: 0.14,
                glowSecondary: 0xE0AE12,
                glowSecondaryOpacity: 0.08,
                title: 0xF0C542,
                subtitle: 0xD7C89B,
                chipBackground: 0xE0AE12,
                chipBackgroundOpacity: 0.16,
                chipText: 0xF0C542,
                brand: 0xF0C542,
                footnote: 0x9F926E
            )
        }
    }
}

private struct SharePosterCardView: View {
    let card: SharePosterCard

    var body: some View {
        let tone = SharePosterTone.resolve(card.tone)
        let labelFont = UIFont.systemFont(ofSize: 22)
        let valueFont = UIFont.systemFont(ofSize: 34)
        let descFont = UIFont.systemFont(ofSize: 18)

        ZStack(alignment: .topLeading) {
            RoundedRectangle(cornerRadius: 24).fill(tone.soft)
            Rectangle()
                .fill(tone.accent)
                .frame(width: 10, height: 146)
            Text(card.label)
                .font(.system(size: 22))
                .foregroundStyle(Color(hex: 0x94A3B8))
                .offset(x: 28, y: 42 - labelFont.ascender)
            Text(card.value)
                .font(.system(size: 34))
                .foregroundStyle(Color(hex: 0xF8FAFC))
                .offset(x: 28, y: 94 - valueFont.ascender)
            if let desc = card.desc {
                Text(desc)
                    .font(.system(size: 18))
                    .foregroundStyle(Color(hex: 0xCBD5E1))
                    .lineLimit(1)
                    .frame(width: 257, alignment: .leading)
                    .offset(x: 28, y: 132 - descFont.ascender)
            }
        }
        .frame(width: 305, height: 146, alignment: .topLeading)
        .clipShape(RoundedRectangle(cornerRadius: 24))
    }
}

private struct SharePosterTone {
    let accent: Color
    let soft: Color

    static func resolve(_ tone: String?) -> SharePosterTone {
        switch tone {
        case "gold": SharePosterTone(accent: Color(hex: 0xF59E0B), soft: Color(hex: 0xF59E0B, opacity: 0.18))
        case "emerald", "green": SharePosterTone(accent: Color(hex: 0x10B981), soft: Color(hex: 0x10B981, opacity: 0.16))
        case "red": SharePosterTone(accent: Color(hex: 0xEF4444), soft: Color(hex: 0xEF4444, opacity: 0.16))
        case "purple": SharePosterTone(accent: Color(hex: 0x8B5CF6), soft: Color(hex: 0x8B5CF6, opacity: 0.16))
        case "orange": SharePosterTone(accent: Color(hex: 0xF97316), soft: Color(hex: 0xF97316, opacity: 0.16))
        case "cyan": SharePosterTone(accent: Color(hex: 0x06B6D4), soft: Color(hex: 0x06B6D4, opacity: 0.16))
        case "slate": SharePosterTone(accent: Color(hex: 0x94A3B8), soft: Color(hex: 0x94A3B8, opacity: 0.16))
        default: SharePosterTone(accent: Color(hex: 0x3B82F6), soft: Color(hex: 0x3B82F6, opacity: 0.16))
        }
    }
}

private struct SharePosterBadgeStyle {
    let fill: Color
    let gradient: [Color]?
    let stroke: Color
    let text: Color

    static func resolve(_ tier: String) -> SharePosterBadgeStyle {
        switch tier {
        case "advanced":
            SharePosterBadgeStyle(
                fill: Color(hex: 0xD9A617),
                gradient: [Color(hex: 0xD9A617), Color(hex: 0xBB8400)],
                stroke: Color(hex: 0xBB8400, opacity: 0.6),
                text: .white
            )
        case "premium":
            SharePosterBadgeStyle(
                fill: Color(hex: 0x3C2B08),
                gradient: [Color(hex: 0x3C2B08), Color(hex: 0x6B4A07)],
                stroke: Color(hex: 0xC69200),
                text: Color(hex: 0xF0C542)
            )
        default:
            SharePosterBadgeStyle(
                fill: Color(hex: 0xE0AE12, opacity: 0.22),
                gradient: nil,
                stroke: Color(hex: 0xE0AE12),
                text: Color(hex: 0xF0C542)
            )
        }
    }
}
