import SwiftUI

/// 战绩海报，版式、文案与配色复刻 `posterGenerator.js` 的 `drawMatchPoster` 标准模板。
struct ShareResultPoster: View {
    let payload: SharePosterPayload
    let qrImage: UIImage?

    var body: some View {
        ZStack(alignment: .topLeading) {
            LinearGradient(
                colors: [Color(hex: 0x0F172A), Color(hex: 0x111827)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )

            Circle()
                .fill(Color(hex: 0x3B82F6, opacity: 0.12))
                .frame(width: 360, height: 360)
                .position(x: 640, y: 110)
            Circle()
                .fill(Color(hex: 0x10B981, opacity: 0.10))
                .frame(width: 280, height: 280)
                .position(x: 80, y: 220)

            centered(ShareResultRules.headerTitle, size: 30, color: Color(hex: 0xCBD5E1), y: 84)
            centered(payload.gameTypeName, size: 24, color: Color(hex: 0x94A3B8), y: 126)

            RoundedRectangle(cornerRadius: 32)
                .fill(Color(hex: 0x0F172A, opacity: 0.72))
                .frame(width: 670, height: 300)
                .position(x: 375, y: 320)

            ZStack {
                RoundedRectangle(cornerRadius: 24).fill(Color.white.opacity(0.08))
                Text(payload.resultText)
                    .font(.system(size: 24))
                    .foregroundStyle(resultColor)
            }
            .frame(width: 132, height: 48)
            .position(x: 126, y: 218)

            ZStack {
                RoundedRectangle(cornerRadius: 24).fill(Color(hex: 0x3B82F6, opacity: 0.14))
                Text(payload.rankChangeLabel)
                    .font(.system(size: 24))
                    .foregroundStyle(Color(hex: 0x93C5FD))
            }
            .frame(width: 112, height: 48)
            .position(x: 614, y: 218)

            if let badge = payload.opponentBadge {
                badgeView(badge, scale: 0.85)
                    .position(x: 594, y: 264)
            }

            centered(payload.myName, size: 28, color: Color(hex: 0xF8FAFC), x: 156, y: 306)
            centered(payload.opponentName, size: 28, color: Color(hex: 0xF8FAFC), x: 594, y: 306)

            centered(ShareResultRules.scoreText(payload.myScore), size: 86, color: Color(hex: 0xF8FAFC), x: 172, y: 392)
            centered(":", size: 42, color: Color(hex: 0x64748B), x: 375, y: 382)
            centered(
                ShareResultRules.scoreText(payload.opponentScore),
                size: 86,
                color: Color(hex: 0xF8FAFC),
                x: 578,
                y: 392
            )

            centered(ShareResultRules.myScoreLabel, size: 22, color: Color(hex: 0x94A3B8), x: 100, y: 438)
            centered(ShareResultRules.opponentScoreLabel, size: 22, color: Color(hex: 0x94A3B8), x: 514, y: 438)

            Path { path in
                path.move(to: CGPoint(x: 70, y: 438))
                path.addLine(to: CGPoint(x: 680, y: 438))
            }
            .stroke(Color.white.opacity(0.08), lineWidth: 1)

            baselineText(
                ShareResultRules.matchTimeText(payload.matchTime),
                size: 22,
                color: Color(hex: 0xCBD5E1),
                width: 590,
                baseline: 510
            )

            baselineText(
                ShareResultRules.summaryNote,
                size: 22,
                color: Color(hex: 0xCBD5E1),
                width: 590,
                alignment: .trailing,
                baseline: 510
            )

            sectionHeader(
                ShareResultRules.highlightsTitle,
                subtitle: ShareResultRules.highlightsSubtitle(gameTypeName: payload.gameTypeName),
                y: 572
            )
            highlightCards

            sectionHeader(ShareResultRules.statsTitle, subtitle: ShareResultRules.statsSubtitle, y: payload.statsTop)
            statCards

            qrBlock
            centered(ShareResultRules.brand, size: 30, color: Color(hex: 0x60A5FA), y: payload.height - 92)
            centered(ShareResultRules.footnote, size: 20, color: Color(hex: 0x64748B), y: payload.height - 54)
        }
        .frame(width: 750, height: payload.height, alignment: .topLeading)
        .clipped()
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
                .position(x: 375, y: 701)
            Text(ShareResultRules.emptyHighlights)
                .font(.system(size: 24))
                .foregroundStyle(Color(hex: 0x94A3B8))
                .position(x: 375, y: 710)
        } else {
            ForEach(Array(payload.highlights.enumerated()), id: \.offset) { index, card in
                SharePosterCardView(card: card)
                    .position(
                        x: 60 + CGFloat(index % 2) * (305 + 20) + 152.5,
                        y: 628 + CGFloat(index / 2) * 170 + 73
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
    static func render(_ payload: SharePosterPayload, qrImage: UIImage?, scale: CGFloat = 2) -> UIImage? {
        let renderer = ImageRenderer(content: ShareResultPoster(payload: payload, qrImage: qrImage))
        renderer.scale = scale
        return renderer.uiImage
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
