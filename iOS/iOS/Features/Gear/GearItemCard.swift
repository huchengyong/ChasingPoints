import SwiftUI

/// 装备卡片：图、名称、副标题、主力标签，以及球杆当前的皮头与巧克。
struct GearItemCard: View {
    @Environment(\.colorScheme) private var colorScheme

    let item: GearItemInfo
    var badges: [String] = []

    private var retired: Bool { item.status == GearItemStatus.retired.rawValue }
    private var subtitle: String { GearDisplayRules.itemSubtitle(item) }
    private var partsLine: String {
        var parts: [String] = []
        if let tip = item.tipProduct { parts.append("皮头 \(GearDisplayRules.productTitle(tip))") }
        if let chalk = item.chalkProduct { parts.append("巧克 \(GearDisplayRules.productTitle(chalk))") }
        return parts.joined(separator: " · ")
    }

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        VStack(alignment: .leading, spacing: 0) {
            GearImageView(source: GearImageSource(item: item))
                .frame(maxWidth: .infinity)
                .frame(height: 170)
                .overlay(alignment: .topLeading) {
                    if retired {
                        Text("已退役")
                            .font(.system(size: 10))
                            .foregroundStyle(palette.textSecondary)
                            .padding(.horizontal, 7)
                            .padding(.vertical, 2)
                            .background(palette.card, in: Capsule())
                            .overlay { Capsule().stroke(palette.border) }
                            .padding(8)
                    }
                }

            VStack(alignment: .leading, spacing: 3) {
                Text(GearDisplayRules.itemTitle(item))
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .fixedSize(horizontal: false, vertical: true)
                if !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                }
                if !badges.isEmpty {
                    WrappingLayout(spacing: 4) {
                        ForEach(badges, id: \.self) { badge in
                            Text(badge)
                                .font(.system(size: 10))
                                .foregroundStyle(palette.brandStrong)
                                .padding(.horizontal, 7)
                                .padding(.vertical, 2)
                                .background(palette.brandTint, in: Capsule())
                        }
                    }
                    .padding(.top, 6)
                }
                if !partsLine.isEmpty {
                    Text(partsLine)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textSecondary)
                        .padding(.top, 6)
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 10)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(palette.card)
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .overlay { RoundedRectangle(cornerRadius: 12).stroke(palette.border) }
        .opacity(retired ? 0.72 : 1)
        .accessibilityElement(children: .combine)
    }
}
